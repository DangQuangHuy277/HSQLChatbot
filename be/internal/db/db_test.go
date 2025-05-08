package db

import (
	"context"
	pgquery "github.com/pganalyze/pg_query_go/v6"
	"testing"
)

func TestExtractTables(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		expected []TableInfo
		wantErr  bool
	}{
		{
			name:  "simple table reference",
			query: "SELECT * FROM student",
			expected: []TableInfo{
				{Name: "student", Alias: ""},
			},
			wantErr: false,
		},
		{
			name:  "table with alias",
			query: "SELECT * FROM student s",
			expected: []TableInfo{
				{Name: "student", Alias: "s"},
			},
			wantErr: false,
		},
		{
			name:  "multiple tables",
			query: "SELECT * FROM student, professor",
			expected: []TableInfo{
				{Name: "student", Alias: ""},
				{Name: "professor", Alias: ""},
			},
			wantErr: false,
		},
		{
			name:  "simple join",
			query: "SELECT * FROM student s JOIN course c ON s.course_id = c.id",
			expected: []TableInfo{
				{Name: "student", Alias: "s"},
				{Name: "course", Alias: "c"},
			},
			wantErr: false,
		},
		{
			name:  "multiple joins",
			query: "SELECT * FROM student s JOIN course c ON s.course_id = c.id JOIN professor p ON c.professor_id = p.id",
			expected: []TableInfo{
				{Name: "student", Alias: "s"},
				{Name: "course", Alias: "c"},
				{Name: "professor", Alias: "p"},
			},
			wantErr: false,
		},
		{
			name:  "multiple tables with aliases",
			query: "SELECT * FROM student s, professor p",
			expected: []TableInfo{
				{Name: "student", Alias: "s"},
				{Name: "professor", Alias: "p"},
			},
			wantErr: false,
		},
		{
			name: "simple subquery",
			query: `SELECT * FROM (
					SELECT name FROM student WHERE age > 20
				) AS adult_students`,
			expected: []TableInfo{
				{Name: "student", Alias: ""},
				{Name: "derived_table", Alias: "adult_students"},
			},
			wantErr: false,
		},
		{
			name: "complex subquery with joins",
			query: `SELECT * FROM (
					SELECT s.name FROM student s 
					JOIN course c ON s.course_id = c.id
				) AS student_courses`,
			expected: []TableInfo{
				{Name: "student", Alias: "s"},
				{Name: "course", Alias: "c"},
				{Name: "derived_table", Alias: "student_courses"},
			},
			wantErr: false,
		},
		{
			name: "join with subquery",
			query: `SELECT * FROM professor p 
					JOIN (SELECT * FROM course WHERE credits > 3) AS c 
					ON p.id = c.professor_id`,
			expected: []TableInfo{
				{Name: "professor", Alias: "p"},
				{Name: "course", Alias: ""},
				{Name: "derived_table", Alias: "c"},
			},
			wantErr: false,
		},
		{
			name: "nested subqueries",
			query: `SELECT * FROM (
					SELECT * FROM (
						SELECT * FROM student WHERE age > 20
					) AS adults WHERE gender = 'F'
				) AS adult_females`,
			expected: []TableInfo{
				{Name: "student", Alias: ""},
				{Name: "derived_table", Alias: "adults"},
				{Name: "derived_table", Alias: "adult_females"},
			},
			wantErr: false,
		},
		{
			name: "complex join structure",
			query: `SELECT * FROM faculty f
					LEFT JOIN department d ON f.id = d.faculty_id
					RIGHT JOIN professor p ON d.id = p.department_id
					FULL OUTER JOIN course c ON p.id = c.professor_id`,
			expected: []TableInfo{
				{Name: "faculty", Alias: "f"},
				{Name: "department", Alias: "d"},
				{Name: "professor", Alias: "p"},
				{Name: "course", Alias: "c"},
			},
			wantErr: false,
		},
		{
			name: "join with multiple conditions",
			query: `SELECT * FROM student s 
					JOIN administrative_class ac 
					ON s.class_id = ac.id AND s.year = ac.year`,
			expected: []TableInfo{
				{Name: "student", Alias: "s"},
				{Name: "administrative_class", Alias: "ac"},
			},
			wantErr: false,
		},
		{
			name: "CTE (Common Table Expression)",
			query: `WITH student_summary AS (
					SELECT program_id, COUNT(*) AS student_count 
					FROM student GROUP BY program_id
				)
				SELECT p.name, ss.student_count 
				FROM program p JOIN student_summary ss ON p.id = ss.program_id`,
			expected: []TableInfo{
				{Name: "program", Alias: "p"},
				{Name: "student_summary", Alias: "ss"},
			},
			wantErr: false,
		},
		{
			name:  "cross join",
			query: "SELECT * FROM student CROSS JOIN course",
			expected: []TableInfo{
				{Name: "student", Alias: ""},
				{Name: "course", Alias: ""},
			},
			wantErr: false,
		},
		{
			name: "lateral join",
			query: `SELECT * FROM department d, 
					LATERAL (SELECT * FROM professor WHERE department_id = d.id) p`,
			expected: []TableInfo{
				{Name: "department", Alias: "d"},
				{Name: "professor", Alias: ""},
				{Name: "derived_table", Alias: "p"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the query
			tree, err := pgquery.Parse(tt.query)
			if err != nil {
				t.Fatalf("Failed to parse query: %v", err)
			}

			if len(tree.Stmts) == 0 {
				t.Fatal("No statements found in parsed query")
			}

			stmt := tree.Stmts[0].Stmt.GetSelectStmt()
			if stmt == nil {
				t.Fatal("Not a SELECT statement")
			}

			// Call the function being tested
			tables, err := extractTables(stmt)

			// Check error condition
			if (err != nil) != tt.wantErr {
				t.Errorf("extractTables() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// If we expect an error but didn't get one, no need to check tables
			if tt.wantErr {
				return
			}

			// Check if we got all expected tables (ignoring order)
			if len(tables) != len(tt.expected) {
				t.Errorf("extractTables() got %d tables, want %d tables", len(tables), len(tt.expected))
				t.Errorf("Got tables: %v", tables)
				t.Errorf("Expected tables: %v", tt.expected)
				return
			}

			// Create maps for easier comparison
			gotMap := make(map[string]string) // name -> alias
			for _, table := range tables {
				key := table.Name
				if table.Alias != "" {
					gotMap[key+":"+table.Alias] = table.Alias
				} else {
					gotMap[key] = table.Alias
				}
			}

			expectedMap := make(map[string]string) // name -> alias
			for _, table := range tt.expected {
				key := table.Name
				if table.Alias != "" {
					expectedMap[key+":"+table.Alias] = table.Alias
				} else {
					expectedMap[key] = table.Alias
				}
			}

			// Compare maps
			for key, val := range expectedMap {
				if gotVal, ok := gotMap[key]; !ok || gotVal != val {
					t.Errorf("Missing or incorrect table entry: %s", key)
				}
			}
		})
	}
}

func TestValidateQuery(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		userRole   string
		userId     string
		wantValid  bool
		wantErrMsg string
	}{
		// Basic query validation tests
		{
			name:       "empty query",
			query:      "",
			userRole:   "student",
			userId:     "1",
			wantValid:  false,
			wantErrMsg: "Empty query",
		},
		{
			name:       "non-select query (INSERT)",
			query:      "INSERT INTO student (name, email) VALUES ('John', 'john@example.com')",
			userRole:   "admin",
			userId:     "1",
			wantValid:  false,
			wantErrMsg: "Only SELECT queries are allowed",
		},
		{
			name:       "non-select query (UPDATE)",
			query:      "UPDATE student SET name = 'John' WHERE id = 1",
			userRole:   "admin",
			userId:     "1",
			wantValid:  false,
			wantErrMsg: "Only SELECT queries are allowed",
		},
		{
			name:       "non-select query (DELETE)",
			query:      "DELETE FROM student WHERE id = 1",
			userRole:   "admin",
			userId:     "1",
			wantValid:  false,
			wantErrMsg: "Only SELECT queries are allowed",
		},
		{
			name:       "student accessing own data with IN clause",
			query:      "SELECT * FROM student WHERE id IN (1, 3, 5)",
			userRole:   "student",
			userId:     "1",
			wantValid:  false,
			wantErrMsg: "Students can only access their own personal info",
		},

		// Role-based access control tests - Admin
		{
			name:       "admin accessing public table",
			query:      "SELECT * FROM course",
			userRole:   "admin",
			userId:     "1",
			wantValid:  true,
			wantErrMsg: "",
		},
		{
			name:       "admin accessing student table",
			query:      "SELECT * FROM student",
			userRole:   "admin",
			userId:     "1",
			wantValid:  true,
			wantErrMsg: "",
		},
		{
			name:       "admin accessing professor table",
			query:      "SELECT * FROM professor",
			userRole:   "admin",
			userId:     "1",
			wantValid:  true,
			wantErrMsg: "",
		},

		// Role-based access control tests - Student
		{
			name:       "student accessing public table",
			query:      "SELECT * FROM course",
			userRole:   "student",
			userId:     "1",
			wantValid:  true,
			wantErrMsg: "",
		},
		{
			name:       "student accessing own data",
			query:      "SELECT * FROM student WHERE id = 1",
			userRole:   "student",
			userId:     "1",
			wantValid:  true,
			wantErrMsg: "",
		},
		{
			name:       "student accessing another student's data",
			query:      "SELECT * FROM student WHERE id = 2",
			userRole:   "student",
			userId:     "1",
			wantValid:  false,
			wantErrMsg: "Students can only access their own personal info",
		},
		{
			name:       "student accessing own course data with JOIN",
			query:      "SELECT * FROM course_class_enrollment scc JOIN student s ON scc.student_id = s.id WHERE s.id = 1",
			userRole:   "student",
			userId:     "1",
			wantValid:  true,
			wantErrMsg: "",
		},
		{
			name:       "student accessing schedule with subquery",
			query:      "SELECT * FROM course_class_schedule ccs WHERE ccs.course_class_id IN (SELECT cc.id FROM course_class cc JOIN course_class_enrollment scc ON cc.id = scc.course_class_id JOIN student s ON scc.student_id = s.id WHERE s.id = 1)",
			userRole:   "student",
			userId:     "1",
			wantValid:  true,
			wantErrMsg: "",
		},

		// Role-based access control tests - Professor
		{
			name:       "professor accessing public table",
			query:      "SELECT * FROM course",
			userRole:   "professor",
			userId:     "1",
			wantValid:  true,
			wantErrMsg: "",
		},
		{
			name:       "professor accessing own data",
			query:      "SELECT * FROM professor WHERE id = 1",
			userRole:   "professor",
			userId:     "1",
			wantValid:  true,
			wantErrMsg: "",
		},
		{
			name:       "professor accessing another professor's data",
			query:      "SELECT * FROM professor WHERE id = 2",
			userRole:   "professor",
			userId:     "1",
			wantValid:  false,
			wantErrMsg: "Professors can only access their own personal info",
		},
		{
			name:       "professor accessing courses they teach",
			query:      "SELECT * FROM course_class WHERE professor_id = 1",
			userRole:   "professor",
			userId:     "1",
			wantValid:  true,
			wantErrMsg: "",
		},
		{
			name:       "professor accessing courses they don't teach",
			query:      "SELECT * FROM course_class WHERE professor_id = 2",
			userRole:   "professor",
			userId:     "1",
			wantValid:  false,
			wantErrMsg: "Professors can only access courses they teach",
		},

		// Complex query tests
		{
			name:      "query with implicit join",
			query:     "SELECT s.name, c.name FROM student s, course c WHERE s.course_id = c.id",
			userRole:  "student",
			userId:    "1",
			wantValid: true,
		},
		{
			name:       "complex query with multiple joins - student",
			query:      "SELECT c.name, cc.code, ccs.day_of_week, ccs.lesson_range FROM course c JOIN course_class cc ON c.id = cc.course_id JOIN course_class_schedule ccs ON cc.id = ccs.course_class_id JOIN course_class_enrollment scc ON cc.id = scc.course_class_id JOIN student s ON scc.student_id = s.id WHERE s.id = 1",
			userRole:   "student",
			userId:     "1",
			wantValid:  true,
			wantErrMsg: "",
		},
		{
			name: "student access with complex join conditions and complex ON clause",
			query: `SELECT c.name, ccs.day_of_week
              FROM student s
              JOIN course_class_enrollment scc ON (scc.student_id = s.id AND s.id = 1) 
                 OR (s.id = 1 AND scc.enrollment_type = 'regular')
                 OR s.id IN (SELECT 1 WHERE TRUE)
              JOIN course_class cc ON cc.id = scc.course_class_id 
                 AND cc.semester = (SELECT 'current' FROM course LIMIT 1)
              JOIN course c ON c.id = cc.course_id
              JOIN course_class_schedule ccs ON ccs.course_class_id = cc.id 
                 AND (ccs.day_of_week = 'Monday' OR ccs.day_of_week = 'Wednesday')
                 AND EXISTS (SELECT 1 FROM student WHERE id = 1)`,
			userRole:   "student",
			userId:     "1",
			wantValid:  true,
			wantErrMsg: "",
		},
		{
			name:      "SEMI JOIN - professor accessing taught courses",
			query:     "SELECT c.* FROM course c WHERE EXISTS (SELECT 1 FROM course_class cc WHERE cc.course_id = c.id AND cc.professor_id = 1)",
			userRole:  "professor",
			userId:    "1",
			wantValid: true,
		},
		{
			name:       "ANTI JOIN - student accessing others' data",
			query:      "SELECT s.* FROM student s WHERE NOT EXISTS (SELECT 1 FROM administrative_class ac WHERE ac.id = s.administrative_class_id AND s.id = 1)",
			userRole:   "student",
			userId:     "1",
			wantValid:  false,
			wantErrMsg: "Students can only access their own personal info",
		},
		{
			name: "subquery in WHERE clause with access control in outer query",
			query: `SELECT s.name, s.email, s.code 
              FROM student s 
              WHERE s.id = 1 
              AND s.program_id IN (
                SELECT cp.program_id 
                FROM course_program cp 
                JOIN course c ON cp.course_id = c.id 
                WHERE c.credits > 3
              )`,
			userRole:   "student",
			userId:     "1",
			wantValid:  true,
			wantErrMsg: "",
		},
		{
			name: "complex subquery in FROM clause",
			query: `SELECT outer_query.* FROM 
                (SELECT s.id, s.name, 
                  (SELECT COUNT(*) FROM course_class_enrollment scc 
                   WHERE scc.student_id = s.id) as course_count,
                  (SELECT AVG(CAST(scc2.grade AS FLOAT)) FROM course_class_enrollment scc2
                   WHERE scc2.student_id = s.id AND scc2.grade ~ '^[0-9]+(\.[0-9]+)?$') as avg_grade
                 FROM student s
                 WHERE s.id = 1) AS outer_query
              WHERE outer_query.course_count > 0`,
			userRole:   "student",
			userId:     "1",
			wantValid:  true,
			wantErrMsg: "",
		},
		{
			name: "complex subquery in FROM clause with outer query access control",
			query: `SELECT student_data.* 
              FROM student s
              JOIN (
                SELECT scc.student_id,
                       c.name as course_name,
                       cc.code as course_code,
                       AVG(CAST(scc.grade AS FLOAT)) as average_grade,
                       COUNT(DISTINCT cc.id) as course_count
                FROM course_class_enrollment scc
                JOIN course_class cc ON scc.course_class_id = cc.id
                JOIN course c ON cc.course_id = c.id
                GROUP BY scc.student_id, c.name, cc.code
              ) AS student_data ON s.id = student_data.student_id
              WHERE s.id = 1`,
			userRole:   "student",
			userId:     "1",
			wantValid:  true,
			wantErrMsg: "",
		},
		{
			name:       "complex query with CTEs",
			query:      `WITH enrolled_students AS (SELECT scc.student_id FROM student_course_class scc JOIN course_class cc ON scc.course_class_id = cc.id WHERE cc.professor_id = 1) SELECT s.* FROM student s JOIN enrolled_students es ON s.id = es.student_id`,
			userRole:   "professor",
			userId:     "1",
			wantValid:  true,
			wantErrMsg: "",
		},
		{
			name:       "query with UNION",
			query:      "SELECT id, name FROM course WHERE id = 1 UNION SELECT id, name FROM program WHERE id = 2",
			userRole:   "student",
			userId:     "1",
			wantValid:  true,
			wantErrMsg: "",
		},

		// Edge cases and special queries
		{
			name:       "query with table aliases",
			query:      "SELECT s.* FROM student s WHERE s.id = 1",
			userRole:   "student",
			userId:     "1",
			wantValid:  true,
			wantErrMsg: "",
		},
		{
			name:       "query with EXISTS subquery",
			query:      "SELECT c.* FROM course c WHERE EXISTS (SELECT 1 FROM course_program cp WHERE cp.course_id = c.id AND cp.program_id = 1)",
			userRole:   "student",
			userId:     "1",
			wantValid:  true,
			wantErrMsg: "",
		},
		{
			name: "complex union with CTEs and nested subqueries",
			query: `WITH student_courses AS (
                SELECT 
                  s.id, 
                  s.name, 
                  c.name as course_name,
                  cc.semester
                FROM student s
                JOIN student_course_class scc ON s.id = scc.student_id
                JOIN course_class cc ON scc.course_class_id = cc.id
                JOIN course c ON cc.course_id = c.id
                WHERE s.id = 1
              ),
              student_grades AS (
                SELECT 
                  s.id, 
                  AVG(CAST(scc.grade AS FLOAT)) as avg_grade,
                  COUNT(scc.id) as graded_courses
                FROM student s
                JOIN student_course_class scc ON s.id = scc.student_id
                WHERE s.id = 1 AND scc.grade IS NOT NULL
                GROUP BY s.id
              )
              SELECT 
                'Current Courses' as data_type,
                sc.name as student_name,
                sc.course_name,
                sc.semester,
                sg.avg_grade,
                NULL as room_number
              FROM student_courses sc
              JOIN student_grades sg ON sc.id = sg.id
              WHERE sc.semester = 'current'
              
              UNION ALL
              
              SELECT 
                'Upcoming Classes' as data_type,
                s.name as student_name,
                c.name as course_name,
                cc.semester,
                NULL as avg_grade,
                (SELECT ccs.room_number 
                 FROM course_class_schedule ccs 
                 WHERE ccs.course_class_id = cc.id 
                 LIMIT 1) as room_number
              FROM student s
              JOIN student_course_class scc ON s.id = scc.student_id
              JOIN course_class cc ON scc.course_class_id = cc.id
              JOIN course c ON cc.course_id = c.id
              WHERE s.id = 1 AND cc.semester = 'upcoming'
              
              ORDER BY semester, course_name`,
			userRole:   "student",
			userId:     "1",
			wantValid:  true,
			wantErrMsg: "",
		},
		{
			name:       "unknown role",
			query:      "SELECT * FROM course",
			userRole:   "guest",
			userId:     "1",
			wantValid:  false,
			wantErrMsg: "Unknown role: guest",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a context with user information
			ctx := context.WithValue(context.Background(), "userId", tt.userId)
			ctx = context.WithValue(ctx, "userRole", tt.userRole)

			// Call the function being tested
			gotValid, gotErrMsg, err := validateQuery(ctx, tt.query)

			// Check for unexpected errors
			if err != nil && tt.wantValid {
				t.Errorf("validateQuery() unexpected error: %v", err)
				return
			}

			// Check if validity matches expected
			if gotValid != tt.wantValid {
				t.Errorf("validateQuery() gotValid = %v, want %v", gotValid, tt.wantValid)
			}

			// Check error message if we expect an error
			if !tt.wantValid && tt.wantErrMsg != gotErrMsg {
				t.Errorf("validateQuery() gotErrMsg = %v, want %v", gotErrMsg, tt.wantErrMsg)
			}
		})
	}
}
