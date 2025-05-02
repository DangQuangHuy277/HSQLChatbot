package db

import (
	"errors"
	"fmt"
	om "github.com/elliotchance/orderedmap/v3"
	pgquery "github.com/pganalyze/pg_query_go/v6"
	"strconv"
	"strings"
)

// AuthorizationService handles SQL query authorization
type AuthorizationService struct {
	schemaService SchemaService
}

// NewAuthorizationService creates a new authorization service
func NewAuthorizationService(service SchemaService) *AuthorizationService {
	return &AuthorizationService{
		schemaService: service,
	}
}

type AuthorizationResult struct {
	Authorized         bool
	UnAuthorizedTables []TableInfo
	// AllDiscoveredTables is a map of alias to array of TableInfo. Note that we should maintain the order of the tables
	AllDiscoveredTables map[string][]*TableInfo

	// Tables is a map of alias to array of TableInfo. Note that we should maintain the order of the tables
	Tables *om.OrderedMap[string, *TableInfoV2]

	TargetList []*pgquery.Node
}

type UserContext interface {
}

// AuthorizeNode checks the authorization for a specific node in the SQL query.
// It validates whether the user with the given role and ID has access to the tables involved.
//
// Parameters:
// - node: the node to check
// - tables: additional tables to check authorization against
// - neg: a boolean indicating if we are in a negation context, in other words, the selected data doesn't contain any authorized data
// - role: the role of the user
// - userId: the ID of the user
//
// Returns:
// - bool: true if the node is authorized, false otherwise
// - []TableInfo: the list of remaining unauthorized tables
// - error: an error if any occurred during the authorization process
func (s *AuthorizationService) AuthorizeNode(node *pgquery.Node, tables *om.OrderedMap[string, *TableInfoV2], neg bool, userInfo UserContext) (*AuthorizationResult, error) {
	if node.GetNode() == nil {
		return nil, errors.New("node is nil")
	}
	// We will ignore the node related to function call, group by, order by, limit, offset, target list etc. and return default false for it
	// Also ignore JSON, XML, and other data types because our database is not using them
	// Also consider PARTITION, WINDOW as unauthorized so that they need to be authorized by a WHERE clause
	// Also only consider SELECT queries and not modifying queries like INSERT, UPDATE, DELETE,etc
	switch node.GetNode().(type) {
	case *pgquery.Node_SelectStmt:
		return s.authorizeSelectStmt(node.GetSelectStmt(), tables, neg, userInfo)
	case *pgquery.Node_BoolExpr:
		return s.authorizeBoolExpr(node.GetBoolExpr(), tables, neg, userInfo)
	case *pgquery.Node_AExpr:
		return s.authorizeAExpr(node.GetAExpr(), tables, neg, userInfo)
	case *pgquery.Node_SubLink:
		return s.authorizeSubLink(node.GetSubLink(), tables, neg, userInfo)
	case *pgquery.Node_JoinExpr:
		return s.authorizeJoinExpr(node.GetJoinExpr(), tables, neg, userInfo)
	case *pgquery.Node_FromExpr:
		return s.authorizeFromExpr(node.GetFromExpr(), tables, neg, userInfo)
	case *pgquery.Node_Query:
	case *pgquery.Node_ColumnRef:
	case *pgquery.Node_AIndirection:
	case *pgquery.Node_AIndices:
	case *pgquery.Node_AArrayExpr:
	case *pgquery.Node_RangeSubselect:
		return s.authorizeRangeSubselect(node.GetRangeSubselect(), tables, neg, userInfo)
	case *pgquery.Node_WithClause:
		return s.authorizeWithClause(node.GetWithClause(), tables, neg, userInfo)
	case *pgquery.Node_CommonTableExpr:
	case *pgquery.Node_FetchStmt:
	case *pgquery.Node_PrepareStmt:
	case *pgquery.Node_ExecuteStmt:
	}

	return &AuthorizationResult{
		Authorized: false,
		Tables:     tables,
	}, nil
}

