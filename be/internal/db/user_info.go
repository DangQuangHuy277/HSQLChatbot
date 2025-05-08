package db

import (
	"context"
	"log"
	"sync"
)

type UserInfo struct {
	ID   int
	Role string
}

type StudentInfo struct {
	UserInfo
	AdministrativeClassID  int
	EnrolledCourseClassIDs []int
	EnrolledScheduleIDs    []int
}

type ProfessorInfo struct {
	UserInfo
	AdvisedClassIDs      []int
	TaughtCourseClassIDs []int
	TaughtScheduleIDs    []int
	AdvisedStudentIDs    []int
}

type AdminInfo struct {
	UserInfo
}

func (u *UserInfo) GetRole() string {
	return u.Role
}

func (u *UserInfo) GetID() int {
	return u.ID
}

func (db *SQLHDb) FetchStudentInfo(ctx context.Context, userID int) (*StudentInfo, error) {
	var info StudentInfo
	info.UserInfo.ID = userID
	info.UserInfo.Role = "student"
	var wg sync.WaitGroup
	wg.Add(3)
	go func() {
		defer wg.Done()
		err := db.GetContext(ctx, &info.AdministrativeClassID, "SELECT administrative_class_id FROM student WHERE id = $1", userID)
		if err != nil {
			log.Println("Error fetching administrative class ID:", err)
			return
		}
	}()

	go func() {
		// Query for course class IDs only
		defer wg.Done()
		err := db.SelectContext(ctx, &info.EnrolledCourseClassIDs,
			`SELECT id FROM course_class_enrollment WHERE student_id = $1`, userID)
		if err != nil {
			log.Println("Error fetching enrolled course class IDs:", err)
			return
		}
	}()

	go func() {
		// Separate query for schedule IDs
		defer wg.Done()
		err := db.SelectContext(ctx, &info.EnrolledScheduleIDs,
			`SELECT sccs.id
         FROM student_course_class_schedule sccs
         JOIN course_class_enrollment scc ON sccs.course_class_enrollment_id = scc.id and scc.student_id = $1`, userID)
		if err != nil {
			log.Println("Error fetching enrolled schedule IDs:", err)
			return
		}
	}()

	wg.Wait()

	return &info, nil
}

func (db *SQLHDb) FetchProfessorInfo(ctx context.Context, userID int) (*ProfessorInfo, error) {
	var info ProfessorInfo
	info.UserInfo.ID = userID
	info.UserInfo.Role = "professor"

	var wg sync.WaitGroup
	wg.Add(4)
	go func() {
		defer wg.Done()
		err := db.SelectContext(ctx, &info.AdvisedClassIDs, "select id from administrative_class where advisor_id = $1", userID)
		if err != nil {
			log.Println("Error fetching advised class ID:", err)
			return
		}
	}()

	go func() {
		defer wg.Done()
		err := db.SelectContext(ctx, &info.TaughtCourseClassIDs,
			`select cc.id
			from course_class cc
			inner join course_class_schedule  ccs on cc.id = ccs.course_class_id
			inner join course_schedule_instructor csi on ccs.id = csi.course_class_schedule_id
			where csi.professor_id = $1`, userID)
		if err != nil {
			log.Println("Error fetching taught course class IDs:", err)
			return
		}
	}()

	go func() {
		defer wg.Done()
		err := db.SelectContext(ctx, &info.TaughtScheduleIDs,
			`select ccs.id
			from course_class_schedule ccs
			inner join course_schedule_instructor csi on ccs.id = csi.course_class_schedule_id
			where csi.professor_id = $1`, userID)
		if err != nil {
			log.Println("Error fetching taught schedule IDs:", err)
			return
		}
	}()

	go func() {
		defer wg.Done()
		err := db.SelectContext(ctx, &info.AdvisedStudentIDs,
			`select s.id
			from student s
			inner join administrative_class ac on s.administrative_class_id = ac.id
			where ac.advisor_id = $1`, userID)
		if err != nil {
			log.Println("Error fetching advised student IDs:", err)
			return
		}
	}()
	wg.Wait()
	return &info, nil
}

func (db *SQLHDb) FetchAdminInfo(ctx context.Context, userID int) (*AdminInfo, error) {
	var info AdminInfo
	info.UserInfo.ID = userID
	info.UserInfo.Role = "admin"
	return &info, nil
}
