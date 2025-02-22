package llm

import (
	"context"
	"errors"
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
	res, err := p.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:    req.Model,
		Messages: toOpenAIMessages(req.Messages),
	})
	if err != nil {
		return Message{}, err
	}
	if len(res.Choices) == 0 {
		return Message{}, errors.New("no choices found")
	}
	return fromOpenAIMessage(res.Choices[0].Message), err
}

func (p *OpenAIProvider) StreamComplete(ctx context.Context, req CompletionRequest) (<-chan StreamChunk, error) {
	stream, err := p.client.CreateChatCompletionStream(ctx, openai.ChatCompletionRequest{
		Model:    req.Model,
		Messages: toOpenAIMessages(req.Messages),
	})
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
		Role:    msg.Role,
		Content: msg.Content,
	}
}

func fromOpenAIMessage(msg openai.ChatCompletionMessage) Message {
	return Message{
		Role:    msg.Role,
		Content: msg.Content,
	}
}

func toOpenAIMessages(messages []Message) []openai.ChatCompletionMessage {
	result := make([]openai.ChatCompletionMessage, len(messages))
	for i, msg := range messages {
		result[i] = toOpenAIMessage(msg)
	}
	return result
}
