package api

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
	"github.com/supersuit-tech/permission-slip/connectors/protonmail"
	"github.com/supersuit-tech/permission-slip/db/testhelper"
)

func TestValidateStandingApprovalConstraintKeys_RejectsUnknownParam(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	testhelper.InsertConnector(t, tx, "protonmail")
	schema := []byte(`{"type":"object","properties":{"message_id":{"type":"integer"},"folder":{"type":"string"}}}`)
	testhelper.InsertConnectorActionFull(t, tx, "protonmail", "protonmail.read_email", "Read Email", testhelper.ConnectorActionOpts{
		ParametersSchema: schema,
	})

	constraints := []byte(`{"bogus_field":"value","message_id":"*"}`)
	err := validateStandingApprovalConstraintKeys(context.Background(), tx, nil, "protonmail.read_email", constraints)
	if err == nil {
		t.Fatal("expected unknown param rejection")
	}
}

func TestValidateStandingApprovalConstraintKeys_SuggestsMetaForKnownField(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	testhelper.InsertConnector(t, tx, "protonmail")
	schema := []byte(`{"type":"object","properties":{"message_id":{"type":"integer"},"folder":{"type":"string"}}}`)
	testhelper.InsertConnectorActionFull(t, tx, "protonmail", "protonmail.read_email", "Read Email", testhelper.ConnectorActionOpts{
		ParametersSchema: schema,
	})

	registry := connectors.NewRegistry()
	registry.Register(protonmail.New())

	constraints := []byte(`{"from":"alice@example.com","message_id":"*"}`)
	err := validateStandingApprovalConstraintKeys(context.Background(), tx, registry, "protonmail.read_email", constraints)
	if err == nil {
		t.Fatal("expected top-level from rejection")
	}
	msg := err.Error()
	if !strings.Contains(msg, `$meta`) || !strings.Contains(msg, "verified metadata constraint") {
		t.Fatalf("expected $meta hint, got: %s", msg)
	}
}

func TestValidateStandingApprovalConstraintKeys_RejectsUnsupportedMetaAction(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	testhelper.InsertConnector(t, tx, "protonmail")
	schema := []byte(`{"type":"object","properties":{"to":{"type":"array"},"subject":{"type":"string"}}}`)
	testhelper.InsertConnectorActionFull(t, tx, "protonmail", "protonmail.send_email", "Send Email", testhelper.ConnectorActionOpts{
		ParametersSchema: schema,
	})

	registry := connectors.NewRegistry()
	registry.Register(protonmail.New())

	constraints := []byte(`{"to":"*","$meta":{"from":"alice@example.com"}}`)
	err := validateStandingApprovalConstraintKeys(context.Background(), tx, registry, "protonmail.send_email", constraints)
	if err == nil {
		t.Fatal("expected unsupported $meta action rejection")
	}
}

func TestValidateStandingApprovalConstraintKeys_RejectsUnknownMetaField(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	testhelper.InsertConnector(t, tx, "protonmail")
	schema := []byte(`{"type":"object","properties":{"message_id":{"type":"integer"},"folder":{"type":"string"}}}`)
	testhelper.InsertConnectorActionFull(t, tx, "protonmail", "protonmail.read_email", "Read Email", testhelper.ConnectorActionOpts{
		ParametersSchema: schema,
	})

	registry := connectors.NewRegistry()
	registry.Register(protonmail.New())

	constraints := []byte(`{"message_id":"*","$meta":{"subject":"Invoice"}}`)
	err := validateStandingApprovalConstraintKeys(context.Background(), tx, registry, "protonmail.read_email", constraints)
	if err == nil {
		t.Fatal("expected unknown $meta field rejection")
	}
}

func TestValidateStandingApprovalConstraintKeys_AllowsValidProtonMeta(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	testhelper.InsertConnector(t, tx, "protonmail")
	schema := []byte(`{"type":"object","properties":{"message_id":{"type":"integer"},"folder":{"type":"string"}}}`)
	testhelper.InsertConnectorActionFull(t, tx, "protonmail", "protonmail.read_email", "Read Email", testhelper.ConnectorActionOpts{
		ParametersSchema: schema,
	})

	registry := connectors.NewRegistry()
	registry.Register(protonmail.New())

	constraints := []byte(`{"message_id":"*","folder":"*","$meta":{"from":{"$pattern":"*@amazon.com"}}}`)
	if err := validateStandingApprovalConstraintKeys(context.Background(), tx, registry, "protonmail.read_email", constraints); err != nil {
		t.Fatalf("expected valid constraints, got: %v", err)
	}
}

func TestValidateStandingApprovalConstraintsForAction_NormalizesPatterns(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	testhelper.InsertConnector(t, tx, "protonmail")
	schema := []byte(`{"type":"object","properties":{"message_id":{"type":"integer"}}}`)
	testhelper.InsertConnectorActionFull(t, tx, "protonmail", "protonmail.read_email", "Read Email", testhelper.ConnectorActionOpts{
		ParametersSchema: schema,
	})

	registry := connectors.NewRegistry()
	registry.Register(protonmail.New())

	raw := json.RawMessage(`{"message_id":"*","$meta":{"from":"*@amazon.com"}}`)
	out, err := validateStandingApprovalConstraintsForAction(context.Background(), tx, registry, "protonmail.read_email", raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var obj map[string]any
	if err := json.Unmarshal(out, &obj); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	meta, ok := obj["$meta"].(map[string]any)
	if !ok {
		t.Fatalf("$meta = %T", obj["$meta"])
	}
	from, ok := meta["from"].(map[string]any)
	if !ok || from["$pattern"] != "*@amazon.com" {
		t.Fatalf("from pattern not normalized: %v", meta["from"])
	}
}
