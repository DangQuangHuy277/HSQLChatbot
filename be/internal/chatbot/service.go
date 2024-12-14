package chatbot

import (
	"HNLP/internal/db"
	"context"
	"database/sql"
	"fmt"
	"github.com/sashabaranov/go-openai"
	"log"
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
	client *openai.Client
	db     *db.HDb
}

// NewChatService creates a new instance of ChatService.
func NewChatService(client *openai.Client, db *db.HDb) *ChatService {
	return &ChatService{client: client, db: db}
}

// QueryByChat handles the chat query logic.
func (cs *ChatService) QueryByChat(request *QueryRequest) (*QueryResponse, error) {
	dbDDL, err := cs.db.LoadDDL()
	if err != nil {
		log.Fatalf("Failed to load DDL from database: %v", err)
	}

	systemContextPromt := CONTEXT_PROMT + dbDDL

	// Create a request to OpenAI
	openAIRequest := openai.ChatCompletionRequest{
		Model: openai.GPT3Dot5Turbo1106,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: systemContextPromt},
			{Role: openai.ChatMessageRoleUser, Content: request.Message},
		},
	}

	// Get the response from OpenAI
	response, err := cs.client.CreateChatCompletion(context.Background(), openAIRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to get response from OpenAI: %v", err)
	}

	query := response.Choices[0].Message.Content
	log.Printf("Query: %s", query)

	// Process read query response
	result := &QueryResponse{}
	rows, err := cs.db.Query(query)
	if err != nil {
		log.Printf("Failed to execute query: %v", err)
		return nil, fmt.Errorf("failed to execute query: %v", err)
	}
	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {
			log.Printf("Failed to close rows: %v", err)
		}
	}(rows)
	cols, _ := rows.Columns()
	result.Columns = cols
	for rows.Next() {
		data := make(map[string]interface{})
		values := make([]interface{}, len(cols))
		for i := range values {
			values[i] = new(interface{})
		}
		if err := rows.Scan(values...); err != nil {
			return nil, fmt.Errorf("failed to scan row: %v", err)
		}
		for i, col := range cols {
			if values[i] != nil {
				data[col] = *(values[i].(*interface{}))
			}
		}
		result.Rows = append(result.Rows, data)
	}
	return result, nil
}

func (cs *ChatService) Communicate() {

}