func (s *AuthorizationService) authorizeSelectStmt(stmt *pgquery.SelectStmt, tables *om.OrderedMap[string, *TableInfoV2], neg bool, userInfo UserContext) (*AuthorizationResult, error) {
	result := &AuthorizationResult{
		Authorized: true,
		Tables:     om.NewOrderedMap[string, *TableInfoV2](),
	}

	if stmt.GetIntoClause() != nil {
		return result, nil
	}

	if stmt.GetWithClause() != nil {
		for _, cte := range stmt.GetWithClause().GetCtes() {
			cteResult, err := s.AuthorizeNode(cte, tables, neg, userInfo)
			if err != nil {
				return nil, err
			}
			result.Tables = s.MergeMaps(cteResult.Tables, result.Tables)
			result.Authorized = result.Authorized && cteResult.Authorized
		}
	}

	if stmt.GetOp() == pgquery.SetOperation_SETOP_UNION {
		// Similar to the OR condition, the remaining unauthorized tables will be union
		// of all unauthorized tables for left and right select
		lResult, err := s.authorizeSelectStmt(stmt.GetLarg(), tables, neg, userInfo)
		if err != nil {
			return nil, err
		}
		rResult, err := s.authorizeSelectStmt(stmt.GetRarg(), tables, neg, userInfo)
		if err != nil {
			return nil, err
		}
		result.Authorized = lResult.Authorized && rResult.Authorized
		// This is a little trick, we only merge the right to left. But because the scope of lResult and rResult is only here
		// So do this to have a little bit better performance and convenience in code
		s.UpdateUnauthorizedTablesByUnion(result.Tables, []*om.OrderedMap[string, *TableInfoV2]{s.MergeMaps(lResult.Tables, rResult.Tables)})
		if lResult.TargetList != nil && rResult.TargetList != nil {
			// The target list should be the same for both sides
			result.TargetList = lResult.TargetList
		}
		return result, nil
	}

	if stmt.GetOp() == pgquery.SetOperation_SETOP_INTERSECT {
		// Similar to the AND condition, the remaining unauthorized tables will be intersection
		lResult, err := s.authorizeSelectStmt(stmt.GetLarg(), tables, neg, userInfo)
		if err != nil {
			return nil, err
		}
		rResult, err := s.authorizeSelectStmt(stmt.GetRarg(), tables, neg, userInfo)
		if err != nil {
			return nil, err
		}
		result.Authorized = lResult.Authorized || rResult.Authorized
		// Update result tables like above
		s.UpdateUnauthorizedTablesByIntersection(result.Tables, []*om.OrderedMap[string, *TableInfoV2]{s.MergeMaps(lResult.Tables, rResult.Tables)})
		if lResult.TargetList != nil && rResult.TargetList != nil {
			// The target list should be the same for both sides
			result.TargetList = lResult.TargetList
		}
		return result, nil
	}

	if stmt.GetOp() == pgquery.SetOperation_SETOP_EXCEPT {
		// We ignore the right SELECT can exclude all unauthorized data of the left Select and make the whole query authorized
		// Because it can be rewritten in as an additional condition in the left select
		// It also is complex case, if this happens, it is usually intended of the hacker
		return s.authorizeSelectStmt(stmt.GetLarg(), tables, neg, userInfo)
	}

	for _, fromItem := range stmt.GetFromClause() {
		fromResult, err := s.AuthorizeNode(fromItem, tables, neg, userInfo)
		if err != nil {
			return nil, err
		}
		result.Tables = s.MergeMaps(result.Tables, fromResult.Tables)
		result.Authorized = result.Authorized && fromResult.Authorized
	}

	// We are certainly that the where clause and having clause don't discover needed new tables.
	// If they discovered new tables, it should be handled inside its own node
	if !result.Authorized && stmt.GetWhereClause() != nil {
		whereResult, err := s.AuthorizeNode(stmt.WhereClause, result.Tables, neg, userInfo)
		if err != nil {
			return nil, err
		}
		// Reassign the tables to the result, because the where clause will filter out authorized tables
		result.Tables = whereResult.Tables
		result.Authorized = whereResult.Authorized
	}

	// The HAVING clause is applied after the WHERE clause
	if !result.Authorized && stmt.GetHavingClause() != nil {
		havingResult, err := s.AuthorizeNode(stmt.HavingClause, result.Tables, neg, userInfo)
		if err != nil {
			return nil, err
		}
		// Also update the result tables
		result.Tables = havingResult.Tables
		result.Authorized = havingResult.Authorized
	}

	if stmt.GetTargetList() != nil {
		// To do: Refactor code to also support authorize subquery in Select (subquery) from table
		result.TargetList = stmt.GetTargetList()
	}

	return result, nil
}

