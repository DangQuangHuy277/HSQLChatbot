package chatbot

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
)

type ChatController struct {
	chatService *ChatService
}

func NewChatController(chatService *ChatService) *ChatController {
	return &ChatController{chatService: chatService}
}

//func (cc *ChatController) QueryByChat(context *gin.Context) {
//	var request QueryRequest
//	if err := context.BindJSON(&request); err != nil {
//		context.JSON(400, gin.H{"error": "Invalid request"})
//		return
//	}
//	// Process the message using the chat service
//	result, err := cc.chatService.QueryByChat(&request)
//	if err != nil {
//		context.JSON(500, gin.H{"error": "Failed to process message"})
//		return
//	}
//	context.JSON(200, result)
//}

func (cc *ChatController) ChatStreamHandler(w http.ResponseWriter, r *http.Request) {
	// 1. Set some Header
	// Set CORS headers to allow all origins. Adjust this for production to allow specific origins.
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Expose-Headers", "Content-Type")

	// Set headers required for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// 2. Decode Request Body
	var request ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// 3. Call Service Layer to stream response
	ctx := r.Context() // Use request context for cancellation

	err := cc.chatService.StreamChatResponse(ctx, request, w)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			// Client disconnected, no need to send error response
			log.Println("Client disconnected during streaming")
			return
		}
		// Handle other errors (e.g., service errors, authentication failures)
		log.Printf("Service error: %v", err)
		http.Error(w, "Failed to stream response: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Println("Request handled successfully")
}
