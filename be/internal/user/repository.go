package user

import (
	"HNLP/be/internal/db"
	"context"
)

type RepositoryImpl struct {
	db db.HDb
}

func NewRepositoryImpl(db db.HDb) *RepositoryImpl {
	return &RepositoryImpl{db: db}
}

func (r *RepositoryImpl) GetById(ctx context.Context, id int) (User, error) {
	var user User
	err := r.db.GetContext(ctx, &user, "SELECT * FROM user_account WHERE id = $1", id)
	return user, err
}

func (r *RepositoryImpl) GetAll(ctx context.Context) ([]User, error) {
	var users []User
	err := r.db.GetContext(ctx, &users, "SELECT * FROM user_account")
	return users, err
}

func (r *RepositoryImpl) GetByUsername(ctx context.Context, username string) (User, error) {
	var user User
	err := r.db.GetContext(ctx, &user, "SELECT * FROM user_account WHERE username = $1", username)
	return user, err
}

func (r *RepositoryImpl) Create(ctx context.Context, user *User) error {
	_, err := r.db.ExecContext(ctx, "INSERT INTO user_account (username, password, role, realname) VALUES ($1, $2, $3, $4)",
		user.Username, user.Password, user.Role, user.Realname)
	return err
}
