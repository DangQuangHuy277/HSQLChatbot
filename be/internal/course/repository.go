package course

import "HNLP/be/internal/db"

type RepositoryImpl struct {
	db *db.HDb
}

func NewRepositoryImpl(db *db.HDb) *RepositoryImpl {
	return &RepositoryImpl{db: db}
}

func (r *RepositoryImpl) GetByCode(code string) (Course, error) {
	var course Course
	err := r.db.Get(course, "SELECT * FROM course WHERE code = $1", code)
	return course, err
}

func (r *RepositoryImpl) GetByName(s string) (Course, error) {
	var course Course
	err := r.db.Get(course, "SELECT * FROM course WHERE name = $1", s)
	return course, err
}

func (r *RepositoryImpl) GetByEnglishName(s string) (Course, error) {
	var course Course
	err := r.db.Get(course, "SELECT * FROM course WHERE english_name = $1", s)
	return course, err
}
