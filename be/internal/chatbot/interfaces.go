package chatbot

type Message struct {
	Role    string `json:"role"` // Optional, should avoid specify
	Content string `json:"content"`
	Type    string `json:"type"` // Optional
	Id      string `json:"id"`   // Optional
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

// CourseKeywords contains extracted information about courses and related keywords
type CourseKeywords struct {
	CourseCode *string  `json:"course_code" jsonschema:"description=The course code (e.g. CS101) if present in the query"`
	CourseName *string  `json:"course_name" jsonschema:"description=The name of the course (e.g. Introduction to Programming) if present in the query"`
	Keywords   []string `json:"keywords" jsonschema:"description=Other key terms related to the query; excluding filler words"`
	IsValid    bool     `json:"is_valid" jsonschema:"description=True if the query relates to study materials or course; false otherwise"`
}
