package instacart

import (
	"encoding/json"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestCreateProductsLink_ParameterAliases(t *testing.T) {
	t.Parallel()
	a := &createProductsLinkAction{}
	aliases := a.ParameterAliases()
	if aliases["items"] != "line_items" {
		t.Fatalf("ParameterAliases items = %q", aliases["items"])
	}
}

func TestCreateProductsLink_Normalize_StringLineItems(t *testing.T) {
	t.Parallel()
	a := &createProductsLinkAction{}
	in := json.RawMessage(`{"line_items":["milk","organic eggs"]}`)
	out := a.Normalize(in)

	var m map[string]json.RawMessage
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	var items []map[string]string
	if err := json.Unmarshal(m["line_items"], &items); err != nil {
		t.Fatalf("unmarshal line_items: %v", err)
	}
	if len(items) != 2 || items[0]["name"] != "milk" || items[1]["name"] != "organic eggs" {
		t.Fatalf("got %#v", items)
	}
}

func TestCreateProductsLink_Normalize_ItemsAliasPipeline(t *testing.T) {
	t.Parallel()
	a := &createProductsLinkAction{}
	raw := json.RawMessage(`{"items":["flour"],"title":"Cake"}`)
	afterAlias := connectors.NormalizeParameters(a.ParameterAliases(), raw)
	out := a.Normalize(afterAlias)

	var m map[string]json.RawMessage
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := m["items"]; ok {
		t.Fatal("items key should be renamed to line_items")
	}
	var items []map[string]string
	if err := json.Unmarshal(m["line_items"], &items); err != nil {
		t.Fatalf("unmarshal line_items: %v", err)
	}
	if len(items) != 1 || items[0]["name"] != "flour" {
		t.Fatalf("got %#v", items)
	}
}

func TestCreateProductsLink_ValidateRequest_StringItems(t *testing.T) {
	t.Parallel()
	a := &createProductsLinkAction{}
	err := a.ValidateRequest(json.RawMessage(`{"line_items":["a","b"]}`))
	if err != nil {
		t.Fatal(err)
	}
}

func TestCreateProductsLink_ValidateRequest_ItemsAlias(t *testing.T) {
	t.Parallel()
	a := &createProductsLinkAction{}
	// ValidateRequest runs after Normalizer in API — simulate with Normalize first.
	raw := connectors.NormalizeParameters(a.ParameterAliases(), json.RawMessage(`{"items":["x"]}`))
	raw = a.Normalize(raw)
	if err := a.ValidateRequest(raw); err != nil {
		t.Fatal(err)
	}
}
