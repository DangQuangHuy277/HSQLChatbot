package db

import (
	"errors"
	"fmt"
	pgquery "github.com/pganalyze/pg_query_go/v6"
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
}

// AuthorizeNode checks the authorization for a specific node in the SQL query.
// It validates whether the user with the given role and ID has access to the tables involved.
//
// Parameters:
// - node: the node to check
// - tables: additional tables to check authorization against
// - role: the role of the user
// - userId: the ID of the user
//
// Returns:
// - bool: true if the node is authorized, false otherwise
// - []TableInfo: the list of remaining unauthorized tables
// - error: an error if any occurred during the authorization process
func (s *AuthorizationService) AuthorizeNode(node *pgquery.Node, tables []TableInfo, role string, userId int) (*AuthorizationResult, error) {
	if node.GetNode() == nil {
		return nil, errors.New("node is nil")
	}
	// We will ignore the node related to function call, group by, order by, limit, offset, target list etc. and return default false for it
	// Also ignore JSON, XML, and other data types because our database is not using them
	// Also consider PARTITION, WINDOW as unauthorized so that they need to be authorized by a WHERE clause
	// Also only consider SELECT queries and not modifying queries like INSERT, UPDATE, DELETE,etc
	switch node.GetNode().(type) {
	case *pgquery.Node_SelectStmt:
		return s.authorizeSelectStmt(node.GetSelectStmt(), tables, role, userId)
	case *pgquery.Node_BoolExpr:
		return s.authorizeBoolExpr(node.GetBoolExpr(), tables, role, userId)
	case *pgquery.Node_AExpr:
		return s.authorizeAExpr(node.GetAExpr(), tables, role, userId)
	case *pgquery.Node_SubLink:
		return s.authorizeSubLink(node.GetSubLink(), tables, role, userId)
	case *pgquery.Node_JoinExpr:
		return s.authorizeJoinExpr(node.GetJoinExpr(), tables, role, userId)
	case *pgquery.Node_FromExpr:
		return s.authorizeFromExpr(node.GetFromExpr(), tables, role, userId)
	case *pgquery.Node_Query:
	case *pgquery.Node_ColumnRef:
	case *pgquery.Node_AIndirection:
	case *pgquery.Node_AIndices:
	case *pgquery.Node_AArrayExpr:
	case *pgquery.Node_RangeSubselect:
		return s.authorizeRangeSubselect(node.GetRangeSubselect(), tables, role, userId)
	case *pgquery.Node_WithClause:
	case *pgquery.Node_CommonTableExpr:
	case *pgquery.Node_FetchStmt:
	case *pgquery.Node_PrepareStmt:
	case *pgquery.Node_ExecuteStmt:
	}

	return nil, nil
}