// authorizeFromExpr authorizes the FROM clause of a SQL statement.
func (s *AuthorizationService) authorizeFromExpr(node *pgquery.FromExpr, tables *om.OrderedMap[string, *TableInfoV2], neg bool, userInfo UserContext) (*AuthorizationResult, error) {
	var combinedTables *om.OrderedMap[string, *TableInfoV2]
	for _, table := range node.GetFromlist() {
		tableResult, err := s.AuthorizeNode(table, tables, neg, userInfo)
		if err != nil {
			return nil, err
		}
		combinedTables = s.MergeMaps(combinedTables, tableResult.Tables)
	}

	return s.AuthorizeNode(node.GetQuals(), combinedTables, neg, userInfo)
}

func (s *AuthorizationService) authorizeCommonTableExpr(expr *pgquery.Node_CommonTableExpr, tables []TableInfo, userInfo UserContext) (*AuthorizationResult, error) {
	return nil, nil
}

func (s *AuthorizationService) authorizeJoinExpr(joinExpr *pgquery.JoinExpr, tables *om.OrderedMap[string, *TableInfoV2], neg bool, userInfo UserContext) (*AuthorizationResult, error) {
	if joinExpr == nil {
		return nil, fmt.Errorf("joinExpr is nil")
	}

	if joinExpr.GetLarg() == nil {
		return nil, fmt.Errorf("left side of join is nil")
	}
	if joinExpr.GetRarg() == nil {
		return nil, fmt.Errorf("right side of join is nil")
	}

	lResult, err := s.AuthorizeNode(joinExpr.GetLarg(), tables, neg, userInfo)
	if err != nil {
		return nil, err
	}

	rResult, err := s.AuthorizeNode(joinExpr.GetRarg(), tables, neg, userInfo)
	if err != nil {
		return nil, err
	}

	var curTables *om.OrderedMap[string, *TableInfoV2]
	curTables = s.MergeMaps(curTables, lResult.Tables)
	curTables = s.MergeMaps(curTables, rResult.Tables)

	// Both sides of the join are authorized
	if lResult.Authorized && rResult.Authorized {
		if joinAlias := joinExpr.GetAlias(); joinAlias != nil {
			curTables = s.createVirtualTableForAlias(joinAlias, nil, curTables)
		}
		return &AuthorizationResult{
			Authorized: true,
			Tables:     curTables,
		}, nil
	}

	// Process natural joins or joins with USING clause
	if joinExpr.GetUsingClause() != nil || joinExpr.GetIsNatural() {
		if joinAlias := joinExpr.GetAlias(); joinAlias != nil {
			curTables = s.createVirtualTableForAlias(joinAlias, nil, curTables)
		}
		if role == "admin" {
			// TODO: Remove all unauthorized tables

			return &AuthorizationResult{
				Authorized: true,
				Tables:     curTables,
			}, nil
		}
		return &AuthorizationResult{
			Authorized: false,
			Tables:     curTables,
		}, nil
	}

	// Determine unauthorized tables based on join type
	switch joinExpr.GetJointype() {
	case pgquery.JoinType_JOIN_INNER, pgquery.JoinType_JOIN_UNIQUE_INNER:
		// We only need to check the unauthorized tables that belong to both sides
		// In other words, we ignore the unauthorized tables that don't exist in both sides
		s.UpdateUnauthorizedTablesByIntersection(lResult.Tables, []*om.OrderedMap[string, *TableInfoV2]{rResult.Tables})

	case pgquery.JoinType_JOIN_LEFT:
		// We ignore the right side of the join
		for _, table := range rResult.Tables.AllFromFront() {
			table.UnAuthorizedTables = om.NewOrderedMap[string, *TableInfoV2]()
		}
	case pgquery.JoinType_JOIN_RIGHT:
		// We ignore the left side of the join
		for _, table := range lResult.Tables.AllFromFront() {
			table.UnAuthorizedTables = om.NewOrderedMap[string, *TableInfoV2]()
		}
	case pgquery.JoinType_JOIN_FULL, pgquery.JoinType_JOIN_UNIQUE_OUTER:
		// We need to check all unauthorized tables from both sides, so curTables is good
		// and we will not change anything in the tables
	default:
		return nil, fmt.Errorf("unsupported join type: %v", joinExpr.GetJointype())
	}

	qualResult, err := s.AuthorizeNode(joinExpr.GetQuals(), curTables, neg, userInfo)
	if err != nil {
		return nil, err
	}

	if joinAlias := joinExpr.GetAlias(); joinAlias != nil {
		// Create and assign the virtual table
		qualResult.Tables = s.createVirtualTableForAlias(joinAlias, nil, qualResult.Tables)
	}

	return qualResult, nil
}

