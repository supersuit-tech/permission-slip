package dynamodb

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/supersuit-tech/permission-slip/connectors"
)

type putItemAction struct {
	conn *DynamoDBConnector
}

type putItemParams struct {
	Region                    string                 `json:"region"`
	Table                     string                 `json:"table"`
	Item                      map[string]interface{} `json:"item"`
	ConditionExpression       *string                `json:"condition_expression"`
	ExpressionAttributeNames  map[string]string      `json:"expression_attribute_names"`
	ExpressionAttributeValues map[string]interface{} `json:"expression_attribute_values"`
	AllowedTables             []string               `json:"allowed_tables"`
	AllowedWriteAttributes    []string               `json:"allowed_write_attributes"`
}

func (p *putItemParams) validate() error {
	if p.Region == "" {
		return &connectors.ValidationError{Message: "missing required parameter: region"}
	}
	if p.Table == "" {
		return &connectors.ValidationError{Message: "missing required parameter: table"}
	}
	if len(p.Item) == 0 {
		return &connectors.ValidationError{Message: "missing required parameter: item"}
	}
	if err := validateAllowedTables(p.Table, p.AllowedTables); err != nil {
		return err
	}
	if len(p.AllowedWriteAttributes) > 0 {
		if err := validateAttrAllowlist(p.AllowedWriteAttributes); err != nil {
			return err
		}
	}
	if err := validateExprAttrNames(p.ExpressionAttributeNames, buildAllowedSet(p.AllowedWriteAttributes)); err != nil {
		return err
	}
	return nil
}

func (a *putItemAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params putItemParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	itemAv, err := attributevalue.MarshalMap(params.Item)
	if err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid item attribute values: %v", err)}
	}
	if err := validateItemWriteSubset(itemAv, buildAllowedSet(params.AllowedWriteAttributes)); err != nil {
		return nil, err
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

	in := &dynamodb.PutItemInput{
		TableName:                 aws.String(params.Table),
		Item:                      itemAv,
		ConditionExpression:       params.ConditionExpression,
		ExpressionAttributeNames:  cloneStringMap(params.ExpressionAttributeNames),
		ExpressionAttributeValues: exprVals,
	}

	_, err = client.PutItem(ctx, in)
	if err != nil {
		return nil, mapDynamoError(err)
	}
	return connectors.JSONResult(map[string]interface{}{"ok": true})
}
