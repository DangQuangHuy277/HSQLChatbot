package llm

import (
	"context"
	"github.com/invopop/jsonschema"
)

type ResponseFormatType string
type FunctionCallingMode string

type ToolType string

const (
	// ResponseFormatTypeText is a text response format
	ResponseFormatTypeText ResponseFormatType = "text"
	// ResponseFormatTypeJson is a json response format
	ResponseFormatTypeJson ResponseFormatType = "json"

	Auto     FunctionCallingMode = "auto"
	Required FunctionCallingMode = "required"
	None     FunctionCallingMode = "none"

	// ToolTypeFunction is a function tool type
	ToolTypeFunction ToolType = "function"
)

// Message -related structs
type Message struct {
	Role     string     `json:"role"`
	Content  string     `json:"content"`
	ToolCall []ToolCall `json:"tool_call,omitempty"`
	Id       string     `json:"id"`
}

// ToolCall represents an actual tool invocation by the model
type ToolCall struct {
	ID       string        `json:"id"`
	Type     ToolType      `json:"type"`
	Function *FunctionCall `json:"function,omitempty"`
}

// FunctionCall represents the function calling information
type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON string containing the function arguments
}

// Tool -related structs
type Tool struct {
	Type     ToolType        `json:"type"`
	Function *FuncDefinition `json:"function,omitempty"`
}

// ResponseFormat  structs
type ResponseFormat struct {
	Type   ResponseFormatType `json:"type"`
	Schema *jsonschema.Schema `json:"schema,omitempty"`
	Name   string             `json:"name"`
}

type CompletionRequest struct {
	Messages            []Message
	Model               string
	ResponseFormat      *ResponseFormat
	Tools               []Tool
	FunctionCallingMode FunctionCallingMode
}

type StreamChunk struct {
	Content string
	Done    bool
}

type AIProvider interface {
	Complete(ctx context.Context, req CompletionRequest) (Message, error)
	StreamComplete(ctx context.Context, req CompletionRequest) (<-chan StreamChunk, error)
}
