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

func TestValidateStandingApprovalConstraintKeys_RejectsEmptyKeyWithMetaHint(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	testhelper.InsertConnector(t, tx, "protonmail")
	schema := []byte(`{"type":"object","properties":{"message_id":{"type":"integer"},"folder":{"type":"string"}}}`)
	testhelper.InsertConnectorActionFull(t, tx, "protonmail", "protonmail.read_email", "Read Email", testhelper.ConnectorActionOpts{
		ParametersSchema: schema,
	})

	registry := connectors.NewRegistry()
	registry.Register(protonmail.New())

	constraints := []byte(`{"":{"from":"automated@airbnb.com"}}`)
	err := validateStandingApprovalConstraintKeys(context.Background(), tx, registry, "protonmail.read_email", constraints)
	if err == nil {
		t.Fatal("expected empty key rejection")
	}
	msg := err.Error()
	if !strings.Contains(msg, `$meta`) || !strings.Contains(msg, "verified metadata constraint") {
		t.Fatalf("expected $meta hint, got: %s", msg)
	}
}

func TestValidateStandingApprovalConstraintKeys_RejectsEmptyKeyWithoutSchema(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	testhelper.InsertConnector(t, tx, "protonmail")
	testhelper.InsertConnectorAction(t, tx, "protonmail", "protonmail.read_email", "Read Email")

	registry := connectors.NewRegistry()
	registry.Register(protonmail.New())

	constraints := []byte(`{"":{"from":"automated@airbnb.com"}}`)
	err := validateStandingApprovalConstraintKeys(context.Background(), tx, registry, "protonmail.read_email", constraints)
	if err == nil {
		t.Fatal("expected empty key rejection without parameters schema")
	}
	if !strings.Contains(err.Error(), `$meta`) {
		t.Fatalf("expected $meta hint, got: %s", err.Error())
	}
}

func TestValidateStandingApprovalConstraintsForAction_RejectsEmptyMetaKey(t *testing.T) {
	t.Parallel()
	raw := json.RawMessage(`{"message_id":"*","$meta":{"":"alice@example.com"}}`)
	_, err := validateStandingApprovalConstraints(raw)
	if err == nil {
		t.Fatal("expected empty $meta key rejection")
	}
	if !strings.Contains(err.Error(), "$meta constraint keys must not be empty") {
		t.Fatalf("unexpected error: %v", err)
	}
}

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

func TestValidateStandingApprovalConstraintsForAction_FillsMissingSchemaWildcards(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	testhelper.InsertConnector(t, tx, "protonmail")
	schema := []byte(`{"type":"object","properties":{"message_id":{"type":"integer"},"folder":{"type":"string"}}}`)
	testhelper.InsertConnectorActionFull(t, tx, "protonmail", "protonmail.read_email", "Read Email", testhelper.ConnectorActionOpts{
		ParametersSchema: schema,
	})

	registry := connectors.NewRegistry()
	registry.Register(protonmail.New())

	raw := json.RawMessage(`{"$meta":{"from":"automated@example.com"}}`)
	out, err := validateStandingApprovalConstraintsForAction(context.Background(), tx, registry, "protonmail.read_email", raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var obj map[string]any
	if err := json.Unmarshal(out, &obj); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if obj["message_id"] != "*" || obj["folder"] != "*" {
		t.Fatalf("expected missing schema params wildcarded, got %#v", obj)
	}
	meta, ok := obj["$meta"].(map[string]any)
	if !ok || meta["from"] != "automated@example.com" {
		t.Fatalf("$meta.from = %#v", obj["$meta"])
	}
}

func TestValidateStandingApprovalConstraintsForAction_PartialSchemaWildcards(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	testhelper.InsertConnector(t, tx, "protonmail")
	schema := []byte(`{"type":"object","properties":{"message_id":{"type":"integer"},"folder":{"type":"string"}}}`)
	testhelper.InsertConnectorActionFull(t, tx, "protonmail", "protonmail.read_email", "Read Email", testhelper.ConnectorActionOpts{
		ParametersSchema: schema,
	})

	registry := connectors.NewRegistry()
	registry.Register(protonmail.New())

	raw := json.RawMessage(`{"folder":"INBOX"}`)
	out, err := validateStandingApprovalConstraintsForAction(context.Background(), tx, registry, "protonmail.read_email", raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var obj map[string]any
	if err := json.Unmarshal(out, &obj); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if obj["message_id"] != "*" {
		t.Fatalf("expected message_id wildcard, got %#v", obj["message_id"])
	}
	if obj["folder"] != "INBOX" {
		t.Fatalf("expected folder INBOX, got %#v", obj["folder"])
	}
}

func TestValidateStandingApprovalConstraintKeys_RejectsObjectValueOnSchemaParam(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	testhelper.InsertConnector(t, tx, "protonmail")
	schema := []byte(`{"type":"object","properties":{"message_id":{"type":"integer"},"limit":{"type":"integer"}}}`)
	testhelper.InsertConnectorActionFull(t, tx, "protonmail", "protonmail.read_email", "Read Email", testhelper.ConnectorActionOpts{
		ParametersSchema: schema,
	})

	constraints := []byte(`{"message_id":"*","limit":{"threshold":20}}`)
	err := validateStandingApprovalConstraintKeys(context.Background(), tx, nil, "protonmail.read_email", constraints)
	if err == nil {
		t.Fatal("expected object value rejection on scalar param")
	}
	if !strings.Contains(err.Error(), "constraint value for \"limit\"") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateStandingApprovalConstraintKeys_AllowsIntegerValueOnSchemaParam(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	testhelper.InsertConnector(t, tx, "protonmail")
	schema := []byte(`{"type":"object","properties":{"message_id":{"type":"integer"},"limit":{"type":"integer"}}}`)
	testhelper.InsertConnectorActionFull(t, tx, "protonmail", "protonmail.read_email", "Read Email", testhelper.ConnectorActionOpts{
		ParametersSchema: schema,
	})

	constraints := []byte(`{"message_id":"*","limit":20}`)
	if err := validateStandingApprovalConstraintKeys(context.Background(), tx, nil, "protonmail.read_email", constraints); err != nil {
		t.Fatalf("expected integer limit constraint, got: %v", err)
	}
}

func TestValidateStandingApprovalConstraintKeys_RejectsObjectValueInStructuredConstraints(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	testhelper.InsertConnector(t, tx, "protonmail")
	schema := []byte(`{"type":"object","properties":{"limit":{"type":"integer"}}}`)
	testhelper.InsertConnectorActionFull(t, tx, "protonmail", "protonmail.read_email", "Read Email", testhelper.ConnectorActionOpts{
		ParametersSchema: schema,
	})

	constraints := []byte(`{
		"$version": 2,
		"match": "any",
		"groups": [{
			"match": "all",
			"conditions": [{"field": "limit", "op": "matches", "value": {"threshold": 20}}]
		}]
	}`)
	err := validateStandingApprovalConstraintKeys(context.Background(), tx, nil, "protonmail.read_email", constraints)
	if err == nil {
		t.Fatal("expected object value rejection in structured constraints")
	}
	if !strings.Contains(err.Error(), "constraint value for \"limit\"") {
		t.Fatalf("unexpected error: %v", err)
	}
}
