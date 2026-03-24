package dynamodb

import (
	"encoding/json"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestDeleteItem_NotFound(t *testing.T) {
	t.Parallel()
	mock := &mockDynamo{}
	conn := newForTest(mock)
	action := conn.Actions()["dynamodb.delete_item"]

	res, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType: "dynamodb.delete_item",
		Parameters: json.RawMessage(`{
			"region":"us-east-1",
			"table":"t1",
			"key":{"pk":"missing"},
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
	if payload["existed"].(bool) {
		t.Fatalf("want existed false: %#v", payload)
	}
}

func TestDeleteItem_WithPrevious(t *testing.T) {
	t.Parallel()
	mock := &mockDynamo{
		deleteItemOut: &dynamodb.DeleteItemOutput{
			Attributes: map[string]types.AttributeValue{
				"pk": &types.AttributeValueMemberS{Value: "gone"},
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
			"key":{"pk":"gone"},
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
	if !payload["existed"].(bool) {
		t.Fatalf("want existed true: %#v", payload)
	}
}
