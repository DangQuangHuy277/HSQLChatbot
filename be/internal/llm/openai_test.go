package llm

import (
	"context"
	"github.com/invopop/jsonschema"
	"github.com/joho/godotenv"
	"github.com/sashabaranov/go-openai"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wk8/go-ordered-map/v2"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestToOpenAITools(t *testing.T) {
	// Create a simple JSON schema for testing
	schema := &jsonschema.Schema{
		Type:        "object",
		Description: "User data",
		Properties:  orderedmap.New[string, *jsonschema.Schema](),
		Required:    []string{"name"},
	}

	// Add properties to the ordered map
	schema.Properties.Set("name", &jsonschema.Schema{
		Type:        "string",
		Description: "The user's name",
	})
	schema.Properties.Set("age", &jsonschema.Schema{
		Type:        "integer",
		Description: "The user's age",
	})

	tools := []Tool{
		{
			Type: "function",
			Function: &FuncDefinition{
				Name:        "getUserData",
				Description: "Get user data from the database",
				//Strict:      true,
				Parameters: schema,
			},
		},
	}

	openAITools := toOpenAITools(tools)

	// Assertions
	require.Len(t, openAITools, 1)
	assert.Equal(t, openai.ToolTypeFunction, openAITools[0].Type)
	assert.Equal(t, "getUserData", openAITools[0].Function.Name)
	assert.Equal(t, "Get user data from the database", openAITools[0].Function.Description)
	assert.True(t, openAITools[0].Function.Strict)
	assert.Equal(t, schema, openAITools[0].Function.Parameters)
}

func TestOpenAIProvider_Complete_WithTools(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return a successful response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"choices": [
				{
					"message": {
						"role": "assistant",
						"content": "I'll help with that function call"
					},
					"finish_reason": "stop",
					"index": 0
				}
			],
			"id": "test-id",
			"object": "chat.completion",
			"created": 1677825464,
			"model": "gpt-3.5-turbo"
		}`))
	}))
	defer server.Close()

	// Create client with mock server URL
	config := openai.DefaultConfig("test-token")
	config.BaseURL = server.URL
	client := openai.NewClientWithConfig(config)

	provider := NewOpenAIProvider(client)

	// Create a simple JSON schema for the function
	schema := &jsonschema.Schema{
		Type:       "object",
		Properties: orderedmap.New[string, *jsonschema.Schema](),
	}
	schema.Properties.Set("query", &jsonschema.Schema{
		Type: "string",
	})

	// Create completion request with tools
	req := CompletionRequest{
		Model: "gpt-3.5-turbo",
		Messages: []Message{
			{Role: "user", Content: "Search for information about Go programming"},
		},
		Tools: []Tool{
			{
				Type: "function",
				Function: &FuncDefinition{
					Name:        "search",
					Description: "Search for information",
					Parameters:  schema,
				},
			},
		},
	}

	// Execute the request
	response, err := provider.Complete(context.Background(), req)

	// Assertions
	require.NoError(t, err)
	assert.Equal(t, "assistant", response.Role)
	assert.Equal(t, "I'll help with that function call", response.Content)
}

func TestOpenAIProvider_StreamComplete_WithTools(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return SSE format for streaming
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		// Write two chunks and EOF
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"I'll \"}}]}\n\n"))
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"help with that function call\"}}]}\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer server.Close()

	// Create client with mock server URL
	config := openai.DefaultConfig("test-token")
	config.BaseURL = server.URL
	client := openai.NewClientWithConfig(config)

	provider := NewOpenAIProvider(client)

	// Create a simple JSON schema for the function
	schema := &jsonschema.Schema{
		Type:       "object",
		Properties: orderedmap.New[string, *jsonschema.Schema](),
	}
	schema.Properties.Set("query", &jsonschema.Schema{
		Type: "string",
	})

	// Create completion request with tools
	req := CompletionRequest{
		Model: "gpt-3.5-turbo",
		Messages: []Message{
			{Role: "user", Content: "Search for information about Go programming"},
		},
		Tools: []Tool{
			{
				Type: "function",
				Function: &FuncDefinition{
					Name:        "search",
					Description: "Search for information",
					Parameters:  schema,
				},
			},
		},
	}

	// Execute the streaming request
	stream, err := provider.StreamComplete(context.Background(), req)
	require.NoError(t, err)

	// Collect all chunks
	var chunks []string
	var isDone bool

	for chunk := range stream {
		if chunk.Done {
			isDone = true
			break
		}
		chunks = append(chunks, chunk.Content)
	}

	// Assertions
	assert.True(t, isDone)
	assert.Equal(t, []string{"I'll ", "help with that function call"}, chunks)
}

func TestOpenAIProvider_RealAPI_Integration(t *testing.T) {
	err := godotenv.Load("/home/huy/Code/Personal/KLTN/be/config/.env")
	if err != nil {
		t.Skipf("Error loading .env file: %v", err)
		return
	}
	if testing.Short() || os.Getenv("OPENAI_API_KEY") == "" {
		t.Skip("Skipping integration test with real OpenAI API")
	}

	//Real API test code here
	client := openai.NewClient(os.Getenv("OPENAI_API_KEY"))

	provider := NewOpenAIProvider(client)
	// Create schema with additionalProperties set to false
	schema := &jsonschema.Schema{
		Type:                 "object",
		Properties:           orderedmap.New[string, *jsonschema.Schema](),
		Required:             []string{"country"},
		AdditionalProperties: jsonschema.FalseSchema,
	}

	// Add country property
	schema.Properties.Set("country", &jsonschema.Schema{
		Type:        "string",
		Description: "The name of the country",
	})

	req := CompletionRequest{
		Model: openai.GPT4oMini20240718,
		Messages: []Message{
			{Role: "user", Content: "What is the capital of France?"},
		},
		Tools: []Tool{
			{
				Type: "function",
				Function: &FuncDefinition{
					Name:        "getCapital",
					Description: "Get the capital of a country",
					Parameters:  schema,
				},
			},
		},
		FunctionCallingMode: Required,
	}
	// Test with actual API calls
	response, err := provider.Complete(context.Background(), req)

	if err != nil {
		t.Fatalf("Error calling OpenAI API: %v", err)
	}

	println("Response from OpenAI API:", response.Content)

}
