package main

import (
	"HNLP/be/internal/auth"
	"HNLP/be/internal/chatbot"
	"HNLP/be/internal/config"
	HDb "HNLP/be/internal/db"
	"HNLP/be/internal/llm"
	"HNLP/be/internal/user"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"github.com/sashabaranov/go-openai"
	"log"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig("/home/huy/Code/Personal/KLTN/be/config/config.yaml", "/home/huy/Code/Personal/KLTN/be/config/.env")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize router
	router := gin.Default()
	router.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "pong"})
	})

	// Initialize database
	db, err := HDb.NewHDb("postgres", cfg.Database.URL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	defer func() {
		if err := db.Close(); err != nil {
			log.Fatalf("Failed to close database: %v", err)
		}
	}()

	// Configure CORS
	router.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.CORS.AllowOrigins,
		AllowMethods:     cfg.CORS.AllowMethods,
		AllowHeaders:     cfg.CORS.AllowHeaders,
		ExposeHeaders:    cfg.CORS.ExposeHeaders,
		AllowCredentials: cfg.CORS.AllowCredentials,
	}))

	// Initialize services
	openAIClient := openai.NewClient(cfg.OpenAI.APIKey)
	openAIProvider := llm.NewOpenAIProvider(openAIClient)

	//geminiAIClient, err := genai.NewClient(context.Background(), option.WithAPIKey(cfg.GeminiAI.APIKey))
	//geminiAIProvider := llm.NewGeminiAIProvider(geminiAIClient)

	chatService := chatbot.NewChatService(openAIProvider, db)
	chatController := chatbot.NewChatController(chatService)

	router.POST("/v1/chat/completions", func(c *gin.Context) {
		chatController.ChatStreamHandler(c.Writer, c.Request)
	})

	// User management
	userRepository := user.NewRepositoryImpl(db)
	userService := user.NewServiceImpl(userRepository) // Pass config here
	userController := user.NewControllerImpl(userService)
	userController.RegisterRoutes(router)

	// Auth management
	authService := auth.NewServiceImpl(userService, cfg.JWT)
	authController := auth.NewControllerImpl(authService)
	authController.RegisterRoutes(router)

	// Start server
	if err := router.Run(":" + cfg.Server.Port); err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}
