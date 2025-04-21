package db

type SchemaService interface {
	// GetColumns returns the list of column names for a given table.
	GetColumns(tableName string) []string
}

type SchemaServiceImpl struct {
}

func NewSchemaServiceImpl() *SchemaServiceImpl {
	return &SchemaServiceImpl{}
}

func (s *SchemaServiceImpl) GetColumns(tableName string) []string {
	return []string{
		"column1",
	}
}
