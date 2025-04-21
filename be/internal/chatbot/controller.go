package chatbot

import (
	"HNLP/be/internal/auth"
	"context"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
)

type ChatController struct {
	chatService *ChatService
}

func NewChatController(chatService *ChatService) *ChatController {
	return &ChatController{chatService: chatService}
}

//	func (cc *ChatController) QueryByChat(context *gin.Context) {
//		var request QueryRequest
//		if err := context.BindJSON(&request); err != nil {
//			context.JSON(400, gin.H{"error": "Invalid request"})
//			return
//		}
//		// Process the message using the chat service
//		result, err := cc.chatService.QueryByChat(&request)
//		if err != nil {
//			context.JSON(500, gin.H{"error": "Failed to process message"})
//			return
//		}
//		context.JSON(200, result)
//	}
func (cc *ChatController) ChatStreamHandler(ctx *gin.Context) {
	// 1. Set SSE Headers
	w := ctx.Writer
	w.Header().Set("Access-Control-Allow-Origin", "*") // Adjust for production
	w.Header().Set("Access-Control-Expose-Headers", "Content-Type")
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// 2. Decode Request Body
	var request ChatRequest
	if err := ctx.ShouldBindJSON(&request); err != nil { // Use Gin's binding
		ctx.String(http.StatusBadRequest, "Invalid request: %v", err)
		return
	}
	//
	//// 3. Get user data from middleware (set earlier)
	//userIdRaw, exists := ctx.Get("userId")
	//if !exists {
	//	ctx.String(http.StatusUnauthorized, "User ID not found")
	//	return
	//}
	//userRole, exists := ctx.Get("userRole")
	//if !exists {
	//	ctx.String(http.StatusUnauthorized, "User role not found")
	//	return
	//}
	//
	//userIdFloat, ok := userIdRaw.(float64)
	//if !ok {
	//	ctx.String(http.StatusInternalServerError, "Invalid user ID type")
	//	return
	//}
	//userId := int(userIdFloat) // Cast float64 to int
	//role, ok := userRole.(string)
	//if !ok {
	//	ctx.String(http.StatusInternalServerError, "Invalid user role type")
	//	return
	//}

	userId := 1
	role := "admin"

	// 4. Enhance context with user data (optional)
	reqCtx := ctx.Request.Context()

	err := cc.chatService.StreamChatResponseV2(reqCtx, request, w, userId, role)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			log.Println("Client disconnected during streaming")
			return
		}
		log.Printf("Service error: %v", err)
		ctx.String(http.StatusInternalServerError, "Failed to stream response: %v", err)
		return
	}

	log.Println("Request handled successfully")
}
func (cc *ChatController) SearchResources(ctx *gin.Context) {
	// Use the same ChatRequest structure for consistency
	var request ChatRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	// Extract search query from the last message's content
	if len(request.Messages) == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "No messages provided"})
		return
	}

	// Use the last message as the search query
	query := request.Messages[len(request.Messages)-1].Content

	// Set default limit
	limit := 10

	// Auth check similar to ChatStreamHandler
	//userIdRaw, exists := ctx.Get("userId")
	//if !exists {
	//	ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found"})
	//	return
	//}

	// Call search service via ChatService
	reqCtx := ctx.Request.Context()
	err := cc.chatService.SearchResources(reqCtx, query, limit, ctx.Writer)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to retrieve resources: %v", err),
		})
		return
	}
	log.Println("Request handled successfully")
}

func (cc *ChatController) RegisterRoutes(router *gin.Engine, jwtService *auth.ServiceImpl) {
	router.POST("/v1/chat/completions", cc.ChatStreamHandler)
	router.POST("/v1/chat/resources", cc.SearchResources)
}