func (s *AuthorizationService) authorizeSelectStmt(stmt *pgquery.SelectStmt, tables []TableInfo, role string, userId int) (*AuthorizationResult, error) {
	result := &AuthorizationResult{
		Authorized:         false,
		UnAuthorizedTables: []TableInfo{},
	}

	if stmt.GetIntoClause() != nil {
		result.UnAuthorizedTables = tables
		return result, nil
	}

	if stmt.GetOp() == pgquery.SetOperation_SETOP_UNION {
		lResult, err := s.authorizeSelectStmt(stmt.GetLarg(), tables, role, userId)
		if err != nil {
			return nil, err
		}
		rResult, err := s.authorizeSelectStmt(stmt.GetRarg(), tables, role, userId)
		if err != nil {
			return nil, err
		}
		result.Authorized = lResult.Authorized && rResult.Authorized
		result.UnAuthorizedTables = s.DeduplicateTables(lResult.UnAuthorizedTables, rResult.UnAuthorizedTables)
		return result, nil
	}

	if stmt.GetOp() == pgquery.SetOperation_SETOP_INTERSECT {
		lResult, err := s.authorizeSelectStmt(stmt.GetLarg(), tables, role, userId)
		if err != nil {
			return nil, err
		}
		rResult, err := s.authorizeSelectStmt(stmt.GetRarg(), tables, role, userId)
		if err != nil {
			return nil, err
		}
		result.Authorized = lResult.Authorized || rResult.Authorized
		if !result.Authorized {
			result.UnAuthorizedTables = s.DeduplicateTables(lResult.UnAuthorizedTables, rResult.UnAuthorizedTables)
		}
		return result, nil
	}

	if stmt.GetOp() == pgquery.SetOperation_SETOP_EXCEPT {
		// We ignore the right SELECT can exclude all unauthorized data of the left Select and make the whole query authorized
		// Because it can be rewritten in as an additional condition in the left select
		// It also is complex case, if this happens, it is usually intended of the hacker
		return s.authorizeSelectStmt(stmt.GetLarg(), tables, role, userId)
	}

	if stmt.GetWithClause() != nil {
		for _, cte := range stmt.GetWithClause().GetCtes() {
			cteResult, err := s.AuthorizeNode(cte, tables, role, userId)
			if err != nil {
				return nil, err
			}
			result.Authorized = cteResult.Authorized
			result.UnAuthorizedTables = append(result.UnAuthorizedTables, cteResult.UnAuthorizedTables...)
			result.AllDiscoveredTables = s.MergeMaps(result.AllDiscoveredTables, cteResult.AllDiscoveredTables)
		}
	}

	for _, fromItem := range stmt.GetFromClause() {
		fromResult, err := s.AuthorizeNode(fromItem, tables, role, userId)
		if err != nil {
			return nil, err
		}
		result.Authorized = fromResult.Authorized
		result.UnAuthorizedTables = append(result.UnAuthorizedTables, fromResult.UnAuthorizedTables...)
		result.AllDiscoveredTables = s.MergeMaps(result.AllDiscoveredTables, fromResult.AllDiscoveredTables)
	}

	// We are certainly that the where clause and having clause don't discover needed new tables.
	// If they discovered new tables, it should be handled inside its own node
	if !result.Authorized && stmt.GetWhereClause() != nil {
		whereResult, err := s.AuthorizeNode(stmt.WhereClause, result.UnAuthorizedTables, role, userId)
		if err != nil {
			return nil, err
		}
		result.Authorized = whereResult.Authorized
		result.UnAuthorizedTables = whereResult.UnAuthorizedTables
	}

	if !result.Authorized && stmt.GetHavingClause() != nil {
		havingResult, err := s.AuthorizeNode(stmt.HavingClause, result.UnAuthorizedTables, role, userId)
		if err != nil {
			return nil, err
		}
		result.Authorized = havingResult.Authorized
		result.UnAuthorizedTables = havingResult.UnAuthorizedTables
	}

	if stmt.GetTargetList() != nil {
		// We need 2 loops, the first one is to handle all the Star cases
		for _, target := range stmt.GetTargetList() {
			resTarget := target.GetResTarget()
			if resTarget == nil {
				// We certainly that the node in the target list is a resTarget
				return nil, fmt.Errorf("resTarget is nil")
			}
			// Ignore case the isn't a column reference
			if resTarget.GetVal() == nil {
				continue
			}
			if resTarget.GetVal().GetColumnRef() == nil {
				continue
			}
			fields := resTarget.GetVal().GetColumnRef().GetFields()
			if len(fields) == 0 {
				continue
			}
			// Case : SELECT * FROM ...
			if fields[0].GetAStar() != nil {
				for _, tableList := range result.AllDiscoveredTables {
					for _, table := range tableList {
						if table == nil {
							continue // Skip nil tables, but this should not happen
						}
						s.PopulateDefaultColumns(table)
						table.HasStar = true
					}
				}
			}
			// Case : SELECT table.* FROM ...
			if len(fields) == 2 && fields[1].GetAStar() != nil && fields[0].GetString_() != nil {
				tableNameRef := fields[0].GetString_().GetSval()
				for _, table := range result.AllDiscoveredTables[tableNameRef] {
					if table == nil {
						continue // Skip nil tables, but this should not happen
					}
					s.PopulateDefaultColumns(table)
					table.HasStar = true
				}
			}
		}

		// The second loop is to handle the column reference cases
		for _, target := range stmt.GetTargetList() {
			resTarget := target.GetResTarget()
			if resTarget == nil {
				// We certainly that the node in the target list is a resTarget
				return nil, fmt.Errorf("resTarget is nil")
			}
			// Ignore case the isn't a column reference
			if resTarget.GetVal() == nil {
				continue
			}
			if resTarget.GetVal().GetColumnRef() == nil {
				continue
			}
			fields := resTarget.GetVal().GetColumnRef().GetFields()
			if len(fields) == 0 {
				continue
			}
			// Case : SELECT table.col (AS alias) FROM ...
			if fields[0].GetString_() != nil {
				tableNameRef := fields[0].GetString_().GetSval()
				colNameRef := fields[1].GetString_().GetSval()
				for _, table := range result.AllDiscoveredTables[tableNameRef] {
					if table == nil {
						continue // Skip nil tables, but this should not happen
					}
					// Skip tables without columns or where the column doesn't exist
					// Since star columns were already processed in the first loop,
					// we can safely skip tables when:
					// 1. The table has no column aliases defined at all (But should not happen after the first loop)
					// 2. The specific column being referenced doesn't exist in this table
					// (Can happen if the table taken from a subquery, all tables from the subquery will have same alias)
					if table.ColumnsAlias == nil || len(table.ColumnsAlias) == 0 {
						continue
					}
					if _, ok := table.ColumnsAlias[colNameRef]; !ok {
						continue
					}

					newColNameRef := colNameRef
					if resTarget.GetName() != "" {
						colNameRef = resTarget.GetName()
					}
					table.ColumnsAlias[colNameRef] = newColNameRef
				}
			}
		}
	}

	return result, nil
}

