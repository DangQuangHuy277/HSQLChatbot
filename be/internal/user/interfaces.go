package user

import "github.com/gin-gonic/gin"

type Controller interface {
	GetUser(ctx *gin.Context) error
	SearchUser(ctx *gin.Context) error
	GetAllUsers(ctx *gin.Context) error
	CreateUser(ctx gin.Context) error
}

type Service interface {
	GetUser(req GetUserRequest) (*GetUserResponse, error)
	GetUserPassword(req GetUserRequest) (*GetUserPasswordResponse, error)
	GetAllUsers() ([]*GetUserResponse, error)
	CreateUser(req *CreateUserRequest) error
	Login(req LoginRequest) (*LoginResponse, error)
}

type Repository interface {
	GetById(id int) (User, error)
	GetAll() ([]User, error)
	GetByUsername(username string) (User, error)
	Create(user *User) error
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
