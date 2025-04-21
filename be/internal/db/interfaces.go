package db

import (
	"github.com/jmoiron/sqlx"
)

// QueryResult represents the result of a SELECT query.
// This will be used to call the LLM to format a natural language response for the user.
// In the future, this will be decoupled into another DTO.
type QueryResult struct {
	Metadata struct {
		RowCount int      `json:"row_count"`
		Columns  []string `json:"columns"`
	} `json:"metadata"`
	Data []map[string]interface{} `json:"data"`
}

type QueryRequest struct {
	Query string `json:"query"`
}

type DDLLoader interface {
	LoadDDL(db *sqlx.DB) (string, error)
}
