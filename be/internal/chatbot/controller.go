package chatbot

import "github.com/gin-gonic/gin"

type ChatController struct {
	chatService *ChatService
}

type QueryRequest struct {
	Message string `json:"message"`
}

type QueryResponse struct {
	Columns []string                 `json:"columns"`
	Rows    []map[string]interface{} `json:"rows"`
}

type CommunicateRequest struct {
	Message string `json:"message"`
}

func NewChatController(chatService *ChatService) *ChatController {
	return &ChatController{chatService: chatService}
}

func (cc *ChatController) QueryByChat(context *gin.Context) {
	var request QueryRequest
	if err := context.BindJSON(&request); err != nil {
		context.JSON(400, gin.H{"error": "Invalid request"})
		return
	}
	// Process the message using the chat service
	result, err := cc.chatService.QueryByChat(&request)
	if err != nil {
		context.JSON(500, gin.H{"error": "Failed to process message"})
		return
	}
	context.JSON(200, result)
}
