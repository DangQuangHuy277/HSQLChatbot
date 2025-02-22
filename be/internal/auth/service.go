package auth

import (
	"HNLP/be/internal/config"
	"HNLP/be/internal/user"
	"errors"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"time"
)

type ServiceImpl struct {
	userService user.Service
	config      config.JWTConfig
}

func NewServiceImpl(userService user.Service, config config.JWTConfig) *ServiceImpl {
	return &ServiceImpl{
		userService: userService,
		config:      config,
	}
}

func (s *ServiceImpl) Login(req LoginRequest) (*LoginResponse, error) {
	userResponse, err := s.userService.GetUserPassword(user.GetUserRequest{Username: req.Username})
	if err != nil {
		return nil, err
	}

	if userResponse == nil {
		return nil, errors.New("user not found")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(userResponse.Password), []byte(req.Password)); err != nil {
		return nil, errors.New("invalid password")
	}

	token, err := s.generateToken(userResponse.ID, userResponse.Username, userResponse.Role)
	if err != nil {
		return nil, err
	}

	return &LoginResponse{Token: token}, nil
}

func (s *ServiceImpl) generateToken(id int, username string, role user.Role) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, &jwt.MapClaims{
		"id":       id,
		"username": username,
		"role":     role,
		"exp":      time.Now().Add(time.Hour * s.config.ExpiryHours).Unix(),
		"iat":      time.Now().Unix(),
	})

	tokenString, err := token.SignedString(s.config.SecretKey)
	if err != nil {
		return "", err
	}
	return tokenString, nil
}