// authorizeSubLink authorizes a SubLink node in the SQL query.
func (s *AuthorizationService) authorizeSubLink(link *pgquery.SubLink, tables *om.OrderedMap[string, *TableInfoV2], neg bool, userInfo UserContext) (*AuthorizationResult, error) {
	// This node only parse fully after analysis phase, so we only can to authorize subquery here
	return s.AuthorizeNode(link.GetSubselect(), tables, false, userInfo)
}

func (s *AuthorizationService) authorizeRangeSubselect(node *pgquery.RangeSubselect, tables *om.OrderedMap[string, *TableInfoV2], neg bool, userInfo UserContext) (*AuthorizationResult, error) {
	if node.GetLateral() {
		// TODO: Recheck
		return nil, fmt.Errorf("lateral is not supported")
	}

	authResult, err := s.AuthorizeNode(node.GetSubquery(), tables, false, userInfo)
	if err != nil {
		return nil, err
	}

	if node.GetAlias() != nil {
		authResult.Tables = s.createVirtualTableForAlias(node.GetAlias(), authResult.TargetList, authResult.Tables)
		authResult.TargetList = nil
	}

	return authResult, nil
}

// authorizeRangeVar get the table name (with authorization information) from the range var
// We should not pass tables to this function because we don't check the authorization of extra tables here
// But keep the tables parameter to align with the other functions
func (s *AuthorizationService) authorizeRangeVar(node *pgquery.RangeVar, tables *om.OrderedMap[string, *TableInfoV2], userInfo UserContext) (*AuthorizationResult, error) {
	if node == nil {
		return nil, fmt.Errorf("range var is nil")
	}

	table := &TableInfoV2{
		Name:       node.GetRelname(),
		Columns:    om.NewOrderedMap[string, *ColumnInfo](), // Need to check with select statement to decide should we add columns here or when we have target list
		Alias:      node.GetRelname(),
		HasStar:    false, // default
		IsDatabase: true,
	}

	// Because we can have col names alias for join expr, etc. which doesn't need to reach TargetList
	// So we populate all columns right here
	for _, colName := range s.schemaService.GetColumns(table.Name) {
		table.Columns.Set(colName, &ColumnInfo{
			Name:        colName,
			Alias:       colName,
			SourceTable: table,
		})
	}

	if node.GetAlias() != nil {
		table.Alias = node.GetAlias().GetAliasname()
	}

	tables = s.MergeMaps(tables, om.NewOrderedMapWithElements(&om.Element[string, *TableInfoV2]{Key: table.Alias, Value: table}))

	return &AuthorizationResult{
		Authorized: isPublicTable(table.Name),
		Tables:     tables,
	}, nil
}

// authorizeWithClause authorizes a WITH clause in the SQL query.
// In most cases, the tables parameter is empty because we don't need to check authorization of extra tables
// But we need to keep it to align with the other functions
func (s *AuthorizationService) authorizeWithClause(clause *pgquery.WithClause, tables *om.OrderedMap[string, *TableInfoV2], neg bool, userInfo UserContext) (*AuthorizationResult, error) {
	result := &AuthorizationResult{
		Authorized: true,
		Tables:     om.NewOrderedMap[string, *TableInfoV2](),
	}
	for _, cte := range clause.GetCtes() {
		cteResult, err := s.AuthorizeNode(cte, tables, neg, userInfo)
		if err != nil {
			return nil, err
		}
		result.Tables = s.MergeMaps(cteResult.Tables, result.Tables)
		result.Authorized = result.Authorized && cteResult.Authorized
	}
	return result, nil
}

