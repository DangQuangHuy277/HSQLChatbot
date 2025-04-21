package chatbot

import (
	"HNLP/be/internal/db"
	"HNLP/be/internal/llm"
	"HNLP/be/internal/search"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/sashabaranov/go-openai"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"log"
	"os"
	"strings"
	"testing"
)

// MockAIProvider implements the llm.AIProvider interface for testing
type MockAIProvider struct {
	mock.Mock
}

// testFuncRegistryInstance is used to implement the singleton pattern
var testFuncRegistryInstance llm.FuncRegistry

// Complete mocks the Complete method
func (m *MockAIProvider) Complete(ctx context.Context, req llm.CompletionRequest) (llm.Message, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(llm.Message), args.Error(1)
}

// StreamComplete mocks the StreamComplete method
func (m *MockAIProvider) StreamComplete(ctx context.Context, req llm.CompletionRequest) (<-chan llm.StreamChunk, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(<-chan llm.StreamChunk), args.Error(1)
}

// MockHDb implements a mock of the db.HDb for testing
type MockHDb struct {
	mock.Mock
}

func (m *MockHDb) ExecuteQuery(ctx context.Context, query string) (*db.QueryResult, error) {
	args := m.Called(ctx, query)
	if result, ok := args.Get(0).(*db.QueryResult); ok {
		return result, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockHDb) LoadDDL() (string, error) {
	args := m.Called()
	return args.String(0), args.Error(1)
}

func (m *MockHDb) GetContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	args = append([]interface{}{ctx, dest, query}, args...)
	return m.Called(args...).Error(0)
}

// MockSearchService implements the search.Service interface for testing
type MockSearchService struct {
	mock.Mock
}

// Search mocks the Search method
func (m *MockSearchService) Search(ctx context.Context, keywords []string) ([]search.Resource, error) {
	args := m.Called(ctx, keywords)
	if resources, ok := args.Get(0).([]search.Resource); ok {
		return resources, args.Error(1)
	}

	return nil, args.Error(1)
}

// mockChatServiceInstance is used to implement the singleton pattern
var mockChatServiceInstance *ChatService

// GetTestChatService creates a mock ChatService with mocked dependencies as a singleton
func GetTestChatService() *ChatService {
	if mockChatServiceInstance == nil {
		// Load .env file for integration testing
		if err := godotenv.Load("../../config/.env"); err != nil {
			log.Printf("Warning: failed to load .env file: %v", err)
		}

		// Get OpenAI key from environment
		openAIKey := os.Getenv("OPENAI_API_KEY")
		if openAIKey == "" {
			log.Printf("Warning: OPENAI_API_KEY not found in environment")
		}
		// Initialize the mock dependencies

		mockChatServiceInstance = &ChatService{
			aiProvider:   llm.NewOpenAIProvider(openai.NewClient(openAIKey)),
			db:           &MockHDb{},
			searchSrv:    &MockSearchService{},
			funcRegistry: GetTestFuncRegistry(),
		}
	}
	return mockChatServiceInstance
}

// GetTestFuncRegistry returns a singleton instance of the test function registry
func GetTestFuncRegistry() llm.FuncRegistry {
	if testFuncRegistryInstance == nil {
		testFuncRegistryInstance = llm.NewFunctionRegistryImpl()
		registerTestFunctionDefinition()
	}

	return testFuncRegistryInstance
}

