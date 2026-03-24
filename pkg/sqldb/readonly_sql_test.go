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

func TestCoerceJSONParamValue_floatWhole(t *testing.T) {
	t.Parallel()
	v := CoerceJSONParamValue(float64(42))
	if i, ok := v.(int64); !ok || i != 42 {
		t.Fatalf("got %T %v", v, v)
	}
}