// authorizeBoolExpr authorizes a BoolExpr node in the SQL query (mostly used in WHERE clause).
func (s *AuthorizationService) authorizeBoolExpr(boolExpr *pgquery.BoolExpr, tables *om.OrderedMap[string, *TableInfoV2], neg bool, userInfo UserContext) (*AuthorizationResult, error) {
	if boolExpr == nil {
		return &AuthorizationResult{
			Authorized: tables.Len() == 0,
			Tables:     tables,
		}, nil
	}

	result := &AuthorizationResult{
		Authorized: true,
		Tables:     DeepCopy(tables),
	}

	switch boolExpr.GetBoolop() {
	case pgquery.BoolExprType_OR_EXPR:
		// All subexpressions must be authorized, so the remaining unauthorized tables will be union
		// of all unauthorized tables for each subexpression
		var authResultTables []*om.OrderedMap[string, *TableInfoV2]
		for _, arg := range boolExpr.Args {
			authResult, err := s.AuthorizeNode(arg, tables, neg, userInfo)
			if err != nil {
				return nil, err
			}
			authResultTables = append(authResultTables, authResult.Tables)
		}
		s.UpdateUnauthorizedTablesByUnion(result.Tables, authResultTables)

	case pgquery.BoolExprType_AND_EXPR:
		// If intersection of unauthorized tables of all subexpressions is empty, then the whole expression is authorized

		var authResultTables []*om.OrderedMap[string, *TableInfoV2]
		for _, arg := range boolExpr.Args {
			authResult, err := s.AuthorizeNode(arg, tables, neg, userInfo)
			if err != nil {
				return nil, err
			}
			authResultTables = append(authResultTables, authResult.Tables)
		}
		s.UpdateUnauthorizedTablesByIntersection(result.Tables, authResultTables)

	case pgquery.BoolExprType_NOT_EXPR:
		// Negation: check if the subexpression is unauthorized,
		// The tables only authorized if contains something like user_id != id
		if len(boolExpr.Args) != 1 {
			return nil, fmt.Errorf("NOT expression must have exactly one argument")
		}
		authResult, err := s.AuthorizeNode(boolExpr.Args[0], tables, !neg, userInfo)
		if err != nil {
			return nil, err
		}
		result.Authorized = !authResult.Authorized
	}

	for _, table := range result.Tables.AllFromFront() {
		if table.UnAuthorizedTables.Len() != 0 {
			result.Authorized = false
			break
		}
	}

	return result, nil
}

// authorizeAExpr authorizes an A_Expr node in the SQL query (mostly used in WHERE clause).
// tables is the list of tables to check authorization against.
// We shouldn't modify the tables parameter because in outer condition (like OR) we need to pass same tables for all member of the condition
func (s *AuthorizationService) authorizeAExpr(expr *pgquery.A_Expr, tables *om.OrderedMap[string, *TableInfoV2], neg bool, userInfo UserContext) (*AuthorizationResult, error) {
	if expr == nil {
		return nil, errors.New("AExpr is nil")
	}

	resultTables := DeepCopy(tables)

	if expr.GetLexpr() == nil || expr.GetRexpr() == nil {
		return &AuthorizationResult{
			Authorized: false,
			Tables:     resultTables,
		}, nil
	}

	wantOperator := "="
	if neg {
		wantOperator = "<>"
	}
	if s.getStringValueFromNode(expr.GetName()[0]) != wantOperator {
		return &AuthorizationResult{
			Authorized: false,
			Tables:     resultTables,
		}, nil
	}

	switch expr.GetKind() {
	case pgquery.A_Expr_Kind_AEXPR_OP:
		// Case: a op b (with op is in https://www.postgresql.org/docs/6.3/c09.htm)
		// Base case: We only allow column = id or id = column
		// Other case like subquery = column, etc don't contribute much to the authorization of the main query
		columnRef, aConst := s.extractColumnRefAndConst(expr)

		if columnRef != nil && aConst != nil {
			// Loop through need to check tables to see which tables this expression can authorize
			for _, table := range resultTables.AllFromFront() {
				for unAuthAlias, unauthorizedTable := range table.UnAuthorizedTables.AllFromFront() {
					authColStrs := s.getAuthorizationColumn(table.Name, userInfo)
					if s.getStringValueFromNode(columnRef.GetFields()[0]) == table.Alias &&
						s.getStringValueFromNode(columnRef.GetFields()[1]) == unauthorizedTable.getAliasOfColumn(authColStr) &&
						int(aConst.GetIval().Ival) == id {
						// Remove the table from the list of unauthorized tables
						table.UnAuthorizedTables.Delete(unAuthAlias)
						break
					}
				}
			}
		}

		// Case: RowExpr = SubLink
		// We will support (t.user_id, t.col1) = (SELECT ... where user_id = id)
		if expr.GetLexpr() != nil && expr.GetLexpr().GetRowExpr() != nil {
			for _, arg := range expr.GetLexpr().GetRowExpr().GetArgs() {
				if arg.GetColumnRef() == nil {
					continue
				}
				fields := arg.GetColumnRef().GetFields()
				if len(fields) <= 1 || fields[0].GetString_() == nil || fields[1].GetString_() == nil {
					continue
				}
				table, ok := tables.Get(fields[0].GetString_().GetSval())
				if !ok {
					continue
				}
				if s.getAuthorizationColumn(table.Name, role) != table.getAliasOfColumn(fields[1].GetString_().GetSval()) {
					// We don't have the authorization column, so we can't authorize this table
					continue
				}
				// Check whether the subquery can authorize the table
				authResult, _ := s.AuthorizeNode(expr.GetRexpr(), om.NewOrderedMapWithElements(&om.Element[string, *TableInfoV2]{Key: table.Alias, Value: table}), neg, userInfo)
				if authResult.Authorized {
					// Remove the table from the list of unauthorized tables
					table.UnAuthorizedTables.Delete(table.Alias)
					break
				}
			}
		}
	case pgquery.A_Expr_Kind_AEXPR_IN:
		// Case: a IN (b, c, d)
		// We only allow column IN (id) or id IN (column)
		if expr.GetLexpr().GetColumnRef() == nil || expr.GetRexpr().GetList() == nil || len(expr.GetRexpr().GetList().GetItems()) != 1 {
			break
		}
		columnRef := expr.GetLexpr().GetColumnRef()
		itemList := expr.GetRexpr().GetList().GetItems()

		for _, table := range resultTables.AllFromFront() {
			for unAuthAlias, unauthorizedTable := range table.UnAuthorizedTables.AllFromFront() {
				authColStr := s.getAuthorizationColumn(unauthorizedTable.Name, role)
				if s.getStringValueFromNode(columnRef.GetFields()[0]) == table.Alias &&
					s.getStringValueFromNode(columnRef.GetFields()[1]) == unauthorizedTable.getAliasOfColumn(authColStr) &&
					s.getStringValueFromNode(itemList[0]) == strconv.Itoa(id) {
					// Remove the table from the list of unauthorized tables
					table.UnAuthorizedTables.Delete(unAuthAlias)
					break
				}
			}
		}
	}

	// Build the result
	authorized := true
	for _, table := range resultTables.AllFromFront() {
		authorized = authorized && table.UnAuthorizedTables.Len() == 0
	}

	return &AuthorizationResult{Authorized: authorized, Tables: resultTables}, nil
}

