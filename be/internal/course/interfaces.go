package course

type Service interface {
	GetCourse(req GetCourseRequest) (*GetCourseResponse, error)
}

type Repository interface {
	GetByCode(code string) (Course, error)
	GetByName(s string) (Course, error)
	GetByEnglishName(s string) (Course, error)
}

type GetCourseRequest struct {
	Code *string `json:"code"`
	Name *string `json:"name"`
}

type GetCourseResponse struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	EnglishName string `json:"english_name"`
}
