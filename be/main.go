package main

import (
	"HNLP/internal/chatbot"
	HDb "HNLP/internal/db"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/sashabaranov/go-openai"
	"log"
	"os"
)

func main() {
	// TODO: separate code for router
	router := gin.Default()
	router.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "pong"})
	})

	LoadEnv()
	openAIKey := os.Getenv("OPENAI_API_KEY_TEST")

	db, err := HDb.NewHDb("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	defer func() {
		err := db.Close()
		if err != nil {
			log.Fatalf("Failed to close database: %v", err)
		}
	}()

	// DI pattern: Just manually inject anything here :P (no DI container)
	openAIClient := openai.NewClient(openAIKey)
	chatService := chatbot.NewChatService(openAIClient, db)
	chatController := chatbot.NewChatController(chatService)
	router.POST("/chat", chatController.QueryByChat)

	StartServer(router)
}

func StartServer(router *gin.Engine) {
	err := router.Run(":8080")
	if err != nil {
		println("Error starting server")
		return
	}
}

func LoadEnv() {
	// TODO: refactor this
	err := godotenv.Load("/home/huy/GolandProjects/HNLP/.env")
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}
}
