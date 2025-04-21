package user

import (
	"context"
	"github.com/gin-gonic/gin"
)

type Controller interface {
	GetUser(ctx *gin.Context) error
	SearchUser(ctx *gin.Context) error
	GetAllUsers(ctx *gin.Context) error
	CreateUser(ctx gin.Context) error
}

type Service interface {
	GetUser(ctx context.Context, req GetUserRequest) (*GetUserResponse, error)
	GetUserPassword(ctx context.Context, req GetUserRequest) (*GetUserPasswordResponse, error)
	GetAllUsers(ctx context.Context) ([]*GetUserResponse, error)
	CreateUser(ctx context.Context, req *CreateUserRequest) error
	Login(ctx context.Context, req LoginRequest) (*LoginResponse, error)
}

type Repository interface {
	GetById(ctx context.Context, id int) (User, error)
	GetAll(ctx context.Context) ([]User, error)
	GetByUsername(ctx context.Context, username string) (User, error)
	Create(ctx context.Context, user *User) error
}

type GetUserResponse struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
}

type GetUserRequest struct {
	ID       int    `json:"id" form:"id" uri:"id"`
	Username string `json:"username" form:"username" uri:"username"`
}

type CreateUserRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Role     string `json:"role" binding:"required"`
	Realname string `json:"realname"`
}

type GetUserPasswordResponse struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Password string `json:"password"`
	Role     Role   `json:"role"`
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	Token string `json:"token"`
}

type Role string

const (
	Student   = "student"
	Professor = "professor"
	Admin     = "admin"
)
