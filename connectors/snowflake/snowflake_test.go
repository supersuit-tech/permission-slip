package snowflake

import (
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestSnowflakeConnector_ID(t *testing.T) {
	t.Parallel()
	c := New()
	if got := c.ID(); got != "snowflake" {
		t.Errorf("ID() = %q, want snowflake", got)
	}
}

func TestSnowflakeConnector_Actions(t *testing.T) {
	t.Parallel()
	c := New()
	actions := c.Actions()
	if _, ok := actions["snowflake.query"]; !ok {
		t.Fatal("missing snowflake.query action")
	}
	if len(actions) != 1 {
		t.Fatalf("len(actions) = %d, want 1", len(actions))
	}
}

func TestSnowflakeConnector_ValidateCredentials(t *testing.T) {
	t.Parallel()
	c := New()
	err := c.ValidateCredentials(t.Context(), connectors.NewCredentials(map[string]string{
		"connection_string": "user:pass@acct/db",
	}))
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	err = c.ValidateCredentials(t.Context(), connectors.NewCredentials(map[string]string{}))
	if err == nil || !connectors.IsValidationError(err) {
		t.Fatalf("expected validation error, got %v", err)
	}
}
