package chatbot

// QueryRequest TODO: Remove this
type QueryRequest struct {
	Message string `json:"message"`
}

// QueryResponse TODO: Remove this
type QueryResponse struct {
	Columns []string                 `json:"columns"`
	Rows    []map[string]interface{} `json:"rows"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
	Type    string `json:"type"`
	Id      string `json:"id"`
}

type ChatRequest struct {
	Messages  []Message `json:"messages"`
	SessionID string    `json:"session_id"`
	Model     string    `json:"model,omitempty"`  // un-support yet
	Stream    bool      `json:"stream,omitempty"` // un-support yet
}

type StreamResponse struct {
	Choices []Choice `json:"choices"`
}

type Choice struct {
	Delta        Delta  `json:"delta"`
	FinishReason string `json:"finish_reason,omitempty"`
}

type Delta struct {
	Content string `json:"content"`
}
