package course

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

func (r *RepositoryImpl) GetByCode(ctx context.Context, code string) (Course, error) {
	var course Course
	err := r.db.GetContext(ctx, &course, "SELECT * FROM course WHERE code = $1", code)
	return course, err
}

func (r *RepositoryImpl) GetByName(ctx context.Context, name string) (Course, error) {
	var course Course
	err := r.db.GetContext(ctx, &course, "SELECT * FROM course WHERE name = $1", name)
	return course, err
}

func (r *RepositoryImpl) GetByEnglishName(ctx context.Context, englishName string) (Course, error) {
	var course Course
	err := r.db.GetContext(ctx, &course, "SELECT * FROM course WHERE english_name = $1", englishName)
	return course, err
}
