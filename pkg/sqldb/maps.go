package sqldb

import "sort"

// SortedKeys returns the keys of a map in sorted order for deterministic
// SQL generation. Used by all SQL connectors for consistent column ordering.
func SortedKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
