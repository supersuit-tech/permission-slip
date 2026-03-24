// Package snowflake implements the Snowflake data warehouse connector. It runs
// read-only SELECT queries with row limits, statement timeouts, and optional
// key-pair auth via private_key_pem in vault credentials.
package snowflake

import (
	"context"
	_ "embed"
	"encoding/json"
	"strings"
	"time"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

const (
	defaultStatementTimeout = 30 * time.Second
	defaultMaxRows          = 1000
)

type SnowflakeConnector struct {
	statementTimeout time.Duration
	maxRows          int
}

func New() *SnowflakeConnector {
	return &SnowflakeConnector{
		statementTimeout: defaultStatementTimeout,
		maxRows:          defaultMaxRows,
	}
}

func (c *SnowflakeConnector) ID() string { return "snowflake" }

//go:embed logo.svg
var logoSVG string

func (c *SnowflakeConnector) Manifest() *connectors.ConnectorManifest {
	return &connectors.ConnectorManifest{
		ID:          "snowflake",
		Name:        "Snowflake",
		Description: "Run read-only SQL queries against Snowflake with parameterized placeholders (?), row caps, and statement timeouts. Password auth uses connection_string only; key-pair auth uses private_key_pem with the RSA key on the driver config (not embedded in the DSN).",
		LogoSVG:     logoSVG,
		Actions: []connectors.ManifestAction{
			{
				ActionType:  "snowflake.query",
				Name:        "Run SQL Query",
				Description: "Execute a read-only SELECT (or WITH … SELECT) with ? placeholders. Results are capped at max_rows.",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["sql"],
					"properties": {
						"sql": {
							"type": "string",
							"description": "Parameterized SELECT query. Use ? for placeholders. Example: SELECT name FROM users WHERE id = ?"
						},
						"params": {
							"type": "array",
							"items": {},
							"description": "Positional parameters for ? placeholders"
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
							"description": "Statement timeout in seconds (default: 30)"
						}
					}
				}`)),
			},
		},
		RequiredCredentials: []connectors.ManifestCredential{
			{
				Service:         "snowflake",
				AuthType:        "custom",
				InstructionsURL: "https://docs.snowflake.com/en/developer-guide/go/go-driver",
			},
		},
		Templates: []connectors.ManifestTemplate{
			{
				ID:          "tpl_snowflake_query_readonly",
				ActionType:  "snowflake.query",
				Name:        "Run read-only queries",
				Description: "Agent can run SELECT queries; SQL and params are agent-controlled.",
				Parameters:  json.RawMessage(`{"sql":"*","params":"*"}`),
			},
		},
	}
}

func (c *SnowflakeConnector) Actions() map[string]connectors.Action {
	return map[string]connectors.Action{
		"snowflake.query": &queryAction{conn: c},
	}
}

func (c *SnowflakeConnector) ValidateCredentials(_ context.Context, creds connectors.Credentials) error {
	connStr, _ := creds.Get("connection_string")
	pkPEM, _ := creds.Get("private_key_pem")
	connStr = strings.TrimSpace(connStr)
	pkPEM = strings.TrimSpace(pkPEM)
	if connStr == "" {
		return &connectors.ValidationError{Message: "missing required credential: connection_string"}
	}
	if !strings.Contains(connStr, "@") {
		return &connectors.ValidationError{Message: "connection_string must be a Snowflake DSN (user@account/...)"}
	}
	if pkPEM != "" {
		if _, err := parseRSAPrivateKeyPEM(pkPEM); err != nil {
			return err
		}
	}
	_, err := buildSnowflakeConfig(connStr, pkPEM)
	return err
}

func (c *SnowflakeConnector) resolveTimeout(paramTimeout int) time.Duration {
	if paramTimeout > 0 {
		t := time.Duration(paramTimeout) * time.Second
		if t > 5*time.Minute {
			t = 5 * time.Minute
		}
		return t
	}
	return c.statementTimeout
}