// createVirtualTableForAlias creates a virtual table for the given alias.
// It should keep reference to the original tables and columns.
// joinAlias is the alias for the join expression.
// targetList is the selected columns for new virtual table (optional).
// curTables is the current tables in the query.
func (s *AuthorizationService) createVirtualTableForAlias(alias *pgquery.Alias, targetList []*pgquery.Node, curTables *om.OrderedMap[string, *TableInfoV2]) *om.OrderedMap[string, *TableInfoV2] {
	// We have an alias for a subscope, so we will create a new wrapper TableInfo corresponding to the alias
	virtualTable := &TableInfoV2{
		Name:    alias.GetAliasname(),
		Columns: om.NewOrderedMap[string, *ColumnInfo](),
		Alias:   alias.GetAliasname(),
		HasStar: true,
	}

	// Convert the column aliases to a slice of strings
	var colAliases []string
	if alias.GetColnames() != nil {
		colAliases = make([]string, 0, len(alias.GetColnames()))
		for _, colName := range alias.GetColnames() {
			if colName.GetString_() != nil {
				colAliases = append(colAliases, colName.GetString_().GetSval())
			}
		}
	}

	allColCnt := 0
	for _, table := range curTables.AllFromFront() {
		allColCnt += table.Columns.Len()
	}

	curIndex := 0
	if targetList != nil {
		for _, colNode := range targetList {
			resTarget := colNode.GetResTarget()
			if resTarget == nil || resTarget.GetVal() == nil || resTarget.GetVal().GetColumnRef() == nil {
				// We only care about column references now
				// Other node will be support in the future
				continue
			}
			fields := resTarget.GetVal().GetColumnRef().GetFields()
			if len(fields) == 0 {
				continue
			}

			if fields[0].GetAStar() != nil {
				// Case : SELECT * FROM ...
				// All tables should populate their columns at RangeVar
				// We only need collect them to the virtual table
				nextIndex := curIndex + allColCnt
				if nextIndex > len(colAliases) {
					nextIndex = len(colAliases)
				}
				virtualTable.createAllColumnsFromSourceTableList(curTables.Values(), colAliases[curIndex:nextIndex])
				curIndex = nextIndex

			} else if len(fields) == 2 && fields[1].GetAStar() != nil && fields[0].GetString_() != nil {
				// Case : SELECT table.* FROM ...
				selectedTableName := fields[0].GetString_().GetSval()
				selectedTable, ok := curTables.Get(selectedTableName)
				if !ok {
					continue // Skip nil tables, but this should not happen
				}
				nextIndex := curIndex + selectedTable.Columns.Len()
				if nextIndex > len(colAliases) {
					nextIndex = len(colAliases)
				}
				virtualTable.createAllColumnFromSourceTable(selectedTable, colAliases[curIndex:nextIndex])
				curIndex = nextIndex
			} else if fields[0].GetString_() != nil {
				// Case : SELECT table.col (AS alias) FROM ...
				selectedTableName := fields[0].GetString_().GetSval()
				selectedColName := fields[1].GetString_().GetSval()
				selectedTable, ok := curTables.Get(selectedTableName)
				if !ok {
					continue // Skip nil tables, but this should not happen
				}

				colAlias := selectedColName
				if resTarget.GetName() != "" {
					colAlias = resTarget.GetName()
				}
				if curIndex < len(colAliases) && colAliases[curIndex] != "" {
					colAlias = colAliases[curIndex]
					curIndex++
				}
				virtualTable.createColumnFromSourceTable(selectedColName, selectedTable, &colAlias)
			}
		}
	} else {
		// when targetList is nil, we create a virtual table like SELECT * FROM ... case
		virtualTable.createAllColumnsFromSourceTableList(curTables.Values(), colAliases)
	}

	// Also populate the unauthorized tables
	// In case all tables are authorized, we should not have any unauthorized tables
	for _, table := range curTables.AllFromFront() {
		if table.UnAuthorizedTables.Len() != 0 {
			for tableAlias, unauthorizedTable := range table.UnAuthorizedTables.AllFromFront() {
				virtualTable.UnAuthorizedTables.Set(tableAlias, unauthorizedTable)
			}
		}
	}

	return om.NewOrderedMapWithElements(&om.Element[string, *TableInfoV2]{Key: virtualTable.Alias, Value: virtualTable})
}