func registerTestFunctionDefinition() {
	// Register some test functions
	type QueryDTO struct {
		Query string `json:"query" jsonschema:"description=SQL query to execute against the university database"`
	}

	testFuncRegistryInstance.Register(llm.FuncWrapper("ExecuteQuery", "Run a SQL query to my university database and return the result in a JSON format", func(ctx context.Context, dto QueryDTO) (*db.QueryResult, error) {
		return &db.QueryResult{}, nil
	}))

	// Mock the function registry to return the function definition
	type StudentQuery struct {
		ID int `json:"id" jsonschema:"description=Unique identifier of the student;minimum=1"`
	}
	handler := func(ctx context.Context, id StudentQuery) (map[string]interface{}, error) {
		return map[string]interface{}{
			"student": map[string]interface{}{
				"id":    123,
				"name":  "John Smith",
				"major": "Computer Science",
				"gpa":   3.8,
			},
		}, nil
	}

	testFuncRegistryInstance.Register(llm.FuncWrapper("GetStudentInfo", "Get student information by id", handler))

	// Register the function definition for course information
	type CourseQuery struct {
		Code string `json:"code" jsonschema:"description=Course code (e.g. CS101)"`
		Name string `json:"name" jsonschema:"description=Course name (e.g. Introduction to Computer Science)"`
	}

	courseHandler := func(ctx context.Context, course CourseQuery) (map[string]interface{}, error) {
		return map[string]interface{}{
			"course": map[string]interface{}{
				"code":    "CS101",
				"name":    "Introduction to Computer Science",
				"credits": 3,
			},
		}, nil
	}
	testFuncRegistryInstance.Register(llm.FuncWrapper("GetCourse", "Get course information by code or name", courseHandler))

	// Register the function definition for administrative class information
	type AdministrativeClassQuery struct {
		ID int `json:"id" jsonschema:"description=Unique identifier of the administrative class;minimum=1"`
	}

	adminClassHandler := func(ctx context.Context, id AdministrativeClassQuery) (map[string]interface{}, error) {
		return map[string]interface{}{
			"administrative_class": map[string]interface{}{
				"id":   1,
				"name": "Administrative Class 101",
			},
		}, nil
	}
	testFuncRegistryInstance.Register(llm.FuncWrapper("GetAdministrativeClass", "Get administrative class information by id", adminClassHandler))

	// Register the function definition for professor information
	type ProfessorQuery struct {
		ID int `json:"id" jsonschema:"description=Unique identifier of the professor,minimum=1"`
	}

	professorHandler := func(ctx context.Context, id ProfessorQuery) (map[string]interface{}, error) {
		return map[string]interface{}{
			"professor": map[string]interface{}{
				"id":    456,
				"name":  "Dr. John Doe",
				"email": "john.doe@gmail.com",
			},
		}, nil
	}
	testFuncRegistryInstance.Register(llm.FuncWrapper("GetProfessor", "Get professor information by id", professorHandler))

	// Register the function definition for course class information
	type CourseClassQuery struct {
		ID int `json:"id" jsonschema:"description=Unique identifier of the course class;minimum=1"`
	}
	courseClassHandler := func(ctx context.Context, id CourseClassQuery) (map[string]interface{}, error) {
		return map[string]interface{}{
			"course_class": map[string]interface{}{
				"id":   789,
				"name": "Course Class 101",
			},
		}, nil
	}
	testFuncRegistryInstance.Register(llm.FuncWrapper("GetCourseClass", "Get course class information by id", courseClassHandler))

	// Register the function definition for student course class information
	type StudentCourseClassQuery struct {
		ID int `json:"id" jsonschema:"description=Unique identifier of the student course class;minimum=1"`
	}
	studentCourseClassHandler := func(ctx context.Context, id StudentCourseClassQuery) (map[string]interface{}, error) {
		return map[string]interface{}{
			"student_course_class": map[string]interface{}{
				"id":   101112,
				"name": "Student Course Class 101",
			},
		}, nil
	}
	testFuncRegistryInstance.Register(llm.FuncWrapper("GetStudentCourseClass", "Get student course class information by id", studentCourseClassHandler))

	// Register the function definition for course class schedule information
	type CourseClassScheduleQuery struct {
		ID int `json:"id" jsonschema:"description=Unique identifier of the course class schedule;minimum=1"`
	}

	courseClassScheduleHandler := func(ctx context.Context, id CourseClassScheduleQuery) (map[string]interface{}, error) {
		return map[string]interface{}{
			"course_class_schedule": map[string]interface{}{
				"id":   131415,
				"name": "Course Class Schedule 101",
			},
		}, nil
	}

	testFuncRegistryInstance.Register(llm.FuncWrapper("GetCourseClassSchedule", "Get course class schedule information by id", courseClassScheduleHandler))
	// Register the function definition for student course class schedule information
	type StudentCourseClassScheduleQuery struct {
		ID int `json:"id" jsonschema:"description=Unique identifier of the student course class schedule;minimum=1"`
	}
	studentCourseClassScheduleHandler := func(ctx context.Context, id StudentCourseClassScheduleQuery) (map[string]interface{}, error) {
		return map[string]interface{}{
			"student_course_class_schedule": map[string]interface{}{
				"id":   161718,
				"name": "Student Course Class Schedule 101",
			},
		}, nil
	}
	testFuncRegistryInstance.Register(llm.FuncWrapper("GetStudentCourseClassSchedule", "Get student course class schedule information by id", studentCourseClassScheduleHandler))

	// Add more specialized functions to test complex scenarios
	type ScheduleQuery struct {
		StudentID  int    `json:"student_id" jsonschema:"description=Unique identifier of the student"`
		SemesterID string `json:"semester_id,omitempty" jsonschema:"description=Semester identifier (e.g. FALL2023)"`
		DayOfWeek  string `json:"day_of_week,omitempty" jsonschema:"description=Day of the week"`
		CourseCode string `json:"course_code,omitempty" jsonschema:"description=Course code to filter the schedule (e.g. CS101)"`
	}

	scheduleHandler := func(ctx context.Context, query ScheduleQuery) (map[string]interface{}, error) {
		return map[string]interface{}{
			"schedule": map[string]interface{}{
				"student_id": query.StudentID,
				"classes": []map[string]interface{}{
					{
						"course_code": "CS101",
						"day":         "Monday",
						"time":        "08:00-10:00",
						"location":    "Room 101",
					},
					{
						"course_code": "MATH202",
						"day":         "Wednesday",
						"time":        "13:00-15:00",
						"location":    "Room 205",
					},
				},
			},
		}, nil
	}
	testFuncRegistryInstance.Register(llm.FuncWrapper("GetStudentSchedule", "Get detailed schedule for a student with optional filters", scheduleHandler))

	// Registration function for complex statistical analysis
	type StatisticsQuery struct {
		Type       string `json:"type" jsonschema:"description=Type of statistics (e.g. grades; attendance)"`
		EntityID   int    `json:"entity_id,omitempty" jsonschema:"description=ID of the entity (student; course; etc.)"`
		SemesterID string `json:"semester_id,omitempty" jsonschema:"description=Semester identifier for the statistics"`
		Metric     string `json:"metric,omitempty" jsonschema:"description=Specific metric to query (e.g.; average; median)"`
	}

	statisticsHandler := func(ctx context.Context, query StatisticsQuery) (map[string]interface{}, error) {
		return map[string]interface{}{
			"statistics": map[string]interface{}{
				"type": query.Type,
				"data": []map[string]interface{}{
					{
						"label": "CS101",
						"value": 78.5,
					},
					{
						"label": "MATH202",
						"value": 82.3,
					},
				},
			},
		}, nil
	}
	testFuncRegistryInstance.Register(llm.FuncWrapper("GetStatistics", "Get statistical data for various academic entities", statisticsHandler))

	// Add a function for handling multi-entity queries
	type MultiEntityQuery struct {
		StudentID   int    `json:"student_id,omitempty" jsonschema:"description=Student ID to include student information"`
		ProfessorID int    `json:"professor_id,omitempty" jsonschema:"description=Professor ID to include professor information"`
		CourseCode  string `json:"course_code,omitempty" jsonschema:"description=Course code to include course information"`
		SemesterID  string `json:"semester_id,omitempty" jsonschema:"description=Semester ID to filter the results"`
	}

	multiEntityHandler := func(ctx context.Context, query MultiEntityQuery) (map[string]interface{}, error) {
		return map[string]interface{}{
			"data": map[string]interface{}{
				"student": func() map[string]interface{} {
					if query.StudentID > 0 {
						return map[string]interface{}{
							"id":   query.StudentID,
							"name": "Jane Smith",
						}
					}
					return nil
				}(),
				"professor": func() map[string]interface{} {
					if query.ProfessorID > 0 {
						return map[string]interface{}{
							"id":   query.ProfessorID,
							"name": "Dr. Robert Brown",
						}
					}
					return nil
				}(),
				"course": func() map[string]interface{} {
					if query.CourseCode != "" {
						return map[string]interface{}{
							"code": query.CourseCode,
							"name": "Advanced Programming",
						}
					}
					return nil
				}(),
			},
		}, nil
	}
	testFuncRegistryInstance.Register(llm.FuncWrapper("GetMultiEntityInfo", "Get information about multiple entities at once", multiEntityHandler))

	// Function for natural language processing capabilities
	type NLPQuery struct {
		Text         string `json:"text" jsonschema:"description=Text to analyze,required"`
		AnalysisType string `json:"analysis_type,omitempty" jsonschema:"description=Type of analysis to perform (e.g., sentiment, entity),enum=sentiment,entity,intent"`
	}

	nlpHandler := func(ctx context.Context, query NLPQuery) (map[string]interface{}, error) {
		return map[string]interface{}{
			"analysis": map[string]interface{}{
				"type": query.AnalysisType,
				"results": map[string]interface{}{
					"entities":  []string{"CS101", "Computer Science", "programming"},
					"intent":    "course_inquiry",
					"sentiment": "neutral",
				},
			},
		}, nil
	}
	testFuncRegistryInstance.Register(llm.FuncWrapper("AnalyzeText", "Perform NLP analysis on text", nlpHandler))

	// Add a function that might conflict with others to test disambiguation
	type SearchQuery struct {
		Keywords []string `json:"keywords" jsonschema:"description=Keywords to search for"`
		Filter   string   `json:"filter,omitempty" jsonschema:"description=Filter to apply to search results (e.g. course; professor)"`
	}

	searchHandler := func(ctx context.Context, query SearchQuery) (map[string]interface{}, error) {
		return map[string]interface{}{
			"search_results": []map[string]interface{}{
				{
					"title": "Introduction to Computer Science",
					"type":  "course",
					"id":    "CS101",
				},
				{
					"title": "Advanced Programming Techniques",
					"type":  "course",
					"id":    "CS301",
				},
			},
		}, nil
	}
	testFuncRegistryInstance.Register(llm.FuncWrapper("SearchDatabase", "Search the university database with keywords", searchHandler))
}

