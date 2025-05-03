package db

import om "github.com/elliotchance/orderedmap/v3"

// CloneUnAuthorizedTables creates a deep copy of an OrderedMap of TableInfoV2 pointers, ensuring the original is not modified.
func CloneUnAuthorizedTables(src *om.OrderedMap[string, *TableInfoV2]) *om.OrderedMap[string, *TableInfoV2] {
	if src == nil {
		return nil
	}
	dst := om.NewOrderedMap[string, *TableInfoV2]()
	for alias, table := range src.AllFromFront() {
		clonedValue := table.Clone()
		dst.Set(alias, clonedValue)
	}
	return dst
}

// containsInt checks if a slice contains a specific integer value
func containsInt(slice []int, val int) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}
