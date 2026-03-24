package bigquery

import (
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestTranslatePlaceholders(t *testing.T) {
	t.Parallel()
	sql, err := translatePlaceholders("SELECT * FROM t WHERE x = ? AND y = ?", 2)
	if err != nil {
		t.Fatal(err)
	}
	want := "SELECT * FROM t WHERE x = @p0 AND y = @p1"
	if sql != want {
		t.Fatalf("got %q want %q", sql, want)
	}
}

func TestTranslatePlaceholders_mismatch(t *testing.T) {
	t.Parallel()
	_, err := translatePlaceholders("SELECT ?", 2)
	if err == nil || !connectors.IsValidationError(err) {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func TestTranslatePlaceholders_paramsWithoutQ(t *testing.T) {
	t.Parallel()
	_, err := translatePlaceholders("SELECT 1", 1)
	if err == nil || !connectors.IsValidationError(err) {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func TestCoerceParamValue_floatWholeNumber(t *testing.T) {
	t.Parallel()
	v := coerceParamValue(float64(42))
	if i, ok := v.(int64); !ok || i != 42 {
		t.Fatalf("got %T %v want int64 42", v, v)
	}
}
