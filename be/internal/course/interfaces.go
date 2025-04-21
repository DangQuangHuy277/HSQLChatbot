package course

import "context"

type Service interface {
	GetCourse(ctx context.Context, req GetCourseRequest) (*GetCourseResponse, error)
}

type Repository interface {
	GetByCode(ctx context.Context, code string) (Course, error)
	GetByName(ctx context.Context, s string) (Course, error)
	GetByEnglishName(ctx context.Context, s string) (Course, error)
}

type GetCourseRequest struct {
	Code *string `json:"code,omitempty"`
	Name *string `json:"name,omitempty"`
}

type GetCourseResponse struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	EnglishName string `json:"english_name"`
}
