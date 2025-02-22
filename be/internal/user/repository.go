package user

import "HNLP/be/internal/db"

type RepositoryImpl struct {
	db *db.HDb
}

func NewRepositoryImpl(db *db.HDb) *RepositoryImpl {
	return &RepositoryImpl{db: db}
}

func (r *RepositoryImpl) GetById(id int) (User, error) {
	var user User
	err := r.db.Get(&user, "SELECT * FROM user_account WHERE id = $1", id)
	return user, err
}

func (r *RepositoryImpl) GetAll() ([]User, error) {
	var users []User
	err := r.db.Select(&users, "SELECT * FROM user_account")
	return users, err
}

func (r *RepositoryImpl) GetByUsername(username string) (User, error) {
	var user User
	err := r.db.Get(&user, "SELECT * FROM user_account WHERE username = $1", username)
	return user, err
}

func (r *RepositoryImpl) Create(user *User) error {
	_, err := r.db.Exec("INSERT INTO user_account (username, password) VALUES ($1, $2)", user.Username, user.Password)
	return err
}
