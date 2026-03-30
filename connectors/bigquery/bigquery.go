// Package bigquery implements read-only BigQuery queries using a service
// account JSON key from vault. Queries support ? placeholders (mapped to @pN),
// row limits, job timeouts, and optional max_bytes_billed caps.
package bigquery

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/supersuit-tech/permission-slip/connectors"
)

const (
	defaultStatementTimeout = 30 * time.Second
	defaultMaxRows          = 1000
	defaultMaxBytesBilled   = int64(10 * 1024 * 1024 * 1024) // 10 GiB
)

type Connector struct {
	statementTimeout time.Duration
	maxRows          int
	maxBytesBilled   int64
}

func New() *Connector {
	return &Connector{
		statementTimeout: defaultStatementTimeout,
		maxRows:          defaultMaxRows,
		maxBytesBilled:   defaultMaxBytesBilled,
	}
}

func (c *Connector) ID() string { return "bigquery" }

//go:embed logo.svg
var logoSVG string

func (c *Connector) Manifest() *connectors.ConnectorManifest {
	return &connectors.ConnectorManifest{
		ID:          "bigquery",
		Name:        "BigQuery",
		Description: "Run read-only Standard SQL against Google BigQuery with ? placeholders (outside strings/comments), row caps, job timeouts, and optional max_bytes_billed. Rejects WITH … DML and other mutating statements. Use a least-privilege service account (e.g. bigquery.jobs.create + dataViewer) in production.",
		LogoSVG:     logoSVG,
		Actions: []connectors.ManifestAction{
			{
				ActionType:  "bigquery.query",
				Name:        "Run SQL Query",
				Description: "Execute a read-only SELECT (or WITH … SELECT). Use ? for positional parameters (bound as @p0, @p1, …).",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["sql"],
					"properties": {
						"sql": {
							"type": "string",
							"description": "Standard SQL SELECT. Use ? for each positional parameter."
						},
						"params": {
							"type": "array",
							"items": {},
							"description": "Values for each ? in order"
						},
						"max_rows": {
							"type": "integer",
							"minimum": 1,
							"maximum": 10000,
							"description": "Maximum rows to return (default: 1000)"
						},
						"timeout_seconds": {
							"type": "integer",
							"minimum": 1,
							"maximum": 300,
							"description": "Query job timeout in seconds (default: 30)"
						},
						"max_bytes_billed": {
							"type": "integer",
							"minimum": 1048576,
							"description": "Cap bytes billed for this query (omit to use connector default, typically 10 GiB)"
						},
						"default_dataset": {
							"type": "string",
							"description": "Default dataset for unqualified table names: project.dataset or dataset (uses key project_id)"
						},
						"query_job_location": {
							"type": "string",
							"description": "BigQuery job location (e.g. US, EU) if your data requires it"
						}
					}
				}`)),
			},
		},
		RequiredCredentials: []connectors.ManifestCredential{
			{
				Service:         "bigquery",
				AuthType:        "custom",
				InstructionsURL: "https://cloud.google.com/bigquery/docs/authentication/service-account-file",
			},
		},
		Templates: []connectors.ManifestTemplate{
			{
				ID:          "tpl_bigquery_query_readonly",
				ActionType:  "bigquery.query",
				Name:        "Run read-only queries",
				Description: "Agent can run SELECT queries; SQL and params are agent-controlled.",
				Parameters:  json.RawMessage(`{"sql":"*","params":"*"}`),
			},
		},
	}
}

func (c *Connector) Actions() map[string]connectors.Action {
	return map[string]connectors.Action{
		"bigquery.query": &queryAction{conn: c},
	}
}

func (c *Connector) ValidateCredentials(_ context.Context, creds connectors.Credentials) error {
	raw, ok := creds.Get("service_account_json")
	if !ok || strings.TrimSpace(raw) == "" {
		return &connectors.ValidationError{Message: "missing required credential: service_account_json"}
	}
	var meta struct {
		ProjectID string `json:"project_id"`
		Type      string `json:"type"`
	}
	if err := json.Unmarshal([]byte(raw), &meta); err != nil {
		return &connectors.ValidationError{Message: fmt.Sprintf("invalid service_account_json: %v", err)}
	}
	if meta.Type != "" && meta.Type != "service_account" {
		return &connectors.ValidationError{Message: "service_account_json must be a Google service account key"}
	}
	if strings.TrimSpace(meta.ProjectID) == "" {
		return &connectors.ValidationError{Message: "service_account_json must include project_id"}
	}
	return nil
}

func (c *Connector) resolveTimeout(paramTimeout int) time.Duration {
	if paramTimeout > 0 {
		t := time.Duration(paramTimeout) * time.Second
		if t > 5*time.Minute {
			t = 5 * time.Minute
		}
		return t
	}
	return c.statementTimeout
}
