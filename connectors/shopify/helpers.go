package shopify

import (
	"sort"
	"strings"
)

// sortedKeys returns the keys of a map[string]bool as a sorted,
// comma-separated string. Used in validation error messages so they
// stay in sync with the allowed-values maps automatically.
func sortedKeys(m map[string]bool) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return strings.Join(keys, ", ")
}
