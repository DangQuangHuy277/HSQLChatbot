package db

import (
	om "github.com/elliotchance/orderedmap/v3"
	pgquery "github.com/pganalyze/pg_query_go/v6"
	"reflect"
	"testing"
)

// MockSchemaService provides a mock implementation of SchemaService for testing.
type MockSchemaService struct{}

func (m *MockSchemaService) GetColumns(tableName string) []string {
	switch tableName {
	case "student":
		return []string{"id", "code", "name", "gender", "birthday", "email", "administrative_class_id"}
	case "professor":
		return []string{"id", "name", "email", "academic_rank", "degree", "department_id"}
	case "course":
		return []string{"id", "code", "name", "english_name", "credits", "practice_hours", "theory_hours", "self_learn_hours", "prerequisite"}
	case "administrative_class":
		return []string{"id", "name", "program_id", "advisor_id"}
	case "course_class_enrollment":
		return []string{"id", "student_id", "course_class_id", "grade"}
	default:
		return []string{}
	}
}

// region TestAuthorizeAExpr
func TestAuthorizeAExpr(t *testing.T) {
	// Initialize mock schema service
	schemaService := &MockSchemaService{}
	authService := NewAuthorizationService(schemaService)

	// Define test cases
	tests := []struct {
		name              string
		aExpr             *pgquery.Node
		tables            *om.OrderedMap[string, *TableInfoV2]
		neg               bool
		userInfo          UserContext
		expectedAuth      bool
		expectedTableAuth map[string]bool
	}{
		{
			name: "Student accessing own data with equality",
			aExpr: pgquery.MakeAExprNode(
				pgquery.A_Expr_Kind_AEXPR_OP,
				[]*pgquery.Node{pgquery.MakeStrNode("=")},
				pgquery.MakeColumnRefNode([]*pgquery.Node{pgquery.MakeStrNode("student"), pgquery.MakeStrNode("id")}, 0),
				pgquery.MakeAConstIntNode(1, 0),
				0,
			),
			tables: func() *om.OrderedMap[string, *TableInfoV2] {
				tbl := &TableInfoV2{
					Name:       "student",
					Columns:    om.NewOrderedMap[string, *ColumnInfo](),
					Alias:      "student",
					Authorized: false,
					IsDatabase: true,
				}
				for _, col := range schemaService.GetColumns("student") {
					tbl.Columns.Set(col, &ColumnInfo{Name: col, SourceTable: tbl})
				}
				return om.NewOrderedMapWithElements(&om.Element[string, *TableInfoV2]{Key: "student", Value: tbl})
			}(),
			neg:               false,
			userInfo:          &StudentInfo{UserInfo: UserInfo{ID: 1, Role: "student"}},
			expectedAuth:      true,
			expectedTableAuth: map[string]bool{"student": true},
		},
		{
			name: "Student accessing another student's data",
			aExpr: pgquery.MakeAExprNode(
				pgquery.A_Expr_Kind_AEXPR_OP,
				[]*pgquery.Node{pgquery.MakeStrNode("=")},
				pgquery.MakeColumnRefNode([]*pgquery.Node{pgquery.MakeStrNode("student"), pgquery.MakeStrNode("id")}, 0),
				pgquery.MakeAConstIntNode(2, 0),
				0,
			),
			tables: func() *om.OrderedMap[string, *TableInfoV2] {
				tbl := &TableInfoV2{
					Name:       "student",
					Columns:    om.NewOrderedMap[string, *ColumnInfo](),
					Alias:      "student",
					Authorized: false,
					IsDatabase: true,
				}
				for _, col := range schemaService.GetColumns("student") {
					tbl.Columns.Set(col, &ColumnInfo{Name: col, SourceTable: tbl})
				}
				return om.NewOrderedMapWithElements(&om.Element[string, *TableInfoV2]{Key: "student", Value: tbl})
			}(),
			neg:               false,
			userInfo:          &StudentInfo{UserInfo: UserInfo{ID: 1, Role: "student"}},
			expectedAuth:      false,
			expectedTableAuth: map[string]bool{"student": false},
		},
		{
			name: "IN expression with student's own ID",
			aExpr: pgquery.MakeAExprNode(
				pgquery.A_Expr_Kind_AEXPR_IN,
				[]*pgquery.Node{pgquery.MakeStrNode("=")},
				pgquery.MakeColumnRefNode([]*pgquery.Node{pgquery.MakeStrNode("student"), pgquery.MakeStrNode("id")}, 0),
				pgquery.MakeListNode([]*pgquery.Node{pgquery.MakeAConstIntNode(1, 0)}),
				0,
			),
			tables: func() *om.OrderedMap[string, *TableInfoV2] {
				tbl := &TableInfoV2{
					Name:       "student",
					Columns:    om.NewOrderedMap[string, *ColumnInfo](),
					Alias:      "student",
					Authorized: false,
					IsDatabase: true,
				}
				for _, col := range schemaService.GetColumns("student") {
					tbl.Columns.Set(col, &ColumnInfo{Name: col, SourceTable: tbl})
				}
				return om.NewOrderedMapWithElements(&om.Element[string, *TableInfoV2]{Key: "student", Value: tbl})
			}(),
			neg:               false,
			userInfo:          &StudentInfo{UserInfo: UserInfo{ID: 1, Role: "student"}},
			expectedAuth:      true,
			expectedTableAuth: map[string]bool{"student": true},
		},
		{
			name: "IN expression with unauthorized ID",
			aExpr: pgquery.MakeAExprNode(
				pgquery.A_Expr_Kind_AEXPR_IN,
				[]*pgquery.Node{pgquery.MakeStrNode("=")},
				pgquery.MakeColumnRefNode([]*pgquery.Node{pgquery.MakeStrNode("student"), pgquery.MakeStrNode("id")}, 0),
				pgquery.MakeListNode([]*pgquery.Node{pgquery.MakeAConstIntNode(2, 0)}),
				0,
			),
			tables: func() *om.OrderedMap[string, *TableInfoV2] {
				tbl := &TableInfoV2{
					Name:       "student",
					Columns:    om.NewOrderedMap[string, *ColumnInfo](),
					Alias:      "student",
					Authorized: false,
					IsDatabase: true,
				}
				for _, col := range schemaService.GetColumns("student") {
					tbl.Columns.Set(col, &ColumnInfo{Name: col, SourceTable: tbl})
				}
				return om.NewOrderedMapWithElements(&om.Element[string, *TableInfoV2]{Key: "student", Value: tbl})
			}(),
			neg:               false,
			userInfo:          &StudentInfo{UserInfo: UserInfo{ID: 1, Role: "student"}},
			expectedAuth:      false,
			expectedTableAuth: map[string]bool{"student": false},
		},
		{
			name: "Public table access by student",
			aExpr: pgquery.MakeAExprNode(
				pgquery.A_Expr_Kind_AEXPR_OP,
				[]*pgquery.Node{pgquery.MakeStrNode("=")},
				pgquery.MakeColumnRefNode([]*pgquery.Node{pgquery.MakeStrNode("course"), pgquery.MakeStrNode("id")}, 0),
				pgquery.MakeAConstIntNode(1, 0),
				0,
			),
			tables: func() *om.OrderedMap[string, *TableInfoV2] {
				tbl := &TableInfoV2{
					Name:       "course",
					Columns:    om.NewOrderedMap[string, *ColumnInfo](),
					Alias:      "course",
					Authorized: false,
					IsDatabase: true,
				}
				for _, col := range schemaService.GetColumns("course") {
					tbl.Columns.Set(col, &ColumnInfo{Name: col, SourceTable: tbl})
				}
				return om.NewOrderedMapWithElements(&om.Element[string, *TableInfoV2]{Key: "course", Value: tbl})
			}(),
			neg:               false,
			userInfo:          &StudentInfo{UserInfo: UserInfo{ID: 1, Role: "student"}},
			expectedAuth:      true,
			expectedTableAuth: map[string]bool{"course": true},
		},
		{
			name: "Admin accessing student data",
			aExpr: pgquery.MakeAExprNode(
				pgquery.A_Expr_Kind_AEXPR_OP,
				[]*pgquery.Node{pgquery.MakeStrNode("=")},
				pgquery.MakeColumnRefNode([]*pgquery.Node{pgquery.MakeStrNode("student"), pgquery.MakeStrNode("id")}, 0),
				pgquery.MakeAConstIntNode(1, 0),
				0,
			),
			tables: func() *om.OrderedMap[string, *TableInfoV2] {
				tbl := &TableInfoV2{
					Name:       "student",
					Columns:    om.NewOrderedMap[string, *ColumnInfo](),
					Alias:      "student",
					Authorized: false,
					IsDatabase: true,
				}
				for _, col := range schemaService.GetColumns("student") {
					tbl.Columns.Set(col, &ColumnInfo{Name: col, SourceTable: tbl})
				}
				return om.NewOrderedMapWithElements(&om.Element[string, *TableInfoV2]{Key: "student", Value: tbl})
			}(),
			neg:               false,
			userInfo:          &AdminInfo{UserInfo: UserInfo{ID: 1, Role: "admin"}},
			expectedAuth:      true,
			expectedTableAuth: map[string]bool{"student": true},
		},
		{
			name: "Negated equality for student's own data",
			aExpr: pgquery.MakeAExprNode(
				pgquery.A_Expr_Kind_AEXPR_OP,
				[]*pgquery.Node{pgquery.MakeStrNode("=")},
				pgquery.MakeColumnRefNode([]*pgquery.Node{pgquery.MakeStrNode("student"), pgquery.MakeStrNode("id")}, 0),
				pgquery.MakeAConstIntNode(1, 0),
				0,
			),
			tables: func() *om.OrderedMap[string, *TableInfoV2] {
				tbl := &TableInfoV2{
					Name:       "student",
					Columns:    om.NewOrderedMap[string, *ColumnInfo](),
					Alias:      "student",
					Authorized: false,
					IsDatabase: true,
				}
				for _, col := range schemaService.GetColumns("student") {
					tbl.Columns.Set(col, &ColumnInfo{Name: col, SourceTable: tbl})
				}
				return om.NewOrderedMapWithElements(&om.Element[string, *TableInfoV2]{Key: "student", Value: tbl})
			}(),
			neg:               true,
			userInfo:          &StudentInfo{UserInfo: UserInfo{ID: 1, Role: "student"}},
			expectedAuth:      false,
			expectedTableAuth: map[string]bool{"student": false},
		},
		{
			name: "Inequality expression",
			aExpr: pgquery.MakeAExprNode(
				pgquery.A_Expr_Kind_AEXPR_OP,
				[]*pgquery.Node{pgquery.MakeStrNode("<>")},
				pgquery.MakeColumnRefNode([]*pgquery.Node{pgquery.MakeStrNode("student"), pgquery.MakeStrNode("id")}, 0),
				pgquery.MakeAConstIntNode(1, 0),
				0,
			),
			tables: func() *om.OrderedMap[string, *TableInfoV2] {
				tbl := &TableInfoV2{
					Name:       "student",
					Columns:    om.NewOrderedMap[string, *ColumnInfo](),
					Alias:      "student",
					Authorized: false,
					IsDatabase: true,
				}
				for _, col := range schemaService.GetColumns("student") {
					tbl.Columns.Set(col, &ColumnInfo{Name: col, SourceTable: tbl})
				}
				return om.NewOrderedMapWithElements(&om.Element[string, *TableInfoV2]{Key: "student", Value: tbl})
			}(),
			neg:               true,
			userInfo:          &StudentInfo{UserInfo: UserInfo{ID: 1, Role: "student"}},
			expectedAuth:      true,
			expectedTableAuth: map[string]bool{"student": true},
		},
		{
			name: "Table with alias",
			aExpr: pgquery.MakeAExprNode(
				pgquery.A_Expr_Kind_AEXPR_OP,
				[]*pgquery.Node{pgquery.MakeStrNode("=")},
				pgquery.MakeColumnRefNode([]*pgquery.Node{pgquery.MakeStrNode("s"), pgquery.MakeStrNode("id")}, 0),
				pgquery.MakeAConstIntNode(1, 0),
				0,
			),
			tables: func() *om.OrderedMap[string, *TableInfoV2] {
				tbl := &TableInfoV2{
					Name:       "student",
					Columns:    om.NewOrderedMap[string, *ColumnInfo](),
					Alias:      "s",
					Authorized: false,
					IsDatabase: true,
				}
				for _, col := range schemaService.GetColumns("student") {
					tbl.Columns.Set(col, &ColumnInfo{Name: col, SourceTable: tbl})
				}
				return om.NewOrderedMapWithElements(&om.Element[string, *TableInfoV2]{Key: "s", Value: tbl})
			}(),
			neg:               false,
			userInfo:          &StudentInfo{UserInfo: UserInfo{ID: 1, Role: "student"}},
			expectedAuth:      true,
			expectedTableAuth: map[string]bool{"s": true},
		},
		{
			name: "Expression with multiple tables",
			aExpr: pgquery.MakeAExprNode(
				pgquery.A_Expr_Kind_AEXPR_OP,
				[]*pgquery.Node{pgquery.MakeStrNode("=")},
				pgquery.MakeColumnRefNode([]*pgquery.Node{pgquery.MakeStrNode("student"), pgquery.MakeStrNode("id")}, 0),
				pgquery.MakeColumnRefNode([]*pgquery.Node{pgquery.MakeStrNode("professor"), pgquery.MakeStrNode("id")}, 0),
				0,
			),
			tables: func() *om.OrderedMap[string, *TableInfoV2] {
				studentTbl := &TableInfoV2{
					Name:       "student",
					Columns:    om.NewOrderedMap[string, *ColumnInfo](),
					Alias:      "student",
					Authorized: false,
					IsDatabase: true,
				}
				for _, col := range schemaService.GetColumns("student") {
					studentTbl.Columns.Set(col, &ColumnInfo{Name: col, SourceTable: studentTbl})
				}
				profTbl := &TableInfoV2{
					Name:       "professor",
					Columns:    om.NewOrderedMap[string, *ColumnInfo](),
					Alias:      "professor",
					Authorized: false,
					IsDatabase: true,
				}
				for _, col := range schemaService.GetColumns("professor") {
					profTbl.Columns.Set(col, &ColumnInfo{Name: col, SourceTable: profTbl})
				}
				return om.NewOrderedMapWithElements(
					&om.Element[string, *TableInfoV2]{Key: "student", Value: studentTbl},
					&om.Element[string, *TableInfoV2]{Key: "professor", Value: profTbl},
				)
			}(),
			neg:               false,
			userInfo:          &ProfessorInfo{UserInfo: UserInfo{ID: 1, Role: "professor"}},
			expectedAuth:      false,
			expectedTableAuth: map[string]bool{"student": false, "professor": false},
		},
		{
			name: "Virtual table from subquery with authorized access",
			aExpr: pgquery.MakeAExprNode(
				pgquery.A_Expr_Kind_AEXPR_OP,
				[]*pgquery.Node{pgquery.MakeStrNode("=")},
				pgquery.MakeColumnRefNode([]*pgquery.Node{pgquery.MakeStrNode("subquery"), pgquery.MakeStrNode("id")}, 0),
				pgquery.MakeAConstIntNode(1, 0),
				0,
			),
			tables: func() *om.OrderedMap[string, *TableInfoV2] {
				// Inner student table
				studentTbl := &TableInfoV2{
					Name:       "student",
					Columns:    om.NewOrderedMap[string, *ColumnInfo](),
					Alias:      "student",
					Authorized: true, // Authorized because student accesses own data
					IsDatabase: true,
				}
				for _, col := range []string{"id", "name"} { // Simplified column setup
					studentTbl.Columns.Set(col, &ColumnInfo{Name: col, SourceTable: studentTbl})
				}

				// Virtual table for subquery
				virtualTbl := &TableInfoV2{
					Name:               "subquery",
					Columns:            om.NewOrderedMap[string, *ColumnInfo](),
					Alias:              "subquery",
					Authorized:         false, // Initially unauthorized, will inherit from studentTbl
					IsDatabase:         false,
					UnAuthorizedTables: om.NewOrderedMap[string, *TableInfoV2](),
				}

				// Populate virtual table columns from student table
				virtualTbl.Columns.Set("id", &ColumnInfo{Name: "id", SourceTable: studentTbl})
				virtualTbl.Columns.Set("name", &ColumnInfo{Name: "name", SourceTable: studentTbl})
				virtualTbl.UnAuthorizedTables.Set("student", studentTbl)

				// Since studentTbl is authorized, virtualTbl should inherit this status
				return om.NewOrderedMapWithElements(&om.Element[string, *TableInfoV2]{Key: "subquery", Value: virtualTbl})
			}(),
			neg:               false,
			userInfo:          &StudentInfo{UserInfo: UserInfo{ID: 1, Role: "student"}},
			expectedAuth:      true,
			expectedTableAuth: map[string]bool{"subquery": true},
		}, {
			name: "Virtual table with IN clause",
			aExpr: pgquery.MakeAExprNode(
				pgquery.A_Expr_Kind_AEXPR_IN,
				[]*pgquery.Node{pgquery.MakeStrNode("=")},
				pgquery.MakeColumnRefNode([]*pgquery.Node{pgquery.MakeStrNode("subquery"), pgquery.MakeStrNode("id")}, 0),
				pgquery.MakeListNode([]*pgquery.Node{pgquery.MakeAConstIntNode(1, 0)}),
				0,
			),
			tables: func() *om.OrderedMap[string, *TableInfoV2] {
				// Inner student table
				studentTbl := &TableInfoV2{
					Name:       "student",
					Columns:    om.NewOrderedMap[string, *ColumnInfo](),
					Alias:      "student",
					Authorized: false,
					IsDatabase: true,
				}
				for _, col := range schemaService.GetColumns("student") {
					studentTbl.Columns.Set(col, &ColumnInfo{Name: col, SourceTable: studentTbl})
				}

				// Virtual table for subquery
				virtualTbl := &TableInfoV2{
					Name:               "subquery",
					Columns:            om.NewOrderedMap[string, *ColumnInfo](),
					Alias:              "subquery",
					Authorized:         false,
					IsDatabase:         false,
					UnAuthorizedTables: om.NewOrderedMap[string, *TableInfoV2](),
				}
				virtualTbl.UnAuthorizedTables.Set("student", studentTbl)
				virtualTbl.Columns.Set("id", &ColumnInfo{Name: "id", SourceTable: studentTbl})

				return om.NewOrderedMapWithElements(&om.Element[string, *TableInfoV2]{Key: "subquery", Value: virtualTbl})
			}(),
			neg:               false,
			userInfo:          &StudentInfo{UserInfo: UserInfo{ID: 1, Role: "student"}},
			expectedAuth:      true,
			expectedTableAuth: map[string]bool{"subquery": true},
		},
		{
			name: "Virtual table with RowExpr = SubLink",
			aExpr: pgquery.MakeAExprNode(
				pgquery.A_Expr_Kind_AEXPR_OP,
				[]*pgquery.Node{pgquery.MakeStrNode("=")},
				&pgquery.Node{
					Node: &pgquery.Node_RowExpr{
						RowExpr: &pgquery.RowExpr{
							Args: []*pgquery.Node{
								pgquery.MakeColumnRefNode([]*pgquery.Node{pgquery.MakeStrNode("subquery"), pgquery.MakeStrNode("id")}, 0),
								pgquery.MakeColumnRefNode([]*pgquery.Node{pgquery.MakeStrNode("subquery"), pgquery.MakeStrNode("name")}, 0),
							},
							RowFormat: pgquery.CoercionForm_COERCE_EXPLICIT_CALL,
						},
					},
				},
				&pgquery.Node{
					Node: &pgquery.Node_SubLink{
						SubLink: &pgquery.SubLink{
							Subselect: &pgquery.Node{
								Node: &pgquery.Node_SelectStmt{
									SelectStmt: &pgquery.SelectStmt{
										TargetList: []*pgquery.Node{
											pgquery.MakeResTargetNodeWithVal(
												pgquery.MakeColumnRefNode([]*pgquery.Node{pgquery.MakeStrNode("s"), pgquery.MakeStrNode("id")}, 0),
												0,
											),
											pgquery.MakeResTargetNodeWithVal(
												pgquery.MakeColumnRefNode([]*pgquery.Node{pgquery.MakeStrNode("s"), pgquery.MakeStrNode("name")}, 0),
												0,
											),
										},
										FromClause: []*pgquery.Node{
											pgquery.MakeFullRangeVarNode("", "student", "s", 0),
										},
										WhereClause: pgquery.MakeAExprNode(
											pgquery.A_Expr_Kind_AEXPR_OP,
											[]*pgquery.Node{pgquery.MakeStrNode("=")},
											pgquery.MakeColumnRefNode([]*pgquery.Node{pgquery.MakeStrNode("s"), pgquery.MakeStrNode("id")}, 0),
											pgquery.MakeAConstIntNode(1, 0),
											0,
										),
									},
								},
							},
						},
					},
				},
				0,
			),
			tables: func() *om.OrderedMap[string, *TableInfoV2] {
				// Inner student table
				studentTbl := &TableInfoV2{
					Name:       "student",
					Columns:    om.NewOrderedMap[string, *ColumnInfo](),
					Alias:      "s",
					Authorized: false,
					IsDatabase: true,
				}
				for _, col := range schemaService.GetColumns("student") {
					studentTbl.Columns.Set(col, &ColumnInfo{Name: col, SourceTable: studentTbl})
				}

				// Virtual table for subquery
				virtualTbl := &TableInfoV2{
					Name:               "subquery",
					Columns:            om.NewOrderedMap[string, *ColumnInfo](),
					Alias:              "subquery",
					Authorized:         false,
					IsDatabase:         false,
					UnAuthorizedTables: om.NewOrderedMap[string, *TableInfoV2](),
				}
				virtualTbl.UnAuthorizedTables.Set("s", studentTbl)
				virtualTbl.Columns.Set("id", &ColumnInfo{Name: "id", SourceTable: studentTbl})
				virtualTbl.Columns.Set("name", &ColumnInfo{Name: "name", SourceTable: studentTbl})

				return om.NewOrderedMapWithElements(&om.Element[string, *TableInfoV2]{Key: "subquery", Value: virtualTbl})
			}(),
			neg:               false,
			userInfo:          &StudentInfo{UserInfo: UserInfo{ID: 1, Role: "student"}},
			expectedAuth:      true,
			expectedTableAuth: map[string]bool{"subquery": true},
		},
	}

	// Run tests
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := authService.AuthorizeNode(tt.aExpr, tt.tables, tt.neg, tt.userInfo)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Check overall authorization
			if result.Authorized != tt.expectedAuth {
				t.Errorf("Expected Authorized=%v, got %v", tt.expectedAuth, result.Authorized)
			}

			// Check table authorization status for expected tables
			for alias, expected := range tt.expectedTableAuth {
				table, ok := result.Tables.Get(alias)
				if !ok {
					t.Errorf("Table %s not found in result", alias)
					continue
				}
				if table.Authorized != expected {
					t.Errorf("Table %s: expected Authorized=%v, got %v", alias, expected, table.Authorized)
				}
			}

			// Check all tables in result to ensure no unexpected unauthorized changes
			for alias, table := range result.Tables.AllFromFront() {
				expected, ok := tt.expectedTableAuth[alias]
				if !ok {
					t.Errorf("Unexpected table %s found in result.Tables", alias)
					continue
				}
				if table.Authorized != expected {
					t.Errorf("Table %s: authorization mismatch, expected %v, got %v", alias, expected, table.Authorized)
				}
			}
		})
	}
}

