package auth

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

type ControllerImpl struct {
	service Service
}

func NewControllerImpl(service Service) *ControllerImpl {
	return &ControllerImpl{service: service}
}

// Login handler
func (c *ControllerImpl) Login(ctx *gin.Context) {
	var req LoginRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: " + err.Error()})
		return
	}

	token, err := c.service.Login(req)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, token)
}

func (c *ControllerImpl) RegisterRoutes(router *gin.Engine) {
	router.POST("/login", c.Login)
}
