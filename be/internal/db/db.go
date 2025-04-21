package db

import (
	"context"
	"database/sql"
	"fmt"
	pgquery "github.com/pganalyze/pg_query_go/v6"
	"time"

	"github.com/jmoiron/sqlx"
)

var ddlLoaders = make(map[string]DDLLoader)

// HDb defines the database operations interface
type HDb interface {
	LoadDDL() (string, error)
	ExecuteQuery(ctx context.Context, query QueryRequest) (*QueryResult, error)
	GetContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

// SQLHDb implements the HDb interface
type SQLHDb struct {
	*sqlx.DB
	DDLLoader
}

// NewHDb creates a new HDb instance
func NewHDb(driverName, dataSourceUrl string) (*SQLHDb, error) {
	db, err := sqlx.Connect(driverName, dataSourceUrl)
	if err != nil {
		return nil, err
	}
	ddlLoader := ddlLoaders[driverName]

	return &SQLHDb{db, ddlLoader}, nil
}

// LoadDDL loads the database schema
func (db *SQLHDb) LoadDDL() (string, error) {
	return db.DDLLoader.LoadDDL(db.DB)
}

// ExecuteQuery executes a SQL query and returns the results
func (db *SQLHDb) ExecuteQuery(ctx context.Context, req QueryRequest) (*QueryResult, error) {
	// Validate the query
	isValid, errMsg, err := validateQuery(ctx, req.Query)
	if err != nil {
		return nil, fmt.Errorf("error validating query: %v", err)
	}
	if !isValid {
		return nil, fmt.Errorf("invalid query: %s", errMsg)
	}

	rows, err := db.QueryxContext(ctx, req.Query)
	if err != nil {
		return nil, fmt.Errorf("error executing query: %v", err)
	}
	defer rows.Close()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("error getting columns: %v", err)
	}

	// Prepare result structure
	result := &QueryResult{}
	result.Metadata.Columns = columns

	// Scan rows into slice of maps
	var allRows []map[string]interface{}
	for rows.Next() {
		row := make(map[string]interface{})
		err := rows.MapScan(row)
		if err != nil {
			return nil, fmt.Errorf("error scanning row: %v", err)
		}

		// Process each value in the row
		for key, val := range row {
			// Handle null values
			if val == nil {
				continue
			}

			// Convert to appropriate type
			switch v := val.(type) {
			case []byte:
				// Try to convert []byte to string
				row[key] = string(v)
			case time.Time:
				// Format time as ISO8601
				row[key] = v.Format(time.RFC3339)
			default:
				// Use value as is
				row[key] = v
			}
		}

		allRows = append(allRows, row)
	}

	// Check for errors from iterating over rows
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %v", err)
	}

	result.Data = allRows
	result.Metadata.RowCount = len(allRows)

	return result, nil
}

func validateQuery(ctx context.Context, query string) (bool, string, error) {
	// Use pg_query_go to parse the SQL query
	tree, err := pgquery.Parse(query)
	if err != nil {
		return false, "Failed to parse SQL query", err
	}

	// Only allow SELECT statements for security
	if len(tree.Stmts) == 0 {
		return false, "Empty query", nil
	}

	stmt := tree.Stmts[0]
	selectStmt := stmt.Stmt.GetSelectStmt()
	if selectStmt == nil {
		return false, "Only SELECT queries are allowed", nil
	}

	// Extract tables being accessed and conditions from the query
	tables, err := extractTables(selectStmt)
	if err != nil {
		return false, "Failed to extract tables from query", err
	}

	// Get user information from context that was set in ExecuteQuery
	//userId := ctx.Value("userId").(string)
	userRole := ctx.Value("userRole").(string)

	// Apply different validation rules based on role
	switch userRole {
	case "admin":
		// Admins can access everything
		return true, "", nil

	case "student":
		// Check if student is only accessing permitted tables
		for _, table := range tables {
			// Public tables are accessible to all
			if isPublicTable(table.Name) {
				continue
			}

			// Add more table-specific access rules for students
		}

	case "professor":
		// Check if professor is only accessing permitted tables
		for _, table := range tables {
			if isPublicTable(table.Name) {
				continue
			}

		}

	default:
		return false, "Unknown role: " + userRole, nil
	}

	return true, "", nil
}

