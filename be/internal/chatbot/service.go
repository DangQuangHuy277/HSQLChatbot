package chatbot

import (
	"HNLP/be/internal/course"
	"HNLP/be/internal/db"
	"HNLP/be/internal/llm"
	"HNLP/be/internal/search"
	"context"
	"encoding/json"
	"fmt"
	"github.com/invopop/jsonschema"
	"github.com/sashabaranov/go-openai"
	"io"
	"log"
	"net/http"
	"strings"
)

const (
	ContextPrompt = `
You translate user requests question into SQL queries for a university relational database with tables like student, professor, course, course_class, etc.
Rules:
- Generate correct, executable SQL queries.
- Respect user role permissions (details below).
- Avoid harmful operations (e.g., DROP, DELETE) unless explicitly allowed.
- Return only the SQL query, no extra text or formatting, not start with sql so that I can use this directly to run.
Schema:
`
)

// IChatService defines the interface for chat services.
type IChatService interface {
}

// ChatService implements the IChatService interface.
type ChatService struct {
	aiProvider llm.AIProvider
	db         *db.HDb
	searchSrv  search.Service
	courseSrv  course.Service
	//funcRegistry llm.FuncRegistry
}

// NewChatService creates a new instance of ChatService.
func NewChatService(aiProvider llm.AIProvider, db *db.HDb, searchSrv search.Service, courseSrv course.Service) *ChatService {
	service := ChatService{
		aiProvider: aiProvider,
		db:         db,
		searchSrv:  searchSrv,
		courseSrv:  courseSrv,
		//funcRegistry: funcRegistry,
	}
	return &service
}

