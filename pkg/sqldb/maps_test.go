package sqldb

import "testing"

func TestSortedKeys(t *testing.T) {
	t.Parallel()

	m := map[string]interface{}{"c": 3, "a": 1, "b": 2}
	keys := SortedKeys(m)
	if len(keys) != 3 || keys[0] != "a" || keys[1] != "b" || keys[2] != "c" {
		t.Errorf("SortedKeys() = %v, want [a b c]", keys)
	}
}

func TestSortedKeys_Empty(t *testing.T) {
	t.Parallel()

	keys := SortedKeys(map[string]interface{}{})
	if len(keys) != 0 {
		t.Errorf("SortedKeys({}) = %v, want []", keys)
	}
}