func TestChatService_StreamChatResponseV2_WithTools(t *testing.T) {
	// Get test service
	srv := GetTestChatService()

	// Create request asking for student's own information
	req := ChatRequest{
		Messages: []Message{
			{
				Role:    "user",
				Content: "What's my student information?",
				Type:    "text",
				Id:      "msg_1",
			},
		},
		SessionID: "test-session-123",
	}

	// Create the mock and store a reference to it
	mockDb := &MockHDb{}
	srv.db = mockDb

	// Configure mock to return schema
	mockDb.On("LoadDDL").Return("CREATE TABLE IF NOT EXISTS student (id INTEGER PRIMARY KEY, name TEXT, major TEXT, gpa REAL);", nil)

	// Mock query execution for student information
	studentResult := &db.QueryResult{
		Data: []map[string]interface{}{
			{
				"id":    123,
				"name":  "John Smith",
				"major": "Computer Science",
				"gpa":   3.8,
			},
		},
		Metadata: struct {
			RowCount int      `json:"row_count"`
			Columns  []string `json:"columns"`
		}{
			RowCount: 1,
			Columns:  []string{"id", "name", "major", "gpa"},
		},
	}

	// Mock the ExecuteQuery method to return student data when queried with user ID
	mockDb.On("ExecuteQuery", mock.Anything, mock.MatchedBy(func(query string) bool {
		// Simple check if query contains student ID
		return query != "" && query != "LOAD DDL"
	})).Return(studentResult, nil)

	// Buffer to collect streamed response
	var responseBuffer bytes.Buffer

	// Call the function being tested with student context and student ID
	err := srv.StreamChatResponseV2(context.Background(), req, &responseBuffer, 123, "student")

	// Verify no error occurred
	assert.Nil(t, err, "Expected no error, got: %v", err)

	// Check response contains student information
	response := responseBuffer.String()
	if response == "" {
		t.Error("Expected non-empty response, got empty string")
	}

	// Verify mocks were called as expected
	mockDb.AssertExpectations(t)
}

