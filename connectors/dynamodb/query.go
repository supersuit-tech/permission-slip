package dynamodb

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamodbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

type queryAction struct {
	conn *DynamoDBConnector
}

type queryParams struct {
	Region                string                 `json:"region"`
	Table                 string                 `json:"table"`
	IndexName             *string                `json:"index_name"`
	PartitionKeyName      string                 `json:"partition_key_name"`
	PartitionKeyValue     json.RawMessage        `json:"partition_key_value"`
	SortKeyName           string                 `json:"sort_key_name"`
	SortKeyValue          json.RawMessage        `json:"sort_key_value"`
	SortKeyCondition      string                 `json:"sort_key_condition"`
	SortKeyBetween        []json.RawMessage      `json:"sort_key_between"`
	Projection            map[string]bool        `json:"projection"`
	Limit                 *int32                 `json:"limit"`
	ScanIndexForward      *bool                  `json:"scan_index_forward"`
	ExclusiveStartKey     map[string]interface{} `json:"exclusive_start_key"`
	AllowedTables         []string               `json:"allowed_tables"`
	AllowedReadAttributes []string               `json:"allowed_read_attributes"`
}

func (p *queryParams) validate() error {
	if p.Region == "" {
		return &connectors.ValidationError{Message: "missing required parameter: region"}
	}
	if p.Table == "" {
		return &connectors.ValidationError{Message: "missing required parameter: table"}
	}
	if p.PartitionKeyName == "" {
		return &connectors.ValidationError{Message: "missing required parameter: partition_key_name"}
	}
	if len(p.PartitionKeyValue) == 0 {
		return &connectors.ValidationError{Message: "missing required parameter: partition_key_value"}
	}
	if !isValidAttrName(p.PartitionKeyName) {
		return &connectors.ValidationError{Message: "invalid partition_key_name"}
	}
	if err := validateAllowedTables(p.Table, p.AllowedTables); err != nil {
		return err
	}
	if len(p.AllowedReadAttributes) > 0 {
		if err := validateAttrAllowlist(p.AllowedReadAttributes); err != nil {
			return err
		}
	}
	if err := validateProjectionSubset(p.Projection, allowedReadSet(p.AllowedReadAttributes)); err != nil {
		return err
	}

	hasSort := p.SortKeyName != "" || p.SortKeyCondition != "" || len(p.SortKeyValue) > 0 || len(p.SortKeyBetween) > 0
	if hasSort {
		if p.SortKeyName == "" {
			return &connectors.ValidationError{Message: "sort_key_name is required when using a sort key condition"}
		}
		if !isValidAttrName(p.SortKeyName) {
			return &connectors.ValidationError{Message: "invalid sort_key_name"}
		}
		if p.SortKeyCondition == "" {
			return &connectors.ValidationError{Message: "sort_key_condition is required when sort_key_name is set"}
		}
		switch strings.ToLower(p.SortKeyCondition) {
		case "between":
			if len(p.SortKeyBetween) != 2 {
				return &connectors.ValidationError{Message: "sort_key_between must contain exactly two values when sort_key_condition is between"}
			}
		case "eq", "lt", "lte", "gt", "gte", "begins_with":
			if len(p.SortKeyValue) == 0 {
				return &connectors.ValidationError{Message: "sort_key_value is required for this sort_key_condition"}
			}
		default:
			return &connectors.ValidationError{Message: fmt.Sprintf("invalid sort_key_condition: %q", p.SortKeyCondition)}
		}
	}

	limit := defaultQueryLimit
	if p.Limit != nil {
		limit = int(*p.Limit)
	}
	if limit < 1 || limit > maxQueryLimit {
		return &connectors.ValidationError{Message: fmt.Sprintf("limit must be between 1 and %d", maxQueryLimit)}
	}

	if p.IndexName != nil && *p.IndexName != "" {
		if !isValidTableName(*p.IndexName) {
			return &connectors.ValidationError{Message: "invalid index_name"}
		}
	}
	return nil
}

