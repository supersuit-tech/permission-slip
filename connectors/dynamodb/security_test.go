package dynamodb

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestDeleteItem_PreviousItemFiltered(t *testing.T) {
	t.Parallel()
	mock := &mockDynamo{
		deleteItemOut: &dynamodb.DeleteItemOutput{
			Attributes: map[string]types.AttributeValue{
				"pk":     &types.AttributeValueMemberS{Value: "k1"},
				"name":   &types.AttributeValueMemberS{Value: "Ada"},
				"secret": &types.AttributeValueMemberS{Value: "classified"},
			},
		},
	}
	conn := newForTest(mock)
	action := conn.Actions()["dynamodb.delete_item"]

	res, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType: "dynamodb.delete_item",
		Parameters: json.RawMessage(`{
			"region":"us-east-1",
			"table":"t1",
			"key":{"pk":"k1"},
			"allowed_tables":["t1"],
			"allowed_read_attributes":["pk","name"]
		}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatal(err)
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(res.Data, &payload); err != nil {
		t.Fatal(err)
	}
	if !payload["existed"].(bool) {
		t.Fatal("want existed true")
	}
	prev := payload["previous_item"].(map[string]interface{})
	if _, has := prev["secret"]; has {
		t.Fatalf("secret should be filtered from previous_item: %#v", prev)
	}
	if prev["pk"] != "k1" || prev["name"] != "Ada" {
		t.Fatalf("allowed attributes missing: %#v", prev)
	}
}

func TestDeleteItem_ExprAttrNamesValidated(t *testing.T) {
	t.Parallel()
	mock := &mockDynamo{}
	conn := newForTest(mock)
	action := conn.Actions()["dynamodb.delete_item"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType: "dynamodb.delete_item",
		Parameters: json.RawMessage(`{
			"region":"us-east-1",
			"table":"t1",
			"key":{"pk":"k1"},
			"allowed_tables":["t1"],
			"allowed_write_attributes":["pk","status"],
			"condition_expression":"#s = :v",
			"expression_attribute_names":{"#s":"secret_field"},
			"expression_attribute_values":{":v":"active"}
		}`),
		Credentials: validCreds(),
	})
	if err == nil || !connectors.IsValidationError(err) {
		t.Fatalf("want validation error for disallowed expr attr name, got %v", err)
	}
	if !strings.Contains(err.Error(), "secret_field") {
		t.Fatalf("error should mention the disallowed attribute: %v", err)
	}
}

func TestPutItem_ExprAttrNamesValidated(t *testing.T) {
	t.Parallel()
	mock := &mockDynamo{}
	conn := newForTest(mock)
	action := conn.Actions()["dynamodb.put_item"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType: "dynamodb.put_item",
		Parameters: json.RawMessage(`{
			"region":"us-east-1",
			"table":"t1",
			"item":{"pk":"a"},
			"allowed_tables":["t1"],
			"allowed_write_attributes":["pk"],
			"condition_expression":"attribute_not_exists(#r)",
			"expression_attribute_names":{"#r":"restricted"}
		}`),
		Credentials: validCreds(),
	})
	if err == nil || !connectors.IsValidationError(err) {
		t.Fatalf("want validation error for disallowed expr attr name, got %v", err)
	}
}

func TestPutItem_ExprAttrNamesAllowed(t *testing.T) {
	t.Parallel()
	mock := &mockDynamo{}
	conn := newForTest(mock)
	action := conn.Actions()["dynamodb.put_item"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType: "dynamodb.put_item",
		Parameters: json.RawMessage(`{
			"region":"us-east-1",
			"table":"t1",
			"item":{"pk":"a","status":"active"},
			"allowed_tables":["t1"],
			"allowed_write_attributes":["pk","status"],
			"condition_expression":"attribute_not_exists(#s)",
			"expression_attribute_names":{"#s":"status"}
		}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
}

func TestQuery_SortKeyBetweenRejectedForNonBetweenCondition(t *testing.T) {
	t.Parallel()
	mock := &mockDynamo{}
	conn := newForTest(mock)
	action := conn.Actions()["dynamodb.query"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType: "dynamodb.query",
		Parameters: json.RawMessage(`{
			"region":"us-east-1",
			"table":"t1",
			"partition_key_name":"pk",
			"partition_key_value":"a",
			"sort_key_name":"sk",
			"sort_key_condition":"eq",
			"sort_key_value":"x",
			"sort_key_between":[1,9],
			"allowed_tables":["t1"]
		}`),
		Credentials: validCreds(),
	})
	if err == nil || !connectors.IsValidationError(err) {
		t.Fatalf("want validation error for sort_key_between with non-between condition, got %v", err)
	}
}

func TestIsValidAttrName_AcceptsHyphensAndDots(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name  string
		input string
		want  bool
	}{
		{"alphanumeric", "myAttr123", true},
		{"underscore", "my_attr", true},
		{"hyphen", "my-attr", true},
		{"dot", "my.attr", true},
		{"space", "my attr", true},
		{"empty", "", false},
		{"control_char", "my\x00attr", false},
		{"newline", "my\nattr", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := isValidAttrName(tc.input)
			if got != tc.want {
				t.Fatalf("isValidAttrName(%q) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}
