package auth

import (
	"github.com/gin-gonic/gin"
)

type Controller interface {
	Login(ctx *gin.Context) error
}

type Service interface {
	Login(req LoginRequest) (*LoginResponse, error)
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	Token string `json:"token"`
}