// authorizeFromExpr authorizes the FROM clause of a SQL statement.
func (s *AuthorizationService) authorizeFromExpr(node *pgquery.FromExpr, tables []TableInfo, role string, id int) (*AuthorizationResult, error) {
	var needToCheckTables []TableInfo
	var discoveredTables map[string][]*TableInfo
	for _, table := range node.GetFromlist() {
		tableResult, err := s.AuthorizeNode(table, tables, role, id)
		if err != nil {
			return nil, err
		}
		needToCheckTables = append(needToCheckTables, tableResult.UnAuthorizedTables...)
		discoveredTables = s.MergeMaps(discoveredTables, tableResult.AllDiscoveredTables)
	}

	authResult, err := s.AuthorizeNode(node.GetQuals(), needToCheckTables, role, id)
	if err != nil {
		return nil, err
	}
	if authResult == nil {
		return nil, fmt.Errorf("authResult is nil")
	}
	// If the quals condition discovered tables (via subquery, etc.), these tables should be checked within the quals node itself and not exposed to the parent node
	authResult.AllDiscoveredTables = discoveredTables
	return authResult, nil
}

func (s *AuthorizationService) authorizeCommonTableExpr(expr *pgquery.Node_CommonTableExpr, tables []TableInfo, role string, id int) (*AuthorizationResult, error) {
	return nil, nil
}

