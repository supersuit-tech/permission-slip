package firestore

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip/connectors"
)

type queryAction struct {
	conn *FirestoreConnector
}

type queryFilterJSON struct {
	Field string          `json:"field"`
	Op    string          `json:"op"`
	Value json.RawMessage `json:"value"`
}

type orderByJSON struct {
	Field     string `json:"field"`
	Direction string `json:"direction"`
}

type queryParams struct {
	CollectionPath      string            `json:"collection_path"`
	Filters             []queryFilterJSON `json:"filters"`
	OrderBy             []orderByJSON     `json:"order_by"`
	Limit               *int              `json:"limit"`
	AllowedPaths        []string          `json:"allowed_paths"`
	AllowedReadFields   []string          `json:"allowed_read_fields"`
}

const maxQueryFilters = 10
const maxOrderBy = 4

func (p *queryParams) validate() error {
	if err := validateCollectionPath(p.CollectionPath); err != nil {
		return err
	}
	if err := validateAllowedPaths(p.CollectionPath, p.AllowedPaths, "collection"); err != nil {
		return err
	}
	if len(p.Filters) > maxQueryFilters {
		return &connectors.ValidationError{Message: fmt.Sprintf("filters must not exceed %d entries", maxQueryFilters)}
	}
	for _, f := range p.Filters {
		if f.Field == "" {
			return &connectors.ValidationError{Message: "each filter must include field"}
		}
		if !isValidFieldName(f.Field) {
			return &connectors.ValidationError{Message: fmt.Sprintf("invalid filter field: %q", f.Field)}
		}
		if f.Op != "==" {
			return &connectors.ValidationError{Message: fmt.Sprintf("filter op must be ==, got %q", f.Op)}
		}
		if len(f.Value) == 0 {
			return &connectors.ValidationError{Message: "each filter must include value"}
		}
	}
	if len(p.OrderBy) > maxOrderBy {
		return &connectors.ValidationError{Message: fmt.Sprintf("order_by must not exceed %d entries", maxOrderBy)}
	}
	for _, o := range p.OrderBy {
		if o.Field == "" {
			return &connectors.ValidationError{Message: "order_by entries must include field"}
		}
		if !isValidFieldName(o.Field) {
			return &connectors.ValidationError{Message: fmt.Sprintf("invalid order_by field: %q", o.Field)}
		}
		dir := o.Direction
		if dir != "asc" && dir != "desc" {
			return &connectors.ValidationError{Message: fmt.Sprintf("order_by direction must be asc or desc, got %q", dir)}
		}
	}
	if p.Limit != nil {
		limit := *p.Limit
		if limit < 1 || limit > maxQueryLimit {
			return &connectors.ValidationError{Message: fmt.Sprintf("limit must be between 1 and %d", maxQueryLimit)}
		}
	}
	if len(p.AllowedReadFields) > 0 {
		return validateFieldAllowlist(p.AllowedReadFields)
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
	limit := defaultQueryLimit
	if params.Limit != nil {
		limit = *params.Limit
	}

	filters := make([]queryFilter, 0, len(params.Filters))
	for _, f := range params.Filters {
		var val interface{}
		if err := json.Unmarshal(f.Value, &val); err != nil {
			return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid filter value for field %q: %v", f.Field, err)}
		}
		filters = append(filters, queryFilter{Field: f.Field, Op: f.Op, Value: val})
	}
	order := make([]orderClause, 0, len(params.OrderBy))
	for _, o := range params.OrderBy {
		order = append(order, orderClause{Field: o.Field, Direction: o.Direction})
	}

	ctx, cancel := a.conn.withTimeout(ctx)
	defer cancel()

	runner, err := a.conn.openRunner(ctx, req.Credentials)
	if err != nil {
		return nil, err
	}
	defer func() { _ = runner.close() }()

	docs, err := runner.queryCollection(ctx, params.CollectionPath, filters, order, limit)
	if err != nil {
		return nil, err
	}
	if docs == nil {
		docs = []map[string]interface{}{}
	}
	allowed := buildFieldSet(params.AllowedReadFields)
	if allowed != nil {
		for i := range docs {
			d, ok := docs[i]["data"].(map[string]interface{})
			if ok {
				docs[i]["data"] = filterMapKeys(d, allowed)
			}
		}
	}
	return connectors.JSONResult(map[string]interface{}{
		"documents": docs,
		"count":     len(docs),
	})
}