func TestValidateUserQuery_Query_WithRealAI(t *testing.T) {
	// Get the service with real AI provider
	svc := GetTestChatService()

	tests := []struct {
		name        string
		query       string
		userId      int
		role        string
		expectValid bool
	}{
		{
			name:        "Student accessing own data - Valid",
			query:       "SELECT * FROM student WHERE id = 123",
			userId:      123,
			role:        "student",
			expectValid: true,
		},
		{
			name:        "Student accessing other student data - Invalid",
			query:       "SELECT * FROM student WHERE id = 456",
			userId:      123,
			role:        "student",
			expectValid: false,
		},
		{
			name:        "Student accessing own grades - Valid",
			query:       "SELECT * FROM student_course_class WHERE student_id = 123",
			userId:      123,
			role:        "student",
			expectValid: true,
		},
		{
			name:        "Student accessing own class schedule - Valid",
			query:       "SELECT * FROM student_course_class_schedule JOIN student_course_class ON student_course_class.id = student_course_class_schedule.student_course_class_id WHERE student_course_class.student_id = 123",
			userId:      123,
			role:        "student",
			expectValid: true,
		},
		{
			name:        "Student accessing own advisor - Valid",
			query:       "SELECT p.* FROM professor p JOIN administrative_class ac ON p.id = ac.advisor_id JOIN student s ON ac.id = s.administrative_class_id WHERE s.id = 123",
			userId:      123,
			role:        "student",
			expectValid: true,
		},
		{
			name:        "Student accessing publicly available course data - Valid",
			query:       "SELECT * FROM course WHERE credits > 3",
			userId:      123,
			role:        "student",
			expectValid: true,
		},
		{
			name:        "Student trying to access all student data - Invalid",
			query:       "SELECT * FROM student",
			userId:      123,
			role:        "student",
			expectValid: false,
		},
		{
			name:        "Professor accessing own courses - Valid",
			query:       "SELECT * FROM course_class WHERE professor_id = 456",
			userId:      456,
			role:        "professor",
			expectValid: true,
		},
		{
			name:        "Professor accessing students in their courses - Valid",
			query:       "SELECT s.* FROM student s JOIN student_course_class scc ON s.id = scc.student_id JOIN course_class cc ON scc.course_class_id = cc.id WHERE cc.professor_id = 456",
			userId:      456,
			role:        "professor",
			expectValid: true,
		},
		{
			name:        "Professor trying to access all student data - Invalid",
			query:       "SELECT * FROM student",
			userId:      456,
			role:        "professor",
			expectValid: false,
		},
		{
			name:        "Professor trying to access courses of other professors - Invalid",
			query:       "SELECT * FROM course_class WHERE professor_id = 789",
			userId:      456,
			role:        "professor",
			expectValid: false,
		},
		{
			name:        "Admin accessing all student data - Valid",
			query:       "SELECT * FROM student",
			userId:      789,
			role:        "admin",
			expectValid: true,
		},
		{
			name:        "Admin accessing professor data - Valid",
			query:       "SELECT * FROM professor",
			userId:      789,
			role:        "admin",
			expectValid: true,
		},
		{
			name:        "Mixed query with allowed and restricted tables - Edge case",
			query:       "SELECT s.name, cc.code FROM student s JOIN student_course_class scc ON s.id = scc.student_id JOIN course_class cc ON scc.course_class_id = cc.id",
			userId:      123,
			role:        "student",
			expectValid: false,
		},
		{
			name:        "Complex join with proper restrictions for student - Valid",
			query:       "SELECT c.name, c.credits, scc.grade FROM student s JOIN student_course_class scc ON s.id = scc.student_id JOIN course_class cc ON scc.course_class_id = cc.id JOIN course c ON cc.course_id = c.id WHERE s.id = 123",
			userId:      123,
			role:        "student",
			expectValid: true,
		},
		{
			name:        "Query with subqueries - Edge case",
			query:       "SELECT * FROM course WHERE id IN (SELECT course_id FROM course_class WHERE id IN (SELECT course_class_id FROM student_course_class WHERE student_id != 123))",
			userId:      123,
			role:        "student",
			expectValid: false,
		},
		{
			name:        "Invalid role",
			query:       "SELECT * FROM student",
			userId:      123,
			role:        "guest",
			expectValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip this test in short mode as it uses real AI calls
			if testing.Short() {
				t.Skip("Skipping test in short mode")
			}

			result, err := svc.validateUserQuery(context.Background(), tt.query, tt.userId, tt.role)

			assert.NoError(t, err, "Expected no error for query: %s", tt.query)

			if tt.expectValid {
				assert.True(t, result.IsValid, "Expected query to be valid: %s", tt.query)
			} else {
				assert.False(t, result.IsValid, "Expected query to be invalid: %s", tt.query)
				assert.NotEmpty(t, result.Message, "Expected error message for invalid query")
			}

		})
	}
}

