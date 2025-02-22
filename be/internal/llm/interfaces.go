package llm

import (
	"context"
)

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
	Type    string `json:"type"`
	Id      string `json:"id"`
}

type CompletionRequest struct {
	Messages []Message
	Model    string
}

type StreamChunk struct {
	Content string
	Done    bool
}

type AIProvider interface {
	Complete(ctx context.Context, req CompletionRequest) (Message, error)
	StreamComplete(ctx context.Context, req CompletionRequest) (<-chan StreamChunk, error)
}
