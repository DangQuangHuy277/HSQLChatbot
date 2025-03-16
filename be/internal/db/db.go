package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

var ddlLoaders = make(map[string]DDLLoader)

type HDb struct {
	*sqlx.DB
	DDLLoader
}

func NewHDb(driverName, dataSourceUrl string) (*HDb, error) {
	db, err := sqlx.Connect(driverName, dataSourceUrl)
	if err != nil {
		return nil, err
	}
	ddlLoader := ddlLoaders[driverName]

	return &HDb{db, ddlLoader}, nil
}

func (hdb *HDb) LoadDDL() (string, error) {
	return hdb.DDLLoader.LoadDDL(hdb.DB)
}

func (hdb *HDb) ExecuteQuery(ctx context.Context, query string) (*QueryResult, error) {
	rows, err := hdb.QueryxContext(ctx, query)
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