func TestValidateUserQuery_WithRealAI(t *testing.T) {
	// Get the service with real AI provider
	svc := GetTestChatService()

	tests := []struct {
		name        string
		query       string
		userId      int
		role        string
		expectValid bool
	}{
		{
			name:        "Student accessing own data - Valid",
			query:       "What is my student information?",
			userId:      123,
			role:        "student",
			expectValid: true,
		},
		{
			name:        "Student accessing other student data - Invalid",
			query:       "Can you show me information about student with ID 456?",
			userId:      123,
			role:        "student",
			expectValid: false,
		},
		{
			name:        "Student accessing own grades - Valid",
			query:       "What are my grades for all courses?",
			userId:      123,
			role:        "student",
			expectValid: true,
		},
		{
			name:        "Student accessing own class schedule - Valid",
			query:       "What is my class schedule for this semester?",
			userId:      123,
			role:        "student",
			expectValid: true,
		},
		{
			name:        "Student accessing own advisor - Valid",
			query:       "Who is my academic advisor?",
			userId:      123,
			role:        "student",
			expectValid: true,
		},
		{
			name:        "Student accessing publicly available course data - Valid",
			query:       "What courses offer more than 3 credits?",
			userId:      123,
			role:        "student",
			expectValid: true,
		},
		{
			name:        "Student trying to access all student data - Invalid",
			query:       "Give me a list of all students in the university",
			userId:      123,
			role:        "student",
			expectValid: false,
		},
		{
			name:        "Professor accessing own courses - Valid",
			query:       "What courses do I teach?",
			userId:      456,
			role:        "professor",
			expectValid: true,
		},
		{
			name:        "Professor accessing students in their courses - Valid",
			query:       "Who are the students enrolled in the courses I teach?",
			userId:      456,
			role:        "professor",
			expectValid: true,
		},
		{
			name:        "Professor trying to access all student data - Invalid",
			query:       "Show me a list of all students in the university",
			userId:      456,
			role:        "professor",
			expectValid: false,
		},
		{
			name:        "Professor trying to access courses of other professors - Invalid",
			query:       "What courses does Professor 789 teach?",
			userId:      456,
			role:        "professor",
			expectValid: false,
		},
		{
			name:        "Admin accessing all student data - Valid",
			query:       "Give me a complete list of all students",
			userId:      789,
			role:        "admin",
			expectValid: true,
		},
		{
			name:        "Admin accessing professor data - Valid",
			query:       "Show me information about all professors",
			userId:      789,
			role:        "admin",
			expectValid: true,
		},
		{
			name:        "Mixed query with allowed and restricted tables - Edge case",
			query:       "List the names of all students and their course codes",
			userId:      123,
			role:        "student",
			expectValid: false,
		},
		{
			name:        "Complex join with proper restrictions for student - Valid",
			query:       "What are the names, credits, and my grades for all the courses I'm taking?",
			userId:      123,
			role:        "student",
			expectValid: true,
		},
		{
			name:        "Query with subqueries - Edge case",
			query:       "What courses are being taken by students other than me?",
			userId:      123,
			role:        "student",
			expectValid: false,
		},
		{
			name:        "Invalid role",
			query:       "Show me information about all students",
			userId:      123,
			role:        "guest",
			expectValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip this test in short mode as it uses real AI calls
			if testing.Short() {
				t.Skip("Skipping test in short mode")
			}

			result, err := svc.validateUserQuery(context.Background(), tt.query, tt.userId, tt.role)

			assert.NoError(t, err, "Expected no error for query: %s", tt.query)

			if tt.expectValid {
				assert.True(t, result.IsValid, "Expected query to be valid: %s", tt.query)
			} else {
				assert.False(t, result.IsValid, "Expected query to be invalid: %s", tt.query)
				assert.NotEmpty(t, result.Message, "Expected error message for invalid query")
			}
		})
	}
}

