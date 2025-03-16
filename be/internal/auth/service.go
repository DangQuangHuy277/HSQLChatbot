package auth

import (
	"HNLP/be/internal/config"
	"github.com/golang-jwt/jwt/v5"
	"time"
)

type ServiceImpl struct {
	config config.JWTConfig
}

func NewServiceImpl(config config.JWTConfig) *ServiceImpl {
	return &ServiceImpl{
		config: config,
	}
}

func (s *ServiceImpl) GenerateToken(id int, username string, role string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, &jwt.MapClaims{
		"id":       id,
		"username": username,
		"role":     role,
		"exp":      time.Now().Add(time.Hour * s.config.ExpiryHours).Unix(),
		"iat":      time.Now().Unix(),
	})

	tokenString, err := token.SignedString([]byte(s.config.SecretKey))
	if err != nil {
		return "", err
	}
	return tokenString, nil
}

func (s *ServiceImpl) ValidateAndParseToken(tokenString string) (*jwt.Token, error) {
	return jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.config.SecretKey), nil
	})
}
