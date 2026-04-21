package api

import (
	"encoding/json"
	"testing"
)

func TestInjectConnectorInstanceIntoParametersSchema(t *testing.T) {
	t.Parallel()
	base := json.RawMessage(`{"type":"object","properties":{"x":{"type":"string"}}}`)
	inst := []connectorInstanceCapability{
		{ID: "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa", Display: "A"},
		{ID: "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb", Display: "B"},
	}
	out := injectConnectorInstanceIntoParametersSchema(base, inst)
	var sch map[string]any
	if err := json.Unmarshal(out, &sch); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	props, _ := sch["properties"].(map[string]any)
	ci, _ := props["connector_instance"].(map[string]any)
	if ci["format"] != "uuid" {
		t.Errorf("expected format uuid, got %#v", ci["format"])
	}
	enum, _ := ci["enum"].([]any)
	if len(enum) != 2 {
		t.Fatalf("enum: %#v", enum)
	}
}

func TestInjectConnectorInstanceIntoParametersSchema_SingleInstanceNoOp(t *testing.T) {
	t.Parallel()
	base := json.RawMessage(`{"type":"object"}`)
	inst := []connectorInstanceCapability{{ID: "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"}}
	out := injectConnectorInstanceIntoParametersSchema(base, inst)
	if string(out) != string(base) {
		t.Errorf("expected unchanged schema, got %s", string(out))
	}
}