// endregion

// region TestAuthorizeBoolExpr
func TestAuthorizeBoolExpr(t *testing.T) {
	// Initialize mock schema service
	schemaService := &MockSchemaService{}
	authService := NewAuthorizationService(schemaService)

	// Define test cases
	tests := []struct {
		name              string
		boolExpr          *pgquery.Node
		tables            *om.OrderedMap[string, *TableInfoV2]
		neg               bool
		userInfo          UserContext
		expectedAuth      bool
		expectedTableAuth map[string]bool
		expectedError     string
	}{
		{
			name: "AND expression with both conditions authorized (student accessing own data and public table)",
			boolExpr: pgquery.MakeBoolExprNode(
				pgquery.BoolExprType_AND_EXPR,
				[]*pgquery.Node{
					pgquery.MakeAExprNode(
						pgquery.A_Expr_Kind_AEXPR_OP,
						[]*pgquery.Node{pgquery.MakeStrNode("=")},
						pgquery.MakeColumnRefNode([]*pgquery.Node{pgquery.MakeStrNode("student"), pgquery.MakeStrNode("id")}, 0),
						pgquery.MakeAConstIntNode(1, 0),
						0,
					),
					pgquery.MakeAExprNode(
						pgquery.A_Expr_Kind_AEXPR_OP,
						[]*pgquery.Node{pgquery.MakeStrNode("=")},
						pgquery.MakeColumnRefNode([]*pgquery.Node{pgquery.MakeStrNode("course"), pgquery.MakeStrNode("id")}, 0),
						pgquery.MakeAConstIntNode(1, 0),
						0,
					),
				},
				0,
			),
			tables: func() *om.OrderedMap[string, *TableInfoV2] {
				studentTbl := &TableInfoV2{
					Name:       "student",
					Columns:    om.NewOrderedMap[string, *ColumnInfo](),
					Alias:      "student",
					Authorized: false,
					IsDatabase: true,
				}
				for _, col := range schemaService.GetColumns("student") {
					studentTbl.Columns.Set(col, &ColumnInfo{Name: col, SourceTable: studentTbl})
				}
				courseTbl := &TableInfoV2{
					Name:       "course",
					Columns:    om.NewOrderedMap[string, *ColumnInfo](),
					Alias:      "course",
					Authorized: false,
					IsDatabase: true,
				}
				for _, col := range schemaService.GetColumns("course") {
					courseTbl.Columns.Set(col, &ColumnInfo{Name: col, SourceTable: courseTbl})
				}
				return om.NewOrderedMapWithElements(
					&om.Element[string, *TableInfoV2]{Key: "student", Value: studentTbl},
					&om.Element[string, *TableInfoV2]{Key: "course", Value: courseTbl},
				)
			}(),
			neg:               false,
			userInfo:          &StudentInfo{UserInfo: UserInfo{ID: 1, Role: "student"}},
			expectedAuth:      true,
			expectedTableAuth: map[string]bool{"student": true, "course": true},
		},
		{
			name: "AND expression with one condition unauthorized (student accessing another student's data)",
			boolExpr: pgquery.MakeBoolExprNode(
				pgquery.BoolExprType_AND_EXPR,
				[]*pgquery.Node{
					pgquery.MakeAExprNode(
						pgquery.A_Expr_Kind_AEXPR_OP,
						[]*pgquery.Node{pgquery.MakeStrNode("=")},
						pgquery.MakeColumnRefNode([]*pgquery.Node{pgquery.MakeStrNode("student"), pgquery.MakeStrNode("id")}, 0),
						pgquery.MakeAConstIntNode(1, 0),
						0,
					),
					pgquery.MakeAExprNode(
						pgquery.A_Expr_Kind_AEXPR_OP,
						[]*pgquery.Node{pgquery.MakeStrNode("=")},
						pgquery.MakeColumnRefNode([]*pgquery.Node{pgquery.MakeStrNode("course_class_enrollment_id"), pgquery.MakeStrNode("student_id")}, 0),
						pgquery.MakeAConstIntNode(2, 0),
						0,
					),
				},
				0,
			),
			tables: func() *om.OrderedMap[string, *TableInfoV2] {
				studentTbl := &TableInfoV2{
					Name:       "student",
					Columns:    om.NewOrderedMap[string, *ColumnInfo](),
					Alias:      "student",
					Authorized: false,
					IsDatabase: true,
				}
				for _, col := range schemaService.GetColumns("student") {
					studentTbl.Columns.Set(col, &ColumnInfo{Name: col, SourceTable: studentTbl})
				}
				sccTbl := &TableInfoV2{
					Name:       "course_class_enrollment",
					Columns:    om.NewOrderedMap[string, *ColumnInfo](),
					Alias:      "course_class_enrollment",
					Authorized: false,
					IsDatabase: true,
				}
				for _, col := range schemaService.GetColumns("course_class_enrollment") {
					sccTbl.Columns.Set(col, &ColumnInfo{Name: col, SourceTable: sccTbl})
				}
				return om.NewOrderedMapWithElements(
					&om.Element[string, *TableInfoV2]{Key: "student", Value: studentTbl},
					&om.Element[string, *TableInfoV2]{Key: "course_class_enrollment", Value: sccTbl},
				)
			}(),
			neg:               false,
			userInfo:          &StudentInfo{UserInfo: UserInfo{ID: 1, Role: "student"}},
			expectedAuth:      false,
			expectedTableAuth: map[string]bool{"student": true, "course_class_enrollment": false},
		},
		{
			name: "OR expression with one condition authorized (student accessing own data or another's data)",
			boolExpr: pgquery.MakeBoolExprNode(
				pgquery.BoolExprType_OR_EXPR,
				[]*pgquery.Node{
					pgquery.MakeAExprNode(
						pgquery.A_Expr_Kind_AEXPR_OP,
						[]*pgquery.Node{pgquery.MakeStrNode("=")},
						pgquery.MakeColumnRefNode([]*pgquery.Node{pgquery.MakeStrNode("student"), pgquery.MakeStrNode("id")}, 0),
						pgquery.MakeAConstIntNode(1, 0),
						0,
					),
					pgquery.MakeAExprNode(
						pgquery.A_Expr_Kind_AEXPR_OP,
						[]*pgquery.Node{pgquery.MakeStrNode("=")},
						pgquery.MakeColumnRefNode([]*pgquery.Node{pgquery.MakeStrNode("course_class_enrollment"), pgquery.MakeStrNode("student_id")}, 0),
						pgquery.MakeAConstIntNode(2, 0),
						0,
					),
				},
				0,
			),
			tables: func() *om.OrderedMap[string, *TableInfoV2] {
				studentTbl := &TableInfoV2{
					Name:       "student",
					Columns:    om.NewOrderedMap[string, *ColumnInfo](),
					Alias:      "student",
					Authorized: false,
					IsDatabase: true,
				}
				for _, col := range schemaService.GetColumns("student") {
					studentTbl.Columns.Set(col, &ColumnInfo{Name: col, SourceTable: studentTbl})
				}
				sccTbl := &TableInfoV2{
					Name:       "course_class_enrollment",
					Columns:    om.NewOrderedMap[string, *ColumnInfo](),
					Alias:      "course_class_enrollment",
					Authorized: false,
					IsDatabase: true,
				}
				for _, col := range schemaService.GetColumns("course_class_enrollment") {
					sccTbl.Columns.Set(col, &ColumnInfo{Name: col, SourceTable: sccTbl})
				}
				return om.NewOrderedMapWithElements(
					&om.Element[string, *TableInfoV2]{Key: "student", Value: studentTbl},
					&om.Element[string, *TableInfoV2]{Key: "course_class_enrollment", Value: sccTbl},
				)
			}(),
			neg:               false,
			userInfo:          &StudentInfo{UserInfo: UserInfo{ID: 1, Role: "student"}},
			expectedAuth:      false,
			expectedTableAuth: map[string]bool{"student": false, "course_class_enrollment": false},
		},
		{
			name: "NOT expression with authorized condition",
			boolExpr: pgquery.MakeBoolExprNode(
				pgquery.BoolExprType_NOT_EXPR,
				[]*pgquery.Node{
					pgquery.MakeAExprNode(
						pgquery.A_Expr_Kind_AEXPR_OP,
						[]*pgquery.Node{pgquery.MakeStrNode("=")},
						pgquery.MakeColumnRefNode([]*pgquery.Node{pgquery.MakeStrNode("student"), pgquery.MakeStrNode("id")}, 0),
						pgquery.MakeAConstIntNode(1, 0),
						0,
					),
				},
				0,
			),
			tables: func() *om.OrderedMap[string, *TableInfoV2] {
				tbl := &TableInfoV2{
					Name:       "student",
					Columns:    om.NewOrderedMap[string, *ColumnInfo](),
					Alias:      "student",
					Authorized: false,
					IsDatabase: true,
				}
				for _, col := range schemaService.GetColumns("student") {
					tbl.Columns.Set(col, &ColumnInfo{Name: col, SourceTable: tbl})
				}
				return om.NewOrderedMapWithElements(&om.Element[string, *TableInfoV2]{Key: "student", Value: tbl})
			}(),
			neg:               false,
			userInfo:          &StudentInfo{UserInfo: UserInfo{ID: 1, Role: "student"}},
			expectedAuth:      false,
			expectedTableAuth: map[string]bool{"student": false},
		},
		{
			name: "NOT expression with unauthorized condition",
			boolExpr: pgquery.MakeBoolExprNode(
				pgquery.BoolExprType_NOT_EXPR,
				[]*pgquery.Node{
					pgquery.MakeAExprNode(
						pgquery.A_Expr_Kind_AEXPR_OP,
						[]*pgquery.Node{pgquery.MakeStrNode("<>")},
						pgquery.MakeColumnRefNode([]*pgquery.Node{pgquery.MakeStrNode("course_class_enrollment"), pgquery.MakeStrNode("student_id")}, 0),
						pgquery.MakeAConstIntNode(1, 0),
						0,
					),
				},
				0,
			),
			tables: func() *om.OrderedMap[string, *TableInfoV2] {
				tbl := &TableInfoV2{
					Name:       "course_class_enrollment",
					Columns:    om.NewOrderedMap[string, *ColumnInfo](),
					Alias:      "course_class_enrollment",
					Authorized: false,
					IsDatabase: true,
				}
				for _, col := range schemaService.GetColumns("course_class_enrollment") {
					tbl.Columns.Set(col, &ColumnInfo{Name: col, SourceTable: tbl})
				}
				return om.NewOrderedMapWithElements(&om.Element[string, *TableInfoV2]{Key: "course_class_enrollment", Value: tbl})
			}(),
			neg:               false,
			userInfo:          &StudentInfo{UserInfo: UserInfo{ID: 1, Role: "student"}},
			expectedAuth:      true,
			expectedTableAuth: map[string]bool{"course_class_enrollment": true},
		},
		{
			name: "Nested AND and OR expression with non-public table",
			boolExpr: pgquery.MakeBoolExprNode(
				pgquery.BoolExprType_AND_EXPR,
				[]*pgquery.Node{
					pgquery.MakeAExprNode(
						pgquery.A_Expr_Kind_AEXPR_OP,
						[]*pgquery.Node{pgquery.MakeStrNode("=")},
						pgquery.MakeColumnRefNode([]*pgquery.Node{pgquery.MakeStrNode("student"), pgquery.MakeStrNode("id")}, 0),
						pgquery.MakeAConstIntNode(1, 0),
						0,
					),
					pgquery.MakeBoolExprNode(
						pgquery.BoolExprType_OR_EXPR,
						[]*pgquery.Node{
							pgquery.MakeAExprNode(
								pgquery.A_Expr_Kind_AEXPR_OP,
								[]*pgquery.Node{pgquery.MakeStrNode("=")},
								pgquery.MakeColumnRefNode([]*pgquery.Node{pgquery.MakeStrNode("course_class_enrollment"), pgquery.MakeStrNode("student_id")}, 0),
								pgquery.MakeAConstIntNode(1, 0),
								0,
							),
							pgquery.MakeAExprNode(
								pgquery.A_Expr_Kind_AEXPR_OP,
								[]*pgquery.Node{pgquery.MakeStrNode("=")},
								pgquery.MakeColumnRefNode([]*pgquery.Node{pgquery.MakeStrNode("course_class_enrollment"), pgquery.MakeStrNode("student_id")}, 0),
								pgquery.MakeAConstIntNode(2, 0),
								0,
							),
						},
						0,
					),
				},
				0,
			),
			tables: func() *om.OrderedMap[string, *TableInfoV2] {
				studentTbl := &TableInfoV2{
					Name:       "student",
					Columns:    om.NewOrderedMap[string, *ColumnInfo](),
					Alias:      "student",
					Authorized: false,
					IsDatabase: true,
				}
				for _, col := range schemaService.GetColumns("student") {
					studentTbl.Columns.Set(col, &ColumnInfo{Name: col, SourceTable: studentTbl})
				}
				sccTbl := &TableInfoV2{
					Name:       "course_class_enrollment",
					Columns:    om.NewOrderedMap[string, *ColumnInfo](),
					Alias:      "course_class_enrollment",
					Authorized: false,
					IsDatabase: true,
				}
				for _, col := range schemaService.GetColumns("course_class_enrollment") {
					sccTbl.Columns.Set(col, &ColumnInfo{Name: col, SourceTable: sccTbl})
				}
				return om.NewOrderedMapWithElements(
					&om.Element[string, *TableInfoV2]{Key: "student", Value: studentTbl},
					&om.Element[string, *TableInfoV2]{Key: "course_class_enrollment", Value: sccTbl},
				)
			}(),
			neg:               false,
			userInfo:          &StudentInfo{UserInfo: UserInfo{ID: 1, Role: "student"}},
			expectedAuth:      false,
			expectedTableAuth: map[string]bool{"student": true, "course_class_enrollment": false},
		},
		{
			name: "Virtual table in AND expression",
			boolExpr: pgquery.MakeBoolExprNode(
				pgquery.BoolExprType_AND_EXPR,
				[]*pgquery.Node{
					pgquery.MakeAExprNode(
						pgquery.A_Expr_Kind_AEXPR_OP,
						[]*pgquery.Node{pgquery.MakeStrNode("=")},
						pgquery.MakeColumnRefNode([]*pgquery.Node{pgquery.MakeStrNode("subquery"), pgquery.MakeStrNode("id")}, 0),
						pgquery.MakeAConstIntNode(1, 0),
						0,
					),
					pgquery.MakeAExprNode(
						pgquery.A_Expr_Kind_AEXPR_OP,
						[]*pgquery.Node{pgquery.MakeStrNode("=")},
						pgquery.MakeColumnRefNode([]*pgquery.Node{pgquery.MakeStrNode("subquery"), pgquery.MakeStrNode("name")}, 0),
						pgquery.MakeAConstStrNode("John", 0),
						0,
					),
				},
				0,
			),
			tables: func() *om.OrderedMap[string, *TableInfoV2] {
				studentTbl := &TableInfoV2{
					Name:       "student",
					Columns:    om.NewOrderedMap[string, *ColumnInfo](),
					Alias:      "student",
					Authorized: false,
					IsDatabase: true,
				}
				for _, col := range schemaService.GetColumns("student") {
					studentTbl.Columns.Set(col, &ColumnInfo{Name: col, SourceTable: studentTbl})
				}
				virtualTbl := &TableInfoV2{
					Name:               "subquery",
					Columns:            om.NewOrderedMap[string, *ColumnInfo](),
					Alias:              "subquery",
					Authorized:         false,
					IsDatabase:         false,
					UnAuthorizedTables: om.NewOrderedMap[string, *TableInfoV2](),
				}
				virtualTbl.UnAuthorizedTables.Set("student", studentTbl)
				virtualTbl.Columns.Set("id", &ColumnInfo{Name: "id", SourceTable: studentTbl})
				virtualTbl.Columns.Set("name", &ColumnInfo{Name: "name", SourceTable: studentTbl})
				return om.NewOrderedMapWithElements(&om.Element[string, *TableInfoV2]{Key: "subquery", Value: virtualTbl})
			}(),
			neg:               false,
			userInfo:          &StudentInfo{UserInfo: UserInfo{ID: 1, Role: "student"}},
			expectedAuth:      true,
			expectedTableAuth: map[string]bool{"subquery": true},
		},
		{
			name:              "Nil BoolExpr",
			boolExpr:          nil,
			tables:            om.NewOrderedMap[string, *TableInfoV2](),
			neg:               false,
			userInfo:          &StudentInfo{UserInfo: UserInfo{ID: 1, Role: "student"}},
			expectedAuth:      true,
			expectedTableAuth: map[string]bool{},
		},
		{
			name: "Invalid NOT expression with multiple arguments",
			boolExpr: pgquery.MakeBoolExprNode(
				pgquery.BoolExprType_NOT_EXPR,
				[]*pgquery.Node{
					pgquery.MakeAExprNode(
						pgquery.A_Expr_Kind_AEXPR_OP,
						[]*pgquery.Node{pgquery.MakeStrNode("=")},
						pgquery.MakeColumnRefNode([]*pgquery.Node{pgquery.MakeStrNode("student"), pgquery.MakeStrNode("id")}, 0),
						pgquery.MakeAConstIntNode(1, 0),
						0,
					),
					pgquery.MakeAExprNode(
						pgquery.A_Expr_Kind_AEXPR_OP,
						[]*pgquery.Node{pgquery.MakeStrNode("=")},
						pgquery.MakeColumnRefNode([]*pgquery.Node{pgquery.MakeStrNode("student"), pgquery.MakeStrNode("id")}, 0),
						pgquery.MakeAConstIntNode(2, 0),
						0,
					),
				},
				0,
			),
			tables: func() *om.OrderedMap[string, *TableInfoV2] {
				tbl := &TableInfoV2{
					Name:       "student",
					Columns:    om.NewOrderedMap[string, *ColumnInfo](),
					Alias:      "student",
					Authorized: false,
					IsDatabase: true,
				}
				for _, col := range schemaService.GetColumns("student") {
					tbl.Columns.Set(col, &ColumnInfo{Name: col, SourceTable: tbl})
				}
				return om.NewOrderedMapWithElements(&om.Element[string, *TableInfoV2]{Key: "student", Value: tbl})
			}(),
			neg:           false,
			userInfo:      &StudentInfo{UserInfo: UserInfo{ID: 1, Role: "student"}},
			expectedAuth:  false,
			expectedError: "NOT expression must have exactly one argument",
		},
		{
			name: "Professor accessing taught course class",
			boolExpr: pgquery.MakeBoolExprNode(
				pgquery.BoolExprType_AND_EXPR,
				[]*pgquery.Node{
					pgquery.MakeAExprNode(
						pgquery.A_Expr_Kind_AEXPR_OP,
						[]*pgquery.Node{pgquery.MakeStrNode("=")},
						pgquery.MakeColumnRefNode([]*pgquery.Node{pgquery.MakeStrNode("course_class_enrollment"), pgquery.MakeStrNode("id")}, 0),
						pgquery.MakeAConstIntNode(10, 0),
						0,
					),
					pgquery.MakeAExprNode(
						pgquery.A_Expr_Kind_AEXPR_OP,
						[]*pgquery.Node{pgquery.MakeStrNode("=")},
						pgquery.MakeColumnRefNode([]*pgquery.Node{pgquery.MakeStrNode("course"), pgquery.MakeStrNode("id")}, 0),
						pgquery.MakeAConstIntNode(1, 0),
						0,
					),
				},
				0,
			),
			tables: func() *om.OrderedMap[string, *TableInfoV2] {
				sccTbl := &TableInfoV2{
					Name:       "course_class_enrollment",
					Columns:    om.NewOrderedMap[string, *ColumnInfo](),
					Alias:      "course_class_enrollment",
					Authorized: false,
					IsDatabase: true,
				}
				for _, col := range schemaService.GetColumns("course_class_enrollment") {
					sccTbl.Columns.Set(col, &ColumnInfo{Name: col, SourceTable: sccTbl})
				}
				courseTbl := &TableInfoV2{
					Name:       "course",
					Columns:    om.NewOrderedMap[string, *ColumnInfo](),
					Alias:      "course",
					Authorized: false,
					IsDatabase: true,
				}
				for _, col := range schemaService.GetColumns("course") {
					courseTbl.Columns.Set(col, &ColumnInfo{Name: col, SourceTable: courseTbl})
				}
				return om.NewOrderedMapWithElements(
					&om.Element[string, *TableInfoV2]{Key: "course_class_enrollment", Value: sccTbl},
					&om.Element[string, *TableInfoV2]{Key: "course", Value: courseTbl},
				)
			}(),
			neg:               false,
			userInfo:          &ProfessorInfo{UserInfo: UserInfo{ID: 1, Role: "professor"}, TaughtCourseClassIDs: []int{10}},
			expectedAuth:      true,
			expectedTableAuth: map[string]bool{"course_class_enrollment": true, "course": true},
		},
		{
			name: "Admin accessing all tables",
			boolExpr: pgquery.MakeBoolExprNode(
				pgquery.BoolExprType_AND_EXPR,
				[]*pgquery.Node{
					pgquery.MakeAExprNode(
						pgquery.A_Expr_Kind_AEXPR_OP,
						[]*pgquery.Node{pgquery.MakeStrNode("=")},
						pgquery.MakeColumnRefNode([]*pgquery.Node{pgquery.MakeStrNode("student"), pgquery.MakeStrNode("id")}, 0),
						pgquery.MakeAConstIntNode(2, 0),
						0,
					),
					pgquery.MakeAExprNode(
						pgquery.A_Expr_Kind_AEXPR_OP,
						[]*pgquery.Node{pgquery.MakeStrNode("=")},
						pgquery.MakeColumnRefNode([]*pgquery.Node{pgquery.MakeStrNode("course_class_enrollment"), pgquery.MakeStrNode("student_id")}, 0),
						pgquery.MakeAConstIntNode(2, 0),
						0,
					),
				},
				0,
			),
			tables: func() *om.OrderedMap[string, *TableInfoV2] {
				studentTbl := &TableInfoV2{
					Name:       "student",
					Columns:    om.NewOrderedMap[string, *ColumnInfo](),
					Alias:      "student",
					Authorized: false,
					IsDatabase: true,
				}
				for _, col := range schemaService.GetColumns("student") {
					studentTbl.Columns.Set(col, &ColumnInfo{Name: col, SourceTable: studentTbl})
				}
				sccTbl := &TableInfoV2{
					Name:       "course_class_enrollment",
					Columns:    om.NewOrderedMap[string, *ColumnInfo](),
					Alias:      "course_class_enrollment",
					Authorized: false,
					IsDatabase: true,
				}
				for _, col := range schemaService.GetColumns("course_class_enrollment") {
					sccTbl.Columns.Set(col, &ColumnInfo{Name: col, SourceTable: sccTbl})
				}
				return om.NewOrderedMapWithElements(
					&om.Element[string, *TableInfoV2]{Key: "student", Value: studentTbl},
					&om.Element[string, *TableInfoV2]{Key: "course_class_enrollment", Value: sccTbl},
				)
			}(),
			neg:               false,
			userInfo:          &UserInfo{ID: 1, Role: "admin"},
			expectedAuth:      true,
			expectedTableAuth: map[string]bool{"student": true, "course_class_enrollment": true},
		},
		{
			name: "Student accessing unauthorized administrative class",
			boolExpr: pgquery.MakeBoolExprNode(
				pgquery.BoolExprType_AND_EXPR,
				[]*pgquery.Node{
					pgquery.MakeAExprNode(
						pgquery.A_Expr_Kind_AEXPR_OP,
						[]*pgquery.Node{pgquery.MakeStrNode("=")},
						pgquery.MakeColumnRefNode([]*pgquery.Node{pgquery.MakeStrNode("student"), pgquery.MakeStrNode("id")}, 0),
						pgquery.MakeAConstIntNode(1, 0),
						0,
					),
					pgquery.MakeAExprNode(
						pgquery.A_Expr_Kind_AEXPR_OP,
						[]*pgquery.Node{pgquery.MakeStrNode("=")},
						pgquery.MakeColumnRefNode([]*pgquery.Node{pgquery.MakeStrNode("administrative_class"), pgquery.MakeStrNode("id")}, 0),
						pgquery.MakeAConstIntNode(2, 0),
						0,
					),
				},
				0,
			),
			tables: func() *om.OrderedMap[string, *TableInfoV2] {
				studentTbl := &TableInfoV2{
					Name:       "student",
					Columns:    om.NewOrderedMap[string, *ColumnInfo](),
					Alias:      "student",
					Authorized: false,
					IsDatabase: true,
				}
				for _, col := range schemaService.GetColumns("student") {
					studentTbl.Columns.Set(col, &ColumnInfo{Name: col, SourceTable: studentTbl})
				}
				adminTbl := &TableInfoV2{
					Name:       "administrative_class",
					Columns:    om.NewOrderedMap[string, *ColumnInfo](),
					Alias:      "administrative_class",
					Authorized: false,
					IsDatabase: true,
				}
				for _, col := range schemaService.GetColumns("administrative_class") {
					adminTbl.Columns.Set(col, &ColumnInfo{Name: col, SourceTable: adminTbl})
				}
				return om.NewOrderedMapWithElements(
					&om.Element[string, *TableInfoV2]{Key: "student", Value: studentTbl},
					&om.Element[string, *TableInfoV2]{Key: "administrative_class", Value: adminTbl},
				)
			}(),
			neg:               false,
			userInfo:          &StudentInfo{UserInfo: UserInfo{ID: 1, Role: "student"}, AdministrativeClassID: 1},
			expectedAuth:      false,
			expectedTableAuth: map[string]bool{"student": true, "administrative_class": false},
		},
		{
			name: "Nested AND with virtual table from authorized subquery",
			boolExpr: pgquery.MakeBoolExprNode(
				pgquery.BoolExprType_AND_EXPR,
				[]*pgquery.Node{
					pgquery.MakeAExprNode(
						pgquery.A_Expr_Kind_AEXPR_OP,
						[]*pgquery.Node{pgquery.MakeStrNode("=")},
						pgquery.MakeColumnRefNode([]*pgquery.Node{pgquery.MakeStrNode("subquery"), pgquery.MakeStrNode("id")}, 0),
						pgquery.MakeAConstIntNode(1, 0),
						0,
					),
					pgquery.MakeBoolExprNode(
						pgquery.BoolExprType_AND_EXPR,
						[]*pgquery.Node{
							pgquery.MakeAExprNode(
								pgquery.A_Expr_Kind_AEXPR_OP,
								[]*pgquery.Node{pgquery.MakeStrNode("=")},
								pgquery.MakeColumnRefNode([]*pgquery.Node{pgquery.MakeStrNode("subquery"), pgquery.MakeStrNode("administrative_class_id")}, 0),
								pgquery.MakeAConstIntNode(1, 0),
								0,
							),
							pgquery.MakeAExprNode(
								pgquery.A_Expr_Kind_AEXPR_OP,
								[]*pgquery.Node{pgquery.MakeStrNode("=")},
								pgquery.MakeColumnRefNode([]*pgquery.Node{pgquery.MakeStrNode("course"), pgquery.MakeStrNode("id")}, 0),
								pgquery.MakeAConstIntNode(1, 0),
								0,
							),
						},
						0,
					),
				},
				0,
			),
			tables: func() *om.OrderedMap[string, *TableInfoV2] {
				// Inner student table (authorized)
				studentTbl := &TableInfoV2{
					Name:       "student",
					Columns:    om.NewOrderedMap[string, *ColumnInfo](),
					Alias:      "student",
					Authorized: true,
					IsDatabase: true,
				}
				for _, col := range schemaService.GetColumns("student") {
					studentTbl.Columns.Set(col, &ColumnInfo{Name: col, SourceTable: studentTbl})
				}
				// Inner administrative class table (authorized)
				adminTbl := &TableInfoV2{
					Name:       "administrative_class",
					Columns:    om.NewOrderedMap[string, *ColumnInfo](),
					Alias:      "admin_class",
					Authorized: true,
					IsDatabase: true,
				}
				for _, col := range schemaService.GetColumns("administrative_class") {
					adminTbl.Columns.Set(col, &ColumnInfo{Name: col, SourceTable: adminTbl})
				}
				// Virtual table for subquery
				virtualTbl := &TableInfoV2{
					Name:               "subquery",
					Columns:            om.NewOrderedMap[string, *ColumnInfo](),
					Alias:              "subquery",
					Authorized:         false,
					IsDatabase:         false,
					UnAuthorizedTables: om.NewOrderedMap[string, *TableInfoV2](),
				}
				virtualTbl.UnAuthorizedTables.Set("student", studentTbl)
				virtualTbl.UnAuthorizedTables.Set("admin_class", adminTbl)
				virtualTbl.Columns.Set("id", &ColumnInfo{Name: "id", SourceTable: studentTbl})
				virtualTbl.Columns.Set("administrative_class_id", &ColumnInfo{Name: "id", SourceTable: adminTbl})
				// Public course table
				courseTbl := &TableInfoV2{
					Name:       "course",
					Columns:    om.NewOrderedMap[string, *ColumnInfo](),
					Alias:      "course",
					Authorized: false,
					IsDatabase: true,
				}
				for _, col := range schemaService.GetColumns("course") {
					courseTbl.Columns.Set(col, &ColumnInfo{Name: col, SourceTable: courseTbl})
				}
				return om.NewOrderedMapWithElements(
					&om.Element[string, *TableInfoV2]{Key: "subquery", Value: virtualTbl},
					&om.Element[string, *TableInfoV2]{Key: "course", Value: courseTbl},
				)
			}(),
			neg:               false,
			userInfo:          &StudentInfo{UserInfo: UserInfo{ID: 1, Role: "student"}, AdministrativeClassID: 1},
			expectedAuth:      true,
			expectedTableAuth: map[string]bool{"subquery": true, "course": true},
		},
		{
			name: "NOT expression with virtual table from unauthorized subquery",
			boolExpr: pgquery.MakeBoolExprNode(
				pgquery.BoolExprType_NOT_EXPR,
				[]*pgquery.Node{
					pgquery.MakeAExprNode(
						pgquery.A_Expr_Kind_AEXPR_OP,
						[]*pgquery.Node{pgquery.MakeStrNode("<>")},
						pgquery.MakeColumnRefNode([]*pgquery.Node{pgquery.MakeStrNode("subquery"), pgquery.MakeStrNode("id")}, 0),
						pgquery.MakeAConstIntNode(1, 0),
						0,
					),
				},
				0,
			),
			tables: func() *om.OrderedMap[string, *TableInfoV2] {
				// Inner student table (unauthorized for ID=2)
				studentTbl := &TableInfoV2{
					Name:       "student",
					Columns:    om.NewOrderedMap[string, *ColumnInfo](),
					Alias:      "student",
					Authorized: false,
					IsDatabase: true,
				}
				for _, col := range schemaService.GetColumns("student") {
					studentTbl.Columns.Set(col, &ColumnInfo{Name: col, SourceTable: studentTbl})
				}
				// Virtual table for subquery
				virtualTbl := &TableInfoV2{
					Name:               "subquery",
					Columns:            om.NewOrderedMap[string, *ColumnInfo](),
					Alias:              "subquery",
					Authorized:         false,
					IsDatabase:         false,
					UnAuthorizedTables: om.NewOrderedMap[string, *TableInfoV2](),
				}
				virtualTbl.UnAuthorizedTables.Set("student", studentTbl)
				virtualTbl.Columns.Set("id", &ColumnInfo{Name: "id", SourceTable: studentTbl})
				return om.NewOrderedMapWithElements(
					&om.Element[string, *TableInfoV2]{Key: "subquery", Value: virtualTbl},
				)
			}(),
			neg:               false,
			userInfo:          &StudentInfo{UserInfo: UserInfo{ID: 1, Role: "student"}},
			expectedAuth:      true,
			expectedTableAuth: map[string]bool{"subquery": true},
		},
		{
			name: "Complex nested boolean with partially authorized virtual table",
			boolExpr: pgquery.MakeBoolExprNode(
				pgquery.BoolExprType_AND_EXPR,
				[]*pgquery.Node{
					pgquery.MakeBoolExprNode(
						pgquery.BoolExprType_OR_EXPR,
						[]*pgquery.Node{
							pgquery.MakeAExprNode(
								pgquery.A_Expr_Kind_AEXPR_OP,
								[]*pgquery.Node{pgquery.MakeStrNode("=")},
								pgquery.MakeColumnRefNode([]*pgquery.Node{pgquery.MakeStrNode("subquery"), pgquery.MakeStrNode("student_id")}, 0),
								pgquery.MakeAConstIntNode(1, 0),
								0,
							),
							pgquery.MakeAExprNode(
								pgquery.A_Expr_Kind_AEXPR_OP,
								[]*pgquery.Node{pgquery.MakeStrNode("=")},
								pgquery.MakeColumnRefNode([]*pgquery.Node{pgquery.MakeStrNode("subquery"), pgquery.MakeStrNode("student_id")}, 0),
								pgquery.MakeAConstIntNode(2, 0),
								0,
							),
						},
						0,
					),
					pgquery.MakeAExprNode(
						pgquery.A_Expr_Kind_AEXPR_OP,
						[]*pgquery.Node{pgquery.MakeStrNode("=")},
						pgquery.MakeColumnRefNode([]*pgquery.Node{pgquery.MakeStrNode("course_class_enrollment"), pgquery.MakeStrNode("student_id")}, 0),
						pgquery.MakeAConstIntNode(1, 0),
						0,
					),
				},
				0,
			),
			tables: func() *om.OrderedMap[string, *TableInfoV2] {
				// Inner student table (partially authorized)
				studentTbl := &TableInfoV2{
					Name:       "student",
					Columns:    om.NewOrderedMap[string, *ColumnInfo](),
					Alias:      "student",
					Authorized: false,
					IsDatabase: true,
				}
				for _, col := range schemaService.GetColumns("student") {
					studentTbl.Columns.Set(col, &ColumnInfo{Name: col, SourceTable: studentTbl})
				}
				// Virtual table for subquery
				virtualTbl := &TableInfoV2{
					Name:               "subquery",
					Columns:            om.NewOrderedMap[string, *ColumnInfo](),
					Alias:              "subquery",
					Authorized:         false,
					IsDatabase:         false,
					UnAuthorizedTables: om.NewOrderedMap[string, *TableInfoV2](),
				}
				virtualTbl.UnAuthorizedTables.Set("student", studentTbl)
				virtualTbl.Columns.Set("student_id", &ColumnInfo{Name: "id", SourceTable: studentTbl})
				// Student course class table
				sccTbl := &TableInfoV2{
					Name:       "course_class_enrollment",
					Columns:    om.NewOrderedMap[string, *ColumnInfo](),
					Alias:      "course_class_enrollment",
					Authorized: false,
					IsDatabase: true,
				}
				for _, col := range schemaService.GetColumns("course_class_enrollment") {
					sccTbl.Columns.Set(col, &ColumnInfo{Name: col, SourceTable: sccTbl})
				}
				return om.NewOrderedMapWithElements(
					&om.Element[string, *TableInfoV2]{Key: "subquery", Value: virtualTbl},
					&om.Element[string, *TableInfoV2]{Key: "course_class_enrollment", Value: sccTbl},
				)
			}(),
			neg:               false,
			userInfo:          &StudentInfo{UserInfo: UserInfo{ID: 1, Role: "student"}},
			expectedAuth:      false,
			expectedTableAuth: map[string]bool{"subquery": false, "course_class_enrollment": true},
		},
		{
			name: "Virtual table with nested NOT and authorized professor access",
			boolExpr: pgquery.MakeBoolExprNode(
				pgquery.BoolExprType_AND_EXPR,
				[]*pgquery.Node{
					pgquery.MakeBoolExprNode(
						pgquery.BoolExprType_NOT_EXPR,
						[]*pgquery.Node{
							pgquery.MakeAExprNode(
								pgquery.A_Expr_Kind_AEXPR_OP,
								[]*pgquery.Node{pgquery.MakeStrNode("=")},
								pgquery.MakeColumnRefNode([]*pgquery.Node{pgquery.MakeStrNode("subquery"), pgquery.MakeStrNode("course_class_id")}, 0),
								pgquery.MakeAConstIntNode(999, 0),
								0,
							),
						},
						0,
					),
					pgquery.MakeAExprNode(
						pgquery.A_Expr_Kind_AEXPR_OP,
						[]*pgquery.Node{pgquery.MakeStrNode("=")},
						pgquery.MakeColumnRefNode([]*pgquery.Node{pgquery.MakeStrNode("subquery"), pgquery.MakeStrNode("id")}, 0),
						pgquery.MakeAConstIntNode(10, 0),
						0,
					),
				},
				0,
			),
			tables: func() *om.OrderedMap[string, *TableInfoV2] {
				// Inner course_class_enrollment table
				sccTbl := &TableInfoV2{
					Name:       "course_class_enrollment",
					Columns:    om.NewOrderedMap[string, *ColumnInfo](),
					Alias:      "scc",
					Authorized: false,
					IsDatabase: true,
				}
				for _, col := range schemaService.GetColumns("course_class_enrollment") {
					sccTbl.Columns.Set(col, &ColumnInfo{Name: col, SourceTable: sccTbl})
				}
				// Virtual table for subquery
				virtualTbl := &TableInfoV2{
					Name:               "subquery",
					Columns:            om.NewOrderedMap[string, *ColumnInfo](),
					Alias:              "subquery",
					Authorized:         false,
					IsDatabase:         false,
					UnAuthorizedTables: om.NewOrderedMap[string, *TableInfoV2](),
				}
				virtualTbl.UnAuthorizedTables.Set("scc", sccTbl)
				virtualTbl.Columns.Set("id", &ColumnInfo{Name: "id", SourceTable: sccTbl})
				virtualTbl.Columns.Set("course_class_id", &ColumnInfo{Name: "course_class_id", SourceTable: sccTbl})
				return om.NewOrderedMapWithElements(
					&om.Element[string, *TableInfoV2]{Key: "subquery", Value: virtualTbl},
				)
			}(),
			neg:               false,
			userInfo:          &ProfessorInfo{UserInfo: UserInfo{ID: 1, Role: "professor"}, TaughtCourseClassIDs: []int{10}},
			expectedAuth:      true,
			expectedTableAuth: map[string]bool{"subquery": true},
		},
		{
			name: "Virtual table with OR and unauthorized inner table",
			boolExpr: pgquery.MakeBoolExprNode(
				pgquery.BoolExprType_OR_EXPR,
				[]*pgquery.Node{
					pgquery.MakeAExprNode(
						pgquery.A_Expr_Kind_AEXPR_OP,
						[]*pgquery.Node{pgquery.MakeStrNode("=")},
						pgquery.MakeColumnRefNode([]*pgquery.Node{pgquery.MakeStrNode("subquery"), pgquery.MakeStrNode("student_id")}, 0),
						pgquery.MakeAConstIntNode(2, 0),
						0,
					),
					pgquery.MakeAExprNode(
						pgquery.A_Expr_Kind_AEXPR_OP,
						[]*pgquery.Node{pgquery.MakeStrNode("=")},
						pgquery.MakeColumnRefNode([]*pgquery.Node{pgquery.MakeStrNode("subquery"), pgquery.MakeStrNode("course_class_id")}, 0),
						pgquery.MakeAConstIntNode(999, 0),
						0,
					),
				},
				0,
			),
			tables: func() *om.OrderedMap[string, *TableInfoV2] {
				// Inner course_class_enrollment table
				sccTbl := &TableInfoV2{
					Name:       "course_class_enrollment",
					Columns:    om.NewOrderedMap[string, *ColumnInfo](),
					Alias:      "scc",
					Authorized: false,
					IsDatabase: true,
				}
				for _, col := range schemaService.GetColumns("course_class_enrollment") {
					sccTbl.Columns.Set(col, &ColumnInfo{Name: col, SourceTable: sccTbl})
				}
				// Virtual table for subquery
				virtualTbl := &TableInfoV2{
					Name:               "subquery",
					Columns:            om.NewOrderedMap[string, *ColumnInfo](),
					Alias:              "subquery",
					Authorized:         false,
					IsDatabase:         false,
					UnAuthorizedTables: om.NewOrderedMap[string, *TableInfoV2](),
				}
				virtualTbl.UnAuthorizedTables.Set("scc", sccTbl)
				virtualTbl.Columns.Set("student_id", &ColumnInfo{Name: "student_id", SourceTable: sccTbl})
				virtualTbl.Columns.Set("course_class_id", &ColumnInfo{Name: "course_class_id", SourceTable: sccTbl})
				return om.NewOrderedMapWithElements(
					&om.Element[string, *TableInfoV2]{Key: "subquery", Value: virtualTbl},
				)
			}(),
			neg:               false,
			userInfo:          &StudentInfo{UserInfo: UserInfo{ID: 1, Role: "student"}},
			expectedAuth:      false,
			expectedTableAuth: map[string]bool{"subquery": false},
		},
		{
			name: "AND with two virtual tables",
			boolExpr: pgquery.MakeBoolExprNode(
				pgquery.BoolExprType_AND_EXPR,
				[]*pgquery.Node{
					pgquery.MakeAExprNode(
						pgquery.A_Expr_Kind_AEXPR_OP,
						[]*pgquery.Node{pgquery.MakeStrNode("=")},
						pgquery.MakeColumnRefNode([]*pgquery.Node{pgquery.MakeStrNode("student_subquery"), pgquery.MakeStrNode("id")}, 0),
						pgquery.MakeAConstIntNode(1, 0),
						0,
					),
					pgquery.MakeAExprNode(
						pgquery.A_Expr_Kind_AEXPR_OP,
						[]*pgquery.Node{pgquery.MakeStrNode("=")},
						pgquery.MakeColumnRefNode([]*pgquery.Node{pgquery.MakeStrNode("scc_subquery"), pgquery.MakeStrNode("student_id")}, 0),
						pgquery.MakeAConstIntNode(1, 0),
						0,
					),
				},
				0,
			),
			tables: func() *om.OrderedMap[string, *TableInfoV2] {
				studentTbl := &TableInfoV2{
					Name:       "student",
					Columns:    om.NewOrderedMap[string, *ColumnInfo](),
					Alias:      "student",
					Authorized: false,
					IsDatabase: true,
				}
				for _, col := range schemaService.GetColumns("student") {
					studentTbl.Columns.Set(col, &ColumnInfo{Name: col, SourceTable: studentTbl})
				}
				sccTbl := &TableInfoV2{
					Name:       "course_class_enrollment",
					Columns:    om.NewOrderedMap[string, *ColumnInfo](),
					Alias:      "scc",
					Authorized: false,
					IsDatabase: true,
				}
				for _, col := range schemaService.GetColumns("course_class_enrollment") {
					sccTbl.Columns.Set(col, &ColumnInfo{Name: col, SourceTable: sccTbl})
				}
				studentVirtual := &TableInfoV2{
					Name: "student_subquery",
					Columns: om.NewOrderedMapWithElements(
						&om.Element[string, *ColumnInfo]{
							Key:   "id",
							Value: &ColumnInfo{Name: "id", SourceTable: studentTbl},
						},
					),
					Alias:              "student_subquery",
					Authorized:         false,
					IsDatabase:         false,
					UnAuthorizedTables: om.NewOrderedMapWithElements(&om.Element[string, *TableInfoV2]{Key: "student", Value: studentTbl}),
				}
				sccVirtual := &TableInfoV2{
					Name: "scc_subquery",
					Columns: om.NewOrderedMapWithElements(
						&om.Element[string, *ColumnInfo]{
							Key:   "student_id",
							Value: &ColumnInfo{Name: "student_id", SourceTable: sccTbl},
						},
					),
					Alias:              "scc_subquery",
					Authorized:         false,
					IsDatabase:         false,
					UnAuthorizedTables: om.NewOrderedMapWithElements(&om.Element[string, *TableInfoV2]{Key: "scc", Value: sccTbl}),
				}
				return om.NewOrderedMapWithElements(
					&om.Element[string, *TableInfoV2]{Key: "student_subquery", Value: studentVirtual},
					&om.Element[string, *TableInfoV2]{Key: "scc_subquery", Value: sccVirtual},
				)
			}(),
			neg:               false,
			userInfo:          &StudentInfo{UserInfo: UserInfo{ID: 1, Role: "student"}},
			expectedAuth:      true,
			expectedTableAuth: map[string]bool{"student_subquery": true, "scc_subquery": true},
		},
	}

	// Run tests
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := authService.authorizeBoolExpr(
				tt.boolExpr.GetBoolExpr(),
				tt.tables,
				tt.neg,
				tt.userInfo,
			)

			// Check for expected error
			if tt.expectedError != "" {
				if err == nil || err.Error() != tt.expectedError {
					t.Errorf("Expected error %q, got %v", tt.expectedError, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Check overall authorization
			if result.Authorized != tt.expectedAuth {
				t.Errorf("Expected Authorized=%v, got %v", tt.expectedAuth, result.Authorized)
			}

			// Check table authorization status
			for alias, expected := range tt.expectedTableAuth {
				table, ok := result.Tables.Get(alias)
				if !ok {
					t.Errorf("Table %s not found in result", alias)
					continue
				}
				if table.Authorized != expected {
					t.Errorf("Table %s: expected Authorized=%v, got %v", alias, expected, table.Authorized)
				}
			}
		})
	}
}