func (s *AuthorizationService) getStringValueFromNode(node *pgquery.Node) string {
	if node == nil {
		return ""
	}

	switch n := node.GetNode().(type) {
	case *pgquery.Node_String_:
		return n.String_.Sval
	case *pgquery.Node_Integer:
		return fmt.Sprintf("%d", n.Integer.Ival)
	case *pgquery.Node_Float:
		return n.Float.Fval
	case *pgquery.Node_ColumnRef:
		// For column references, concatenate field names
		var parts []string
		for _, field := range n.ColumnRef.Fields {
			if str := field.GetString_(); str != nil {
				parts = append(parts, str.GetSval())
			}
		}
		return strings.Join(parts, ".")
	case *pgquery.Node_AConst:
		// Extract value from constant
		return ""
	}

	// Default case - return empty string for unsupported node types
	return ""
}

// extractColumnRefAndConst extracts the column reference and constant from an A_Expr node.
// It returns the column reference and constant as separate values and nil if not applicable.
func (s *AuthorizationService) extractColumnRefAndConst(aExpr *pgquery.A_Expr) (*pgquery.ColumnRef, *pgquery.A_Const) {
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

// UpdateUnauthorizedTablesByUnion updates the unauthorized tables in the destination list
func (s *AuthorizationService) UpdateUnauthorizedTablesByUnion(dst *om.OrderedMap[string, *TableInfoV2], tablesList []*om.OrderedMap[string, *TableInfoV2]) {
	// All subexpressions must be authorized, so the remaining unauthorized tables will be union
	// of all unauthorized tables for each subexpression

	// Map table alias to a set of unauthorized table aliases (in go it is a map)
	tableUnAuth := make(map[string]map[string]bool)
	for _, table := range dst.AllFromFront() {
		tableUnAuth[table.Alias] = make(map[string]bool)
	}

	// First pass: Collect all unauthorized tables from the tablesList
	for _, tables := range tablesList {
		for _, table := range tables.AllFromFront() {
			for _, unauthorizedTable := range table.UnAuthorizedTables.AllFromFront() {
				tableUnAuth[table.Alias][unauthorizedTable.Alias] = true
			}
		}
	}

	for _, table := range dst.AllFromFront() {
		for _, unauthorizedTable := range table.UnAuthorizedTables.AllFromFront() {
			if _, unAuth := tableUnAuth[table.Alias][unauthorizedTable.Alias]; !unAuth {
				// We can't find the table in the union, it means it is authorized
				// Remove the table from the list of unauthorized tables
				table.UnAuthorizedTables.Delete(unauthorizedTable.Alias)
			}
		}
	}
}

// UpdateUnauthorizedTablesByIntersection updates the unauthorized tables in the destination list
func (s *AuthorizationService) UpdateUnauthorizedTablesByIntersection(dst *om.OrderedMap[string, *TableInfoV2], tablesList []*om.OrderedMap[string, *TableInfoV2]) {
	// If intersection of unauthorized tables of all subexpressions is empty, then the whole expression is authorized
	tableAuth := make(map[string]map[string]bool)
	for _, table := range dst.AllFromFront() {
		tableAuth[table.Alias] = make(map[string]bool)
	}

	for _, tables := range tablesList {

		// The authorizeAExpr should already clean the authorized tables
		// Keep track the unauthorized tables
		for _, table := range tables.AllFromFront() {
			for _, unauthorizedTable := range table.UnAuthorizedTables.AllFromFront() {
				tableAuth[table.Alias][unauthorizedTable.Alias] = true
			}
		}
	}

	for _, table := range dst.AllFromFront() {
		for _, unauthorizedTable := range table.UnAuthorizedTables.AllFromFront() {
			if _, auth := tableAuth[table.Alias][unauthorizedTable.Alias]; auth {
				// Remove the table from the list of unauthorized tables
				table.UnAuthorizedTables.Delete(unauthorizedTable.Alias)
				break
			}
		}
	}
}

func (s *AuthorizationService) DeduplicateTables(tables ...[]TableInfo) []TableInfo {
	seen := make(map[string]bool)
	var result []TableInfo

	for _, tableList := range tables {
		for _, table := range tableList {
			// Create a unique key based on Name and Alias
			key := table.Name
			if table.Alias != "" {
				key += ":" + table.Alias
			}

			if !seen[key] {
				seen[key] = true
				result = append(result, table)
			}
		}
	}

	return result
}

func (s *AuthorizationService) GetIntersection(tables1, tables2 []TableInfo) []TableInfo {
	intersection := make([]TableInfo, 0)

	for _, table1 := range tables1 {
		for _, table2 := range tables2 {
			if table1.Name == table2.Name {
				intersection = append(intersection, table1)
			}
		}
	}

	return intersection
}

func (s *AuthorizationService) GetTablesException(tables1, tables2 []TableInfo) []TableInfo {
	exception := make([]TableInfo, 0)

	for _, table1 := range tables1 {
		found := false
		for _, table2 := range tables2 {
			if table1.Name == table2.Name {
				found = true
				break
			}
		}
		if !found {
			exception = append(exception, table1)
		}
	}

	return exception
}

// MergeMaps merges the contents of src map into dst map
// If dst is nil, it will be initialized
// In case of duplicate keys, the value from src will overwrite the value in dst
func (s *AuthorizationService) MergeMaps(dst, src *om.OrderedMap[string, *TableInfoV2]) *om.OrderedMap[string, *TableInfoV2] {
	for key, value := range src.AllFromFront() {
		dst.Set(key, value)
	}
	return dst
}

// getAuthorizationColumn returns the column name used for authorization filtering
// for a given table and role. Returns an empty string for public tables, admin role,
// or when no specific authorization column applies.
func (s *AuthorizationService) getAuthorizationColumn(name string, userInfo UserContext) []string {
	// Admin role has unrestricted access to all tables
	if role == "admin" {
		return ""
	}

	// Public tables accessible to all roles
	switch name {
	case "program", "semester", "course", "course_program", "course_class",
		"course_class_schedule", "course_schedule_instructor", "faculty":
		return ""
	}

	// Role-specific authorization columns
	switch role {
	case "student":
		switch name {
		case "student":
			return "id"
		case "administrative_class":
			return "id" // Filter by student's administrative_class_id
		case "student_course_class":
			return "student_id"
		case "student_scholarship":
			return "student_id"
		}
	case "professor":
		switch name {
		case "professor":
			return "id"
		case "administrative_class":
			return "advisor_id"
		case "course_class":
			return "professor_id"
		case "course_schedule_instructor":
			return "professor_id"
		}
	}

	// Default: No access for unspecified tables/roles
	return ""
}
