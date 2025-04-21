package main

import (
	"HNLP/be/internal/auth"
	"HNLP/be/internal/chatbot"
	"HNLP/be/internal/config"
	"HNLP/be/internal/course"
	HDb "HNLP/be/internal/db"
	"HNLP/be/internal/llm"
	"HNLP/be/internal/search"
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
	// Configure CORS
	router.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.CORS.AllowOrigins,
		AllowMethods:     cfg.CORS.AllowMethods,
		AllowHeaders:     cfg.CORS.AllowHeaders,
		ExposeHeaders:    cfg.CORS.ExposeHeaders,
		AllowCredentials: cfg.CORS.AllowCredentials,
	}))

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

	// Initialize services
	openAIClient := openai.NewClient(cfg.OpenAI.APIKey)
	openAIProvider := llm.NewOpenAIProvider(openAIClient)

	//geminiAIClient, err := genai.NewClient(context.Background(), option.WithAPIKey(cfg.GeminiAI.APIKey))
	//geminiAIProvider := llm.NewGeminiAIProvider(geminiAIClient)

	// User management
	jwtService := auth.NewServiceImpl(cfg.JWT)
	userRepository := user.NewRepositoryImpl(db)
	userService := user.NewServiceImpl(jwtService, userRepository) // Pass config here
	userController := user.NewControllerImpl(userService)
	userController.RegisterRoutes(router, jwtService)

	// Course management
	courseRepo := course.NewRepositoryImpl(db)
	courseService := course.NewServiceImpl(courseRepo)

	// Search
	searchService := search.NewSearchService(cfg.SerpApi)

	// Init function registry, after we inits all the services and before we inits the chatbot
	funcRegistry := llm.NewFunctionRegistryImpl()
	funcRegistry.Register(llm.FuncWrapper("ExecuteQuery", "Run a SQL query to my university database and return the result in a JSON format", db.ExecuteQuery))
	funcRegistry.Register(llm.FuncWrapper("GetCourse", "Get course information by code or name", courseService.GetCourse))

	chatService := chatbot.NewChatService(openAIProvider, db, searchService, funcRegistry)
	chatController := chatbot.NewChatController(chatService)
	chatController.RegisterRoutes(router, jwtService)

	// Start server
	if err := router.Run(":" + cfg.Server.Port); err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}
