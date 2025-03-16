package auth

import (
	"github.com/golang-jwt/jwt/v5"
)

type Service interface {
	GenerateToken(id int, username string, role string) (string, error)
	ValidateAndParseToken(tokenString string) (*jwt.Token, error)
}