func (a *queryAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params queryParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	var pkVal interface{}
	if err := json.Unmarshal(params.PartitionKeyValue, &pkVal); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid partition_key_value: %v", err)}
	}
	pkAv, err := attributevalue.Marshal(pkVal)
	if err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid partition_key_value: %v", err)}
	}

	exprNames := map[string]string{
		"#pk": params.PartitionKeyName,
	}
	exprVals := map[string]dynamodbtypes.AttributeValue{
		":pk": pkAv,
	}
	keyCond := "#pk = :pk"

	if params.SortKeyName != "" {
		exprNames["#sk"] = params.SortKeyName
		switch strings.ToLower(params.SortKeyCondition) {
		case "eq":
			var skVal interface{}
			if err := json.Unmarshal(params.SortKeyValue, &skVal); err != nil {
				return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid sort_key_value: %v", err)}
			}
			skAv, err := attributevalue.Marshal(skVal)
			if err != nil {
				return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid sort_key_value: %v", err)}
			}
			exprVals[":sk"] = skAv
			keyCond += " AND #sk = :sk"
		case "lt":
			var skVal interface{}
			if err := json.Unmarshal(params.SortKeyValue, &skVal); err != nil {
				return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid sort_key_value: %v", err)}
			}
			skAv, err := attributevalue.Marshal(skVal)
			if err != nil {
				return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid sort_key_value: %v", err)}
			}
			exprVals[":sk"] = skAv
			keyCond += " AND #sk < :sk"
		case "lte":
			var skVal interface{}
			if err := json.Unmarshal(params.SortKeyValue, &skVal); err != nil {
				return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid sort_key_value: %v", err)}
			}
			skAv, err := attributevalue.Marshal(skVal)
			if err != nil {
				return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid sort_key_value: %v", err)}
			}
			exprVals[":sk"] = skAv
			keyCond += " AND #sk <= :sk"
		case "gt":
			var skVal interface{}
			if err := json.Unmarshal(params.SortKeyValue, &skVal); err != nil {
				return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid sort_key_value: %v", err)}
			}
			skAv, err := attributevalue.Marshal(skVal)
			if err != nil {
				return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid sort_key_value: %v", err)}
			}
			exprVals[":sk"] = skAv
			keyCond += " AND #sk > :sk"
		case "gte":
			var skVal interface{}
			if err := json.Unmarshal(params.SortKeyValue, &skVal); err != nil {
				return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid sort_key_value: %v", err)}
			}
			skAv, err := attributevalue.Marshal(skVal)
			if err != nil {
				return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid sort_key_value: %v", err)}
			}
			exprVals[":sk"] = skAv
			keyCond += " AND #sk >= :sk"
		case "begins_with":
			var skVal interface{}
			if err := json.Unmarshal(params.SortKeyValue, &skVal); err != nil {
				return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid sort_key_value: %v", err)}
			}
			skAv, err := attributevalue.Marshal(skVal)
			if err != nil {
				return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid sort_key_value: %v", err)}
			}
			exprVals[":sk"] = skAv
			keyCond += " AND begins_with(#sk, :sk)"
		case "between":
			var lo, hi interface{}
			if err := json.Unmarshal(params.SortKeyBetween[0], &lo); err != nil {
				return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid sort_key_between[0]: %v", err)}
			}
			if err := json.Unmarshal(params.SortKeyBetween[1], &hi); err != nil {
				return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid sort_key_between[1]: %v", err)}
			}
			loAv, err := attributevalue.Marshal(lo)
			if err != nil {
				return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid sort_key_between[0]: %v", err)}
			}
			hiAv, err := attributevalue.Marshal(hi)
			if err != nil {
				return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid sort_key_between[1]: %v", err)}
			}
			exprVals[":sklo"] = loAv
			exprVals[":skhi"] = hiAv
			keyCond += " AND #sk BETWEEN :sklo AND :skhi"
		}
	}

	limit := int32(defaultQueryLimit)
	if params.Limit != nil {
		limit = *params.Limit
	}

	ctx, cancel := a.conn.withTimeout(ctx)
	defer cancel()

	client, err := a.conn.newClient(ctx, params.Region, req.Credentials)
	if err != nil {
		return nil, err
	}

	in := &dynamodb.QueryInput{
		TableName:                 aws.String(params.Table),
		KeyConditionExpression:    aws.String(keyCond),
		ExpressionAttributeNames:  exprNames,
		ExpressionAttributeValues: exprVals,
		Limit:                     aws.Int32(limit),
		ScanIndexForward:          params.ScanIndexForward,
	}
	if params.IndexName != nil && *params.IndexName != "" {
		in.IndexName = params.IndexName
	}
	if len(params.Projection) > 0 {
		var names []string
		for k, v := range params.Projection {
			if !v {
				return nil, &connectors.ValidationError{Message: "projection values must be true only"}
			}
			names = append(names, k)
		}
		// Avoid collisions with #pk / #sk — use #p0, #p1, ...
		projPlaceholders, projNames := projectionPlaceholders(names)
		for k, v := range projNames {
			if _, exists := in.ExpressionAttributeNames[k]; exists {
				return nil, &connectors.ValidationError{Message: fmt.Sprintf("projection attribute %q conflicts with reserved key condition name", v)}
			}
			in.ExpressionAttributeNames[k] = v
		}
		in.ProjectionExpression = aws.String(strings.Join(projPlaceholders, ", "))
	}
	if len(params.ExclusiveStartKey) > 0 {
		esk, err := attributevalue.MarshalMap(params.ExclusiveStartKey)
		if err != nil {
			return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid exclusive_start_key: %v", err)}
		}
		in.ExclusiveStartKey = esk
	}

	out, err := client.Query(ctx, in)
	if err != nil {
		return nil, mapDynamoError(err)
	}

	allowed := allowedReadSet(params.AllowedReadAttributes)
	items := make([]map[string]interface{}, 0, len(out.Items))
	for _, it := range out.Items {
		if allowed != nil {
			it = filterItemAttrs(it, allowed)
		}
		var row map[string]interface{}
		if err := attributevalue.UnmarshalMap(it, &row); err != nil {
			return nil, &connectors.ExternalError{Message: fmt.Sprintf("unmarshaling query item: %v", err)}
		}
		items = append(items, row)
	}

	resp := map[string]interface{}{
		"items":         items,
		"count":         out.Count,
		"scanned_count": out.ScannedCount,
		"has_more":      len(out.LastEvaluatedKey) > 0,
	}
	if len(out.LastEvaluatedKey) > 0 {
		var lk map[string]interface{}
		if err := attributevalue.UnmarshalMap(out.LastEvaluatedKey, &lk); err != nil {
			return nil, &connectors.ExternalError{Message: fmt.Sprintf("unmarshaling LastEvaluatedKey: %v", err)}
		}
		resp["last_evaluated_key"] = lk
	} else {
		resp["last_evaluated_key"] = nil
	}

	return connectors.JSONResult(resp)
}

func projectionPlaceholders(attrNames []string) ([]string, map[string]string) {
	placeholders := make([]string, len(attrNames))
	names := make(map[string]string, len(attrNames))
	for i, n := range attrNames {
		ph := "#p" + strconv.Itoa(i)
		placeholders[i] = ph
		names[ph] = n
	}
	return placeholders, names
}
