package llm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/invopop/jsonschema"
	"github.com/sashabaranov/go-openai"
	"io"
)

type OpenAIProvider struct {
	client *openai.Client
}

func NewOpenAIProvider(client *openai.Client) *OpenAIProvider {
	return &OpenAIProvider{client: client}
}

func (p *OpenAIProvider) Complete(ctx context.Context, req CompletionRequest) (Message, error) {
	// Create the request object
	if req.Model == "" {
		req.Model = openai.GPT4oMini20240718
	}

	chatRequest := openai.ChatCompletionRequest{
		Model:          req.Model,
		Messages:       toOpenAIMessages(req.Messages),
		ResponseFormat: toOpenAIResponseFormat(req.ResponseFormat),
		Tools:          toOpenAITools(req.Tools),
	}

	if req.FunctionCallingMode != "" {
		chatRequest.ToolChoice = req.FunctionCallingMode
	}

	// Log request for debugging
	requestJSON, _ := json.MarshalIndent(chatRequest, "", "  ")
	fmt.Printf("OpenAI Request Body: %s\n", requestJSON)

	res, err := p.client.CreateChatCompletion(ctx, chatRequest)

	// Log response for debugging
	responseJSON, _ := json.MarshalIndent(res, "", "  ")
	fmt.Printf("OpenAI Response Body: %s\n", responseJSON)

	if err != nil {
		return Message{}, err
	}
	if len(res.Choices) == 0 {
		return Message{}, errors.New("no choices found")
	}
	return fromOpenAIMessage(res.Choices[0].Message), err
}

func (p *OpenAIProvider) StreamComplete(ctx context.Context, req CompletionRequest) (<-chan StreamChunk, error) {
	// Create the request object
	chatRequest := openai.ChatCompletionRequest{
		Model:          req.Model,
		Messages:       toOpenAIMessages(req.Messages),
		ResponseFormat: toOpenAIResponseFormat(req.ResponseFormat),
		Tools:          toOpenAITools(req.Tools),
	}

	// Marshal the request to JSON for debugging
	requestJSON, _ := json.MarshalIndent(chatRequest, "", "  ")
	fmt.Printf("OpenAI Request Body: %s\n", requestJSON)

	stream, err := p.client.CreateChatCompletionStream(ctx, chatRequest)
	if err != nil {
		return nil, err
	}

	chunks := make(chan StreamChunk)
	go func() {
		defer close(chunks)
		defer stream.Close()

		for {
			response, err := stream.Recv()
			if errors.Is(err, io.EOF) {
				chunks <- StreamChunk{Done: true}
				return
			}
			if err != nil {
				// Consider error handling channel
				return
			}

			if len(response.Choices) > 0 {
				chunks <- StreamChunk{
					Content: response.Choices[0].Delta.Content,
				}
			}
		}
	}()

	return chunks, nil
}

// ------------------Private helper function------------------

func toOpenAIMessage(msg Message) openai.ChatCompletionMessage {
	return openai.ChatCompletionMessage{
		Role:       msg.Role,
		Content:    msg.Content,
		ToolCalls:  toOpenAIToolCalls(msg.ToolCalls),
		ToolCallID: msg.ToolCallId,
	}
}

func fromOpenAIMessage(msg openai.ChatCompletionMessage) Message {
	return Message{
		Role:       msg.Role,
		Content:    msg.Content,
		ToolCalls:  toToolCall(msg.ToolCalls),
		ToolCallId: msg.ToolCallID,
	}
}

func toToolCall(toolCalls []openai.ToolCall) []ToolCall {
	result := make([]ToolCall, len(toolCalls))
	for i, toolCall := range toolCalls {
		result[i] = ToolCall{
			ID:       toolCall.ID,
			Type:     ToolType(toolCall.Type),
			Function: &FunctionCall{Name: toolCall.Function.Name, Arguments: toolCall.Function.Arguments},
		}
	}
	return result
}

func toOpenAIMessages(messages []Message) []openai.ChatCompletionMessage {
	result := make([]openai.ChatCompletionMessage, len(messages))
	for i, msg := range messages {
		result[i] = toOpenAIMessage(msg)
	}
	return result
}

func toOpenAITools(tools []Tool) []openai.Tool {
	result := make([]openai.Tool, len(tools))
	for i, tool := range tools {
		result[i] = openai.Tool{
			Type:     convertToolType(tool.Type),
			Function: toOpenAIFunction(tool.Function),
		}
	}
	return result
}

func toOpenAIToolCalls(toolCalls []ToolCall) []openai.ToolCall {
	result := make([]openai.ToolCall, len(toolCalls))
	for i, toolCall := range toolCalls {
		result[i] = openai.ToolCall{
			ID:       toolCall.ID,
			Type:     convertToolType(toolCall.Type),
			Function: openai.FunctionCall{Name: toolCall.Function.Name, Arguments: toolCall.Function.Arguments},
		}
	}
	return result

}

func toOpenAIFunction(function *FuncDefinition) *openai.FunctionDefinition {
	return &openai.FunctionDefinition{
		Name:        function.Name,
		Description: function.Description,
		Strict:      false,
		Parameters:  function.Parameters,
	}
}

// convertToolType converts a string tool type to openai.ToolType
func convertToolType(toolType ToolType) openai.ToolType {
	if toolType == ToolTypeFunction {
		return openai.ToolTypeFunction
	}
	// Default or handle other tool types if needed
	return openai.ToolTypeFunction
}

func toOpenAIResponseFormat(format *ResponseFormat) *openai.ChatCompletionResponseFormat {
	if format == nil {
		return nil
	}
	switch format.Type {
	case ResponseFormatTypeText:
		return &openai.ChatCompletionResponseFormat{
			Type: openai.ChatCompletionResponseFormatTypeText,
		}
	case ResponseFormatTypeJson:
		return &openai.ChatCompletionResponseFormat{
			Type:       openai.ChatCompletionResponseFormatTypeJSONSchema,
			JSONSchema: convertJsonSchemaToOpenAISchema(format.Schema, format.Name),
		}
	default:
		return nil
	}
}

// SchemaMarshaler is a wrapper for jsonschema.Schema that implements json.Marshaler
type SchemaMarshaler struct {
	Schema *jsonschema.Schema `json:"schema"`
}

// NewSchemaMarshaler creates a new SchemaMarshaler that wraps a jsonschema.Schema
func NewSchemaMarshaler(schema *jsonschema.Schema) *SchemaMarshaler {
	return &SchemaMarshaler{Schema: schema}
}

// MarshalJSON implements the json.Marshaler interface
func (m *SchemaMarshaler) MarshalJSON() ([]byte, error) {
	if m.Schema == nil {
		return []byte("{}"), nil // Return empty JSON object instead of null
	}
	return json.Marshal(m.Schema)
}
func convertJsonSchemaToOpenAISchema(schema *jsonschema.Schema, name string) *openai.ChatCompletionResponseFormatJSONSchema {
	if schema == nil {
		return nil
	}

	return &openai.ChatCompletionResponseFormatJSONSchema{
		Name:        name,
		Description: schema.Description,
		Schema:      NewSchemaMarshaler(schema),
		Strict:      true, // Default to strict validation
	}
}
