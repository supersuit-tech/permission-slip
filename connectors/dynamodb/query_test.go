package dynamodb

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestQuery_BuildsKeyCondition(t *testing.T) {
	t.Parallel()
	mock := &mockDynamo{
		queryOut: &dynamodb.QueryOutput{
			Items: []map[string]types.AttributeValue{
				{"pk": &types.AttributeValueMemberS{Value: "a"}},
			},
			Count: 1,
		},
	}
	conn := newForTest(mock)
	action := conn.Actions()["dynamodb.query"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType: "dynamodb.query",
		Parameters: json.RawMessage(`{
			"region":"eu-west-1",
			"table":"t1",
			"partition_key_name":"pk",
			"partition_key_value":"part-1",
			"sort_key_name":"sk",
			"sort_key_condition":"begins_with",
			"sort_key_value":"pre",
			"allowed_tables":["t1"],
			"limit":5
		}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if mock.lastQueryIn == nil {
		t.Fatal("no query input")
	}
	cond := *mock.lastQueryIn.KeyConditionExpression
	if !strings.Contains(cond, "#pk = :pk") || !strings.Contains(cond, "begins_with(#sk, :sk)") {
		t.Fatalf("unexpected condition: %s", cond)
	}
	if *mock.lastQueryIn.Limit != 5 {
		t.Fatalf("limit = %d", *mock.lastQueryIn.Limit)
	}
}

func TestQuery_BetweenSortKey(t *testing.T) {
	t.Parallel()
	mock := &mockDynamo{queryOut: &dynamodb.QueryOutput{Items: []map[string]types.AttributeValue{}, Count: 0}}
	conn := newForTest(mock)
	action := conn.Actions()["dynamodb.query"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType: "dynamodb.query",
		Parameters: json.RawMessage(`{
			"region":"us-east-1",
			"table":"t1",
			"partition_key_name":"pk",
			"partition_key_value":10,
			"sort_key_name":"sk",
			"sort_key_condition":"between",
			"sort_key_between":[1,9],
			"allowed_tables":["t1"]
		}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatal(err)
	}
	cond := *mock.lastQueryIn.KeyConditionExpression
	if !strings.Contains(cond, "BETWEEN :sklo AND :skhi") {
		t.Fatalf("want between in condition, got %q", cond)
	}
}
