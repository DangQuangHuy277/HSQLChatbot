package course

import (
	"context"
	"errors"
)

type ServiceImpl struct {
	repo Repository
}

func NewServiceImpl(repo Repository) *ServiceImpl {
	return &ServiceImpl{
		repo: repo,
	}
}

func (s *ServiceImpl) GetCourse(ctx context.Context, req GetCourseRequest) (*GetCourseResponse, error) {
	var course Course
	var err error

	// Try ID first if provided
	if req.Code != nil {
		course, err = s.repo.GetByCode(ctx, *req.Code)
		if err == nil {
			return &GetCourseResponse{
				Code:        course.Code,
				Name:        course.Name,
				EnglishName: course.EnglishName,
			}, nil
		}
	}

	// Try name if provided (either as fallback from ID or direct request)
	if req.Name != nil {
		// Try regular name
		course, err = s.repo.GetByName(ctx, *req.Name)
		if err == nil {
			return &GetCourseResponse{
				Code:        course.Code,
				Name:        course.Name,
				EnglishName: course.EnglishName,
			}, nil
		}

		// Try English name
		course, err = s.repo.GetByEnglishName(ctx, *req.Name)
		if err == nil {
			return &GetCourseResponse{
				Code:        course.Code,
				Name:        course.Name,
				EnglishName: course.EnglishName,
			}, nil
		}
	}

	// If we reach here, no course was found
	return nil, errors.New("course not found")
}
