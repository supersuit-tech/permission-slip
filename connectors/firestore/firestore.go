// Package firestore implements the Google Cloud Firestore connector for the Permission Slip
// connector execution layer. It uses the official Firestore Go client with a service account
// JSON credential stored in the vault.
//
// Security model:
//   - Path allowlists constrain which documents and collections an agent may access
//   - Optional read/write field allowlists limit returned and written map keys
//   - Query limit caps documents read per request
package firestore

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"

	"github.com/supersuit-tech/permission-slip/connectors"
)

const (
	defaultTimeout    = 30 * time.Second
	defaultQueryLimit = 50
	maxQueryLimit     = 500
	maxAllowedPaths   = 64
	maxFieldAllowlist = 100
)

// fsRunner is the Firestore operations used by actions (implemented by *realRunner, mocked in tests).
type fsRunner interface {
	getDocument(ctx context.Context, path string) (map[string]interface{}, error)
	setDocument(ctx context.Context, path string, data map[string]interface{}, merge bool) error
	updateDocument(ctx context.Context, path string, data map[string]interface{}) error
	deleteDocument(ctx context.Context, path string) error
	queryCollection(ctx context.Context, collectionPath string, filters []queryFilter, order []orderClause, limit int) ([]map[string]interface{}, error)
	close() error
}

// FirestoreConnector owns shared configuration for Firestore actions.
type FirestoreConnector struct {
	timeout   time.Duration
	newRunner func(ctx context.Context, projectID string, credJSON []byte, emulatorHost string) (fsRunner, error)
}

// New creates a FirestoreConnector with production defaults.
func New() *FirestoreConnector {
	c := &FirestoreConnector{timeout: defaultTimeout}
	c.newRunner = c.buildRunner
	return c
}

func newForTest(r fsRunner) *FirestoreConnector {
	return &FirestoreConnector{
		timeout: defaultTimeout,
		newRunner: func(context.Context, string, []byte, string) (fsRunner, error) {
			return r, nil
		},
	}
}

// ID returns "firestore", matching connectors.id in the database.
func (c *FirestoreConnector) ID() string { return "firestore" }

//go:embed logo.svg
var logoSVG string