// Helper functions
func isPublicTable(tableName string) bool {
	publicTables := []string{
		"faculty",
		"program",
		"course",
		"course_program",
		"course_class",
		"course_class_schedule",
	}

	for _, table := range publicTables {
		if table == tableName {
			return true
		}
	}
	return false
}

func validateSelectQueryWithRole(query *pgquery.SelectStmt, role string, userId int) (bool, error) {
	if query == nil {
		return false, fmt.Errorf("query is nil")
	}
	// First handle simple SELECT cases
	if query.GetFromClause() != nil && !hasSubquery(query) {
		//Check join conditions first
		unAuthTables := make([]TableInfo, 0)
		for _, fromItem := range query.FromClause {
			if joinExpr := fromItem.GetJoinExpr(); joinExpr != nil {
				resTables, err := validateJoinTree(joinExpr, role, userId)
				if err != nil {
					return false, err
				}

				unAuthTables = append(unAuthTables, resTables...)
			} else if rangeVar := fromItem.GetRangeVar(); rangeVar != nil {
				currentTable := TableInfo{
					Name:  rangeVar.Relname,
					Alias: rangeVar.Alias.Aliasname,
				}

				if isPublicTable(currentTable.Name) {
					continue
				}
				unAuthTables = append(unAuthTables, currentTable)
			}
		}

		if len(unAuthTables) > 0 {
			// Check if the user has access to the tables
			for _, table := range unAuthTables {
				if isPublicTable(table.Name) {
					// No need to check but be paranoid
					continue
				}

				if !validateCondition(query.WhereClause, table, role, userId) {
					return false, fmt.Errorf("unauthorized access to table %s", table.Name)
				}
			}
		}
		// For remain unauthorized tables, check the where clause
	}

	result := true

	//// Check if all CTEs have proper role conditions
	//if query.GetWithClause() != nil {
	//	for _, cte := range query.WithClause.Ctes {
	//		if cteSelect := cte.GetSelectStmt(); cteSelect != nil {
	//			// Recursively validate CTEs
	//			cteIsValid, err := validateSelectQueryWithRole(cteSelect, role)
	//			if err != nil {
	//				return false, err
	//			}
	//			if !cteIsValid {
	//				hasRoleCond = false
	//				break
	//			}
	//		}
	//	}
	//}
	//
	//// Check for each select clause in UNION queries
	//if query.GetOp() == pgquery.SetOperation_SETOP_UNION {
	//	lValid, lErr := validateSelectQueryWithRole(query.Larg, role)
	//	if lErr != nil || !lValid {
	//		return false, lErr
	//	}
	//	rValid, rErr := validateSelectQueryWithRole(query.Rarg, role)
	//	if rErr != nil || !rValid {
	//		return false, rErr
	//	}
	//}
	//
	//// Check if the main query has proper role conditions
	//if query.FromClause != nil {
	//	for _, fromItem := range query.FromClause {
	//		if joinExpr := fromItem.GetJoinExpr(); joinExpr != nil {
	//			if joinSelect := joinExpr.Larg.GetSelectStmt(); joinSelect != nil {
	//				joinIsValid, err := validateSelectQueryWithRole(joinSelect, role)
	//				if err != nil {
	//					return false, err
	//				}
	//				if !joinIsValid {
	//					hasRoleCond = false
	//					break
	//				}
	//			}
	//		}
	//	}
	//}

	return result, nil
}