// StreamChatResponse streams responses from the chat service using Server-Sent Events (SSE).
// It performs the following steps:
// 1. Retrieves the database DDL and constructs a system context prompt.
// 2. Appends each user message to the context for the AI provider.
// 3. Requests an SQL query based on the complete prompt.
// 4. Executes the generated SQL query and logs the result.
// 5. Converts the SQL query result to a JSON string.
// 6. Requests a natural language answer based on the original query and its result.
// 7. Streams the natural language answer back to the client using SSE.
//
// Parameters:
//
//	ctx  - The request context, used for cancellation and deadlines.
//	req  - The chat request containing user messages and session details.
//	w    - The writer to stream SSE messages to the client.
//	role - The role of the user making the request.
//
// Returns:
//
//	error - An error if any step in processing the request fails.
func (cs *ChatService) StreamChatResponse(ctx context.Context, req ChatRequest, w io.Writer, userId int, role string) error {
	// Step 1: Define role-specific prompt
	rolePrompt := ""
	switch role {
	case "student":
		rolePrompt = fmt.Sprintf("The user is a student with ID %d, so your query should only access data for their personal info, courses, administrative class and corresponding advisor, grades, etc. The filter should include something like 'student.id = %d' or 'student_course_class.student_id = %d' to specify only data related to them, except data about program, course, and course_class is general.", userId, userId, userId)
	case "professor":
		rolePrompt = fmt.Sprintf("The user is a professor with ID %d, so your query should only access data for their personal info, courses they teach, corresponding students and grades, schedules, etc. The filter should include something like 'course_class.professor_id = %d' or 'course_schedule_instructor.professor_id = %d' to specify only data related to them, except data about program, course, and student is general.", userId, userId, userId)
	case "admin":
		rolePrompt = fmt.Sprintf("The user is an admin with ID %d, so your query can access all data including students, professors, courses, grades, schedules, etc. No specific filter like 'id = %d' is required unless requested, as all data about program, course, course_class, etc., is accessible.", userId, userId)
	default:
		return fmt.Errorf("invalid role: %s", role)
	}

	// Step 2: Prepare LLM messages
	dbDDL, err := cs.db.LoadDDL()
	if err != nil {
		log.Fatalf("Failed to load DDL from database: %v", err)
	}
	systemContextPrompt := ContextPrompt + dbDDL + "\n\n" + rolePrompt + "Here is the user question: \n" + req.Messages[len(req.Messages)-1].Content
	var llmMessages []llm.Message

	llmMessages = append(llmMessages, llm.Message{
		Role:    openai.ChatMessageRoleUser,
		Content: systemContextPrompt,
	})
	getSQLQueryRequest := llm.CompletionRequest{
		Messages: llmMessages,           // Use a slice literal
		Model:    openai.O1Mini20240912, // TODO: Fix the miss match and hardcode between OpenAI and Gemini
	}

	// Step 3: Get and validate SQL query
	sqlQuery, err := cs.aiProvider.Complete(ctx, getSQLQueryRequest)
	if err != nil {
		log.Printf("Failed to get response from AI provider: %v", err)
	}
	if strings.HasPrefix(sqlQuery.Content, "```sql") {
		sqlQuery.Content = strings.TrimPrefix(sqlQuery.Content, "```sql")
	}
	sqlQuery.Content = strings.ReplaceAll(sqlQuery.Content, "`", "")
	sqlQuery.Content = strings.TrimSpace(sqlQuery.Content)
	log.Printf("SQL Query: %s", sqlQuery.Content)

	// Step 4: Execute the SQL query
	queryResult, err := cs.db.ExecuteQuery(ctx, sqlQuery.Content)
	if err != nil {
		log.Printf("Failed to execute query: %v", err)
	}
	log.Printf("Result: %v", queryResult)
	// TODO: Write the queryResult to the frontend, (maybe will need to use normal JSON response instead of SSE or a advanced way in SSE)

	// Step 5: Convert result to JSON
	queryResultJSON, err := convertQueryResultToJSONString(queryResult)
	if err != nil {
		log.Printf("Failed to convert query queryResult to JSON: %v", err)
	}

	// Step 6: Get natural language answer
	getNaturalAnswerRequest := llm.CompletionRequest{
		Messages: []llm.Message{
			{
				Role: "user",
				Content: fmt.Sprintf(`You are a chatbot for the university database. Your task is to provide a natural language answer for the user question based on the database query result (in JSON format).

**The User question:** %s

**The Database Result:** %v`, req.Messages[len(req.Messages)-1].Content, queryResultJSON),
			},
		},
		Model: openai.O1Mini20240912,
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

func (cs *ChatService) SearchResources(ctx context.Context, query string, limit int, w io.Writer) error {
	//Step1 :extract courseKeywords from the user query using the AI provider
	//courseKeywords, err := cs.extractKeywords(ctx, query)
	//if err != nil {
	//	w.Write([]byte("data: [DONE]\n\n"))
	//	return err
	//}
	//
	//// Validation check
	//if !courseKeywords.IsValid {
	//	w.Write([]byte("data: [DONE]\n\n"))
	//	return fmt.Errorf("invalid query: %s", query)
	//}
	//
	//// Step 2: Get course info from course service
	//courseResp, _ := cs.courseSrv.GetCourse(course.GetCourseRequest{
	//	Code: courseKeywords.CourseCode,
	//	Name: courseKeywords.CourseName,
	//})
	//
	//// If we have course info, then add this to the search query
	//keywords := courseKeywords.Keywords
	//if courseResp != nil {
	//	keywords = append(keywords, courseResp.EnglishName)
	//}

	keywords := []string{"machine learning", "AI"}

	// Step 2 get resources from search service
	resources, err := cs.searchSrv.Search(ctx, keywords)
	if err != nil {
		return err
	}

	// Apply limit
	if limit > 0 && len(resources) > limit {
		resources = resources[:limit]
	}

	// Convert resources to JSON for AI context
	resourcesJSON, err := json.MarshalIndent(resources, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal resources: %w", err)
	}

	// Create prompt for natural language response
	naturalLangRequest := llm.CompletionRequest{
		Messages: []llm.Message{
			{
				Role: openai.ChatMessageRoleUser,
				Content: fmt.Sprintf(`You are a learning resource assistant.You are provided some searched resource from internet.
You need to recommend resource for student.

Student search query: "%s"

Searched Resource results (JSON format): %s

Please provide a natural language response.`,
					query, string(resourcesJSON)),
			},
		},
		Model: openai.GPT4oMini20240718,
	}
	// Stream the response
	chunks, err := cs.aiProvider.StreamComplete(ctx, naturalLangRequest)
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

func (cs *ChatService) extractKeywords(ctx context.Context, query string) (CourseKeywords, error) {

	keywordRequest := llm.CompletionRequest{
		Messages: []llm.Message{
			{
				Role: openai.ChatMessageRoleUser,
				Content: `Extract relevant keywords from the user's input for searching educational resources (books, lectures, papers, articles, etc.). Return a JSON object with:
A query is valid if it mentions a course (code/name) or study-related terms (e.g., "articles", "lectures", "programming"). Invalid if unrelated (e.g., "cat videos").

Examples:
- "Find articles for CS101" → {"course_code": "CS101", "course_name": null, "keywords": ["articles"], "is_valid": true}
- "Resources about machine learning for AI" → {"course_code": null, "course_name": "AI", "keywords": ["machine learning", "resources"], "is_valid": true}
- "Show funny cat videos" → {"course_code": null, "course_name": null, "keywords": ["cat videos", "funny"], "is_valid": false}

Input: ` + query,
			},
		},
		Model: openai.GPT4oMini20240718,
		ResponseFormat: &llm.ResponseFormat{
			Type:   llm.ResponseFormatTypeJson,
			Schema: jsonschema.Reflect(&CourseKeywords{}).Definitions["CourseKeywords"],
			Name:   "CourseKeywords",
		},
	}

	response, err := cs.aiProvider.Complete(ctx, keywordRequest)
	if err != nil {
		return CourseKeywords{}, fmt.Errorf("failed to extract keywords: %v", err)
	}

	var result CourseKeywords
	if err := json.Unmarshal([]byte(response.Content), &result); err != nil {
		return CourseKeywords{}, fmt.Errorf("failed to parse LLM response: %v", err)
	}

	return result, nil
}

//------------------Private helper functions------------------

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
