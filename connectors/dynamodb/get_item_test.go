package dynamodb

import (
	"encoding/json"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestGetItem_TableAllowlist(t *testing.T) {
	t.Parallel()
	mock := &mockDynamo{}
	conn := newForTest(mock)
	action := conn.Actions()["dynamodb.get_item"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType: "dynamodb.get_item",
		Parameters: json.RawMessage(`{
			"region":"us-east-1",
			"table":"orders",
			"key":{"pk":"a"},
			"allowed_tables":["users"]
		}`),
		Credentials: validCreds(),
	})
	if err == nil || !connectors.IsValidationError(err) {
		t.Fatalf("want validation error, got %v", err)
	}
}

func TestGetItem_Success(t *testing.T) {
	t.Parallel()
	mock := &mockDynamo{
		getItemOut: &dynamodb.GetItemOutput{
			Item: map[string]types.AttributeValue{
				"pk":   &types.AttributeValueMemberS{Value: "k1"},
				"name": &types.AttributeValueMemberS{Value: "Ada"},
			},
		},
	}
	conn := newForTest(mock)
	action := conn.Actions()["dynamodb.get_item"]

	res, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType: "dynamodb.get_item",
		Parameters: json.RawMessage(`{
			"region":"us-east-1",
			"table":"t1",
			"key":{"pk":"k1"},
			"allowed_tables":["t1"]
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
	if !payload["found"].(bool) {
		t.Fatalf("want found true, got %#v", payload)
	}
	item := payload["item"].(map[string]interface{})
	if item["pk"] != "k1" || item["name"] != "Ada" {
		t.Fatalf("unexpected item: %#v", item)
	}
	if mock.lastGetItemIn == nil || *mock.lastGetItemIn.TableName != "t1" {
		t.Fatalf("unexpected input: %+v", mock.lastGetItemIn)
	}
}

func TestGetItem_ReadAllowlistFilters(t *testing.T) {
	t.Parallel()
	mock := &mockDynamo{
		getItemOut: &dynamodb.GetItemOutput{
			Item: map[string]types.AttributeValue{
				"pk":     &types.AttributeValueMemberS{Value: "k1"},
				"secret": &types.AttributeValueMemberS{Value: "x"},
			},
		},
	}
	conn := newForTest(mock)
	action := conn.Actions()["dynamodb.get_item"]

	res, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType: "dynamodb.get_item",
		Parameters: json.RawMessage(`{
			"region":"us-east-1",
			"table":"t1",
			"key":{"pk":"k1"},
			"allowed_tables":["t1"],
			"allowed_read_attributes":["pk"]
		}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatal(err)
	}
	var payload map[string]interface{}
	_ = json.Unmarshal(res.Data, &payload)
	item := payload["item"].(map[string]interface{})
	if _, has := item["secret"]; has {
		t.Fatalf("secret should be filtered out: %#v", item)
	}
}