// validateJoinTree validates the join tree for a given role and user ID
// Returns the remain unauthorized tables
func validateJoinTree(joinExpr *pgquery.JoinExpr, role string, userId int) ([]TableInfo, error) {
	if joinExpr == nil {
		return nil, fmt.Errorf("joinExpr is nil")
	}
	var unAuthTables []TableInfo

	if joinExpr.Larg == nil {
		return nil, fmt.Errorf("left side of join is nil")
	}
	if joinExpr.Rarg == nil {
		return nil, fmt.Errorf("right side of join is nil")
	}

	// Recursively validate the left side of the join
	if joinExpr.Larg.GetJoinExpr() != nil {
		// Still have join on the left side ?
		childUnAuthTables, err := validateJoinTree(joinExpr.Larg.GetJoinExpr(), role, userId)
		if err == nil {
			unAuthTables = append(unAuthTables, childUnAuthTables...)
		}
	}

	// Validate the base case where left and right are both RangeVar
	for index, node := range []*pgquery.Node{joinExpr.Larg, joinExpr.Rarg} {
		if rangeVar := node.GetRangeVar(); rangeVar != nil {
			currentTable := TableInfo{
				Name:  rangeVar.Relname,
				Alias: rangeVar.Alias.Aliasname,
			}

			if isPublicTable(currentTable.Name) {
				continue
			}

			if isForceCheckOutside(joinExpr.Jointype, index == 0) {
				unAuthTables = append(unAuthTables, currentTable)
			}

			if joinExpr.GetQuals() != nil {
				// Check if the join condition is valid for the role
				if !validateCondition(joinExpr.GetQuals(), currentTable, role, userId) {
					unAuthTables = append(unAuthTables, currentTable)
				}
			}

		}
	}
	return nil, nil
}
func validateCondition(qualNode *pgquery.Node, table TableInfo, role string, id int) bool {
	if qualNode == nil {
		return false
	}

	switch qualNode.Node.(type) {
	case *pgquery.Node_BoolExpr:
		return validateBoolExpr(qualNode.GetBoolExpr(), table, role, id)
	case *pgquery.Node_SubLink:
		return validateSubLink(qualNode.GetSubLink(), role, id)
	case *pgquery.Node_AExpr:
		return validateAExpr(qualNode.GetAExpr(), table, role, id)
	default:
		return false
	}
}

// Handle boolean expressions (AND, OR, NOT)
func validateBoolExpr(boolExpr *pgquery.BoolExpr, table TableInfo, role string, id int) bool {
	if boolExpr == nil {
		return true
	}

	switch boolExpr.GetBoolop() {
	case pgquery.BoolExprType_OR_EXPR:
		// All subexpressions must be authorized
		for _, arg := range boolExpr.Args {
			if !validateCondition(arg, table, role, id) {
				return false
			}
		}
		return true

	case pgquery.BoolExprType_AND_EXPR:
		// At least one subexpression must be authorized
		for _, arg := range boolExpr.Args {
			if validateCondition(arg, table, role, id) {
				return true
			}
		}
		return false

	case pgquery.BoolExprType_NOT_EXPR:
		// Negation: check if the subexpression is unauthorized
		if len(boolExpr.Args) == 1 {
			return !validateCondition(boolExpr.Args[0], table, role, id)
		}
		return false

	default:
		return false
	}
}

// Handle subqueries
func validateSubLink(subLink *pgquery.SubLink, role string, id int) bool {
	if subLink == nil {
		return true
	}

	subquery := subLink.GetSubselect()
	if subquery == nil {
		return true
	}

	selectStmt := subquery.GetSelectStmt()
	if selectStmt == nil {
		return true
	}

	authorized, err := validateSelectQueryWithRole(selectStmt, role, id)
	return authorized && err == nil
}

// Handle A_Expr nodes (comparison operators, IN clauses, etc.)
func validateAExpr(aExpr *pgquery.A_Expr, table TableInfo, role string, id int) bool {
	if aExpr == nil || aExpr.GetName() == nil {
		return true
	}

	switch aExpr.GetKind() {
	case pgquery.A_Expr_Kind_AEXPR_OP:
		operator := aExpr.GetName()[0].String()
		if operator != "=" {
			return false // Only equality operator is authorized
		}

		columnRef, aConst := extractColumnRefAndConst(aExpr)
		if columnRef == nil || aConst == nil {
			return false
		}

		return validateColumnRef(columnRef, table, role) &&
			validateConstValue(aConst, role, id)

		// TODO: Handle other operators

	case pgquery.A_Expr_Kind_AEXPR_IN:
		// The condition need to be of the form "column IN (value)"
		if aExpr.GetLexpr() == nil || aExpr.GetRexpr() == nil {
			return false
		}

		// Validate left side (should be a column reference)
		lexpr := aExpr.GetLexpr()
		columnRef := lexpr.GetColumnRef()
		if columnRef == nil {
			return false
		}

		lResult := validateColumnRef(columnRef, table, role)
		if !lResult {
			return false
		}

		// Validate right side (should be a list with only one constant matching user ID)
		rexpr := aExpr.GetRexpr()
		list := rexpr.GetList()
		if list == nil || list.GetItems() == nil || len(list.GetItems()) != 1 {
			return false
		}

		if list.GetItems()[0].GetAConst() != nil {
			aConst := list.GetItems()[0].GetAConst()
			return validateConstValue(aConst, role, id)
		}

		return role == "admin" // Admins can access all values

	default:
		return role == "admin"
	}
}

