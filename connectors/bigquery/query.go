package bigquery

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	bq "cloud.google.com/go/bigquery"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"

	"github.com/supersuit-tech/permission-slip-web/connectors"
	"github.com/supersuit-tech/permission-slip-web/pkg/sqldb"
)

type queryAction struct {
	conn *Connector
}

type queryParams struct {
	SQL              string        `json:"sql"`
	Params           []interface{} `json:"params"`
	MaxRows          int           `json:"max_rows"`
	TimeoutSeconds   int           `json:"timeout_seconds"`
	MaxBytesBilled   int64         `json:"max_bytes_billed"`
	DefaultDataset   string        `json:"default_dataset"`
	QueryJobLocation string        `json:"query_job_location"`
}

func (p *queryParams) validate() error {
	if err := validateReadOnlyBigQuerySQL(p.SQL); err != nil {
		return err
	}
	if p.MaxRows < 0 {
		return &connectors.ValidationError{Message: "max_rows must be positive"}
	}
	if p.MaxRows > 10000 {
		return &connectors.ValidationError{Message: "max_rows cannot exceed 10000"}
	}
	if p.MaxBytesBilled < 0 {
		return &connectors.ValidationError{Message: "max_bytes_billed must be non-negative"}
	}
	if p.MaxBytesBilled > 0 && p.MaxBytesBilled < 1048576 {
		return &connectors.ValidationError{Message: "max_bytes_billed must be at least 1048576 (1 MiB) when set"}
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

	sqlText, err := translatePlaceholders(params.SQL, len(params.Params))
	if err != nil {
		return nil, err
	}

	saJSON, projectID, err := serviceAccountAndProject(req)
	if err != nil {
		return nil, err
	}

	timeout := a.conn.resolveTimeout(params.TimeoutSeconds)
	maxRows := a.conn.maxRows
	if params.MaxRows > 0 {
		maxRows = params.MaxRows
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	client, err := bq.NewClient(ctx, projectID, option.WithCredentialsJSON(saJSON))
	if err != nil {
		return nil, mapBQError(err, "creating client")
	}
	defer client.Close()

	q := client.Query(sqlText)
	q.Parameters = buildNamedParameters(params.Params)
	defProj, defDS := resolveDefaultDataset(projectID, params.DefaultDataset, req)
	q.DefaultProjectID = defProj
	q.DefaultDatasetID = defDS
	if loc := strings.TrimSpace(params.QueryJobLocation); loc != "" {
		q.JobIDConfig.Location = loc
	}
	maxBytes := a.conn.maxBytesBilled
	if params.MaxBytesBilled > 0 {
		maxBytes = params.MaxBytesBilled
	}
	if maxBytes > 0 {
		q.MaxBytesBilled = maxBytes
	}
	q.JobTimeout = timeout

	it, err := q.Read(ctx)
	if err != nil {
		return nil, mapBQError(err, "running query")
	}

	var results []map[string]interface{}
	for len(results) < maxRows+1 {
		var row []bq.Value
		err := it.Next(&row)
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, mapBQError(err, "reading rows")
		}
		results = append(results, rowToMap(it.Schema, row))
	}

	columns := schemaColumnNames(it.Schema)
	if len(columns) == 0 && len(results) > 0 {
		columns = sortedMapKeys(results[0])
	}

	results, truncated := sqldb.DetectTruncation(results, maxRows)

	return connectors.JSONResult(map[string]interface{}{
		"columns":   columns,
		"rows":      results,
		"row_count": len(results),
		"truncated": truncated,
	})
}

func serviceAccountAndProject(req connectors.ActionRequest) ([]byte, string, error) {
	raw, ok := req.Credentials.Get("service_account_json")
	if !ok || strings.TrimSpace(raw) == "" {
		return nil, "", &connectors.ValidationError{Message: "missing credential: service_account_json"}
	}
	var meta struct {
		ProjectID string `json:"project_id"`
		Type      string `json:"type"`
	}
	if err := json.Unmarshal([]byte(raw), &meta); err != nil {
		return nil, "", &connectors.ValidationError{Message: fmt.Sprintf("invalid service_account_json: %v", err)}
	}
	if meta.Type != "" && meta.Type != "service_account" {
		return nil, "", &connectors.ValidationError{Message: "service_account_json must be a Google service account key (type service_account)"}
	}
	if strings.TrimSpace(meta.ProjectID) == "" {
		return nil, "", &connectors.ValidationError{Message: "service_account_json must include project_id"}
	}
	return []byte(raw), meta.ProjectID, nil
}

func resolveDefaultDataset(fallbackProject, param string, req connectors.ActionRequest) (projectID, datasetID string) {
	projectID = fallbackProject
	ds := strings.TrimSpace(param)
	if ds == "" {
		ds, _ = req.Credentials.Get("default_dataset")
	}
	ds = strings.TrimSpace(ds)
	if ds == "" {
		return projectID, ""
	}
	if strings.Contains(ds, ".") {
		parts := strings.SplitN(ds, ".", 2)
		return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
	}
	return projectID, ds
}

func buildNamedParameters(params []interface{}) []bq.QueryParameter {
	if len(params) == 0 {
		return nil
	}
	out := make([]bq.QueryParameter, 0, len(params))
	for i, p := range params {
		out = append(out, bq.QueryParameter{
			Name:  fmt.Sprintf("p%d", i),
			Value: coerceParamValue(p),
		})
	}
	return out
}

func coerceParamValue(v interface{}) interface{} {
	switch x := v.(type) {
	case float64:
		if !math.IsNaN(x) && !math.IsInf(x, 0) && x == math.Trunc(x) && x >= math.MinInt64 && x <= math.MaxInt64 {
			return int64(x)
		}
		return x
	case json.Number:
		if i, err := x.Int64(); err == nil {
			return i
		}
		f, _ := x.Float64()
		return f
	case []interface{}:
		out := make([]interface{}, len(x))
		for i := range x {
			out[i] = coerceParamValue(x[i])
		}
		return out
	case map[string]interface{}:
		out := make(map[string]interface{}, len(x))
		for k, val := range x {
			out[k] = coerceParamValue(val)
		}
		return out
	default:
		return v
	}
}

func schemaColumnNames(schema bq.Schema) []string {
	if len(schema) == 0 {
		return nil
	}
	names := make([]string, len(schema))
	for i, f := range schema {
		names[i] = f.Name
	}
	return names
}

func sortedMapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func rowToMap(schema bq.Schema, row []bq.Value) map[string]interface{} {
	m := make(map[string]interface{}, len(row))
	if len(schema) > 0 {
		for i, f := range schema {
			if i >= len(row) {
				break
			}
			m[f.Name] = cellToJSONable(row[i])
		}
		return m
	}
	for i := range row {
		m[fmt.Sprintf("col_%d", i)] = cellToJSONable(row[i])
	}
	return m
}

func cellToJSONable(v interface{}) interface{} {
	if v == nil {
		return nil
	}
	switch x := v.(type) {
	case []byte:
		return string(x)
	case time.Time:
		return x.Format(time.RFC3339Nano)
	case []bq.Value:
		out := make([]interface{}, len(x))
		for i := range x {
			out[i] = cellToJSONable(x[i])
		}
		return out
	case map[string]bq.Value:
		out := make(map[string]interface{}, len(x))
		for k, val := range x {
			out[k] = cellToJSONable(val)
		}
		return out
	case []interface{}:
		out := make([]interface{}, len(x))
		for i := range x {
			out[i] = cellToJSONable(x[i])
		}
		return out
	case map[string]interface{}:
		out := make(map[string]interface{}, len(x))
		for k, val := range x {
			out[k] = cellToJSONable(val)
		}
		return out
	default:
		return x
	}
}
