package dynamodb

import (
	"encoding/json"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestPutItem_WriteAllowlist(t *testing.T) {
	t.Parallel()
	mock := &mockDynamo{}
	conn := newForTest(mock)
	action := conn.Actions()["dynamodb.put_item"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType: "dynamodb.put_item",
		Parameters: json.RawMessage(`{
			"region":"us-east-1",
			"table":"t1",
			"item":{"pk":"a","extra":"b"},
			"allowed_tables":["t1"],
			"allowed_write_attributes":["pk"]
		}`),
		Credentials: validCreds(),
	})
	if err == nil || !connectors.IsValidationError(err) {
		t.Fatalf("want validation error, got %v", err)
	}
}

func TestPutItem_Success(t *testing.T) {
	t.Parallel()
	mock := &mockDynamo{}
	conn := newForTest(mock)
	action := conn.Actions()["dynamodb.put_item"]

	res, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType: "dynamodb.put_item",
		Parameters: json.RawMessage(`{
			"region":"us-west-2",
			"table":"t1",
			"item":{"pk":"a","v":1},
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
	if payload["ok"] != true {
		t.Fatalf("want ok true: %#v", payload)
	}
	if mock.lastPutItemIn == nil || *mock.lastPutItemIn.TableName != "t1" {
		t.Fatal("put not invoked as expected")
	}
}
