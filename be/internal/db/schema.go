package db

import (
	"sync"
)

type SchemaService interface {
	// GetColumns returns the list of column names for a given table.
	GetColumns(tableName string) []string
}

type SchemaServiceImpl struct {
	db          *SQLHDb
	columnCache map[string][]string
	cacheMutex  sync.RWMutex
}

func NewSchemaServiceImpl(db *SQLHDb) *SchemaServiceImpl {
	service := &SchemaServiceImpl{
		db:          db,
		columnCache: make(map[string][]string),
	}
	return service
}

func (s *SchemaServiceImpl) GetColumns(tableName string) []string {
	// First check cache
	s.cacheMutex.RLock()
	if columns, exists := s.columnCache[tableName]; exists {
		s.cacheMutex.RUnlock()
		return columns
	}
	s.cacheMutex.RUnlock()

	// If not in cache, query database if available
	if s.db != nil {
		columns := s.queryColumnsFromDB(tableName)
		if len(columns) > 0 {
			// Cache the result
			s.cacheMutex.Lock()
			s.columnCache[tableName] = columns
			s.cacheMutex.Unlock()
			return columns
		}
	}

	// Return empty slice if table not found
	return []string{}
}

func (s *SchemaServiceImpl) queryColumnsFromDB(tableName string) []string {
	var columns []string
	err := s.db.Select(columns,
		`SELECT column_name 
		FROM information_schema.columns 
		WHERE table_name = $1 
		ORDER BY ordinal_position`, tableName)

	if err != nil {
		return []string{}
	}
	return columns
}