func (s *AuthorizationService) authorizeJoinExpr(joinExpr *pgquery.JoinExpr, tables []TableInfo, role string, id int) (*AuthorizationResult, error) {
	if joinExpr == nil {
		return nil, fmt.Errorf("joinExpr is nil")
	}

	if joinExpr.GetLarg() == nil {
		return nil, fmt.Errorf("left side of join is nil")
	}
	if joinExpr.GetRarg() == nil {
		return nil, fmt.Errorf("right side of join is nil")
	}

	lResult, err := s.AuthorizeNode(joinExpr.GetLarg(), tables, role, id)
	if err != nil {
		return nil, err
	}

	rResult, err := s.AuthorizeNode(joinExpr.GetRarg(), tables, role, id)
	if err != nil {
		return nil, err
	}

	discoveredTables := lResult.AllDiscoveredTables
	discoveredTables = s.MergeMaps(discoveredTables, rResult.AllDiscoveredTables)
	if lResult.Authorized && rResult.Authorized {
		return &AuthorizationResult{Authorized: true, UnAuthorizedTables: []TableInfo{}, AllDiscoveredTables: discoveredTables}, nil
	}

	// If the join is a natural join or using clause, we will not have any quals condition to check, only need to check role if needed
	remainingTables := s.DeduplicateTables(lResult.UnAuthorizedTables, rResult.UnAuthorizedTables)
	if joinExpr.GetUsingClause() != nil || joinExpr.GetIsNatural() {
		if role == "admin" {
			return &AuthorizationResult{Authorized: true, UnAuthorizedTables: nil, AllDiscoveredTables: discoveredTables}, nil
		}
		return &AuthorizationResult{Authorized: false, UnAuthorizedTables: remainingTables, AllDiscoveredTables: discoveredTables}, nil
	}

	switch joinExpr.GetJointype() {
	case pgquery.JoinType_JOIN_INNER:
	case pgquery.JoinType_JOIN_UNIQUE_INNER:
		remainingTables = s.GetIntersection(lResult.UnAuthorizedTables, rResult.UnAuthorizedTables)
	case pgquery.JoinType_JOIN_LEFT:
		remainingTables = lResult.UnAuthorizedTables
	case pgquery.JoinType_JOIN_RIGHT:
		remainingTables = rResult.UnAuthorizedTables
	case pgquery.JoinType_JOIN_FULL:
	case pgquery.JoinType_JOIN_UNIQUE_OUTER:
		remainingTables = s.DeduplicateTables(lResult.UnAuthorizedTables, rResult.UnAuthorizedTables)
	default:
		return nil, fmt.Errorf("unsupported join type: %v", joinExpr.GetJointype())

	}

	qualResult, err := s.AuthorizeNode(joinExpr.GetQuals(), remainingTables, role, id)
	if err != nil {
		return nil, err
	}

	if joinAlias := joinExpr.GetAlias(); joinAlias != nil {
		// We have an alias for a subscope, so we need to convert all the discovered tables alias to the alias of this subscope
		for _, tableInfos := range discoveredTables {
			for _, tableInfo := range tableInfos {
				if tableInfo == nil {
					continue // Skip nil tables, but this should not happen
				}
				tableInfo.Alias = joinAlias.GetAliasname()
			}
		}

		// Also update the UnAuthorizedTables with the alias
		for i := range remainingTables {
			qualResult.UnAuthorizedTables[i].Alias = joinAlias.GetAliasname()
		}

		// If the alias has colnames, we need to update the columns alias of the tables
		if joinAlias.GetColnames() != nil {
			curIndex := 0
			// Loop with order to maintain the order of the tables
			for _, tableList := range qualResult.AllDiscoveredTables {
				for _, table := range tableList {
					if table.HasStar {
						// Populate all columns of the table, just to be sure
						s.PopulateDefaultColumns(table)
						// How can we know the order of the columns?
						// Solution 1: Use ordered map or slice of tuples to maintain column order.
						// Challenge: How do we correctly map columns when the selected columns in a subquery
						// don't match the order of tables discovered in the FROM clause?
						// Solution 2: Use old draft logic, virtual table
					}
				}
			}
		}
	}

	return qualResult, nil
}

func (s *AuthorizationService) authorizeSubLink(link *pgquery.SubLink, tables []TableInfo, role string, id int) (*AuthorizationResult, error) {
	subResult, err := s.AuthorizeNode(link.GetSubselect(), tables, role, id)
	if err != nil {
		return nil, err
	}

	return subResult, nil
}