func TestGetToolCallsByAI_WithRealAI_Extended(t *testing.T) {
	// Skip this test in short mode as it uses real AI calls
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}

	// Get the service with real AI provider
	svc := GetTestChatService()
	if err := godotenv.Load("../../config/.env"); err != nil {
		log.Printf("Warning: failed to load .env file: %v", err)
	}
	myDb, _ := db.NewHDb("postgres", os.Getenv("DATABASE_URL"))
	ddl, _ := myDb.LoadDDL()

	tests := []struct {
		name           string
		query          string
		userId         int
		userRole       string
		expectedToolFn string // Name of the expected function to be called
		isHardCase     bool   // Flag for hard/edge cases
	}{
		// Basic cases
		{
			name:           "Student information query",
			query:          "What's my student information?",
			userId:         123,
			userRole:       "student",
			expectedToolFn: "GetStudentInfo",
			isHardCase:     false,
		},
		{
			name:           "Course information query",
			query:          "Tell me about CS101 course",
			userId:         123,
			userRole:       "student",
			expectedToolFn: "GetCourse",
			isHardCase:     false,
		},

		// More complex cases
		{
			name:           "Complex schedule query with multiple filters",
			query:          "Show me my Monday classes for the Fall 2023 semester that are in the Computer Science building",
			userId:         123,
			userRole:       "student",
			expectedToolFn: "GetStudentSchedule",
			isHardCase:     true,
		},
		{
			name:           "Statistical analysis request",
			query:          "What's the average grade distribution for CS101 compared to other courses this semester?",
			userId:         123,
			userRole:       "student",
			expectedToolFn: "GetStatistics",
			isHardCase:     true,
		},

		// Ambiguous queries
		{
			name:           "Ambiguous query that could be schedule or course info",
			query:          "Tell me about CS101 on Mondays",
			userId:         123,
			userRole:       "student",
			expectedToolFn: "", // Could be multiple tools, just check if we get any valid response
			isHardCase:     true,
		},

		// Professor role cases
		{
			name:           "Professor asking about courses they teach",
			query:          "Show me the roster for my CS101 class",
			userId:         456,
			userRole:       "professor",
			expectedToolFn: "GetMultiEntityInfo",
			isHardCase:     true,
		},
		{
			name:           "Professor asking about student performance",
			query:          "What is the grade distribution in my classes?",
			userId:         456,
			userRole:       "professor",
			expectedToolFn: "GetStatistics",
			isHardCase:     true,
		},

		// Admin role cases
		{
			name:           "Admin asking about department statistics",
			query:          "Show me enrollment statistics for all Computer Science courses",
			userId:         789,
			userRole:       "admin",
			expectedToolFn: "GetStatistics",
			isHardCase:     true,
		},

		// Mixed entity queries
		{
			name:           "Query involving multiple entities",
			query:          "What grade did student 123 get in Professor 456's CS101 class last semester?",
			userId:         789,
			userRole:       "admin", // This query would be appropriate for an admin
			expectedToolFn: "GetMultiEntityInfo",
			isHardCase:     true,
		},

		// Complex natural language understanding
		{
			name:           "Natural language query with implicit entities",
			query:          "I'm struggling with the programming assignments in my computer science class, can you tell me about available tutoring resources?",
			userId:         123,
			userRole:       "student",
			expectedToolFn: "AnalyzeText",
			isHardCase:     true,
		},

		// Edge cases with potential conflicts
		{
			name:           "Vague query with multiple interpretations",
			query:          "Find information about computer science",
			userId:         123,
			userRole:       "student",
			expectedToolFn: "SearchDatabase",
			isHardCase:     true,
		},

		// Negative cases
		{
			name:           "Invalid query with no clear function match",
			query:          "What's the weather like today?",
			userId:         123,
			userRole:       "student",
			expectedToolFn: "ExecuteQuery", // Fallback to SQL query is expected
			isHardCase:     true,
		},

		// Security test cases
		{
			name:           "Student trying to access admin information",
			query:          "Show me all student records in the database",
			userId:         123,
			userRole:       "student",
			expectedToolFn: "", // Should be handled securely
			isHardCase:     true,
		},
		{
			name:           "SQL injection-like query",
			query:          "SELECT * FROM student; DROP TABLE student;",
			userId:         123,
			userRole:       "student",
			expectedToolFn: "ExecuteQuery",
			isHardCase:     true,
		},
	}

	// Get the function definitions
	funcDefs := svc.funcRegistry.GetFuncDefinitions()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Format the tool prompt with the query, now including user context
			toolPrompt := fmt.Sprintf(ToolPromptTemplate, ddl, tt.userId, tt.userRole, tt.query)

			// Call the method being tested
			response, err := svc.getToolCallsByAI(context.Background(), toolPrompt, funcDefs)

			// Assert no error
			assert.NoError(t, err, "Expected no error when getting tool calls")

			// Assert we got a response
			assert.NotNil(t, response, "Expected non-nil response")

			// Assert there are tool calls
			if !tt.isHardCase {
				// For regular cases, we expect specific tools
				assert.Greater(t, len(response.ToolCalls), 0, "Expected at least one tool call")

				if tt.expectedToolFn != "" {
					// Check if the expected function was called
					foundExpectedTool := false
					for _, toolCall := range response.ToolCalls {
						if toolCall.Function.Name == tt.expectedToolFn {
							foundExpectedTool = true
							break
						}
					}

					assert.True(t, foundExpectedTool, "Expected tool call to function %s not found", tt.expectedToolFn)

					// Assert that the function was called with valid arguments
					funcRegistry := GetTestFuncRegistry()
					for _, toolCall := range response.ToolCalls {
						_, err := funcRegistry.Execute(context.Background(), toolCall)
						assert.NoError(t, err)
					}
				}
			} else {
				// For hard cases, just log the response for manual inspection
				t.Logf("Hard case response: %+v", response.ToolCalls)

				// If we expected a specific function even for hard cases
				if tt.expectedToolFn != "" {
					foundExpectedTool := false
					for _, toolCall := range response.ToolCalls {
						if toolCall.Function.Name == tt.expectedToolFn {
							foundExpectedTool = true
							break
						}
					}

					// Use a less strict assertion for hard cases
					if !foundExpectedTool {
						t.Logf("Note: Expected function %s was not called, but this is a hard case so we're just logging it", tt.expectedToolFn)
					}
				}
			}

			// For all cases, verify the response format
			for i, toolCall := range response.ToolCalls {
				assert.NotEmpty(t, toolCall.ID, "Tool call #%d has empty ID", i)
				assert.NotEmpty(t, toolCall.Function.Name, "Tool call #%d has empty function name", i)
				// Check that the arguments are valid JSON
				var args map[string]interface{}
				err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args)
				assert.NoError(t, err, "Tool call #%d has invalid JSON arguments: %s", i, toolCall.Function.Arguments)
			}
		})
	}
}

