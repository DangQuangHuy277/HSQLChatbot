package user

import (
	"HNLP/be/internal/auth"
	"HNLP/be/internal/middleware"
	"github.com/gin-gonic/gin"
	"net/http"
)

type ControllerImpl struct {
	service Service
}

func NewControllerImpl(service Service) *ControllerImpl {
	return &ControllerImpl{service: service}
}

// GetUser handler
func (c *ControllerImpl) GetUser(ctx *gin.Context) {
	var req GetUserRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}
	user, err := c.service.GetUser(ctx, req)
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, user)
}

// SearchUser handler
func (c *ControllerImpl) SearchUser(ctx *gin.Context) {
	var req GetUserRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID"})
		return
	}
	user, err := c.service.GetUser(ctx, req)
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, user)
}

// GetAllUsers handler
func (c *ControllerImpl) GetAllUsers(ctx *gin.Context) {
	users, err := c.service.GetAllUsers(ctx)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if len(users) == 0 {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "no users found"})
		return
	}
	ctx.JSON(http.StatusOK, users)
}

// CreateUser handler
func (c *ControllerImpl) CreateUser(ctx *gin.Context) {
	var req CreateUserRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	err := c.service.CreateUser(ctx, &req)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusCreated, gin.H{"message": "user created"})
}

// Login handler
func (c *ControllerImpl) Login(ctx *gin.Context) {
	var req LoginRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	token, err := c.service.Login(ctx, req)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, token)
}

// RegisterRoutes to set up Gin routes
func (c *ControllerImpl) RegisterRoutes(router *gin.Engine, service auth.Service) {
	router.GET("/users/:id", middleware.Authenticate(service), c.GetUser)
	router.GET("/users/search", c.SearchUser)
	router.GET("/users", c.GetAllUsers)
	router.POST("/users", c.CreateUser)
	router.POST("/login", c.Login)
}