// Extract column reference and constant from an expression
func extractColumnRefAndConst(aExpr *pgquery.A_Expr) (*pgquery.ColumnRef, *pgquery.A_Const) {
	if aExpr.GetLexpr() == nil || aExpr.GetRexpr() == nil {
		return nil, nil
	}

	var columnRef *pgquery.ColumnRef
	var aConst *pgquery.A_Const

	// Check if column is on left and constant is on right
	if aExpr.GetLexpr().GetColumnRef() != nil && aExpr.GetRexpr().GetAConst() != nil {
		columnRef = aExpr.GetLexpr().GetColumnRef()
		aConst = aExpr.GetRexpr().GetAConst()
	} else if aExpr.GetLexpr().GetAConst() != nil && aExpr.GetRexpr().GetColumnRef() != nil {
		columnRef = aExpr.GetRexpr().GetColumnRef()
		aConst = aExpr.GetLexpr().GetAConst()
	}

	return columnRef, aConst
}

// Validate if a column reference matches table/role conditions
func validateColumnRef(columnRef *pgquery.ColumnRef, table TableInfo, role string) bool {
	if columnRef == nil {
		return false
	}

	fields := columnRef.GetFields()
	if fields == nil || len(fields) == 0 {
		return false
	}

	// Check table/alias name
	if string_ := fields[0].GetString_(); string_ == nil ||
		(string_.GetSval() != table.Name && string_.GetSval() != table.Alias) {
		return false
	}

	// No column name specified
	if len(fields) <= 1 {
		return false
	}

	// Check column name for role-specific permissions
	if string_ := fields[1].GetString_(); string_ != nil {
		switch role {
		case "admin":
			return true
		case "student":
			return string_.GetSval() == "student_id"
		case "professor":
			return string_.GetSval() == "professor_id"
		}
	}

	return false
}

// Validate if a constant value matches the user's ID for their role
func validateConstValue(aConst *pgquery.A_Const, role string, id int) bool {
	if aConst == nil || aConst.GetVal() == nil {
		return false
	}

	val := aConst.GetVal()

	switch role {
	case "admin":
		return true
	case "student", "professor":
		// Check for integer value matching ID
		if iVal, ok := val.(*pgquery.A_Const_Ival); ok && iVal != nil {
			return iVal.Ival.Ival == int32(id)
		}
	}

	return false
}

// Checks if the join type requires checking outside the join condition
// Returns true if the table (left or right) needs additional WHERE clause validation
func isForceCheckOutside(joinType pgquery.JoinType, isLeftExpr bool) bool {
	switch joinType {
	case pgquery.JoinType_JOIN_INNER, pgquery.JoinType_JOIN_UNIQUE_INNER:
		// Inner joins filter both tables with ON clause, no need for outside checking
		return false

	case pgquery.JoinType_JOIN_LEFT:
		// LEFT JOIN: Check left side outside (preserved)
		return isLeftExpr

	case pgquery.JoinType_JOIN_RIGHT:
		// RIGHT JOIN: Check right side outside (preserved)
		return !isLeftExpr

	case pgquery.JoinType_JOIN_FULL, pgquery.JoinType_JOIN_UNIQUE_OUTER:
		// FULL/UNIQUE OUTER JOIN: Both sides preserved, check both outside
		return true

	default:
		// Other join is internal postgresql join types, so we use the normal check
		return false
	}

}