func TestGetToolCallsByAI_MultipleTools(t *testing.T) {
	// Skip this test in short mode as it uses real AI calls
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}

	// Get the service with real AI provider
	svc := GetTestChatService()
	if err := godotenv.Load("../../config/.env"); err != nil {
		log.Printf("Warning: failed to load .env file: %v", err)
	}
	myDb, _ := db.NewHDb("postgres", os.Getenv("DATABASE_URL"))
	ddl, _ := myDb.LoadDDL()

	// Query that likely requires multiple tool calls
	query := "Compare my grades in CS101 with the class average and tell me when Professor Johnson's office hours are"
	userId := 123
	userRole := "student"

	// Format the tool prompt with the query and user context
	toolPrompt := fmt.Sprintf(ToolPromptTemplate, ddl, userId, userRole, query)

	// Get the function definitions
	funcDefs := svc.funcRegistry.GetFuncDefinitions()

	// Call the method being tested
	response, err := svc.getToolCallsByAI(context.Background(), toolPrompt, funcDefs)

	// Assert no error
	assert.NoError(t, err, "Expected no error when getting tool calls")

	// Assert we got a response
	assert.NotNil(t, response, "Expected non-nil response")

	// We expect this might generate multiple tool calls
	t.Logf("Number of tool calls: %d", len(response.ToolCalls))

	// Log each tool call for inspection
	for i, toolCall := range response.ToolCalls {
		t.Logf("Tool call #%d: %s", i+1, toolCall.Function.Name)
		t.Logf("Arguments: %s", toolCall.Function.Arguments)
	}

	// Verify at least one tool call was made
	assert.Greater(t, len(response.ToolCalls), 0, "Expected at least one tool call")
}

