package bigquery

import (
	"testing"
)

func TestTranslatePlaceholders_skipsStringLiteral(t *testing.T) {
	t.Parallel()
	sql := "SELECT * FROM t WHERE note = 'Did you mean ?%'"
	out, err := translatePlaceholders(sql, 0)
	if err != nil {
		t.Fatal(err)
	}
	if out != sql {
		t.Fatalf("got %q want unchanged", out)
	}
}

func TestTranslatePlaceholders_skipsComment(t *testing.T) {
	t.Parallel()
	sql := "SELECT 1 -- is this ok?\nWHERE x = ?"
	out, err := translatePlaceholders(sql, 1)
	if err != nil {
		t.Fatal(err)
	}
	want := "SELECT 1 -- is this ok?\nWHERE x = @p0"
	if out != want {
		t.Fatalf("got %q want %q", out, want)
	}
}

func TestCoerceParamValue_delegates(t *testing.T) {
	t.Parallel()
	v := coerceParamValue(float64(7))
	if i, ok := v.(int64); !ok || i != 7 {
		t.Fatalf("got %T %v", v, v)
	}
}
