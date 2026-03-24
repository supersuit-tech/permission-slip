package bigquery

import (
	"strings"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestValidateReadOnlyBigQuerySQL_withDeleteRejected(t *testing.T) {
	t.Parallel()
	sql := "WITH cte AS (SELECT 1 AS id) DELETE FROM `p.d.t` WHERE id IN (SELECT id FROM cte)"
	err := validateReadOnlyBigQuerySQL(sql)
	if err == nil || !connectors.IsValidationError(err) {
		t.Fatalf("expected validation error, got %v", err)
	}
	if !strings.Contains(err.Error(), "DELETE") {
		t.Fatalf("error should mention DELETE: %v", err)
	}
}

func TestValidateReadOnlyBigQuerySQL_selectOk(t *testing.T) {
	t.Parallel()
	err := validateReadOnlyBigQuerySQL("WITH a AS (SELECT 1) SELECT * FROM a")
	if err != nil {
		t.Fatal(err)
	}
}

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

func TestRejectMultiStatement(t *testing.T) {
	t.Parallel()
	err := rejectMultiStatement("SELECT 1; DROP TABLE x")
	if err == nil || !connectors.IsValidationError(err) {
		t.Fatalf("expected error, got %v", err)
	}
}
