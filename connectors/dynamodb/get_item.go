package dynamodb

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"

	"github.com/supersuit-tech/permission-slip/connectors"
)

type getItemAction struct {
	conn *DynamoDBConnector
}

type getItemParams struct {
	Region                string                 `json:"region"`
	Table                 string                 `json:"table"`
	Key                   map[string]interface{} `json:"key"`
	ConsistentRead        *bool                  `json:"consistent_read"`
	Projection            map[string]bool        `json:"projection"`
	AllowedTables         []string               `json:"allowed_tables"`
	AllowedReadAttributes []string               `json:"allowed_read_attributes"`
}

func (p *getItemParams) validate() error {
	if p.Region == "" {
		return &connectors.ValidationError{Message: "missing required parameter: region"}
	}
	if p.Table == "" {
		return &connectors.ValidationError{Message: "missing required parameter: table"}
	}
	if len(p.Key) == 0 {
		return &connectors.ValidationError{Message: "missing required parameter: key"}
	}
	if err := validateAllowedTables(p.Table, p.AllowedTables); err != nil {
		return err
	}
	if len(p.AllowedReadAttributes) > 0 {
		if err := validateAttrAllowlist(p.AllowedReadAttributes); err != nil {
			return err
		}
	}
	allowed := buildAllowedSet(p.AllowedReadAttributes)
	return validateProjectionSubset(p.Projection, allowed)
}

func (a *getItemAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params getItemParams
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

	ctx, cancel := a.conn.withTimeout(ctx)
	defer cancel()

	client, err := a.conn.newClient(ctx, params.Region, req.Credentials)
	if err != nil {
		return nil, err
	}

	in := &dynamodb.GetItemInput{
		TableName:      aws.String(params.Table),
		Key:            keyAv,
		ConsistentRead: params.ConsistentRead,
	}
	if len(params.Projection) > 0 {
		var names []string
		for k, v := range params.Projection {
			if !v {
				return nil, &connectors.ValidationError{Message: "projection values must be true only"}
			}
			names = append(names, k)
		}
		in.ProjectionExpression = aws.String(joinExprNames(names))
		in.ExpressionAttributeNames = exprNamePlaceholders(names)
	}

	out, err := client.GetItem(ctx, in)
	if err != nil {
		return nil, mapDynamoError(err)
	}
	if out.Item == nil {
		return connectors.JSONResult(map[string]interface{}{"item": nil, "found": false})
	}

	item := out.Item
	if allowed := buildAllowedSet(params.AllowedReadAttributes); allowed != nil {
		item = filterItemAttrs(item, allowed)
	}

	var itemJSON map[string]interface{}
	if err := attributevalue.UnmarshalMap(item, &itemJSON); err != nil {
		return nil, &connectors.ExternalError{Message: fmt.Sprintf("unmarshaling DynamoDB item: %v", err)}
	}
	return connectors.JSONResult(map[string]interface{}{
		"item":  itemJSON,
		"found": true,
	})
}

func joinExprNames(names []string) string {
	if len(names) == 0 {
		return ""
	}
	s := "#n0"
	for i := 1; i < len(names); i++ {
		s += ", #n" + strconv.Itoa(i)
	}
	return s
}

func exprNamePlaceholders(names []string) map[string]string {
	m := make(map[string]string, len(names))
	for i, n := range names {
		m["#n"+strconv.Itoa(i)] = n
	}
	return m
}