func hasSubquery(stmt *pgquery.SelectStmt) bool {
	if stmt == nil {
		return false
	}

	// 1. Check FROM clause (tables and derived tables)
	for _, fromItem := range stmt.FromClause {
		// Check for subquery in FROM clause (RangeSubselect)
		if subquery := fromItem.GetRangeSubselect(); subquery != nil {
			return true
		}

		// Check for subquery in JOIN expression
		if joinExpr := fromItem.GetJoinExpr(); joinExpr != nil {
			// Check subquery in left or right side of JOIN
			if larg := joinExpr.Larg; larg != nil {
				if larg.GetRangeSubselect() != nil {
					return true
				}
			}
			if rarg := joinExpr.Rarg; rarg != nil {
				if rarg.GetRangeSubselect() != nil {
					return true
				}
			}
		}
	}

	// 2. Check WHERE clause for subqueries
	if stmt.WhereClause != nil {
		if hasSubexpressionSubquery(stmt.WhereClause) {
			return true
		}
	}

	// 3. Check SELECT list for scalar subqueries
	for _, target := range stmt.TargetList {
		if target != nil && hasSubexpressionSubquery(target) {
			return true
		}
	}

	// 4. Check HAVING clause
	if stmt.HavingClause != nil {
		if hasSubexpressionSubquery(stmt.HavingClause) {
			return true
		}
	}

	return false
}

// Helper function to recursively check expressions for subqueries
func hasSubexpressionSubquery(expr *pgquery.Node) bool {
	if expr == nil {
		return false
	}

	// Check if this node is a sublink (subquery)
	if sublink := expr.GetSubLink(); sublink != nil {
		return true
	}

	// Recursively check subexpressions based on node type
	switch n := expr.Node.(type) {
	case *pgquery.Node_AExpr:
		// Check binary expression operands
		if n.AExpr.Lexpr != nil && hasSubexpressionSubquery(n.AExpr.Lexpr) {
			return true
		}
		if n.AExpr.Rexpr != nil && hasSubexpressionSubquery(n.AExpr.Rexpr) {
			return true
		}
	case *pgquery.Node_BoolExpr:
		// Check boolean expression args
		for _, arg := range n.BoolExpr.Args {
			if hasSubexpressionSubquery(arg) {
				return true
			}
		}
	}

	return false
}

type TableInfo struct {
	Name         string
	Alias        string            // Empty if no alias
	ColumnsAlias map[string]string // Map of alias to column name
	HasStar      bool              // Should default to true
}

func extractTables(stmt *pgquery.SelectStmt) ([]TableInfo, error) {
	// Extract table names from the FROM clause
	var tables []TableInfo

	for _, fromItem := range stmt.FromClause {
		// Handle RangeVar (simple table reference)
		if rangeVar := fromItem.GetRangeVar(); rangeVar != nil {
			tableInfo := TableInfo{Name: rangeVar.Relname}

			// Check if the table has an alias
			if rangeVar.Alias != nil {
				tableInfo.Alias = rangeVar.Alias.Aliasname
			}

			tables = append(tables, tableInfo)
		}

		// Handle JoinExpr
		if joinExpr := fromItem.GetJoinExpr(); joinExpr != nil {
			// Extract from left side
			if larg := joinExpr.Larg; larg != nil {
				if rv := larg.GetRangeVar(); rv != nil {
					tableInfo := TableInfo{Name: rv.Relname}
					if rv.Alias != nil {
						tableInfo.Alias = rv.Alias.Aliasname
					}
					tables = append(tables, tableInfo)
				}
			}

			// Extract from right side
			if rarg := joinExpr.Rarg; rarg != nil {
				if rv := rarg.GetRangeVar(); rv != nil {
					tableInfo := TableInfo{Name: rv.Relname}
					if rv.Alias != nil {
						tableInfo.Alias = rv.Alias.Aliasname
					}
					tables = append(tables, tableInfo)
				}
			}
		}

		// Handle subqueries in FROM clause
		if subquery := fromItem.GetRangeSubselect(); subquery != nil {
			if subquery.Subquery != nil && subquery.Subquery.GetSelectStmt() != nil {
				// Recursively extract tables from subquery
				subTables, err := extractTables(subquery.Subquery.GetSelectStmt())
				if err != nil {
					return nil, err
				}
				tables = append(tables, subTables...)

				// If the subquery itself has an alias, note it
				if subquery.Alias != nil {
					// This is a derived table with an alias
					tables = append(tables, TableInfo{
						Name:  "derived_table",
						Alias: subquery.Alias.Aliasname,
					})
				}
			}
		}
	}

	return tables, nil
}
