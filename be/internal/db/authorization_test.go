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
			name: "AND expression with both conditions authorized",
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
			neg:          false,
			userInfo:     &StudentInfo{UserInfo: UserInfo{ID: 1, Role: "student"}},
			expectedAuth: true,
			expectedTableAuth: map[string]bool{
				"student": true,
				"course":  true,
			},
		},
		{
			name: "AND expression with one condition unauthorized",
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
			neg:          false,
			userInfo:     &StudentInfo{UserInfo: UserInfo{ID: 1, Role: "student"}},
			expectedAuth: false,
			expectedTableAuth: map[string]bool{
				"student": false,
			},
		},
		{
			name: "OR expression with one condition authorized",
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
			neg:          false,
			userInfo:     &StudentInfo{UserInfo: UserInfo{ID: 1, Role: "student"}},
			expectedAuth: false,
			expectedTableAuth: map[string]bool{
				"student": false,
			},
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
			neg:          false,
			userInfo:     &StudentInfo{UserInfo: UserInfo{ID: 1, Role: "student"}},
			expectedAuth: false,
			expectedTableAuth: map[string]bool{
				"student": false,
			},
		},
		{
			name: "NOT expression with unauthorized condition",
			boolExpr: pgquery.MakeBoolExprNode(
				pgquery.BoolExprType_NOT_EXPR,
				[]*pgquery.Node{
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
			neg:          false,
			userInfo:     &StudentInfo{UserInfo: UserInfo{ID: 1, Role: "student"}},
			expectedAuth: true,
			expectedTableAuth: map[string]bool{
				"student": true,
			},
		},
		{
			name: "Nested AND and OR expression",
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
								pgquery.MakeColumnRefNode([]*pgquery.Node{pgquery.MakeStrNode("course"), pgquery.MakeStrNode("id")}, 0),
								pgquery.MakeAConstIntNode(1, 0),
								0,
							),
							pgquery.MakeAExprNode(
								pgquery.A_Expr_Kind_AEXPR_OP,
								[]*pgquery.Node{pgquery.MakeStrNode("=")},
								pgquery.MakeColumnRefNode([]*pgquery.Node{pgquery.MakeStrNode("course"), pgquery.MakeStrNode("id")}, 0),
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
			neg:          false,
			userInfo:     &StudentInfo{UserInfo: UserInfo{ID: 1, Role: "student"}},
			expectedAuth: true,
			expectedTableAuth: map[string]bool{
				"student": true,
				"course":  true,
			},
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
			neg:          false,
			userInfo:     &StudentInfo{UserInfo: UserInfo{ID: 1, Role: "student"}},
			expectedAuth: true,
			expectedTableAuth: map[string]bool{
				"subquery": true,
			},
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
