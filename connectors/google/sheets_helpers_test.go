package google

import (
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestValidateValues_TooManyRows(t *testing.T) {
	t.Parallel()

	// Create an array with maxSheetsRows+1 rows.
	values := make([][]any, maxSheetsRows+1)
	for i := range values {
		values[i] = []any{"data"}
	}

	err := validateValues(values)
	if err == nil {
		t.Fatal("expected error for too many rows")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestValidateValues_TooManyCells(t *testing.T) {
	t.Parallel()

	// Create 1000 rows × 501 columns = 501,000 cells > maxSheetsCells.
	row := make([]any, 501)
	for i := range row {
		row[i] = "x"
	}
	values := make([][]any, 1000)
	for i := range values {
		r := make([]any, len(row))
		copy(r, row)
		values[i] = r
	}

	err := validateValues(values)
	if err == nil {
		t.Fatal("expected error for too many cells")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestValidateValues_RaggedRows(t *testing.T) {
	t.Parallel()

	values := [][]any{
		{"A", "B", "C"},
		{"D", "E"},
	}

	err := validateValues(values)
	if err == nil {
		t.Fatal("expected error for ragged rows")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestValidateValues_Empty(t *testing.T) {
	t.Parallel()

	err := validateValues(nil)
	if err != nil {
		t.Fatalf("expected no error for empty values, got: %v", err)
	}
}

func TestValidateValues_ValidData(t *testing.T) {
	t.Parallel()

	values := [][]any{
		{"A", "B", "C"},
		{"D", "E", "F"},
	}

	err := validateValues(values)
	if err != nil {
		t.Fatalf("expected no error for valid data, got: %v", err)
	}
}
