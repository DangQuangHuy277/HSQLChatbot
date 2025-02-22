package chatbot

import (
	"HNLP/be/internal/db"
	"HNLP/be/internal/llm"
	"context"
	"encoding/json"
	"fmt"
	"github.com/sashabaranov/go-openai"
	"io"
	"log"
	"net/http"
)

const (
	CONTEXT_PROMT = `
	You act like a natural language processing to SQL query translator for a university database. I will provide you some requirements of user and you will provide me the SQL query that I can bring this query to query in database.
	Note that return the query so that I can pass directly to the database (Without triple backtick).
	Here is context of the database:
`
)

// IChatService defines the interface for chat services.
type IChatService interface {
	QueryByChat(*QueryRequest) error
}

// ChatService implements the IChatService interface.
type ChatService struct {
	aiProvider llm.AIProvider
	db         *db.HDb
}

// NewChatService creates a new instance of ChatService.
func NewChatService(aiProvider llm.AIProvider, db *db.HDb) *ChatService {
	return &ChatService{aiProvider: aiProvider, db: db}
}

func (cs *ChatService) StreamChatResponse(ctx context.Context, req ChatRequest, w io.Writer) error {
	// Step 1: Get SQL Query from LLM
	dbDDL, err := cs.db.LoadDDL()
	if err != nil {
		log.Fatalf("Failed to load DDL from database: %v", err)
	}
	systemContextPrompt := CONTEXT_PROMT + dbDDL
	var llmMessages []llm.Message

	llmMessages = append(llmMessages, llm.Message{
		Role:    openai.ChatMessageRoleSystem,
		Content: systemContextPrompt,
	})
	for _, msg := range req.Messages {

		llmMessages = append(llmMessages, llm.Message{
			Role:    openai.ChatMessageRoleUser, // Hardcoded for now
			Content: msg.Content,
			Type:    msg.Type,
			Id:      msg.Id,
		})
	}

	getSQLQueryRequest := llm.CompletionRequest{
		Messages: llmMessages,              // Use a slice literal
		Model:    openai.GPT4oMini20240718, // TODO: Fix the miss match and hardcode between OpenAI and Gemini
	}

	sqlQuery, err := cs.aiProvider.Complete(ctx, getSQLQueryRequest)
	if err != nil {
		log.Printf("Failed to get response from AI provider: %v", err)
	}
	log.Printf("SQL Query: %s", sqlQuery.Content)

	// Step 2: Execute the SQL Query and get the Result
	queryResult, err := cs.db.ExecuteAndFormatQuery(sqlQuery.Content)
	if err != nil {
		log.Printf("Failed to execute query: %v", err)
	}
	log.Printf("Result: %v", queryResult)
	// TODO: Write the queryResult to the frontend, (maybe will need to use normal JSON response instead of SSE or a advanced way in SSE)

	// Step 3: Get natural language answer from question and query queryResult by LLM
	queryResultJSON, err := convertQueryResultToJSONString(queryResult)
	if err != nil {
		log.Printf("Failed to convert query queryResult to JSON: %v", err)
	}

	getNaturalAnswerRequest := llm.CompletionRequest{
		Messages: []llm.Message{
			{
				Role: "system",
				Content: `You are chatbot for the university database. 
Your task is to provide a natural language answer for the user question base on the database query queryResult (at JSON format).`,
			},
			{
				Role: "user",
				Content: fmt.Sprintf(`**The User question:** %s

**The Database Result:** %v`, req.Messages[len(req.Messages)-1].Content, queryResultJSON), // Hardcoded for now, need to sure that this is question
			},
		},
		Model: openai.GPT4oMini20240718,
	}
	chunks, err := cs.aiProvider.StreamComplete(ctx, getNaturalAnswerRequest)
	if err != nil {
		return err
	}

	for chunk := range chunks {
		if chunk.Done {
			fmt.Fprintf(w, "data: [DONE]\n\n")
			return nil
		}

		// Format SSE response
		resp := StreamResponse{
			Choices: []Choice{{Delta: Delta{Content: chunk.Content}}},
		}

		if err := writeSSEResponse(w, resp); err != nil {
			return err
		}
	}
	return nil
}

func convertQueryResultToJSONString(queryResult *db.QueryResult) (string, error) {
	jsonBytes, err := json.MarshalIndent(queryResult, "", "  ") // Use MarshalIndent for pretty JSON
	if err != nil {
		return "", err
	}
	return string(jsonBytes), nil
}

func writeSSEResponse(w io.Writer, resp StreamResponse) error {
	// Marshal the response to JSON
	jsonData, err := json.Marshal(resp)
	if err != nil {
		return fmt.Errorf("failed to marshal SSE response: %w", err)
	}

	// Write the SSE formatted message
	if _, err := fmt.Fprintf(w, "data: %s\n\n", jsonData); err != nil {
		return fmt.Errorf("failed to write SSE message: %w", err)
	}

	// If the writer supports flushing (like http.ResponseWriter), flush it
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}

	return nil
}