func (s *AuthorizationService) authorizeRangeSubselect(node *pgquery.RangeSubselect, tables []TableInfo, role string, id int) (*AuthorizationResult, error) {
	if node.GetLateral() {
		// TODO: Recheck
		return nil, fmt.Errorf("lateral is not supported")
	}

	authResult, err := s.AuthorizeNode(node.GetSubquery(), tables, role, id)
	if err != nil {
		return nil, err
	}
	// We sure that the subquery of RangeSubselect is a SelectStmt
	if node.GetSubquery().GetSelectStmt() == nil {
		return nil, fmt.Errorf("subquery in range subselect is not a select statement")
	}

	subselectStmt := node.GetSubquery().GetSelectStmt()
	authResult, err = s.authorizeSelectStmt(subselectStmt, tables, role, id)
	if err != nil {
		return nil, err
	}
	return authResult, nil
}
func (s *AuthorizationService) authorizeBoolExpr(boolExpr *pgquery.BoolExpr, tables []TableInfo, role string, id int) (*AuthorizationResult, error) {
	if boolExpr == nil {
		return &AuthorizationResult{
			Authorized:          len(tables) == 0,
			UnAuthorizedTables:  tables,
			AllDiscoveredTables: make(map[string][]*TableInfo),
		}, nil
	}

	result := &AuthorizationResult{}

	switch boolExpr.GetBoolop() {
	case pgquery.BoolExprType_OR_EXPR:
		// All subexpressions must be authorized
		result.Authorized = true
		for _, arg := range boolExpr.Args {
			authResult, err := s.AuthorizeNode(arg, tables, role, id)
			if err != nil {
				return nil, err
			}
			result.Authorized = authResult.Authorized
			result.UnAuthorizedTables = s.DeduplicateTables(result.UnAuthorizedTables, authResult.UnAuthorizedTables)
		}

	case pgquery.BoolExprType_AND_EXPR:
		// At least one subexpression must be authorized
		result.Authorized = false
		for _, arg := range boolExpr.Args {
			authResult, err := s.AuthorizeNode(arg, tables, role, id)
			if err != nil {
				return nil, err
			}
			result.UnAuthorizedTables = s.GetIntersection(result.UnAuthorizedTables, authResult.UnAuthorizedTables)
		}
		result.Authorized = len(result.UnAuthorizedTables) == 0

	case pgquery.BoolExprType_NOT_EXPR:
		// Negation: check if the subexpression is unauthorized
		if len(boolExpr.Args) != 1 {
			return nil, fmt.Errorf("NOT expression must have exactly one argument")
		}
		authResult, err := s.AuthorizeNode(boolExpr.Args[0], tables, role, id)
		if err != nil {
			return nil, err
		}
		result.Authorized = !authResult.Authorized
		result.UnAuthorizedTables = s.GetTablesException(tables, authResult.UnAuthorizedTables)
	}
	return result, nil
}

func (s *AuthorizationService) authorizeAExpr(expr *pgquery.A_Expr, tables []TableInfo, role string, id int) (*AuthorizationResult, error) {
	return &AuthorizationResult{Authorized: false, UnAuthorizedTables: nil}, nil
}

// PopulateDefaultColumns ensures all columns from the schema are represented
// in the TableInfo's ColumnsAlias map.
// If a column from the schema is not already in the ColumnsAlias map,
// it's added with itself as the alias.
// This function is idempotent, so it should be safe to call multiple times.
func (s *AuthorizationService) PopulateDefaultColumns(table *TableInfo) {
	if table == nil {
		return
	}

	// Initialize ColumnsAlias if it's nil
	if table.ColumnsAlias == nil {
		table.ColumnsAlias = make(map[string]string)
	}

	// Add all columns from schema if not already present
	for _, col := range s.schemaService.GetColumns(table.Name) {
		if _, ok := table.ColumnsAlias[col]; !ok {
			table.ColumnsAlias[col] = col
		}
	}
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
func (s *AuthorizationService) MergeMaps(dst, src map[string][]*TableInfo) map[string][]*TableInfo {
	if dst == nil {
		dst = make(map[string][]*TableInfo)
	}

	for key, values := range src {
		dst[key] = append(dst[key], values...)
	}

	return dst
}
