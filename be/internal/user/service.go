package user

import (
	"HNLP/be/internal/auth"
	"context"
	"errors"
	"golang.org/x/crypto/bcrypt"
)

type ServiceImpl struct {
	jwtService auth.Service
	repo       Repository
}

func NewServiceImpl(jwtService auth.Service, repo Repository) *ServiceImpl {
	return &ServiceImpl{
		jwtService: jwtService,
		repo:       repo,
	}
}

func (s *ServiceImpl) GetUser(ctx context.Context, req GetUserRequest) (*GetUserResponse, error) {
	user, err := s.repo.GetById(ctx, req.ID)
	if user == (User{}) {
		user, err = s.repo.GetByUsername(ctx, req.Username)
	}
	if err != nil {
		return nil, errors.New("user not found")
	}
	return &GetUserResponse{
		ID:       user.ID,
		Username: user.Username,
	}, nil
}

func (s *ServiceImpl) GetUserPassword(ctx context.Context, req GetUserRequest) (*GetUserPasswordResponse, error) {
	user, err := s.repo.GetById(ctx, req.ID)
	if user == (User{}) {
		user, err = s.repo.GetByUsername(ctx, req.Username)
	}
	if err != nil {
		return nil, errors.New("user not found")
	}
	return &GetUserPasswordResponse{
		ID:       user.ID,
		Username: user.Username,
		Password: user.Password,
		Role:     user.Role,
	}, nil
}

func (s *ServiceImpl) GetAllUsers(ctx context.Context) ([]*GetUserResponse, error) {
	users, err := s.repo.GetAll(ctx)
	if err != nil {
		return nil, errors.New("failed to get users")
	}
	var response []*GetUserResponse
	for _, user := range users {
		response = append(response, &GetUserResponse{
			ID:       user.ID,
			Username: user.Username,
		})
	}
	return response, nil
}

func (s *ServiceImpl) CreateUser(ctx context.Context, req *CreateUserRequest) error {
	// Validate req
	existingUser, err := s.repo.GetByUsername(ctx, req.Username)
	if existingUser != (User{}) {
		return errors.New("user already exists")
	}
	if err != nil && err.Error() != "sql: no rows in result set" { // TODO: Avoid hardcoding
		return errors.New("failed to check if user exists")
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return errors.New("failed to hash password")
	}

	user := &User{
		Username: req.Username,
		Password: string(hashedPassword),
		Role:     Role(req.Role),
		Realname: req.Realname,
	}
	err = s.repo.Create(ctx, user)
	if err != nil {
		return errors.New("failed to create user")
	}
	return nil
}

func (s *ServiceImpl) Login(ctx context.Context, req LoginRequest) (*LoginResponse, error) {
	userResponse, err := s.GetUserPassword(ctx, GetUserRequest{Username: req.Username})
	if err != nil {
		return nil, err
	}

	if userResponse == nil {
		return nil, errors.New("user not found")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(userResponse.Password), []byte(req.Password)); err != nil {
		return nil, errors.New("invalid password")
	}

	token, err := s.jwtService.GenerateToken(userResponse.ID, userResponse.Username, string(userResponse.Role))
	if err != nil {
		return nil, err
	}

	return &LoginResponse{Token: token}, nil
}
