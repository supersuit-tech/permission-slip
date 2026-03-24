package dynamodb

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

type deleteItemAction struct {
	conn *DynamoDBConnector
}

type deleteItemParams struct {
	Region                    string                 `json:"region"`
	Table                     string                 `json:"table"`
	Key                       map[string]interface{} `json:"key"`
	ConditionExpression       *string                `json:"condition_expression"`
	ExpressionAttributeNames  map[string]string      `json:"expression_attribute_names"`
	ExpressionAttributeValues map[string]interface{} `json:"expression_attribute_values"`
	AllowedTables             []string               `json:"allowed_tables"`
}

func (p *deleteItemParams) validate() error {
	if p.Region == "" {
		return &connectors.ValidationError{Message: "missing required parameter: region"}
	}
	if p.Table == "" {
		return &connectors.ValidationError{Message: "missing required parameter: table"}
	}
	if len(p.Key) == 0 {
		return &connectors.ValidationError{Message: "missing required parameter: key"}
	}
	return validateAllowedTables(p.Table, p.AllowedTables)
}

func (a *deleteItemAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params deleteItemParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	keyAv, err := attributevalue.MarshalMap(params.Key)
	if err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid key attribute values: %v", err)}
	}

	var exprVals map[string]dynamodbtypes.AttributeValue
	if len(params.ExpressionAttributeValues) > 0 {
		exprVals, err = attributevalue.MarshalMap(params.ExpressionAttributeValues)
		if err != nil {
			return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid expression_attribute_values: %v", err)}
		}
	}

	ctx, cancel := a.conn.withTimeout(ctx)
	defer cancel()

	client, err := a.conn.newClient(ctx, params.Region, req.Credentials)
	if err != nil {
		return nil, err
	}

	in := &dynamodb.DeleteItemInput{
		TableName:                 aws.String(params.Table),
		Key:                       keyAv,
		ConditionExpression:       params.ConditionExpression,
		ExpressionAttributeNames:  cloneStringMap(params.ExpressionAttributeNames),
		ExpressionAttributeValues: exprVals,
		ReturnValues:              dynamodbtypes.ReturnValueAllOld,
	}

	out, err := client.DeleteItem(ctx, in)
	if err != nil {
		return nil, mapDynamoError(err)
	}

	existed := len(out.Attributes) > 0
	resp := map[string]interface{}{"ok": true, "existed": existed}
	if !existed {
		resp["previous_item"] = nil
		return connectors.JSONResult(resp)
	}
	var prev map[string]interface{}
	if err := attributevalue.UnmarshalMap(out.Attributes, &prev); err != nil {
		return nil, &connectors.ExternalError{Message: fmt.Sprintf("unmarshaling deleted item: %v", err)}
	}
	resp["previous_item"] = prev
	return connectors.JSONResult(resp)
}
