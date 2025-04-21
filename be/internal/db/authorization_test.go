package db

import (
	"fmt"
	pgquery "github.com/pganalyze/pg_query_go/v6"
	"testing"
)

func TestAuthorizeSelectStmt(t *testing.T) {
	tests := []struct {
		name           string
		query          string
		role           string
		userId         int
		wantAuthorized bool
		wantRemaining  []TableInfo
		wantErr        bool
	}{
		// Simple SELECT tests
		{
			name:           "simple select from public table",
			query:          "SELECT * FROM course",
			role:           "student",
			userId:         1,
			wantAuthorized: true,
			wantRemaining:  []TableInfo{},
			wantErr:        false,
		},
		{
			name:           "student accessing own data",
			query:          "SELECT * FROM student WHERE id = 1",
			role:           "student",
			userId:         1,
			wantAuthorized: true,
			wantRemaining:  []TableInfo{},
			wantErr:        false,
		},
		{
			name:           "student accessing other student data",
			query:          "SELECT * FROM student WHERE id = 2",
			role:           "student",
			userId:         1,
			wantAuthorized: false,
			wantRemaining:  []TableInfo{{Name: "student"}},
			wantErr:        false,
		},

		// UNION tests
		{
			name:           "union with both sides authorized",
			query:          "SELECT * FROM course UNION SELECT * FROM course_class WHERE professor_id = 1",
			role:           "professor",
			userId:         1,
			wantAuthorized: true,
			wantRemaining:  []TableInfo{},
			wantErr:        false,
		},
		{
			name:           "union with one side unauthorized",
			query:          "SELECT * FROM professor WHERE id = 2 UNION SELECT * FROM course",
			role:           "professor",
			userId:         1,
			wantAuthorized: false,
			wantRemaining:  []TableInfo{{Name: "professor"}},
			wantErr:        false,
		},

		// INTERSECT tests
		{
			name:           "intersect with both sides authorized",
			query:          "SELECT * FROM course INTERSECT SELECT * FROM course_program",
			role:           "student",
			userId:         1,
			wantAuthorized: true,
			wantRemaining:  []TableInfo{},
			wantErr:        false,
		},
		{
			name:           "intersect with only one side authorized",
			query:          "SELECT * FROM course INTERSECT SELECT * FROM student WHERE id = 2",
			role:           "student",
			userId:         1,
			wantAuthorized: true,
			wantRemaining:  []TableInfo{},
			wantErr:        false,
		},

		// EXCEPT tests
		{
			name:           "except with left side authorized",
			query:          "SELECT * FROM course EXCEPT SELECT * FROM student",
			role:           "student",
			userId:         1,
			wantAuthorized: true,
			wantRemaining:  []TableInfo{},
			wantErr:        false,
		},
		{
			name:           "except operation with unauthorized data that becomes authorized",
			query:          "SELECT course_id FROM student_course_class WHERE student_id = 2 EXCEPT SELECT course_id FROM student_course_class WHERE student_id = 1",
			role:           "student",
			userId:         1,
			wantAuthorized: false,
			wantRemaining:  []TableInfo{},
			wantErr:        false,
		},

		// CTE (WITH) tests
		{
			name:           "simple CTE with authorized tables",
			query:          "WITH course_counts AS (SELECT COUNT(*) FROM course) SELECT * FROM course_counts",
			role:           "student",
			userId:         1,
			wantAuthorized: true,
			wantRemaining:  []TableInfo{},
			wantErr:        false,
		},
		{
			name:           "CTE with unauthorized table",
			query:          "WITH student_data AS (SELECT * FROM student WHERE id = 2) SELECT * FROM student_data",
			role:           "student",
			userId:         1,
			wantAuthorized: false,
			wantRemaining:  []TableInfo{{Name: "student"}, {Name: "student_data"}},
			wantErr:        false,
		},

		// Complex FROM clause tests
		{
			name:           "join with all tables authorized",
			query:          "SELECT * FROM course c JOIN course_program cp ON c.id = cp.course_id",
			role:           "student",
			userId:         1,
			wantAuthorized: true,
			wantRemaining:  []TableInfo{},
			wantErr:        false,
		},
		{
			name:           "join with one table unauthorized",
			query:          "SELECT * FROM course c JOIN student s ON c.id = s.id",
			role:           "student",
			userId:         1,
			wantAuthorized: false,
			wantRemaining:  []TableInfo{{Name: "student", Alias: "s"}},
			wantErr:        false,
		},

		// WHERE clause tests
		{
			name:           "where clause with conditions authorizing access",
			query:          "SELECT * FROM student WHERE id = 1",
			role:           "student",
			userId:         1,
			wantAuthorized: true,
			wantRemaining:  []TableInfo{},
			wantErr:        false,
		},
		{
			name:           "where clause with subquery",
			query:          "SELECT * FROM course WHERE id IN (SELECT course_id FROM student_course_class WHERE student_id = 1)",
			role:           "student",
			userId:         1,
			wantAuthorized: true,
			wantRemaining:  []TableInfo{},
			wantErr:        false,
		},

		// Complex combined tests
		{
			name:           "complex query with joins and subqueries",
			query:          "SELECT c.name, cc.code, c.*, * FROM course c JOIN course_class cc ON c.id = cc.course_id WHERE cc.id IN (SELECT course_class_id FROM student_course_class WHERE student_id = 1)",
			role:           "student",
			userId:         1,
			wantAuthorized: true,
			wantRemaining:  []TableInfo{},
			wantErr:        false,
		},
		{
			name:           "query with CTE, joins, and where clause",
			query:          "WITH courses_taken AS (SELECT course_class_id FROM student_course_class WHERE student_id = 1) SELECT c.name FROM course c JOIN course_class cc ON c.id = cc.course_id JOIN courses_taken ct ON cc.id = ct.course_class_id",
			role:           "student",
			userId:         1,
			wantAuthorized: true,
			wantRemaining:  []TableInfo{},
			wantErr:        false,
		},

		// Edge cases
		{
			name:           "query with INTO clause (not permitted)",
			query:          "SELECT * INTO temp_table FROM course",
			role:           "admin",
			userId:         1,
			wantAuthorized: false,
			wantRemaining:  nil,
			wantErr:        false,
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

			// Extract tables from the query for initial tables input
			tables, err := extractTables(stmt)
			if err != nil {
				t.Fatalf("Failed to extract tables: %v", err)
			}

			// Call the function being tested
			service := NewAuthorizationService(NewSchemaServiceImpl())
			authRes, err := service.authorizeSelectStmt(stmt, tables, tt.role, tt.userId)

			// Check for expected errors
			if (err != nil) != tt.wantErr {
				t.Errorf("authorizeSelectStmt() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Check authorization result
			if authRes.Authorized != tt.wantAuthorized {
				t.Errorf("authorizeSelectStmt() gotAuthorized = %v, want %v", authRes.Authorized, tt.wantAuthorized)
			}

			// Check remaining tables (ignoring order)
			if !compareTableInfoSlices(authRes.UnAuthorizedTables, tt.wantRemaining) {
				t.Errorf("authorizeSelectStmt() gotRemaining = %v, want %v", authRes.UnAuthorizedTables, tt.wantRemaining)
			}
		})
	}
}

func TestAuthorizeJoinExpr(t *testing.T) {
	tests := []struct {
		name           string
		query          string
		role           string
		userId         int
		wantAuthorized bool
		wantRemaining  []TableInfo
		wantErr        bool
	}{
		// Simple JOIN tests
		{
			name:           "simple join with authorized tables",
			query:          "SELECT * FROM course c JOIN course_program cp ON c.id = cp.course_id",
			role:           "student",
			userId:         1,
			wantAuthorized: true,
			wantRemaining:  []TableInfo{},
			wantErr:        false,
		},
		{
			name:           "simple join with one table unauthorized",
			query:          "SELECT * FROM course c JOIN student s ON c.id = s.id",
			role:           "student",
			userId:         1,
			wantAuthorized: false,
			wantRemaining:  []TableInfo{{Name: "student", Alias: "s"}},
			wantErr:        false,
		},
		{
			name:           "join with alias",
			query:          "SELECT public.mytable.mycolumn,cc.code FROM (course c JOIN course_class AS cc ON c.id = cc.course_id) AS t1",
			role:           "student",
			userId:         1,
			wantAuthorized: true,
			wantRemaining:  []TableInfo{},
			wantErr:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			tables, err := extractTables(stmt)
			if err != nil {
				t.Fatalf("Failed to extract tables: %v", err)
			}

			service := NewAuthorizationService(NewSchemaServiceImpl())
			authRes, err := service.authorizeJoinExpr(stmt.GetFromClause()[0].GetJoinExpr(), tables, tt.role, tt.userId)

			if (err != nil) != tt.wantErr {
				t.Errorf("authorizeJoinExpr() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if authRes.Authorized != tt.wantAuthorized {
				t.Errorf("authorizeJoinExpr() gotAuthorized = %v, want %v", authRes.Authorized, tt.wantAuthorized)
			}

			if !compareTableInfoSlices(authRes.UnAuthorizedTables, tt.wantRemaining) {
				t.Errorf("authorizeJoinExpr() gotRemaining = %v, want %v", authRes.UnAuthorizedTables, tt.wantRemaining)
			}
		})
	}
}

func TestAuthorizationService_authorizeSubLink(t *testing.T) {
	tests := []struct {
		name           string
		query          string
		role           string
		userId         int
		wantAuthorized bool
		wantTables     []TableInfo
		wantErr        bool
	}{
		{
			name: "EXISTS subquery - professor courses",
			query: `SELECT id, name 
                    FROM professor p
                    WHERE EXISTS (
                        SELECT 1 
                        FROM course_class cc 
                        WHERE cc.professor_id = p.id
                    )`,
			role:           "professor",
			userId:         1,
			wantAuthorized: true,
			wantTables:     []TableInfo{},
			wantErr:        false,
		},
		{
			name: "IN subquery - professors by semester",
			query: `SELECT id, name 
                    FROM professor 
                    WHERE id IN (
                        SELECT professor_id 
                        FROM course_class 
                        WHERE semester_id = '2023-1'
                    )`,
			role:           "professor",
			userId:         1,
			wantAuthorized: true,
			wantTables:     []TableInfo{},
			wantErr:        false,
		},
		{
			name: "ANY subquery - student course access",
			query: `SELECT id, name 
                    FROM student 
                    WHERE id = ANY (
                        SELECT student_id 
                        FROM student_course_class 
                        WHERE student_id != 1
                    )`,
			role:           "student",
			userId:         1,
			wantAuthorized: false,
			wantTables:     []TableInfo{{Name: "student_course_class"}},
			wantErr:        false,
		},
		{
			name: "Scalar subquery - professor class count",
			query: `SELECT id, name,
                    (SELECT COUNT(*) FROM course_class cc WHERE cc.professor_id = p.id) AS class_count
                    FROM professor p`,
			role:           "professor",
			userId:         1,
			wantAuthorized: true,
			wantTables:     []TableInfo{},
			wantErr:        false,
		},
		{
			name: "ALL subquery - student scholarships",
			query: `SELECT id, name 
                    FROM student 
                    WHERE id > ALL (
                        SELECT student_id 
                        FROM student_scholarship 
                        WHERE scholarship_id = 1
                    )`,
			role:           "student",
			userId:         1,
			wantAuthorized: false,
			wantTables:     []TableInfo{{Name: "student_scholarship"}},
			wantErr:        false,
		},
		{
			name: "Row subquery - student enrollment",
			query: `SELECT id, name, * 
                    FROM student 
                    WHERE (id, administrative_class_id) = ANY (
                        SELECT student_id, course_class_id 
                        FROM student_course_class
                        WHERE student_id = 1
                    )`,
			role:           "student",
			userId:         1,
			wantAuthorized: true,
			wantTables:     []TableInfo{},
			wantErr:        false,
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

			// Extract the SubLink node
			stmt := tree.Stmts[0].Stmt.GetSelectStmt()
			if stmt == nil {
				t.Fatal("Not a SELECT statement")
			}

			var subLink *pgquery.SubLink
			// Find SubLink in different parts of the query
			if stmt.WhereClause != nil {
				subLink = extractSubLink(stmt.WhereClause)
			}

			if subLink == nil && len(stmt.GetTargetList()) > 0 {
				// Try to find in target list (for scalar subqueries)
				for _, target := range stmt.GetTargetList() {
					if target.GetResTarget().GetVal() != nil {
						extracted := extractSubLink(target.GetResTarget().GetVal())
						if extracted != nil {
							subLink = extracted
							break
						}
					}
				}
			}

			if subLink == nil {
				t.Fatal("No SubLink found in query")
			}

			// Create the service
			service := NewAuthorizationService(NewSchemaServiceImpl())

			// Extract tables for testing
			tables := extractTablesFromQuery(t, tt.query)

			// Create a mock for AuthorizeNode

			// Call the function being tested
			result, err := service.authorizeSubLink(subLink, tables, tt.role, tt.userId)

			// Check for expected errors
			if (err != nil) != tt.wantErr {
				t.Errorf("authorizeSubLink() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil {
				// Check authorization result
				if result.Authorized != tt.wantAuthorized {
					t.Errorf("authorizeSubLink() Authorized = %v, want %v", result.Authorized, tt.wantAuthorized)
				}

				// Check unauthorized tables
				if !compareTableInfoSlices(result.UnAuthorizedTables, tt.wantTables) {
					t.Errorf("authorizeSubLink() UnAuthorizedTables = %v, want %v", result.UnAuthorizedTables, tt.wantTables)
				}
			}
		})
	}
}

// Helper function to extract SubLink from a node
func extractSubLink(node *pgquery.Node) *pgquery.SubLink {
	if node == nil {
		return nil
	}

	switch n := node.GetNode().(type) {
	case *pgquery.Node_SubLink:
		return n.SubLink
	case *pgquery.Node_BoolExpr:
		for _, arg := range n.BoolExpr.GetArgs() {
			if subLink := extractSubLink(arg); subLink != nil {
				return subLink
			}
		}
	case *pgquery.Node_AExpr:
		if n.AExpr.GetRexpr() != nil {
			if subLink := extractSubLink(n.AExpr.GetRexpr()); subLink != nil {
				return subLink
			}
		}
		if n.AExpr.GetLexpr() != nil {
			if subLink := extractSubLink(n.AExpr.GetLexpr()); subLink != nil {
				return subLink
			}
		}
	}
	return nil
}

// Helper function to extract tables from a query for testing
func extractTablesFromQuery(t *testing.T, query string) []TableInfo {
	tree, err := pgquery.Parse(query)
	if err != nil {
		t.Fatalf("Failed to parse query: %v", err)
	}

	if len(tree.Stmts) == 0 {
		t.Fatal("No statements found in parsed query")
	}

	stmt := tree.Stmts[0].Stmt.GetSelectStmt()
	tables, err := extractTables(stmt)
	if err != nil {
		t.Fatalf("Failed to extract tables: %v", err)
	}

	return tables
}

// Helper to check if two nodes are equal
func isNodeEqual(n1, n2 *pgquery.Node) bool {
	// Simple equality check - in a real implementation you might need a more sophisticated approach
	return n1 == n2 || (n1 != nil && n2 != nil && fmt.Sprintf("%v", n1) == fmt.Sprintf("%v", n2))
}

// MockSchemaService for testing
type MockSchemaService struct {
	GetColumnsFunc func(tableName string) ([]string, error)
}

func (m *MockSchemaService) GetColumns(tableName string) ([]string, error) {
	return m.GetColumnsFunc(tableName)
}

// Helper function to compare slices of TableInfo regardless of order
func compareTableInfoSlices(a, b []TableInfo) bool {
	if len(a) != len(b) {
		return false
	}

	aMap := make(map[string]string)
	for _, table := range a {
		key := table.Name
		if table.Alias != "" {
			key += ":" + table.Alias
		}
		aMap[key] = table.Alias
	}

	bMap := make(map[string]string)
	for _, table := range b {
		key := table.Name
		if table.Alias != "" {
			key += ":" + table.Alias
		}
		bMap[key] = table.Alias
	}

	// Compare maps
	for key, val := range aMap {
		if bVal, ok := bMap[key]; !ok || bVal != val {
			return false
		}
	}

	for key, val := range bMap {
		if aVal, ok := aMap[key]; !ok || aVal != val {
			return false
		}
	}

	return true
}