// Manifest returns connector metadata for DB seeding.
func (c *FirestoreConnector) Manifest() *connectors.ConnectorManifest {
	return &connectors.ConnectorManifest{
		ID:          "firestore",
		Name:        "Google Cloud Firestore",
		Description: "Firestore document reads, writes, and queries for Firebase / GCP mobile backends",
		LogoSVG:     logoSVG,
		Actions: []connectors.ManifestAction{
			{
				ActionType:      "firestore.get",
				Name:            "Get document",
				Description:     "Read a document by path as a JSON field map",
				RiskLevel:       "low",
				DisplayTemplate: "Get {{path}}",
				Preview: &connectors.ActionPreview{
					Layout: "record",
					Fields: map[string]string{"title": "path"},
				},
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["path", "allowed_paths"],
					"properties": {
						"path": {
							"type": "string",
							"description": "Document path relative to the database root (e.g. users/alice)"
						},
						"allowed_paths": {
							"type": "array",
							"items": {"type": "string"},
							"minItems": 1,
							"maxItems": 64,
							"description": "Path allowlist — each entry is a full document path OR a collection path (odd segments) that prefixes allowed documents; request path must equal an entry or start with entry + /"
						},
						"allowed_read_fields": {
							"type": "array",
							"items": {"type": "string"},
							"maxItems": 100,
							"description": "When set, only these top-level fields are returned"
						}
					}
				}`)),
			},
			{
				ActionType:      "firestore.set",
				Name:            "Set document",
				Description:     "Create or overwrite a document with a JSON field map",
				RiskLevel:       "medium",
				DisplayTemplate: "Set {{path}}",
				Preview: &connectors.ActionPreview{
					Layout: "record",
					Fields: map[string]string{"title": "path"},
				},
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["path", "data", "allowed_paths"],
					"properties": {
						"path": {"type": "string", "description": "Document path relative to the database root"},
						"data": {
							"type": "object",
							"description": "Top-level fields to store (JSON-serializable values)"
						},
						"merge": {
							"type": "boolean",
							"description": "When true, merge into existing document instead of replacing"
						},
						"allowed_paths": {
							"type": "array",
							"items": {"type": "string"},
							"minItems": 1,
							"maxItems": 64,
							"description": "Document path allowlist — full document paths or collection paths (odd segments) as prefixes"
						},
						"allowed_write_fields": {
							"type": "array",
							"items": {"type": "string"},
							"maxItems": 100,
							"description": "When set, data may only contain these top-level field names"
						}
					}
				}`)),
			},
			{
				ActionType:      "firestore.update",
				Name:            "Update document",
				Description:     "Update fields via Firestore field paths (map keys are paths; dots denote nesting, e.g. address.city)",
				RiskLevel:       "medium",
				DisplayTemplate: "Update {{path}}",
				Preview: &connectors.ActionPreview{
					Layout: "record",
					Fields: map[string]string{"title": "path"},
				},
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["path", "data", "allowed_paths"],
					"properties": {
						"path": {"type": "string"},
						"data": {"type": "object", "description": "Field paths to values — each JSON object key is a Firestore field path (dot segments target nested fields, same as Firestore Update)"},
						"allowed_paths": {
							"type": "array",
							"items": {"type": "string"},
							"minItems": 1,
							"maxItems": 64,
							"description": "Document path allowlist — full document paths or collection paths (odd segments) as prefixes"
						},
						"allowed_write_fields": {
							"type": "array",
							"items": {"type": "string"},
							"maxItems": 100,
							"description": "When set, data keys must be listed here; values are Firestore field paths (e.g. address.city updates nested city)"
						}
					}
				}`)),
			},
			{
				ActionType:      "firestore.delete",
				Name:            "Delete document",
				Description:     "Delete a document by path",
				RiskLevel:       "high",
				DisplayTemplate: "Delete {{path}}",
				Preview: &connectors.ActionPreview{
					Layout: "record",
					Fields: map[string]string{"title": "path"},
				},
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["path", "allowed_paths"],
					"properties": {
						"path": {"type": "string"},
						"allowed_paths": {
							"type": "array",
							"items": {"type": "string"},
							"minItems": 1,
							"maxItems": 64,
							"description": "Document path allowlist — full document paths or collection paths (odd segments) as prefixes"
						}
					}
				}`)),
			},
			{
				ActionType:      "firestore.query",
				Name:            "Query collection",
				Description:     "Run a structured query on a collection (equality / order / limit)",
				RiskLevel:       "low",
				DisplayTemplate: "Query {{collection_path}}",
				Preview: &connectors.ActionPreview{
					Layout: "record",
					Fields: map[string]string{"title": "collection_path"},
				},
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["collection_path", "allowed_paths"],
					"properties": {
						"collection_path": {
							"type": "string",
							"description": "Collection path relative to the database root (e.g. users or users/alice/posts)"
						},
						"filters": {
							"type": "array",
							"description": "Equality filters: each item is {\"field\",\"op\",\"value\"} where op is \"==\"",
							"items": {
								"type": "object",
								"required": ["field", "op", "value"],
								"properties": {
									"field": {"type": "string"},
									"op": {"type": "string", "enum": ["=="]},
									"value": {}
								}
							},
							"maxItems": 10
						},
						"order_by": {
							"type": "array",
							"items": {
								"type": "object",
								"required": ["field", "direction"],
								"properties": {
									"field": {"type": "string"},
									"direction": {"type": "string", "enum": ["asc", "desc"]}
								}
							},
							"maxItems": 4
						},
						"limit": {
							"type": "integer",
							"minimum": 1,
							"maximum": 500,
							"default": 50,
							"description": "Maximum documents to return"
						},
						"allowed_paths": {
							"type": "array",
							"items": {"type": "string"},
							"minItems": 1,
							"maxItems": 64,
							"description": "Collection path allowlist — full collection paths or parent document paths (even segments) as prefixes"
						},
						"allowed_read_fields": {
							"type": "array",
							"items": {"type": "string"},
							"maxItems": 100
						}
					}
				}`)),
			},
		},
		RequiredCredentials: []connectors.ManifestCredential{
			{
				Service:         "firestore",
				AuthType:        "custom",
				InstructionsURL: "https://firebase.google.com/docs/firestore/quickstart",
			},
		},
		Templates: []connectors.ManifestTemplate{
			{
				ID:          "tpl_firestore_read_one_doc",
				ActionType:  "firestore.get",
				Name:        "Read one document",
				Description: "Agent may read a single document path only.",
				Parameters:  json.RawMessage(`{"path":"users/USER_ID","allowed_paths":["users/USER_ID"]}`),
			},
			{
				ID:          "tpl_firestore_write_profile",
				ActionType:  "firestore.update",
				Name:        "Update profile fields",
				Description: "Agent may patch allowed fields on a user document.",
				Parameters:  json.RawMessage(`{"path":"users/USER_ID","data":{"displayName":"*"},"allowed_paths":["users/USER_ID"],"allowed_write_fields":["displayName"]}`),
			},
			{
				ID:          "tpl_firestore_query_posts",
				ActionType:  "firestore.query",
				Name:        "Query posts subcollection",
				Description: "Agent may list posts under a user with a limit.",
				Parameters:  json.RawMessage(`{"collection_path":"users/USER_ID/posts","allowed_paths":["users/USER_ID/posts"],"limit":50,"order_by":[{"field":"createdAt","direction":"desc"}]}`),
			},
		},
	}
}

// Actions returns registered handlers.
func (c *FirestoreConnector) Actions() map[string]connectors.Action {
	return map[string]connectors.Action{
		"firestore.get":    &getAction{conn: c},
		"firestore.set":    &setAction{conn: c},
		"firestore.update": &updateAction{conn: c},
		"firestore.delete": &deleteAction{conn: c},
		"firestore.query":  &queryAction{conn: c},
	}
}

type saFile struct {
	Type        string `json:"type"`
	ProjectID   string `json:"project_id"`
	ClientEmail string `json:"client_email"`
	PrivateKey  string `json:"private_key"`
}

// ValidateCredentials checks service account JSON and optional overrides.
func (c *FirestoreConnector) ValidateCredentials(_ context.Context, creds connectors.Credentials) error {
	raw, ok := creds.Get("service_account_json")
	if !ok || strings.TrimSpace(raw) == "" {
		return &connectors.ValidationError{Message: "missing required credential: service_account_json (GCP service account key JSON)"}
	}
	var sa saFile
	if err := json.Unmarshal([]byte(raw), &sa); err != nil {
		return &connectors.ValidationError{Message: fmt.Sprintf("service_account_json is not valid JSON: %v", err)}
	}
	if sa.Type != "service_account" {
		return &connectors.ValidationError{Message: "service_account_json must be a GCP service account key (type service_account)"}
	}
	if sa.ClientEmail == "" || sa.PrivateKey == "" {
		return &connectors.ValidationError{Message: "service_account_json must include client_email and private_key"}
	}
	if host, ok := creds.Get("emulator_host"); ok && strings.TrimSpace(host) != "" {
		if err := validateEmulatorHost(host); err != nil {
			return err
		}
	}
	if pid, ok := creds.Get("project_id"); ok && strings.TrimSpace(pid) != "" {
		return nil
	}
	if sa.ProjectID == "" {
		return &connectors.ValidationError{Message: "missing project_id: set credential project_id or include project_id in the service account JSON"}
	}
	return nil
}

func resolveProjectID(creds connectors.Credentials, credJSON []byte) (string, error) {
	if pid, ok := creds.Get("project_id"); ok && strings.TrimSpace(pid) != "" {
		return strings.TrimSpace(pid), nil
	}
	var sa saFile
	if err := json.Unmarshal(credJSON, &sa); err != nil {
		return "", &connectors.ValidationError{Message: fmt.Sprintf("invalid service account JSON: %v", err)}
	}
	if sa.ProjectID == "" {
		return "", &connectors.ValidationError{Message: "could not determine project_id from credentials"}
	}
	return sa.ProjectID, nil
}

func (c *FirestoreConnector) buildRunner(ctx context.Context, projectID string, credJSON []byte, emulatorHost string) (fsRunner, error) {
	if _, err := google.CredentialsFromJSON(ctx, credJSON, "https://www.googleapis.com/auth/datastore"); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid service account credentials: %v", err)}
	}
	emulatorHost = strings.TrimSpace(emulatorHost)
	if emulatorHost != "" {
		if err := validateEmulatorHost(emulatorHost); err != nil {
			return nil, err
		}
		// Per-client gRPC dial — never use os.Setenv(FIRESTORE_EMULATOR_HOST) (process-wide, multi-tenant unsafe).
		return newRealRunnerEmulator(ctx, projectID, emulatorHost)
	}
	opts := []option.ClientOption{option.WithCredentialsJSON(credJSON)}
	return newRealRunner(ctx, projectID, opts)
}

func validateEmulatorHost(host string) error {
	host = strings.TrimSpace(host)
	if host == "" {
		return &connectors.ValidationError{Message: "emulator_host must not be empty when set"}
	}
	// host:port, no scheme
	if strings.Contains(host, "://") {
		return &connectors.ValidationError{Message: "emulator_host must be host:port without a scheme (e.g. 127.0.0.1:8080)"}
	}
	if !strings.Contains(host, ":") {
		return &connectors.ValidationError{Message: "emulator_host must include a port (e.g. 127.0.0.1:8080)"}
	}
	return nil
}
