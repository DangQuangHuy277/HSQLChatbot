package db

import (
	om "github.com/elliotchance/orderedmap/v3"
	"github.com/stretchr/testify/assert"
	"testing"
)

// region TestColumnInfo_Clone
func TestColumnInfo_Clone(t *testing.T) {
	tests := []struct {
		name     string
		input    *ColumnInfo
		expected *ColumnInfo
	}{
		{
			name:     "Nil column info",
			input:    nil,
			expected: nil,
		},
		{
			name: "Basic column info without source table",
			input: &ColumnInfo{
				Name: "col1",
			},
			expected: &ColumnInfo{
				Name: "col1",
			},
		},
		{
			name: "Column info with source table",
			input: &ColumnInfo{
				Name: "col1",
				SourceTable: &TableInfoV2{
					Name:       "table1",
					Alias:      "t1",
					IsDatabase: true,
				},
			},
			expected: &ColumnInfo{
				Name: "col1",
				SourceTable: &TableInfoV2{
					Name:       "table1",
					Alias:      "t1",
					IsDatabase: true,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.input.Clone(nil)
			if tt.expected == nil {
				assert.Nil(t, result)
				return
			}

			assert.Equal(t, tt.expected.Name, result.Name)

			if tt.expected.SourceTable == nil {
				assert.Nil(t, result.SourceTable)
			} else {
				assert.Equal(t, tt.expected.SourceTable.Name, result.SourceTable.Name)
				assert.Equal(t, tt.expected.SourceTable.Alias, result.SourceTable.Alias)
				assert.Equal(t, tt.expected.SourceTable.IsDatabase, result.SourceTable.IsDatabase)
			}

			// Check that it's actually a clone (different pointers)
			if tt.input != nil {
				assert.NotSame(t, tt.input, result)
				if tt.input.SourceTable != nil {
					assert.NotSame(t, tt.input.SourceTable, result.SourceTable)
				}
			}
		})
	}
}

// endregion

// region TestTableInfoV2_Clone
func TestTableInfoV2_Clone(t *testing.T) {
	t.Run("Nil table", func(t *testing.T) {
		var table *TableInfoV2
		clone := table.Clone()
		assert.Nil(t, clone)
	})

	t.Run("Empty table", func(t *testing.T) {
		table := &TableInfoV2{
			Name:       "table1",
			Alias:      "t1",
			HasStar:    false,
			IsDatabase: true,
		}
		clone := table.Clone()

		assert.Equal(t, table.Name, clone.Name)
		assert.Equal(t, table.Alias, clone.Alias)
		assert.Equal(t, table.HasStar, clone.HasStar)
		assert.Equal(t, table.IsDatabase, clone.IsDatabase)
		assert.Nil(t, clone.Columns)
		assert.Nil(t, clone.UnAuthorizedTables)
		assert.NotSame(t, table, clone)
	})

	t.Run("Table with columns", func(t *testing.T) {
		table := &TableInfoV2{
			Name:       "table1",
			Alias:      "t1",
			IsDatabase: true,
			Columns:    om.NewOrderedMap[string, *ColumnInfo](),
		}

		// Add columns to the table
		sourceTable := &TableInfoV2{
			Name:       "source",
			IsDatabase: true,
		}
		table.Columns.Set("col1", &ColumnInfo{
			Name:        "col1",
			SourceTable: sourceTable,
		})

		clone := table.Clone()

		// Check basic properties
		assert.Equal(t, table.Name, clone.Name)
		assert.Equal(t, table.Alias, clone.Alias)
		assert.NotNil(t, clone.Columns)
		assert.Equal(t, 1, clone.Columns.Len())

		// Check column cloning
		col, _ := clone.Columns.Get("col1")
		assert.Equal(t, "col1", col.Name)
		assert.Equal(t, "source", col.SourceTable.Name)

		// Check that it's a deep copy
		originalCol, _ := table.Columns.Get("col1")
		assert.NotSame(t, originalCol, col)
		assert.NotSame(t, originalCol.SourceTable, col.SourceTable)
	})

	t.Run("Table with unauthorized tables", func(t *testing.T) {
		table := &TableInfoV2{
			Name:               "table1",
			UnAuthorizedTables: om.NewOrderedMap[string, *TableInfoV2](),
		}

		unauth := &TableInfoV2{
			Name:       "unauth",
			IsDatabase: true,
		}
		table.UnAuthorizedTables.Set("ua1", unauth)

		clone := table.Clone()

		assert.NotNil(t, clone.UnAuthorizedTables)
		assert.Equal(t, 1, clone.UnAuthorizedTables.Len())

		unauthClone, _ := clone.UnAuthorizedTables.Get("ua1")
		assert.Equal(t, "unauth", unauthClone.Name)
		assert.Equal(t, true, unauthClone.IsDatabase)

		// Check that it's a deep copy
		originalUnauth, _ := table.UnAuthorizedTables.Get("ua1")
		assert.NotSame(t, originalUnauth, unauthClone)
	})
}

// endregion

// region TestTableInfoV2_createColumnFromSourceTable
func TestTableInfoV2_createColumnFromSourceTable(t *testing.T) {
	t.Run("Create column with nil columns map", func(t *testing.T) {
		sourceTable := &TableInfoV2{
			Name:    "source",
			Columns: om.NewOrderedMap[string, *ColumnInfo](),
		}
		sourceTable.Columns.Set("col1", &ColumnInfo{
			Name:        "col1",
			SourceTable: sourceTable,
		})

		targetTable := &TableInfoV2{Name: "target"}
		targetTable.createColumnFromSourceTable("col1", sourceTable, nil)

		assert.NotNil(t, targetTable.Columns)
		assert.Equal(t, 1, targetTable.Columns.Len())

		col, exists := targetTable.Columns.Get("col1")
		assert.True(t, exists)
		assert.Equal(t, "col1", col.Name)
		assert.Same(t, sourceTable, col.SourceTable)
	})

	t.Run("Create column with custom alias", func(t *testing.T) {
		sourceTable := &TableInfoV2{
			Name:    "source",
			Columns: om.NewOrderedMap[string, *ColumnInfo](),
		}
		sourceTable.Columns.Set("col1", &ColumnInfo{
			Name:        "col1",
			SourceTable: sourceTable,
		})

		targetTable := &TableInfoV2{
			Name:    "target",
			Columns: om.NewOrderedMap[string, *ColumnInfo](),
		}
		newAlias := "new_alias"
		targetTable.createColumnFromSourceTable("col1", sourceTable, &newAlias)

		col, exists := targetTable.Columns.Get("new_alias")
		assert.True(t, exists)
		assert.Equal(t, "col1", col.Name)
	})

	t.Run("Create non-existing column", func(t *testing.T) {
		sourceTable := &TableInfoV2{
			Name:    "source",
			Columns: om.NewOrderedMap[string, *ColumnInfo](),
		}

		targetTable := &TableInfoV2{Name: "target"}
		targetTable.createColumnFromSourceTable("nonexistent", sourceTable, nil)

		assert.NotNil(t, targetTable.Columns)
		assert.Equal(t, 0, targetTable.Columns.Len())
	})
}

// endregion

// region TestTableInfoV2_createAllColumnFromSourceTable
func TestTableInfoV2_createAllColumnFromSourceTable(t *testing.T) {
	t.Run("Nil source table", func(t *testing.T) {
		targetTable := &TableInfoV2{Name: "target"}
		targetTable.createAllColumnFromSourceTable(nil, nil)
		assert.Nil(t, targetTable.Columns)
	})

	t.Run("Source table with nil columns", func(t *testing.T) {
		sourceTable := &TableInfoV2{Name: "source"}
		targetTable := &TableInfoV2{Name: "target"}
		targetTable.createAllColumnFromSourceTable(sourceTable, nil)
		assert.Nil(t, targetTable.Columns)
	})

	t.Run("Create columns without aliases", func(t *testing.T) {
		sourceTable := &TableInfoV2{
			Name:    "source",
			Columns: om.NewOrderedMap[string, *ColumnInfo](),
		}
		sourceTable.Columns.Set("col1", &ColumnInfo{Name: "col1", SourceTable: sourceTable})
		sourceTable.Columns.Set("col2", &ColumnInfo{Name: "col2", SourceTable: sourceTable})

		targetTable := &TableInfoV2{Name: "target"}
		targetTable.createAllColumnFromSourceTable(sourceTable, nil)

		assert.Equal(t, 2, targetTable.Columns.Len())
		col1, _ := targetTable.Columns.Get("col1")
		assert.Equal(t, "col1", col1.Name)
		col2, _ := targetTable.Columns.Get("col2")
		assert.Equal(t, "col2", col2.Name)
	})

	t.Run("Create columns with aliases", func(t *testing.T) {
		sourceTable := &TableInfoV2{
			Name:    "source",
			Columns: om.NewOrderedMap[string, *ColumnInfo](),
		}
		sourceTable.Columns.Set("col1", &ColumnInfo{Name: "col1", SourceTable: sourceTable})
		sourceTable.Columns.Set("col2", &ColumnInfo{Name: "col2", SourceTable: sourceTable})

		targetTable := &TableInfoV2{Name: "target"}
		targetTable.createAllColumnFromSourceTable(sourceTable, []string{"a1", "a2"})

		assert.Equal(t, 2, targetTable.Columns.Len())
		col1, _ := targetTable.Columns.Get("a1")
		assert.Equal(t, "col1", col1.Name)
		col2, _ := targetTable.Columns.Get("a2")
		assert.Equal(t, "col2", col2.Name)
	})

	t.Run("Create columns with fewer aliases than columns", func(t *testing.T) {
		sourceTable := &TableInfoV2{
			Name:    "source",
			Columns: om.NewOrderedMap[string, *ColumnInfo](),
		}
		sourceTable.Columns.Set("col1", &ColumnInfo{Name: "col1", SourceTable: sourceTable})
		sourceTable.Columns.Set("col2", &ColumnInfo{Name: "col2", SourceTable: sourceTable})

		targetTable := &TableInfoV2{Name: "target"}
		targetTable.createAllColumnFromSourceTable(sourceTable, []string{"a1"})

		assert.Equal(t, 2, targetTable.Columns.Len())
		col1, _ := targetTable.Columns.Get("a1")
		assert.Equal(t, "col1", col1.Name)
		col2, _ := targetTable.Columns.Get("col2")
		assert.Equal(t, "col2", col2.Name)
	})
}

// endregion

// region TestTableInfoV2_createAllColumnsFromSourceTableList
func TestTableInfoV2_createAllColumnsFromSourceTableList(t *testing.T) {
	t.Run("Empty source table list", func(t *testing.T) {
		targetTable := &TableInfoV2{Name: "target"}
		tables := om.NewOrderedMap[string, *TableInfoV2]()
		targetTable.createAllColumnsFromSourceTableList(tables.Values(), []string{"a1", "a2"})
		targetTable.createAllColumnsFromSourceTableList(tables.Values(), nil)
		assert.Nil(t, targetTable.Columns)
	})

	t.Run("Multiple tables without aliases", func(t *testing.T) {
		sourceTable1 := &TableInfoV2{
			Name:    "source1",
			Columns: om.NewOrderedMap[string, *ColumnInfo](),
		}
		sourceTable1.Columns.Set("col1", &ColumnInfo{Name: "col1", SourceTable: sourceTable1})

		sourceTable2 := &TableInfoV2{
			Name:    "source2",
			Columns: om.NewOrderedMap[string, *ColumnInfo](),
		}
		sourceTable2.Columns.Set("col2", &ColumnInfo{Name: "col2", SourceTable: sourceTable2})

		targetTable := &TableInfoV2{Name: "target"}
		tables := om.NewOrderedMapWithElements(
			&om.Element[string, *TableInfoV2]{Key: "source1", Value: sourceTable1},
			&om.Element[string, *TableInfoV2]{Key: "source2", Value: sourceTable2})
		targetTable.createAllColumnsFromSourceTableList(tables.Values(), nil)

		assert.Equal(t, 2, targetTable.Columns.Len())
		col1, _ := targetTable.Columns.Get("col1")
		assert.Equal(t, "col1", col1.Name)
		assert.Equal(t, sourceTable1, col1.SourceTable)

		col2, _ := targetTable.Columns.Get("col2")
		assert.Equal(t, "col2", col2.Name)
		assert.Equal(t, sourceTable2, col2.SourceTable)
	})

	t.Run("Multiple tables with aliases", func(t *testing.T) {
		sourceTable1 := &TableInfoV2{
			Name:    "source1",
			Columns: om.NewOrderedMap[string, *ColumnInfo](),
		}
		sourceTable1.Columns.Set("col1", &ColumnInfo{Name: "col1", SourceTable: sourceTable1})

		sourceTable2 := &TableInfoV2{
			Name:    "source2",
			Columns: om.NewOrderedMap[string, *ColumnInfo](),
		}
		sourceTable2.Columns.Set("col2", &ColumnInfo{Name: "col2", SourceTable: sourceTable2})

		targetTable := &TableInfoV2{Name: "target"}
		tables := om.NewOrderedMapWithElements(
			&om.Element[string, *TableInfoV2]{Key: "source1", Value: sourceTable1},
			&om.Element[string, *TableInfoV2]{Key: "source2", Value: sourceTable2})
		targetTable.createAllColumnsFromSourceTableList(tables.Values(), []string{"a1", "a2"})

		assert.Equal(t, 2, targetTable.Columns.Len())
		col1, _ := targetTable.Columns.Get("a1")
		assert.Equal(t, "col1", col1.Name)

		col2, _ := targetTable.Columns.Get("a2")
		assert.Equal(t, "col2", col2.Name)
	})

	t.Run("Multiple tables with fewer aliases than columns", func(t *testing.T) {
		sourceTable1 := &TableInfoV2{
			Name:    "source1",
			Columns: om.NewOrderedMap[string, *ColumnInfo](),
		}
		sourceTable1.Columns.Set("col1", &ColumnInfo{Name: "col1", SourceTable: sourceTable1})
		sourceTable1.Columns.Set("col2", &ColumnInfo{Name: "col2", SourceTable: sourceTable1})

		sourceTable2 := &TableInfoV2{
			Name:    "source2",
			Columns: om.NewOrderedMap[string, *ColumnInfo](),
		}
		sourceTable2.Columns.Set("col3", &ColumnInfo{Name: "col3", SourceTable: sourceTable2})

		targetTable := &TableInfoV2{Name: "target"}
		tables := om.NewOrderedMapWithElements(
			&om.Element[string, *TableInfoV2]{Key: "source1", Value: sourceTable1},
			&om.Element[string, *TableInfoV2]{Key: "source2", Value: sourceTable2})
		targetTable.createAllColumnsFromSourceTableList(tables.Values(), []string{"a1", "a2"})

		assert.Equal(t, 3, targetTable.Columns.Len())
		col1, _ := targetTable.Columns.Get("a1")
		assert.Equal(t, "col1", col1.Name)

		col2, _ := targetTable.Columns.Get("a2")
		assert.Equal(t, "col2", col2.Name)

		col3, _ := targetTable.Columns.Get("col3")
		assert.Equal(t, "col3", col3.Name)
	})
}

// endregion

// region TestTableInfoV2_getAliasOfColumn
func TestTab(t *testing.T) {
	t.Run("Direct column in database table", func(t *testing.T) {
		table := &TableInfoV2{
			Name:       "table",
			IsDatabase: true,
			Columns:    om.NewOrderedMap[string, *ColumnInfo](),
		}
		table.Columns.Set("c1", &ColumnInfo{
			Name:        "col1",
			SourceTable: table,
		})

		alias := table.getAliasOfColumn("col1")
		assert.Equal(t, "c1", alias)
	})

	t.Run("Column in nested subquery", func(t *testing.T) {
		dbTable := &TableInfoV2{
			Name:       "db_table",
			IsDatabase: true,
			Columns:    om.NewOrderedMap[string, *ColumnInfo](),
		}
		dbTable.Columns.Set("col1", &ColumnInfo{
			Name:        "col1",
			SourceTable: dbTable,
		})

		subquery := &TableInfoV2{
			Name:       "subquery",
			IsDatabase: false,
			Columns:    om.NewOrderedMap[string, *ColumnInfo](),
		}
		subquery.Columns.Set("sq_col", &ColumnInfo{
			Name:        "col1",
			SourceTable: dbTable,
		})

		mainQuery := &TableInfoV2{
			Name:       "main",
			IsDatabase: false,
			Columns:    om.NewOrderedMap[string, *ColumnInfo](),
		}
		mainQuery.Columns.Set("main_col", &ColumnInfo{
			Name:        "sq_col",
			SourceTable: subquery,
		})

		alias := mainQuery.getAliasOfColumn("col1")
		assert.Equal(t, "main_col", alias)
	})

	t.Run("Non-existent column", func(t *testing.T) {
		table := &TableInfoV2{
			Name:       "table",
			IsDatabase: true,
			Columns:    om.NewOrderedMap[string, *ColumnInfo](),
		}
		table.Columns.Set("c1", &ColumnInfo{
			Name:        "col1",
			SourceTable: table,
		})

		alias := table.getAliasOfColumn("nonexistent")
		assert.Equal(t, "", alias)
	})
}

// endregion