// endregion

// region TestUpdateUnauthorizedTablesByIntersection
func TestUpdateUnauthorizedTablesByIntersection(t *testing.T) {
	// Initialize mock schema service
	schemaService := &MockSchemaService{}
	authService := NewAuthorizationService(schemaService)

	tests := []struct {
		name              string
		dst               *om.OrderedMap[string, *TableInfoV2]
		tablesList        []*om.OrderedMap[string, *TableInfoV2]
		expectedDstTables map[string]struct {
			Authorized         bool
			UnAuthorizedTables map[string]bool // Map of alias to presence in UnAuthorizedTables
		}
	}{
		{
			name: "Database table unauthorized in all",
			dst: func() *om.OrderedMap[string, *TableInfoV2] {
				studentTbl := &TableInfoV2{
					Name:               "student",
					Columns:            om.NewOrderedMap[string, *ColumnInfo](),
					Alias:              "student",
					Authorized:         false,
					IsDatabase:         true,
					UnAuthorizedTables: om.NewOrderedMap[string, *TableInfoV2](),
				}
				for _, col := range schemaService.GetColumns("student") {
					studentTbl.Columns.Set(col, &ColumnInfo{Name: col, SourceTable: studentTbl})
				}
				return om.NewOrderedMapWithElements(
					&om.Element[string, *TableInfoV2]{Key: "student", Value: studentTbl},
				)
			}(),
			tablesList: []*om.OrderedMap[string, *TableInfoV2]{
				func() *om.OrderedMap[string, *TableInfoV2] {
					tbl := &TableInfoV2{
						Name:               "student",
						Columns:            om.NewOrderedMap[string, *ColumnInfo](),
						Alias:              "student",
						Authorized:         false,
						IsDatabase:         true,
						UnAuthorizedTables: om.NewOrderedMap[string, *TableInfoV2](),
					}
					for _, col := range schemaService.GetColumns("student") {
						tbl.Columns.Set(col, &ColumnInfo{Name: col, SourceTable: tbl})
					}
					return om.NewOrderedMapWithElements(
						&om.Element[string, *TableInfoV2]{Key: "student", Value: tbl},
					)
				}(),
				func() *om.OrderedMap[string, *TableInfoV2] {
					tbl := &TableInfoV2{
						Name:               "student",
						Columns:            om.NewOrderedMap[string, *ColumnInfo](),
						Alias:              "student",
						Authorized:         false,
						IsDatabase:         true,
						UnAuthorizedTables: om.NewOrderedMap[string, *TableInfoV2](),
					}
					for _, col := range schemaService.GetColumns("student") {
						tbl.Columns.Set(col, &ColumnInfo{Name: col, SourceTable: tbl})
					}
					return om.NewOrderedMapWithElements(
						&om.Element[string, *TableInfoV2]{Key: "student", Value: tbl},
					)
				}(),
			},
			expectedDstTables: map[string]struct {
				Authorized         bool
				UnAuthorizedTables map[string]bool
			}{
				"student": {Authorized: false, UnAuthorizedTables: map[string]bool{}},
			},
		},
		{
			name: "Database table authorized in one",
			dst: func() *om.OrderedMap[string, *TableInfoV2] {
				studentTbl := &TableInfoV2{
					Name:               "student",
					Columns:            om.NewOrderedMap[string, *ColumnInfo](),
					Alias:              "student",
					Authorized:         false,
					IsDatabase:         true,
					UnAuthorizedTables: om.NewOrderedMap[string, *TableInfoV2](),
				}
				for _, col := range schemaService.GetColumns("student") {
					studentTbl.Columns.Set(col, &ColumnInfo{Name: col, SourceTable: studentTbl})
				}
				return om.NewOrderedMapWithElements(
					&om.Element[string, *TableInfoV2]{Key: "student", Value: studentTbl},
				)
			}(),
			tablesList: []*om.OrderedMap[string, *TableInfoV2]{
				func() *om.OrderedMap[string, *TableInfoV2] {
					tbl := &TableInfoV2{
						Name:               "student",
						Columns:            om.NewOrderedMap[string, *ColumnInfo](),
						Alias:              "student",
						Authorized:         true,
						IsDatabase:         true,
						UnAuthorizedTables: om.NewOrderedMap[string, *TableInfoV2](),
					}
					for _, col := range schemaService.GetColumns("student") {
						tbl.Columns.Set(col, &ColumnInfo{Name: col, SourceTable: tbl})
					}
					return om.NewOrderedMapWithElements(
						&om.Element[string, *TableInfoV2]{Key: "student", Value: tbl},
					)
				}(),
				func() *om.OrderedMap[string, *TableInfoV2] {
					tbl := &TableInfoV2{
						Name:               "student",
						Columns:            om.NewOrderedMap[string, *ColumnInfo](),
						Alias:              "student",
						Authorized:         false,
						IsDatabase:         true,
						UnAuthorizedTables: om.NewOrderedMap[string, *TableInfoV2](),
					}
					for _, col := range schemaService.GetColumns("student") {
						tbl.Columns.Set(col, &ColumnInfo{Name: col, SourceTable: tbl})
					}
					return om.NewOrderedMapWithElements(
						&om.Element[string, *TableInfoV2]{Key: "student", Value: tbl},
					)
				}(),
			},
			expectedDstTables: map[string]struct {
				Authorized         bool
				UnAuthorizedTables map[string]bool
			}{
				"student": {Authorized: true, UnAuthorizedTables: map[string]bool{}},
			},
		},
		{
			name: "Virtual table with all inner tables unauthorized",
			dst: func() *om.OrderedMap[string, *TableInfoV2] {
				studentTbl := &TableInfoV2{
					Name:               "student",
					Columns:            om.NewOrderedMap[string, *ColumnInfo](),
					Alias:              "student",
					Authorized:         false,
					IsDatabase:         true,
					UnAuthorizedTables: om.NewOrderedMap[string, *TableInfoV2](),
				}
				for _, col := range schemaService.GetColumns("student") {
					studentTbl.Columns.Set(col, &ColumnInfo{Name: col, SourceTable: studentTbl})
				}
				courseTbl := &TableInfoV2{
					Name:       "course",
					Columns:    om.NewOrderedMap[string, *ColumnInfo](),
					Alias:      "course",
					Authorized: false,
					IsDatabase: true,
				}
				for _, col := range schemaService.GetColumns("course") {
					courseTbl.Columns.Set(col, &ColumnInfo{Name: col, SourceTable: courseTbl})
				}
				virtualTbl := &TableInfoV2{
					Name:               "subquery",
					Columns:            om.NewOrderedMap[string, *ColumnInfo](),
					Alias:              "subquery",
					Authorized:         false,
					IsDatabase:         false,
					UnAuthorizedTables: om.NewOrderedMap[string, *TableInfoV2](),
				}
				virtualTbl.UnAuthorizedTables.Set("student", studentTbl)
				virtualTbl.UnAuthorizedTables.Set("course", courseTbl)
				virtualTbl.Columns.Set("id", &ColumnInfo{Name: "id", SourceTable: studentTbl})
				virtualTbl.Columns.Set("name", &ColumnInfo{Name: "name", SourceTable: courseTbl})
				return om.NewOrderedMapWithElements(
					&om.Element[string, *TableInfoV2]{Key: "subquery", Value: virtualTbl},
				)
			}(),
			tablesList: []*om.OrderedMap[string, *TableInfoV2]{
				func() *om.OrderedMap[string, *TableInfoV2] {
					studentTbl := &TableInfoV2{
						Name:       "student",
						Columns:    om.NewOrderedMap[string, *ColumnInfo](),
						Alias:      "student",
						Authorized: false,
						IsDatabase: true,
					}
					for _, col := range schemaService.GetColumns("student") {
						studentTbl.Columns.Set(col, &ColumnInfo{Name: col, SourceTable: studentTbl})
					}
					courseTbl := &TableInfoV2{
						Name:       "course",
						Columns:    om.NewOrderedMap[string, *ColumnInfo](),
						Alias:      "course",
						Authorized: false,
						IsDatabase: true,
					}
					for _, col := range schemaService.GetColumns("course") {
						courseTbl.Columns.Set(col, &ColumnInfo{Name: col, SourceTable: courseTbl})
					}
					virtualTbl := &TableInfoV2{
						Name:               "subquery",
						Columns:            om.NewOrderedMap[string, *ColumnInfo](),
						Alias:              "subquery",
						Authorized:         false,
						IsDatabase:         false,
						UnAuthorizedTables: om.NewOrderedMap[string, *TableInfoV2](),
					}
					virtualTbl.UnAuthorizedTables.Set("student", studentTbl)
					virtualTbl.UnAuthorizedTables.Set("course", courseTbl)
					virtualTbl.Columns.Set("id", &ColumnInfo{Name: "id", SourceTable: studentTbl})
					virtualTbl.Columns.Set("name", &ColumnInfo{Name: "name", SourceTable: courseTbl})
					return om.NewOrderedMapWithElements(
						&om.Element[string, *TableInfoV2]{Key: "subquery", Value: virtualTbl},
					)
				}(),
				func() *om.OrderedMap[string, *TableInfoV2] {
					studentTbl := &TableInfoV2{
						Name:       "student",
						Columns:    om.NewOrderedMap[string, *ColumnInfo](),
						Alias:      "student",
						Authorized: false,
						IsDatabase: true,
					}
					for _, col := range schemaService.GetColumns("student") {
						studentTbl.Columns.Set(col, &ColumnInfo{Name: col, SourceTable: studentTbl})
					}
					courseTbl := &TableInfoV2{
						Name:       "course",
						Columns:    om.NewOrderedMap[string, *ColumnInfo](),
						Alias:      "course",
						Authorized: false,
						IsDatabase: true,
					}
					for _, col := range schemaService.GetColumns("course") {
						courseTbl.Columns.Set(col, &ColumnInfo{Name: col, SourceTable: courseTbl})
					}
					virtualTbl := &TableInfoV2{
						Name:               "subquery",
						Columns:            om.NewOrderedMap[string, *ColumnInfo](),
						Alias:              "subquery",
						Authorized:         false,
						IsDatabase:         false,
						UnAuthorizedTables: om.NewOrderedMap[string, *TableInfoV2](),
					}
					virtualTbl.UnAuthorizedTables.Set("student", studentTbl)
					virtualTbl.UnAuthorizedTables.Set("course", courseTbl)
					virtualTbl.Columns.Set("id", &ColumnInfo{Name: "id", SourceTable: studentTbl})
					virtualTbl.Columns.Set("name", &ColumnInfo{Name: "name", SourceTable: courseTbl})
					return om.NewOrderedMapWithElements(
						&om.Element[string, *TableInfoV2]{Key: "subquery", Value: virtualTbl},
					)
				}(),
			},
			expectedDstTables: map[string]struct {
				Authorized         bool
				UnAuthorizedTables map[string]bool
			}{
				"subquery": {Authorized: false, UnAuthorizedTables: map[string]bool{"student": true, "course": true}},
			},
		},
		{
			name: "Virtual table with one inner table authorized",
			dst: func() *om.OrderedMap[string, *TableInfoV2] {
				studentTbl := &TableInfoV2{
					Name:       "student",
					Columns:    om.NewOrderedMap[string, *ColumnInfo](),
					Alias:      "student",
					Authorized: false,
					IsDatabase: true,
				}
				for _, col := range schemaService.GetColumns("student") {
					studentTbl.Columns.Set(col, &ColumnInfo{Name: col, SourceTable: studentTbl})
				}
				courseTbl := &TableInfoV2{
					Name:       "course",
					Columns:    om.NewOrderedMap[string, *ColumnInfo](),
					Alias:      "course",
					Authorized: false,
					IsDatabase: true,
				}
				for _, col := range schemaService.GetColumns("course") {
					courseTbl.Columns.Set(col, &ColumnInfo{Name: col, SourceTable: courseTbl})
				}
				virtualTbl := &TableInfoV2{
					Name:               "subquery",
					Columns:            om.NewOrderedMap[string, *ColumnInfo](),
					Alias:              "subquery",
					Authorized:         false,
					IsDatabase:         false,
					UnAuthorizedTables: om.NewOrderedMap[string, *TableInfoV2](),
				}
				virtualTbl.UnAuthorizedTables.Set("student", studentTbl)
				virtualTbl.UnAuthorizedTables.Set("course", courseTbl)
				virtualTbl.Columns.Set("id", &ColumnInfo{Name: "id", SourceTable: studentTbl})
				virtualTbl.Columns.Set("name", &ColumnInfo{Name: "name", SourceTable: courseTbl})
				return om.NewOrderedMapWithElements(
					&om.Element[string, *TableInfoV2]{Key: "subquery", Value: virtualTbl},
				)
			}(),
			tablesList: []*om.OrderedMap[string, *TableInfoV2]{
				func() *om.OrderedMap[string, *TableInfoV2] {
					studentTbl := &TableInfoV2{
						Name:       "student",
						Columns:    om.NewOrderedMap[string, *ColumnInfo](),
						Alias:      "student",
						Authorized: true,
						IsDatabase: true,
					}
					for _, col := range schemaService.GetColumns("student") {
						studentTbl.Columns.Set(col, &ColumnInfo{Name: col, SourceTable: studentTbl})
					}
					courseTbl := &TableInfoV2{
						Name:       "course",
						Columns:    om.NewOrderedMap[string, *ColumnInfo](),
						Alias:      "course",
						Authorized: false,
						IsDatabase: true,
					}
					for _, col := range schemaService.GetColumns("course") {
						courseTbl.Columns.Set(col, &ColumnInfo{Name: col, SourceTable: courseTbl})
					}
					virtualTbl := &TableInfoV2{
						Name:               "subquery",
						Columns:            om.NewOrderedMap[string, *ColumnInfo](),
						Alias:              "subquery",
						Authorized:         false,
						IsDatabase:         false,
						UnAuthorizedTables: om.NewOrderedMap[string, *TableInfoV2](),
					}
					virtualTbl.UnAuthorizedTables.Set("student", studentTbl)
					virtualTbl.UnAuthorizedTables.Set("course", courseTbl)
					virtualTbl.Columns.Set("id", &ColumnInfo{Name: "id", SourceTable: studentTbl})
					virtualTbl.Columns.Set("name", &ColumnInfo{Name: "name", SourceTable: courseTbl})
					return om.NewOrderedMapWithElements(
						&om.Element[string, *TableInfoV2]{Key: "subquery", Value: virtualTbl},
					)
				}(),
				func() *om.OrderedMap[string, *TableInfoV2] {
					studentTbl := &TableInfoV2{
						Name:       "student",
						Columns:    om.NewOrderedMap[string, *ColumnInfo](),
						Alias:      "student",
						Authorized: false,
						IsDatabase: true,
					}
					for _, col := range schemaService.GetColumns("student") {
						studentTbl.Columns.Set(col, &ColumnInfo{Name: col, SourceTable: studentTbl})
					}
					courseTbl := &TableInfoV2{
						Name:       "course",
						Columns:    om.NewOrderedMap[string, *ColumnInfo](),
						Alias:      "course",
						Authorized: false,
						IsDatabase: true,
					}
					for _, col := range schemaService.GetColumns("course") {
						courseTbl.Columns.Set(col, &ColumnInfo{Name: col, SourceTable: courseTbl})
					}
					virtualTbl := &TableInfoV2{
						Name:               "subquery",
						Columns:            om.NewOrderedMap[string, *ColumnInfo](),
						Alias:              "subquery",
						Authorized:         false,
						IsDatabase:         false,
						UnAuthorizedTables: om.NewOrderedMap[string, *TableInfoV2](),
					}
					virtualTbl.UnAuthorizedTables.Set("student", studentTbl)
					virtualTbl.UnAuthorizedTables.Set("course", courseTbl)
					virtualTbl.Columns.Set("id", &ColumnInfo{Name: "id", SourceTable: studentTbl})
					virtualTbl.Columns.Set("name", &ColumnInfo{Name: "name", SourceTable: courseTbl})
					return om.NewOrderedMapWithElements(
						&om.Element[string, *TableInfoV2]{Key: "subquery", Value: virtualTbl},
					)
				}(),
			},
			expectedDstTables: map[string]struct {
				Authorized         bool
				UnAuthorizedTables map[string]bool
			}{
				"subquery": {Authorized: false, UnAuthorizedTables: map[string]bool{"course": true}},
			},
		},
		{
			name: "Virtual table referencing another virtual table",
			dst: func() *om.OrderedMap[string, *TableInfoV2] {
				studentTbl := &TableInfoV2{
					Name:               "student",
					Columns:            om.NewOrderedMap[string, *ColumnInfo](),
					Alias:              "student",
					Authorized:         false,
					IsDatabase:         true,
					UnAuthorizedTables: om.NewOrderedMap[string, *TableInfoV2](),
				}
				for _, col := range schemaService.GetColumns("student") {
					studentTbl.Columns.Set(col, &ColumnInfo{Name: col, SourceTable: studentTbl})
				}
				innerVirtualTbl := &TableInfoV2{
					Name:               "inner_subquery",
					Columns:            om.NewOrderedMap[string, *ColumnInfo](),
					Alias:              "inner_subquery",
					Authorized:         false,
					IsDatabase:         false,
					UnAuthorizedTables: om.NewOrderedMap[string, *TableInfoV2](),
				}
				innerVirtualTbl.UnAuthorizedTables.Set("student", studentTbl)
				innerVirtualTbl.Columns.Set("id", &ColumnInfo{Name: "id", SourceTable: studentTbl})
				virtualTbl := &TableInfoV2{
					Name:               "subquery",
					Columns:            om.NewOrderedMap[string, *ColumnInfo](),
					Alias:              "subquery",
					Authorized:         false,
					IsDatabase:         false,
					UnAuthorizedTables: om.NewOrderedMap[string, *TableInfoV2](),
				}
				virtualTbl.UnAuthorizedTables.Set("student", studentTbl)
				virtualTbl.Columns.Set("id", &ColumnInfo{Name: "id", SourceTable: innerVirtualTbl})
				return om.NewOrderedMapWithElements(
					&om.Element[string, *TableInfoV2]{Key: "subquery", Value: virtualTbl},
				)
			}(),
			tablesList: []*om.OrderedMap[string, *TableInfoV2]{
				func() *om.OrderedMap[string, *TableInfoV2] {
					studentTbl := &TableInfoV2{
						Name:               "student",
						Columns:            om.NewOrderedMap[string, *ColumnInfo](),
						Alias:              "student",
						Authorized:         true,
						IsDatabase:         true,
						UnAuthorizedTables: om.NewOrderedMap[string, *TableInfoV2](),
					}
					for _, col := range schemaService.GetColumns("student") {
						studentTbl.Columns.Set(col, &ColumnInfo{Name: col, SourceTable: studentTbl})
					}
					innerVirtualTbl := &TableInfoV2{
						Name:               "inner_subquery",
						Columns:            om.NewOrderedMap[string, *ColumnInfo](),
						Alias:              "inner_subquery",
						Authorized:         false,
						IsDatabase:         false,
						UnAuthorizedTables: om.NewOrderedMap[string, *TableInfoV2](),
					}
					innerVirtualTbl.UnAuthorizedTables.Set("student", studentTbl)
					innerVirtualTbl.Columns.Set("id", &ColumnInfo{Name: "id", SourceTable: studentTbl})
					virtualTbl := &TableInfoV2{
						Name:               "subquery",
						Columns:            om.NewOrderedMap[string, *ColumnInfo](),
						Alias:              "subquery",
						Authorized:         false,
						IsDatabase:         false,
						UnAuthorizedTables: om.NewOrderedMap[string, *TableInfoV2](),
					}
					virtualTbl.UnAuthorizedTables.Set("student", studentTbl)
					virtualTbl.Columns.Set("id", &ColumnInfo{Name: "id", SourceTable: innerVirtualTbl})
					return om.NewOrderedMapWithElements(
						&om.Element[string, *TableInfoV2]{Key: "subquery", Value: virtualTbl},
					)
				}(),
				func() *om.OrderedMap[string, *TableInfoV2] {
					studentTbl := &TableInfoV2{
						Name:               "student",
						Columns:            om.NewOrderedMap[string, *ColumnInfo](),
						Alias:              "student",
						Authorized:         false,
						IsDatabase:         true,
						UnAuthorizedTables: om.NewOrderedMap[string, *TableInfoV2](),
					}
					for _, col := range schemaService.GetColumns("student") {
						studentTbl.Columns.Set(col, &ColumnInfo{Name: col, SourceTable: studentTbl})
					}
					innerVirtualTbl := &TableInfoV2{
						Name:               "inner_subquery",
						Columns:            om.NewOrderedMap[string, *ColumnInfo](),
						Alias:              "inner_subquery",
						Authorized:         false,
						IsDatabase:         false,
						UnAuthorizedTables: om.NewOrderedMap[string, *TableInfoV2](),
					}
					innerVirtualTbl.UnAuthorizedTables.Set("student", studentTbl)
					innerVirtualTbl.Columns.Set("id", &ColumnInfo{Name: "id", SourceTable: studentTbl})
					virtualTbl := &TableInfoV2{
						Name:               "subquery",
						Columns:            om.NewOrderedMap[string, *ColumnInfo](),
						Alias:              "subquery",
						Authorized:         false,
						IsDatabase:         false,
						UnAuthorizedTables: om.NewOrderedMap[string, *TableInfoV2](),
					}
					virtualTbl.UnAuthorizedTables.Set("student", studentTbl)
					virtualTbl.Columns.Set("id", &ColumnInfo{Name: "id", SourceTable: innerVirtualTbl})
					return om.NewOrderedMapWithElements(
						&om.Element[string, *TableInfoV2]{Key: "subquery", Value: virtualTbl},
					)
				}(),
			},
			expectedDstTables: map[string]struct {
				Authorized         bool
				UnAuthorizedTables map[string]bool
			}{
				"subquery": {Authorized: true, UnAuthorizedTables: map[string]bool{}},
			},
		},
		{
			name: "Multiple tables with mixed authorization",
			dst: func() *om.OrderedMap[string, *TableInfoV2] {
				studentTbl := &TableInfoV2{
					Name:               "student",
					Columns:            om.NewOrderedMap[string, *ColumnInfo](),
					Alias:              "student",
					Authorized:         false,
					IsDatabase:         true,
					UnAuthorizedTables: om.NewOrderedMap[string, *TableInfoV2](),
				}
				for _, col := range schemaService.GetColumns("student") {
					studentTbl.Columns.Set(col, &ColumnInfo{Name: col, SourceTable: studentTbl})
				}
				courseTbl := &TableInfoV2{
					Name:               "course",
					Columns:            om.NewOrderedMap[string, *ColumnInfo](),
					Alias:              "course",
					Authorized:         false,
					IsDatabase:         true,
					UnAuthorizedTables: om.NewOrderedMap[string, *TableInfoV2](),
				}
				for _, col := range schemaService.GetColumns("course") {
					courseTbl.Columns.Set(col, &ColumnInfo{Name: col, SourceTable: courseTbl})
				}
				return om.NewOrderedMapWithElements(
					&om.Element[string, *TableInfoV2]{Key: "student", Value: studentTbl},
					&om.Element[string, *TableInfoV2]{Key: "course", Value: courseTbl},
				)
			}(),
			tablesList: []*om.OrderedMap[string, *TableInfoV2]{
				func() *om.OrderedMap[string, *TableInfoV2] {
					studentTbl := &TableInfoV2{
						Name:               "student",
						Columns:            om.NewOrderedMap[string, *ColumnInfo](),
						Alias:              "student",
						Authorized:         true,
						IsDatabase:         true,
						UnAuthorizedTables: om.NewOrderedMap[string, *TableInfoV2](),
					}
					for _, col := range schemaService.GetColumns("student") {
						studentTbl.Columns.Set(col, &ColumnInfo{Name: col, SourceTable: studentTbl})
					}
					courseTbl := &TableInfoV2{
						Name:               "course",
						Columns:            om.NewOrderedMap[string, *ColumnInfo](),
						Alias:              "course",
						Authorized:         false,
						IsDatabase:         true,
						UnAuthorizedTables: om.NewOrderedMap[string, *TableInfoV2](),
					}
					for _, col := range schemaService.GetColumns("course") {
						courseTbl.Columns.Set(col, &ColumnInfo{Name: col, SourceTable: courseTbl})
					}
					return om.NewOrderedMapWithElements(
						&om.Element[string, *TableInfoV2]{Key: "student", Value: studentTbl},
						&om.Element[string, *TableInfoV2]{Key: "course", Value: courseTbl},
					)
				}(),
				func() *om.OrderedMap[string, *TableInfoV2] {
					studentTbl := &TableInfoV2{
						Name:               "student",
						Columns:            om.NewOrderedMap[string, *ColumnInfo](),
						Alias:              "student",
						Authorized:         false,
						IsDatabase:         true,
						UnAuthorizedTables: om.NewOrderedMap[string, *TableInfoV2](),
					}
					for _, col := range schemaService.GetColumns("student") {
						studentTbl.Columns.Set(col, &ColumnInfo{Name: col, SourceTable: studentTbl})
					}
					courseTbl := &TableInfoV2{
						Name:               "course",
						Columns:            om.NewOrderedMap[string, *ColumnInfo](),
						Alias:              "course",
						Authorized:         false,
						IsDatabase:         true,
						UnAuthorizedTables: om.NewOrderedMap[string, *TableInfoV2](),
					}
					for _, col := range schemaService.GetColumns("course") {
						courseTbl.Columns.Set(col, &ColumnInfo{Name: col, SourceTable: courseTbl})
					}
					return om.NewOrderedMapWithElements(
						&om.Element[string, *TableInfoV2]{Key: "student", Value: studentTbl},
						&om.Element[string, *TableInfoV2]{Key: "course", Value: courseTbl},
					)
				}(),
			},
			expectedDstTables: map[string]struct {
				Authorized         bool
				UnAuthorizedTables map[string]bool
			}{
				"student": {Authorized: true, UnAuthorizedTables: map[string]bool{}},
				"course":  {Authorized: false, UnAuthorizedTables: map[string]bool{}},
			},
		},
		{
			name: "Empty tables list",
			dst: func() *om.OrderedMap[string, *TableInfoV2] {
				studentTbl := &TableInfoV2{
					Name:               "student",
					Columns:            om.NewOrderedMap[string, *ColumnInfo](),
					Alias:              "student",
					Authorized:         true,
					IsDatabase:         true,
					UnAuthorizedTables: om.NewOrderedMap[string, *TableInfoV2](),
				}
				for _, col := range schemaService.GetColumns("student") {
					studentTbl.Columns.Set(col, &ColumnInfo{Name: col, SourceTable: studentTbl})
				}
				return om.NewOrderedMapWithElements(
					&om.Element[string, *TableInfoV2]{Key: "student", Value: studentTbl},
				)
			}(),
			tablesList: []*om.OrderedMap[string, *TableInfoV2]{},
			expectedDstTables: map[string]struct {
				Authorized         bool
				UnAuthorizedTables map[string]bool
			}{
				"student": {Authorized: true, UnAuthorizedTables: map[string]bool{}},
			},
		},
		{
			name: "No unauthorized tables in virtual table",
			dst: func() *om.OrderedMap[string, *TableInfoV2] {
				virtualTbl := &TableInfoV2{
					Name:               "subquery",
					Columns:            om.NewOrderedMap[string, *ColumnInfo](),
					Alias:              "subquery",
					Authorized:         false,
					IsDatabase:         false,
					UnAuthorizedTables: om.NewOrderedMap[string, *TableInfoV2](),
				}
				virtualTbl.Columns.Set("id", &ColumnInfo{Name: "id", SourceTable: virtualTbl})
				return om.NewOrderedMapWithElements(
					&om.Element[string, *TableInfoV2]{Key: "subquery", Value: virtualTbl},
				)
			}(),
			tablesList: []*om.OrderedMap[string, *TableInfoV2]{
				func() *om.OrderedMap[string, *TableInfoV2] {
					virtualTbl := &TableInfoV2{
						Name:               "subquery",
						Columns:            om.NewOrderedMap[string, *ColumnInfo](),
						Alias:              "subquery",
						Authorized:         false,
						IsDatabase:         false,
						UnAuthorizedTables: om.NewOrderedMap[string, *TableInfoV2](),
					}
					virtualTbl.Columns.Set("id", &ColumnInfo{Name: "id", SourceTable: virtualTbl})
					return om.NewOrderedMapWithElements(
						&om.Element[string, *TableInfoV2]{Key: "subquery", Value: virtualTbl},
					)
				}(),
			},
			expectedDstTables: map[string]struct {
				Authorized         bool
				UnAuthorizedTables map[string]bool
			}{
				"subquery": {Authorized: true, UnAuthorizedTables: map[string]bool{}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authService.UpdateUnauthorizedTablesByIntersection(tt.dst, tt.tablesList)

			// Verify the state of dst tables
			for alias, expected := range tt.expectedDstTables {
				table, ok := tt.dst.Get(alias)
				if !ok {
					t.Errorf("Table %s not found in dst", alias)
					continue
				}

				// Check Authorized status
				if table.Authorized != expected.Authorized {
					t.Errorf("Table %s: expected Authorized=%v, got %v", alias, expected.Authorized, table.Authorized)
				}

				// Check UnAuthorizedTables
				actualUnAuth := make(map[string]bool)
				for _, unAuthTable := range table.UnAuthorizedTables.AllFromFront() {
					actualUnAuth[unAuthTable.Alias] = true
				}
				if !reflect.DeepEqual(actualUnAuth, expected.UnAuthorizedTables) {
					t.Errorf("Table %s: expected UnAuthorizedTables=%v, got %v", alias, expected.UnAuthorizedTables, actualUnAuth)
				}
			}
		})
	}
}

// endregion

// region TestUpdateUnauthorizedTablesByUnion
func TestUpdateUnauthorizedTablesByUnion(t *testing.T) {
	// Initialize mock schema service
	schemaService := &MockSchemaService{}
	authService := NewAuthorizationService(schemaService)

	tests := []struct {
		name              string
		dst               *om.OrderedMap[string, *TableInfoV2]
		tablesList        []*om.OrderedMap[string, *TableInfoV2]
		expectedDstTables map[string]struct {
			Authorized         bool
			UnAuthorizedTables map[string]bool // Map of alias to presence in UnAuthorizedTables
		}
	}{
		{
			name: "Database table unauthorized in one",
			dst: func() *om.OrderedMap[string, *TableInfoV2] {
				studentTbl := &TableInfoV2{
					Name:               "student",
					Columns:            om.NewOrderedMap[string, *ColumnInfo](),
					Alias:              "student",
					Authorized:         true,
					IsDatabase:         true,
					UnAuthorizedTables: om.NewOrderedMap[string, *TableInfoV2](),
				}
				for _, col := range schemaService.GetColumns("student") {
					studentTbl.Columns.Set(col, &ColumnInfo{Name: col, SourceTable: studentTbl})
				}
				return om.NewOrderedMapWithElements(
					&om.Element[string, *TableInfoV2]{Key: "student", Value: studentTbl},
				)
			}(),
			tablesList: []*om.OrderedMap[string, *TableInfoV2]{
				func() *om.OrderedMap[string, *TableInfoV2] {
					tbl := &TableInfoV2{
						Name:               "student",
						Columns:            om.NewOrderedMap[string, *ColumnInfo](),
						Alias:              "student",
						Authorized:         true,
						IsDatabase:         true,
						UnAuthorizedTables: om.NewOrderedMap[string, *TableInfoV2](),
					}
					for _, col := range schemaService.GetColumns("student") {
						tbl.Columns.Set(col, &ColumnInfo{Name: col, SourceTable: tbl})
					}
					return om.NewOrderedMapWithElements(
						&om.Element[string, *TableInfoV2]{Key: "student", Value: tbl},
					)
				}(),
				func() *om.OrderedMap[string, *TableInfoV2] {
					tbl := &TableInfoV2{
						Name:               "student",
						Columns:            om.NewOrderedMap[string, *ColumnInfo](),
						Alias:              "student",
						Authorized:         false,
						IsDatabase:         true,
						UnAuthorizedTables: om.NewOrderedMap[string, *TableInfoV2](),
					}
					for _, col := range schemaService.GetColumns("student") {
						tbl.Columns.Set(col, &ColumnInfo{Name: col, SourceTable: tbl})
					}
					return om.NewOrderedMapWithElements(
						&om.Element[string, *TableInfoV2]{Key: "student", Value: tbl},
					)
				}(),
			},
			expectedDstTables: map[string]struct {
				Authorized         bool
				UnAuthorizedTables map[string]bool
			}{
				"student": {Authorized: false, UnAuthorizedTables: map[string]bool{}},
			},
		},
		{
			name: "Database table authorized in all",
			dst: func() *om.OrderedMap[string, *TableInfoV2] {
				studentTbl := &TableInfoV2{
					Name:               "student",
					Columns:            om.NewOrderedMap[string, *ColumnInfo](),
					Alias:              "student",
					Authorized:         false,
					IsDatabase:         true,
					UnAuthorizedTables: om.NewOrderedMap[string, *TableInfoV2](),
				}
				for _, col := range schemaService.GetColumns("student") {
					studentTbl.Columns.Set(col, &ColumnInfo{Name: col, SourceTable: studentTbl})
				}
				return om.NewOrderedMapWithElements(
					&om.Element[string, *TableInfoV2]{Key: "student", Value: studentTbl},
				)
			}(),
			tablesList: []*om.OrderedMap[string, *TableInfoV2]{
				func() *om.OrderedMap[string, *TableInfoV2] {
					tbl := &TableInfoV2{
						Name:               "student",
						Columns:            om.NewOrderedMap[string, *ColumnInfo](),
						Alias:              "student",
						Authorized:         true,
						IsDatabase:         true,
						UnAuthorizedTables: om.NewOrderedMap[string, *TableInfoV2](),
					}
					for _, col := range schemaService.GetColumns("student") {
						tbl.Columns.Set(col, &ColumnInfo{Name: col, SourceTable: tbl})
					}
					return om.NewOrderedMapWithElements(
						&om.Element[string, *TableInfoV2]{Key: "student", Value: tbl},
					)
				}(),
				func() *om.OrderedMap[string, *TableInfoV2] {
					tbl := &TableInfoV2{
						Name:               "student",
						Columns:            om.NewOrderedMap[string, *ColumnInfo](),
						Alias:              "student",
						Authorized:         true,
						IsDatabase:         true,
						UnAuthorizedTables: om.NewOrderedMap[string, *TableInfoV2](),
					}
					for _, col := range schemaService.GetColumns("student") {
						tbl.Columns.Set(col, &ColumnInfo{Name: col, SourceTable: tbl})
					}
					return om.NewOrderedMapWithElements(
						&om.Element[string, *TableInfoV2]{Key: "student", Value: tbl},
					)
				}(),
			},
			expectedDstTables: map[string]struct {
				Authorized         bool
				UnAuthorizedTables map[string]bool
			}{
				"student": {Authorized: true, UnAuthorizedTables: map[string]bool{}},
			},
		},
		{
			name: "Virtual table with one unauthorized table removed",
			dst: func() *om.OrderedMap[string, *TableInfoV2] {
				studentTbl := &TableInfoV2{
					Name:               "student",
					Columns:            om.NewOrderedMap[string, *ColumnInfo](),
					Alias:              "student",
					Authorized:         true,
					IsDatabase:         true,
					UnAuthorizedTables: om.NewOrderedMap[string, *TableInfoV2](),
				}
				for _, col := range schemaService.GetColumns("student") {
					studentTbl.Columns.Set(col, &ColumnInfo{Name: col, SourceTable: studentTbl})
				}
				courseTbl := &TableInfoV2{
					Name:               "course",
					Columns:            om.NewOrderedMap[string, *ColumnInfo](),
					Alias:              "course",
					Authorized:         false,
					IsDatabase:         true,
					UnAuthorizedTables: om.NewOrderedMap[string, *TableInfoV2](),
				}
				for _, col := range schemaService.GetColumns("course") {
					courseTbl.Columns.Set(col, &ColumnInfo{Name: col, SourceTable: courseTbl})
				}
				virtualTbl := &TableInfoV2{
					Name:               "subquery",
					Columns:            om.NewOrderedMap[string, *ColumnInfo](),
					Alias:              "subquery",
					Authorized:         false,
					IsDatabase:         false,
					UnAuthorizedTables: om.NewOrderedMap[string, *TableInfoV2](),
				}
				virtualTbl.UnAuthorizedTables.Set("student", studentTbl)
				virtualTbl.UnAuthorizedTables.Set("course", courseTbl)
				virtualTbl.Columns.Set("id", &ColumnInfo{Name: "id", SourceTable: studentTbl})
				virtualTbl.Columns.Set("name", &ColumnInfo{Name: "name", SourceTable: courseTbl})
				return om.NewOrderedMapWithElements(
					&om.Element[string, *TableInfoV2]{Key: "subquery", Value: virtualTbl},
				)
			}(),
			tablesList: []*om.OrderedMap[string, *TableInfoV2]{
				func() *om.OrderedMap[string, *TableInfoV2] {
					studentTbl := &TableInfoV2{
						Name:               "student",
						Columns:            om.NewOrderedMap[string, *ColumnInfo](),
						Alias:              "student",
						Authorized:         true,
						IsDatabase:         true,
						UnAuthorizedTables: om.NewOrderedMap[string, *TableInfoV2](),
					}
					for _, col := range schemaService.GetColumns("student") {
						studentTbl.Columns.Set(col, &ColumnInfo{Name: col, SourceTable: studentTbl})
					}
					courseTbl := &TableInfoV2{
						Name:               "course",
						Columns:            om.NewOrderedMap[string, *ColumnInfo](),
						Alias:              "course",
						Authorized:         false,
						IsDatabase:         true,
						UnAuthorizedTables: om.NewOrderedMap[string, *TableInfoV2](),
					}
					for _, col := range schemaService.GetColumns("course") {
						courseTbl.Columns.Set(col, &ColumnInfo{Name: col, SourceTable: courseTbl})
					}
					virtualTbl := &TableInfoV2{
						Name:               "subquery",
						Columns:            om.NewOrderedMap[string, *ColumnInfo](),
						Alias:              "subquery",
						Authorized:         false,
						IsDatabase:         false,
						UnAuthorizedTables: om.NewOrderedMap[string, *TableInfoV2](),
					}
					virtualTbl.UnAuthorizedTables.Set("student", studentTbl)
					virtualTbl.UnAuthorizedTables.Set("course", courseTbl)
					virtualTbl.Columns.Set("id", &ColumnInfo{Name: "id", SourceTable: studentTbl})
					virtualTbl.Columns.Set("name", &ColumnInfo{Name: "name", SourceTable: courseTbl})
					return om.NewOrderedMapWithElements(
						&om.Element[string, *TableInfoV2]{Key: "subquery", Value: virtualTbl},
					)
				}(),
				func() *om.OrderedMap[string, *TableInfoV2] {
					studentTbl := &TableInfoV2{
						Name:               "student",
						Columns:            om.NewOrderedMap[string, *ColumnInfo](),
						Alias:              "student",
						Authorized:         true,
						IsDatabase:         true,
						UnAuthorizedTables: om.NewOrderedMap[string, *TableInfoV2](),
					}
					for _, col := range schemaService.GetColumns("student") {
						studentTbl.Columns.Set(col, &ColumnInfo{Name: col, SourceTable: studentTbl})
					}
					courseTbl := &TableInfoV2{
						Name:               "course",
						Columns:            om.NewOrderedMap[string, *ColumnInfo](),
						Alias:              "course",
						Authorized:         false,
						IsDatabase:         true,
						UnAuthorizedTables: om.NewOrderedMap[string, *TableInfoV2](),
					}
					for _, col := range schemaService.GetColumns("course") {
						courseTbl.Columns.Set(col, &ColumnInfo{Name: col, SourceTable: courseTbl})
					}
					virtualTbl := &TableInfoV2{
						Name:               "subquery",
						Columns:            om.NewOrderedMap[string, *ColumnInfo](),
						Alias:              "subquery",
						Authorized:         false,
						IsDatabase:         false,
						UnAuthorizedTables: om.NewOrderedMap[string, *TableInfoV2](),
					}
					virtualTbl.UnAuthorizedTables.Set("student", studentTbl)
					virtualTbl.UnAuthorizedTables.Set("course", courseTbl)
					virtualTbl.Columns.Set("id", &ColumnInfo{Name: "id", SourceTable: studentTbl})
					virtualTbl.Columns.Set("name", &ColumnInfo{Name: "name", SourceTable: courseTbl})
					return om.NewOrderedMapWithElements(
						&om.Element[string, *TableInfoV2]{Key: "subquery", Value: virtualTbl},
					)
				}(),
			},
			expectedDstTables: map[string]struct {
				Authorized         bool
				UnAuthorizedTables map[string]bool
			}{
				"subquery": {Authorized: false, UnAuthorizedTables: map[string]bool{"course": true}},
			},
		},
		{
			name: "Virtual table with multiple authorized tables fully removed",
			dst: func() *om.OrderedMap[string, *TableInfoV2] {
				studentTbl := &TableInfoV2{
					Name:               "student",
					Columns:            om.NewOrderedMap[string, *ColumnInfo](),
					Alias:              "student",
					Authorized:         true,
					IsDatabase:         true,
					UnAuthorizedTables: om.NewOrderedMap[string, *TableInfoV2](),
				}
				for _, col := range schemaService.GetColumns("student") {
					studentTbl.Columns.Set(col, &ColumnInfo{Name: col, SourceTable: studentTbl})
				}
				courseTbl := &TableInfoV2{
					Name:               "course",
					Columns:            om.NewOrderedMap[string, *ColumnInfo](),
					Alias:              "course",
					Authorized:         true,
					IsDatabase:         true,
					UnAuthorizedTables: om.NewOrderedMap[string, *TableInfoV2](),
				}
				for _, col := range schemaService.GetColumns("course") {
					courseTbl.Columns.Set(col, &ColumnInfo{Name: col, SourceTable: courseTbl})
				}
				virtualTbl := &TableInfoV2{
					Name:               "subquery",
					Columns:            om.NewOrderedMap[string, *ColumnInfo](),
					Alias:              "subquery",
					Authorized:         false,
					IsDatabase:         false,
					UnAuthorizedTables: om.NewOrderedMap[string, *TableInfoV2](),
				}
				virtualTbl.UnAuthorizedTables.Set("student", studentTbl)
				virtualTbl.UnAuthorizedTables.Set("course", courseTbl)
				virtualTbl.Columns.Set("id", &ColumnInfo{Name: "id", SourceTable: studentTbl})
				virtualTbl.Columns.Set("name", &ColumnInfo{Name: "name", SourceTable: courseTbl})
				return om.NewOrderedMapWithElements(
					&om.Element[string, *TableInfoV2]{Key: "subquery", Value: virtualTbl},
				)
			}(),
			tablesList: []*om.OrderedMap[string, *TableInfoV2]{
				func() *om.OrderedMap[string, *TableInfoV2] {
					studentTbl := &TableInfoV2{
						Name:               "student",
						Columns:            om.NewOrderedMap[string, *ColumnInfo](),
						Alias:              "student",
						Authorized:         true,
						IsDatabase:         true,
						UnAuthorizedTables: om.NewOrderedMap[string, *TableInfoV2](),
					}
					for _, col := range schemaService.GetColumns("student") {
						studentTbl.Columns.Set(col, &ColumnInfo{Name: col, SourceTable: studentTbl})
					}
					courseTbl := &TableInfoV2{
						Name:               "course",
						Columns:            om.NewOrderedMap[string, *ColumnInfo](),
						Alias:              "course",
						Authorized:         true,
						IsDatabase:         true,
						UnAuthorizedTables: om.NewOrderedMap[string, *TableInfoV2](),
					}
					for _, col := range schemaService.GetColumns("course") {
						courseTbl.Columns.Set(col, &ColumnInfo{Name: col, SourceTable: courseTbl})
					}
					virtualTbl := &TableInfoV2{
						Name:               "subquery",
						Columns:            om.NewOrderedMap[string, *ColumnInfo](),
						Alias:              "subquery",
						Authorized:         false,
						IsDatabase:         false,
						UnAuthorizedTables: om.NewOrderedMap[string, *TableInfoV2](),
					}
					virtualTbl.UnAuthorizedTables.Set("student", studentTbl)
					virtualTbl.UnAuthorizedTables.Set("course", courseTbl)
					virtualTbl.Columns.Set("id", &ColumnInfo{Name: "id", SourceTable: studentTbl})
					virtualTbl.Columns.Set("name", &ColumnInfo{Name: "name", SourceTable: courseTbl})
					return om.NewOrderedMapWithElements(
						&om.Element[string, *TableInfoV2]{Key: "subquery", Value: virtualTbl},
					)
				}(),
				func() *om.OrderedMap[string, *TableInfoV2] {
					studentTbl := &TableInfoV2{
						Name:               "student",
						Columns:            om.NewOrderedMap[string, *ColumnInfo](),
						Alias:              "student",
						Authorized:         true,
						IsDatabase:         true,
						UnAuthorizedTables: om.NewOrderedMap[string, *TableInfoV2](),
					}
					for _, col := range schemaService.GetColumns("student") {
						studentTbl.Columns.Set(col, &ColumnInfo{Name: col, SourceTable: studentTbl})
					}
					courseTbl := &TableInfoV2{
						Name:               "course",
						Columns:            om.NewOrderedMap[string, *ColumnInfo](),
						Alias:              "course",
						Authorized:         true,
						IsDatabase:         true,
						UnAuthorizedTables: om.NewOrderedMap[string, *TableInfoV2](),
					}
					for _, col := range schemaService.GetColumns("course") {
						courseTbl.Columns.Set(col, &ColumnInfo{Name: "name", SourceTable: courseTbl})
					}
					virtualTbl := &TableInfoV2{
						Name:               "subquery",
						Columns:            om.NewOrderedMap[string, *ColumnInfo](),
						Alias:              "subquery",
						Authorized:         false,
						IsDatabase:         false,
						UnAuthorizedTables: om.NewOrderedMap[string, *TableInfoV2](),
					}
					virtualTbl.UnAuthorizedTables.Set("student", studentTbl)
					virtualTbl.UnAuthorizedTables.Set("course", courseTbl)
					virtualTbl.Columns.Set("id", &ColumnInfo{Name: "id", SourceTable: studentTbl})
					virtualTbl.Columns.Set("name", &ColumnInfo{Name: "name", SourceTable: courseTbl})
					return om.NewOrderedMapWithElements(
						&om.Element[string, *TableInfoV2]{Key: "subquery", Value: virtualTbl},
					)
				}(),
			},
			expectedDstTables: map[string]struct {
				Authorized         bool
				UnAuthorizedTables map[string]bool
			}{
				"subquery": {Authorized: true, UnAuthorizedTables: map[string]bool{}},
			},
		},
		{
			name: "Multiple tables with mixed authorization",
			dst: func() *om.OrderedMap[string, *TableInfoV2] {
				studentTbl := &TableInfoV2{
					Name:               "student",
					Columns:            om.NewOrderedMap[string, *ColumnInfo](),
					Alias:              "student",
					Authorized:         true,
					IsDatabase:         true,
					UnAuthorizedTables: om.NewOrderedMap[string, *TableInfoV2](),
				}
				for _, col := range schemaService.GetColumns("student") {
					studentTbl.Columns.Set(col, &ColumnInfo{Name: col, SourceTable: studentTbl})
				}
				courseTbl := &TableInfoV2{
					Name:               "course",
					Columns:            om.NewOrderedMap[string, *ColumnInfo](),
					Alias:              "course",
					Authorized:         true,
					IsDatabase:         true,
					UnAuthorizedTables: om.NewOrderedMap[string, *TableInfoV2](),
				}
				for _, col := range schemaService.GetColumns("course") {
					courseTbl.Columns.Set(col, &ColumnInfo{Name: col, SourceTable: courseTbl})
				}
				return om.NewOrderedMapWithElements(
					&om.Element[string, *TableInfoV2]{Key: "student", Value: studentTbl},
					&om.Element[string, *TableInfoV2]{Key: "course", Value: courseTbl},
				)
			}(),
			tablesList: []*om.OrderedMap[string, *TableInfoV2]{
				func() *om.OrderedMap[string, *TableInfoV2] {
					studentTbl := &TableInfoV2{
						Name:               "student",
						Columns:            om.NewOrderedMap[string, *ColumnInfo](),
						Alias:              "student",
						Authorized:         true,
						IsDatabase:         true,
						UnAuthorizedTables: om.NewOrderedMap[string, *TableInfoV2](),
					}
					for _, col := range schemaService.GetColumns("student") {
						studentTbl.Columns.Set(col, &ColumnInfo{Name: col, SourceTable: studentTbl})
					}
					courseTbl := &TableInfoV2{
						Name:               "course",
						Columns:            om.NewOrderedMap[string, *ColumnInfo](),
						Alias:              "course",
						Authorized:         false,
						IsDatabase:         true,
						UnAuthorizedTables: om.NewOrderedMap[string, *TableInfoV2](),
					}
					for _, col := range schemaService.GetColumns("course") {
						courseTbl.Columns.Set(col, &ColumnInfo{Name: col, SourceTable: courseTbl})
					}
					return om.NewOrderedMapWithElements(
						&om.Element[string, *TableInfoV2]{Key: "student", Value: studentTbl},
						&om.Element[string, *TableInfoV2]{Key: "course", Value: courseTbl},
					)
				}(),
				func() *om.OrderedMap[string, *TableInfoV2] {
					studentTbl := &TableInfoV2{
						Name:               "student",
						Columns:            om.NewOrderedMap[string, *ColumnInfo](),
						Alias:              "student",
						Authorized:         false,
						IsDatabase:         true,
						UnAuthorizedTables: om.NewOrderedMap[string, *TableInfoV2](),
					}
					for _, col := range schemaService.GetColumns("student") {
						studentTbl.Columns.Set(col, &ColumnInfo{Name: col, SourceTable: studentTbl})
					}
					courseTbl := &TableInfoV2{
						Name:               "course",
						Columns:            om.NewOrderedMap[string, *ColumnInfo](),
						Alias:              "course",
						Authorized:         true,
						IsDatabase:         true,
						UnAuthorizedTables: om.NewOrderedMap[string, *TableInfoV2](),
					}
					for _, col := range schemaService.GetColumns("course") {
						courseTbl.Columns.Set(col, &ColumnInfo{Name: col, SourceTable: courseTbl})
					}
					return om.NewOrderedMapWithElements(
						&om.Element[string, *TableInfoV2]{Key: "student", Value: studentTbl},
						&om.Element[string, *TableInfoV2]{Key: "course", Value: courseTbl},
					)
				}(),
			},
			expectedDstTables: map[string]struct {
				Authorized         bool
				UnAuthorizedTables map[string]bool
			}{
				"student": {Authorized: false, UnAuthorizedTables: map[string]bool{}},
				"course":  {Authorized: false, UnAuthorizedTables: map[string]bool{}},
			},
		},
		{
			name: "Empty tables list",
			dst: func() *om.OrderedMap[string, *TableInfoV2] {
				studentTbl := &TableInfoV2{
					Name:               "student",
					Columns:            om.NewOrderedMap[string, *ColumnInfo](),
					Alias:              "student",
					Authorized:         false,
					IsDatabase:         true,
					UnAuthorizedTables: om.NewOrderedMap[string, *TableInfoV2](),
				}
				for _, col := range schemaService.GetColumns("student") {
					studentTbl.Columns.Set(col, &ColumnInfo{Name: col, SourceTable: studentTbl})
				}
				return om.NewOrderedMapWithElements(
					&om.Element[string, *TableInfoV2]{Key: "student", Value: studentTbl},
				)
			}(),
			tablesList: []*om.OrderedMap[string, *TableInfoV2]{},
			expectedDstTables: map[string]struct {
				Authorized         bool
				UnAuthorizedTables map[string]bool
			}{
				"student": {Authorized: false, UnAuthorizedTables: map[string]bool{}},
			},
		},
		{
			name: "Nil tables list",
			dst: func() *om.OrderedMap[string, *TableInfoV2] {
				studentTbl := &TableInfoV2{
					Name:               "student",
					Columns:            om.NewOrderedMap[string, *ColumnInfo](),
					Alias:              "student",
					Authorized:         false,
					IsDatabase:         true,
					UnAuthorizedTables: om.NewOrderedMap[string, *TableInfoV2](),
				}
				for _, col := range schemaService.GetColumns("student") {
					studentTbl.Columns.Set(col, &ColumnInfo{Name: col, SourceTable: studentTbl})
				}
				return om.NewOrderedMapWithElements(
					&om.Element[string, *TableInfoV2]{Key: "student", Value: studentTbl},
				)
			}(),
			tablesList: nil,
			expectedDstTables: map[string]struct {
				Authorized         bool
				UnAuthorizedTables map[string]bool
			}{
				"student": {Authorized: false, UnAuthorizedTables: map[string]bool{}},
			},
		},
		{
			name: "No unauthorized tables in virtual table",
			dst: func() *om.OrderedMap[string, *TableInfoV2] {
				studentTbl := &TableInfoV2{
					Name:               "student",
					Columns:            om.NewOrderedMap[string, *ColumnInfo](),
					Alias:              "student",
					Authorized:         true,
					IsDatabase:         true,
					UnAuthorizedTables: om.NewOrderedMap[string, *TableInfoV2](),
				}
				for _, col := range schemaService.GetColumns("student") {
					studentTbl.Columns.Set(col, &ColumnInfo{Name: col, SourceTable: studentTbl})
				}
				virtualTbl := &TableInfoV2{
					Name:               "subquery",
					Columns:            om.NewOrderedMap[string, *ColumnInfo](),
					Alias:              "subquery",
					Authorized:         false,
					IsDatabase:         false,
					UnAuthorizedTables: om.NewOrderedMap[string, *TableInfoV2](),
				}
				virtualTbl.Columns.Set("id", &ColumnInfo{Name: "id", SourceTable: studentTbl})
				return om.NewOrderedMapWithElements(
					&om.Element[string, *TableInfoV2]{Key: "subquery", Value: virtualTbl},
				)
			}(),
			tablesList: []*om.OrderedMap[string, *TableInfoV2]{
				func() *om.OrderedMap[string, *TableInfoV2] {
					studentTbl := &TableInfoV2{
						Name:               "student",
						Columns:            om.NewOrderedMap[string, *ColumnInfo](),
						Alias:              "student",
						Authorized:         true,
						IsDatabase:         true,
						UnAuthorizedTables: om.NewOrderedMap[string, *TableInfoV2](),
					}
					for _, col := range schemaService.GetColumns("student") {
						studentTbl.Columns.Set(col, &ColumnInfo{Name: col, SourceTable: studentTbl})
					}
					virtualTbl := &TableInfoV2{
						Name:               "subquery",
						Columns:            om.NewOrderedMap[string, *ColumnInfo](),
						Alias:              "subquery",
						Authorized:         false,
						IsDatabase:         false,
						UnAuthorizedTables: om.NewOrderedMap[string, *TableInfoV2](),
					}
					virtualTbl.Columns.Set("id", &ColumnInfo{Name: "id", SourceTable: studentTbl})
					return om.NewOrderedMapWithElements(
						&om.Element[string, *TableInfoV2]{Key: "subquery", Value: virtualTbl},
					)
				}(),
			},
			expectedDstTables: map[string]struct {
				Authorized         bool
				UnAuthorizedTables map[string]bool
			}{
				"subquery": {Authorized: true, UnAuthorizedTables: map[string]bool{}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authService.UpdateUnauthorizedTablesByUnion(tt.dst, tt.tablesList)

			// Verify the state of dst tables
			for alias, expected := range tt.expectedDstTables {
				table, ok := tt.dst.Get(alias)
				if !ok {
					t.Errorf("Table %s not found in dst", alias)
					continue
				}

				// Check Authorized status
				if table.Authorized != expected.Authorized {
					t.Errorf("Table %s: expected Authorized=%v, got %v", alias, expected.Authorized, table.Authorized)
				}

				// Check UnAuthorizedTables
				actualUnAuth := make(map[string]bool)
				for _, unAuthTable := range table.UnAuthorizedTables.AllFromFront() {
					actualUnAuth[unAuthTable.Alias] = true
				}
				if !reflect.DeepEqual(actualUnAuth, expected.UnAuthorizedTables) {
					t.Errorf("Table %s: expected UnAuthorizedTables=%v, got %v", alias, expected.UnAuthorizedTables, actualUnAuth)
				}
			}
		})
	}
}

// endregion

// region TestIsAuthorizedExpression
func TestIsAuthorizedExpression(t *testing.T) {
	// Initialize mock schema service and authorization service
	schemaService := &MockSchemaService{}
	authService := NewAuthorizationService(schemaService)

	tests := []struct {
		name            string
		columnRef       *pgquery.ColumnRef
		aConst          *pgquery.A_Const
		targetTable     *TableInfoV2
		realUnAuthTable *TableInfoV2
		userInfo        UserContext
		expected        bool
	}{
		{
			name: "Public table bypasses authorization",
			columnRef: pgquery.MakeColumnRefNode([]*pgquery.Node{
				pgquery.MakeStrNode("course"),
				pgquery.MakeStrNode("id"),
			}, 0).GetColumnRef(),
			aConst: pgquery.MakeAConstIntNode(1, 0).GetAConst(),
			targetTable: &TableInfoV2{
				Name:       "course",
				Columns:    om.NewOrderedMap[string, *ColumnInfo](),
				Alias:      "course",
				IsDatabase: true,
			},
			realUnAuthTable: &TableInfoV2{
				Name:       "course",
				Columns:    om.NewOrderedMap[string, *ColumnInfo](),
				Alias:      "course",
				IsDatabase: true,
			},
			userInfo: &StudentInfo{UserInfo: UserInfo{ID: 1, Role: "student"}},
			expected: true,
		},
		{
			name: "Admin role bypasses authorization",
			columnRef: pgquery.MakeColumnRefNode([]*pgquery.Node{
				pgquery.MakeStrNode("student"),
				pgquery.MakeStrNode("id"),
			}, 0).GetColumnRef(),
			aConst: pgquery.MakeAConstIntNode(2, 0).GetAConst(),
			targetTable: &TableInfoV2{
				Name:       "student",
				Columns:    om.NewOrderedMap[string, *ColumnInfo](),
				Alias:      "student",
				IsDatabase: true,
			},
			realUnAuthTable: &TableInfoV2{
				Name:       "student",
				Columns:    om.NewOrderedMap[string, *ColumnInfo](),
				Alias:      "student",
				IsDatabase: true,
			},
			userInfo: &UserInfo{ID: 1, Role: "admin"},
			expected: true,
		},
		{
			name: "Student accessing own data in student table",
			columnRef: pgquery.MakeColumnRefNode([]*pgquery.Node{
				pgquery.MakeStrNode("student"),
				pgquery.MakeStrNode("id"),
			}, 0).GetColumnRef(),
			aConst: pgquery.MakeAConstIntNode(1, 0).GetAConst(),
			targetTable: &TableInfoV2{
				Name: "student",
				Columns: om.NewOrderedMapWithElements(
					&om.Element[string, *ColumnInfo]{
						Key:   "id",
						Value: &ColumnInfo{Name: "id", SourceTable: &TableInfoV2{Name: "student"}},
					},
				),
				Alias:      "student",
				IsDatabase: true,
			},
			realUnAuthTable: &TableInfoV2{
				Name:       "student",
				Columns:    om.NewOrderedMap[string, *ColumnInfo](),
				Alias:      "student",
				IsDatabase: true,
			},
			userInfo: &StudentInfo{UserInfo: UserInfo{ID: 1, Role: "student"}},
			expected: true,
		},
		{
			name: "Student accessing another student's data",
			columnRef: pgquery.MakeColumnRefNode([]*pgquery.Node{
				pgquery.MakeStrNode("student"),
				pgquery.MakeStrNode("id"),
			}, 0).GetColumnRef(),
			aConst: pgquery.MakeAConstIntNode(2, 0).GetAConst(),
			targetTable: &TableInfoV2{
				Name: "student",
				Columns: om.NewOrderedMapWithElements(
					&om.Element[string, *ColumnInfo]{
						Key:   "id",
						Value: &ColumnInfo{Name: "id", SourceTable: &TableInfoV2{Name: "student"}},
					},
				),
				Alias:      "student",
				IsDatabase: true,
			},
			realUnAuthTable: &TableInfoV2{
				Name:       "student",
				Columns:    om.NewOrderedMap[string, *ColumnInfo](),
				Alias:      "student",
				IsDatabase: true,
			},
			userInfo: &StudentInfo{UserInfo: UserInfo{ID: 1, Role: "student"}},
			expected: false,
		},
		{
			name: "Professor accessing taught course class",
			columnRef: pgquery.MakeColumnRefNode([]*pgquery.Node{
				pgquery.MakeStrNode("course_class_enrollment"),
				pgquery.MakeStrNode("id"),
			}, 0).GetColumnRef(),
			aConst: pgquery.MakeAConstIntNode(1, 0).GetAConst(),
			targetTable: &TableInfoV2{
				Name: "course_class_enrollment",
				Columns: om.NewOrderedMapWithElements(
					&om.Element[string, *ColumnInfo]{
						Key:   "id",
						Value: &ColumnInfo{Name: "id", SourceTable: &TableInfoV2{Name: "course_class_enrollment"}},
					},
				),
				Alias:      "course_class_enrollment",
				IsDatabase: true,
			},
			realUnAuthTable: &TableInfoV2{
				Name:       "course_class_enrollment",
				Columns:    om.NewOrderedMap[string, *ColumnInfo](),
				Alias:      "course_class_enrollment",
				IsDatabase: true,
			},
			userInfo: &ProfessorInfo{UserInfo: UserInfo{ID: 1, Role: "professor"}, TaughtCourseClassIDs: []int{1}},
			expected: true,
		},
		{
			name: "Column name conflict across tables - student_id in student vs course_class_enrollment",
			columnRef: pgquery.MakeColumnRefNode([]*pgquery.Node{
				pgquery.MakeStrNode("course_class_enrollment"),
				pgquery.MakeStrNode("student_id"),
			}, 0).GetColumnRef(),
			aConst: pgquery.MakeAConstIntNode(1, 0).GetAConst(),
			targetTable: &TableInfoV2{
				Name: "course_class_enrollment",
				Columns: om.NewOrderedMapWithElements(
					&om.Element[string, *ColumnInfo]{
						Key:   "student_id",
						Value: &ColumnInfo{Name: "student_id", SourceTable: &TableInfoV2{Name: "course_class_enrollment"}},
					},
				),
				Alias:      "course_class_enrollment",
				IsDatabase: true,
			},
			realUnAuthTable: &TableInfoV2{
				Name:       "course_class_enrollment",
				Columns:    om.NewOrderedMap[string, *ColumnInfo](),
				Alias:      "course_class_enrollment",
				IsDatabase: true,
			},
			userInfo: &StudentInfo{UserInfo: UserInfo{ID: 1, Role: "student"}},
			expected: true,
		},
		{
			name: "Column name conflict - student accessing student_id in wrong table",
			columnRef: pgquery.MakeColumnRefNode([]*pgquery.Node{
				pgquery.MakeStrNode("student"),
				pgquery.MakeStrNode("student_id"),
			}, 0).GetColumnRef(),
			aConst: pgquery.MakeAConstIntNode(1, 0).GetAConst(),
			targetTable: &TableInfoV2{
				Name: "student",
				Columns: om.NewOrderedMapWithElements(
					&om.Element[string, *ColumnInfo]{
						Key:   "student_id",
						Value: &ColumnInfo{Name: "student_id", SourceTable: &TableInfoV2{Name: "student"}},
					},
				),
				Alias:      "student",
				IsDatabase: true,
			},
			realUnAuthTable: &TableInfoV2{
				Name:       "course_class_enrollment",
				Columns:    om.NewOrderedMap[string, *ColumnInfo](),
				Alias:      "course_class_enrollment",
				IsDatabase: true,
			},
			userInfo: &StudentInfo{UserInfo: UserInfo{ID: 1, Role: "student"}},
			expected: false,
		},
		{
			name: "Missing column info",
			columnRef: pgquery.MakeColumnRefNode([]*pgquery.Node{
				pgquery.MakeStrNode("student"),
				pgquery.MakeStrNode("non_existent_column"),
			}, 0).GetColumnRef(),
			aConst: pgquery.MakeAConstIntNode(1, 0).GetAConst(),
			targetTable: &TableInfoV2{
				Name:       "student",
				Columns:    om.NewOrderedMap[string, *ColumnInfo](),
				Alias:      "student",
				IsDatabase: true,
			},
			realUnAuthTable: &TableInfoV2{
				Name:       "student",
				Columns:    om.NewOrderedMap[string, *ColumnInfo](),
				Alias:      "student",
				IsDatabase: true,
			},
			userInfo: &StudentInfo{UserInfo: UserInfo{ID: 1, Role: "student"}},
			expected: false,
		},
	}

	// Run tests
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := authService.isAuthorizedExpression(tt.columnRef, tt.aConst, tt.targetTable, tt.realUnAuthTable, tt.userInfo)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// endregion
