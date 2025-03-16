package llm

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/generative-ai-go/genai"
	"github.com/invopop/jsonschema"
	"google.golang.org/api/iterator"
	"log"
)

type GeminiProvider struct {
	client *genai.Client
}

func NewGeminiAIProvider(client *genai.Client) *GeminiProvider {
	return &GeminiProvider{client: client}
}

func (p *GeminiProvider) Complete(ctx context.Context, req CompletionRequest) (Message, error) {
	model := p.client.GenerativeModel("gemini-2.0-pro-exp-02-05")
	if req.ResponseFormat != nil {
		model.ResponseMIMEType = toResponseMIMEType(req.ResponseFormat.Type)
		model.ResponseSchema = convertJsonSchemaToGeminiSchema(req.ResponseFormat.Schema)
	}
	chatStream := model.StartChat()
	res, err := chatStream.SendMessage(ctx, p.extractParts(req.Messages)...)
	if err != nil {
		return Message{}, err
	}

	return Message{
		Content: fmt.Sprintf("%v", res.Candidates[0].Content.Parts[0]),
	}, nil
}

func (p *GeminiProvider) StreamComplete(ctx context.Context, req CompletionRequest) (<-chan StreamChunk, error) {
	model := p.client.GenerativeModel("gemini-2.0-pro-exp-02-05")
	chatStream := model.StartChat()
	resIterator := chatStream.SendMessageStream(ctx, p.extractParts(req.Messages)...)

	chunks := make(chan StreamChunk)

	go func() {
		defer close(chunks)

		for {
			resp, err := resIterator.Next()
			if errors.Is(err, iterator.Done) {
				chunks <- StreamChunk{Done: true}
				return
			}
			if err != nil {
				log.Printf("Error in StreamComplete: %v", err)
				return
			}

			chunks <- StreamChunk{
				Content: fmt.Sprintf("%v", resp.Candidates[0].Content.Parts[0]),
			}
		}
	}()

	return chunks, nil
}

// -----------------Private Helper Functions-----------------
func (p *GeminiProvider) extractParts(messages []Message) []genai.Part {
	var parts []genai.Part
	for _, msg := range messages {
		parts = append(parts, genai.Text(msg.Content))
	}
	return parts
}

func toResponseMIMEType(formatType ResponseFormatType) string {
	switch formatType {
	case ResponseFormatTypeText:
		return "text/plain"
	case ResponseFormatTypeJson:
		return "application/json"
	default:
		return "text/plain"
	}
}

func convertJsonSchemaToGeminiSchema(schema *jsonschema.Schema) *genai.Schema {
	if schema == nil {
		return nil
	}

	result := &genai.Schema{
		Description: schema.Description,
		Format:      schema.Format,
		Nullable:    false, // Default value
	}

	// Convert type
	result.Type = convertSchemaType(schema.Type)

	// Convert enum if present
	if schema.Enum != nil {
		result.Enum = convertEnumToStringSlice(schema.Enum)
	}

	// Convert Items for array type
	if schema.Items != nil {
		result.Items = convertJsonSchemaToGeminiSchema(schema.Items)
	}

	// Convert Properties for object type
	if schema.Properties != nil {
		result.Properties = make(map[string]*genai.Schema)
		for pair := schema.Properties.Oldest(); pair != nil; pair = pair.Next() {
			result.Properties[pair.Key] = convertJsonSchemaToGeminiSchema(pair.Value)
		}
	}

	// Copy required properties
	if schema.Required != nil {
		result.Required = append([]string{}, schema.Required...)
	}

	return result
}

// convertSchemaType converts jsonschema type string to genai.Type
func convertSchemaType(schemaType string) genai.Type {
	switch schemaType {
	case "string":
		return genai.TypeString
	case "number":
		return genai.TypeNumber
	case "integer":
		return genai.TypeInteger
	case "boolean":
		return genai.TypeBoolean
	case "array":
		return genai.TypeArray
	case "object":
		return genai.TypeObject
	default:
		return genai.TypeUnspecified
	}
}

// convertEnumToStringSlice converts []any to []string
func convertEnumToStringSlice(enumValues []any) []string {
	result := make([]string, 0, len(enumValues))
	for _, v := range enumValues {
		if str, ok := v.(string); ok {
			result = append(result, str)
		} else {
			// Convert non-string values to string representation
			result = append(result, fmt.Sprintf("%v", v))
		}
	}
	return result
}
