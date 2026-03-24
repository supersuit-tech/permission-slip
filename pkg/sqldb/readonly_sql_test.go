package sqldb

import (
	"strings"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestValidateReadOnlyWarehouseSQL_withDeleteRejected(t *testing.T) {
	t.Parallel()
	sql := "WITH cte AS (SELECT 1 AS id) DELETE FROM `p.d.t` WHERE id IN (SELECT id FROM cte)"
	err := ValidateReadOnlyWarehouseSQL(sql)
	if err == nil || !connectors.IsValidationError(err) {
		t.Fatalf("expected validation error, got %v", err)
	}
	if !strings.Contains(err.Error(), "DELETE") {
		t.Fatalf("error should mention DELETE: %v", err)
	}
}

func TestValidateReadOnlyWarehouseSQL_selectOk(t *testing.T) {
	t.Parallel()
	if err := ValidateReadOnlyWarehouseSQL("WITH a AS (SELECT 1) SELECT * FROM a"); err != nil {
		t.Fatal(err)
	}
}

func TestValidateReadOnlyWarehouseSQL_withAsNoSpace(t *testing.T) {
	t.Parallel()
	if err := ValidateReadOnlyWarehouseSQL("WITH cte AS(SELECT 1) SELECT * FROM cte"); err != nil {
		t.Fatal(err)
	}
}

func TestValidateReadOnlyWarehouseSQL_withIncompleteAS(t *testing.T) {
	t.Parallel()
	err := ValidateReadOnlyWarehouseSQL("WITH cte AS")
	if err == nil || !connectors.IsValidationError(err) {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func TestValidateReadOnlyWarehouseSQL_notWITHIN(t *testing.T) {
	t.Parallel()
	err := ValidateReadOnlyWarehouseSQL("WITHIN(1,2)")
	if err == nil || !connectors.IsValidationError(err) {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func TestValidateReadOnlyWarehouseSQL_backtickCTEName(t *testing.T) {
	t.Parallel()
	if err := ValidateReadOnlyWarehouseSQL("WITH `my-dataset` AS (SELECT 1) SELECT * FROM `my-dataset`"); err != nil {
		t.Fatalf("backtick CTE name should be accepted: %v", err)
	}
}

func TestValidateReadOnlyWarehouseSQL_tripleQuotedString(t *testing.T) {
	t.Parallel()
	// Triple-quoted string containing DELETE should not trigger keyword rejection
	if err := ValidateReadOnlyWarehouseSQL(`SELECT * FROM t WHERE x = """DELETE"""`); err != nil {
		t.Fatalf("triple-quoted string should be masked: %v", err)
	}
}

func TestScrubSQLMasking_unterminatedBlockComment(t *testing.T) {
	t.Parallel()
	sql := "SELECT 1 /* note: DELET"
	scrubbed := scrubSQLMasking(sql)
	// The final byte of the unterminated comment must be masked
	if strings.ContainsAny(scrubbed[len("SELECT 1 "):], "DELET/*") {
		t.Fatalf("unterminated block comment leaked bytes: %q", scrubbed)
	}
}

func TestCoerceJSONParamValue_floatWhole(t *testing.T) {
	t.Parallel()
	v := CoerceJSONParamValue(float64(42))
	if i, ok := v.(int64); !ok || i != 42 {
		t.Fatalf("got %T %v", v, v)
	}
}

func TestCoerceJSONParamValue_maxInt64Boundary(t *testing.T) {
	t.Parallel()
	// float64(math.MaxInt64) rounds up to 9223372036854775808.0 which overflows int64.
	// It must remain a float64 and NOT be converted to int64.
	v := CoerceJSONParamValue(float64(9.223372036854776e18))
	if _, ok := v.(float64); !ok {
		t.Fatalf("value near MaxInt64 boundary should stay float64, got %T %v", v, v)
	}
}