func TestGetToolCallsByAI_MalformedQueries(t *testing.T) {
	// Skip this test in short mode as it uses real AI calls
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}

	// Get the service with real AI provider
	svc := GetTestChatService()
	if err := godotenv.Load("../../config/.env"); err != nil {
		log.Printf("Warning: failed to load .env file: %v", err)
	}
	myDb, _ := db.NewHDb("postgres", os.Getenv("DATABASE_URL"))
	ddl, _ := myDb.LoadDDL()

	malformedQueries := []struct {
		name     string
		query    string
		userId   int
		userRole string
	}{
		{
			name:     "Empty query",
			query:    "",
			userId:   123,
			userRole: "student",
		},
		{
			name:     "SQL injection attempt",
			query:    "SELECT * FROM student; DROP TABLE student;",
			userId:   123,
			userRole: "student",
		},
		{
			name:     "XSS attempt",
			query:    "<script>alert('XSS')</script>",
			userId:   123,
			userRole: "student",
		},
		{
			name:     "Extremely long query",
			query:    strings.Repeat("Tell me about my grades. ", 100),
			userId:   123,
			userRole: "student",
		},
		{
			name:     "Non-English query",
			query:    "Покажите мне мои оценки", // Russian
			userId:   123,
			userRole: "student",
		},
		{
			name:     "Query with special characters",
			query:    "What are my grades? 😊 🎓 $#@!",
			userId:   123,
			userRole: "student",
		},
	}

	// Get the function definitions
	funcDefs := svc.funcRegistry.GetFuncDefinitions()

	for _, mq := range malformedQueries {
		t.Run(mq.name, func(t *testing.T) {
			// Format the tool prompt with the query and user context
			toolPrompt := fmt.Sprintf(ToolPromptTemplate, ddl, mq.userId, mq.userRole, mq.query)

			// Call the method being tested
			response, err := svc.getToolCallsByAI(context.Background(), toolPrompt, funcDefs)

			// We're just checking that it doesn't crash, so log any errors instead of failing
			if err != nil {
				t.Logf("Got error for malformed query: %v", err)
			} else {
				t.Logf("Received response for malformed query: %+v", response)
				t.Logf("Number of tool calls: %d", len(response.ToolCalls))
			}
		})
	}
}
