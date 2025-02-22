package llm

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/generative-ai-go/genai"
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
