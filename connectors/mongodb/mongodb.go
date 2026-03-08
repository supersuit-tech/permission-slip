// Package mongodb implements the MongoDB connector for the Permission Slip
// connector execution layer. It uses the official Go MongoDB driver to execute
// CRUD operations against document databases on behalf of agents.
//
// Security model:
//   - Connection URI is stored in the vault and passed via credentials
//   - Collection and field allowlists constrain what agents can access
//   - Query filter operators are restricted to prevent expensive scans
//   - Result sizes are capped via configurable limits
//   - Default read preference uses secondaryPreferred where appropriate
package mongodb

import (
	_ "embed"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

const (
	defaultTimeout    = 30 * time.Second
	defaultResultLimit = 100
	maxResultLimit     = 1000
)

// allowedFilterOperators is the set of MongoDB query operators that agents are
// permitted to use in filters. This prevents expensive operations like $where
// or $regex that could be used for ReDoS or full collection scans.
var allowedFilterOperators = map[string]bool{
	"$eq":  true,
	"$ne":  true,
	"$gt":  true,
	"$gte": true,
	"$lt":  true,
	"$lte": true,
	"$in":  true,
	"$nin": true,
	"$and": true,
	"$or":  true,
	"$not": true,
	"$exists": true,
}

// MongoDBConnector owns the shared configuration used by all MongoDB actions.
// The actual connection is established per-request using credentials from the
// vault so no long-lived connection pool is maintained.
type MongoDBConnector struct {
	timeout time.Duration
	// dialFunc is used in tests to inject a custom client constructor.
	dialFunc func(ctx context.Context, uri string) (*mongo.Client, error)
}

// New creates a MongoDBConnector with sensible defaults.
func New() *MongoDBConnector {
	return &MongoDBConnector{
		timeout: defaultTimeout,
	}
}

// ID returns "mongodb", matching the connectors.id in the database.
func (c *MongoDBConnector) ID() string { return "mongodb" }

// Manifest returns the connector's metadata manifest. Used by the server to
// auto-seed DB rows on startup.
//go:embed logo.svg
var logoSVG string

func (c *MongoDBConnector) Manifest() *connectors.ConnectorManifest {
	return &connectors.ConnectorManifest{
		ID:          "mongodb",
		Name:        "MongoDB",
		Description: "MongoDB integration for document database operations",
		LogoSVG:     logoSVG,
		Actions: []connectors.ManifestAction{
			{
				ActionType:  "mongodb.find",
				Name:        "Find Documents",
				Description: "Query documents from a collection using filters",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["database", "collection"],
					"properties": {
						"database": {
							"type": "string",
							"description": "Database name"
						},
						"collection": {
							"type": "string",
							"description": "Collection name"
						},
						"filter": {
							"type": "object",
							"description": "MongoDB query filter (restricted operators only)",
							"default": {}
						},
						"projection": {
							"type": "object",
							"description": "Fields to include or exclude (1 to include, 0 to exclude)"
						},
						"sort": {
							"type": "object",
							"description": "Sort specification (1 for ascending, -1 for descending)"
						},
						"limit": {
							"type": "integer",
							"description": "Maximum number of documents to return (max 1000)",
							"default": 100,
							"minimum": 1,
							"maximum": 1000
						},
						"skip": {
							"type": "integer",
							"description": "Number of documents to skip",
							"default": 0,
							"minimum": 0
						}
					}
				}`)),
			},
			{
				ActionType:  "mongodb.insert",
				Name:        "Insert Documents",
				Description: "Insert one or more documents into a collection",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["database", "collection", "documents"],
					"properties": {
						"database": {
							"type": "string",
							"description": "Database name"
						},
						"collection": {
							"type": "string",
							"description": "Collection name"
						},
						"documents": {
							"type": "array",
							"description": "Array of documents to insert",
							"items": { "type": "object" },
							"minItems": 1,
							"maxItems": 100
						}
					}
				}`)),
			},
			{
				ActionType:  "mongodb.update",
				Name:        "Update Documents",
				Description: "Update documents matching a filter using update operators",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["database", "collection", "filter", "update"],
					"properties": {
						"database": {
							"type": "string",
							"description": "Database name"
						},
						"collection": {
							"type": "string",
							"description": "Collection name"
						},
						"filter": {
							"type": "object",
							"description": "Query filter to match documents to update"
						},
						"update": {
							"type": "object",
							"description": "Update operators (e.g. {\"$set\": {\"field\": \"value\"}})"
						},
						"multi": {
							"type": "boolean",
							"description": "Update all matching documents (default: false, updates only first match)",
							"default": false
						}
					}
				}`)),
			},
			{
				ActionType:  "mongodb.delete",
				Name:        "Delete Documents",
				Description: "Delete documents matching a filter",
				RiskLevel:   "high",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["database", "collection", "filter"],
					"properties": {
						"database": {
							"type": "string",
							"description": "Database name"
						},
						"collection": {
							"type": "string",
							"description": "Collection name"
						},
						"filter": {
							"type": "object",
							"description": "Query filter to match documents to delete (must not be empty)"
						},
						"multi": {
							"type": "boolean",
							"description": "Delete all matching documents (default: false, deletes only first match)",
							"default": false
						}
					}
				}`)),
			},
		},
		RequiredCredentials: []connectors.ManifestCredential{
			{Service: "mongodb", AuthType: "custom", InstructionsURL: "https://www.mongodb.com/docs/manual/reference/connection-string/"},
		},
		Templates: []connectors.ManifestTemplate{
			{
				ID:          "tpl_mongodb_find_readonly",
				ActionType:  "mongodb.find",
				Name:        "Read-only queries",
				Description: "Agent can query any collection with filters. Good for analytics and reporting agents.",
				Parameters:  json.RawMessage(`{"database":"*","collection":"*","filter":"*","projection":"*","sort":"*","limit":"*","skip":"*"}`),
			},
			{
				ID:          "tpl_mongodb_insert_all",
				ActionType:  "mongodb.insert",
				Name:        "Insert documents",
				Description: "Agent can insert documents into any collection.",
				Parameters:  json.RawMessage(`{"database":"*","collection":"*","documents":"*"}`),
			},
			{
				ID:          "tpl_mongodb_update_all",
				ActionType:  "mongodb.update",
				Name:        "Update documents",
				Description: "Agent can update documents in any collection.",
				Parameters:  json.RawMessage(`{"database":"*","collection":"*","filter":"*","update":"*","multi":"*"}`),
			},
			{
				ID:          "tpl_mongodb_delete_single",
				ActionType:  "mongodb.delete",
				Name:        "Delete single documents",
				Description: "Agent can delete individual documents. Multi-delete is disabled for safety.",
				Parameters:  json.RawMessage(`{"database":"*","collection":"*","filter":"*","multi":false}`),
			},
		},
	}
}

// Actions returns the registered action handlers keyed by action_type.
func (c *MongoDBConnector) Actions() map[string]connectors.Action {
	return map[string]connectors.Action{
		"mongodb.find":   &findAction{conn: c},
		"mongodb.insert": &insertAction{conn: c},
		"mongodb.update": &updateAction{conn: c},
		"mongodb.delete": &deleteAction{conn: c},
	}
}

// ValidateCredentials checks that the provided credentials contain a
// non-empty connection_uri for MongoDB connections.
func (c *MongoDBConnector) ValidateCredentials(_ context.Context, creds connectors.Credentials) error {
	uri, ok := creds.Get("connection_uri")
	if !ok || uri == "" {
		return &connectors.ValidationError{Message: "missing required credential: connection_uri"}
	}
	return nil
}

// connect creates a MongoDB client from credentials. The caller is responsible
// for calling client.Disconnect when done.
func (c *MongoDBConnector) connect(ctx context.Context, creds connectors.Credentials) (*mongo.Client, error) {
	uri, ok := creds.Get("connection_uri")
	if !ok || uri == "" {
		return nil, &connectors.ValidationError{Message: "connection_uri credential is missing or empty"}
	}

	if c.dialFunc != nil {
		return c.dialFunc(ctx, uri)
	}

	opts := options.Client().ApplyURI(uri).
		SetReadPreference(readpref.SecondaryPreferred()).
		SetTimeout(c.timeout)

	client, err := mongo.Connect(opts)
	if err != nil {
		return nil, &connectors.ExternalError{Message: fmt.Sprintf("MongoDB connection failed: %v", err)}
	}

	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		_ = client.Disconnect(ctx)
		if connectors.IsTimeout(err) {
			return nil, &connectors.TimeoutError{Message: fmt.Sprintf("MongoDB ping timed out: %v", err)}
		}
		return nil, &connectors.ExternalError{Message: fmt.Sprintf("MongoDB ping failed: %v", err)}
	}

	return client, nil
}

// validateFilter checks that a filter document only uses allowed operators.
// This prevents agents from using expensive or dangerous operators like $where.
func validateFilter(filter map[string]interface{}) error {
	for key, val := range filter {
		if len(key) > 0 && key[0] == '$' {
			if !allowedFilterOperators[key] {
				return &connectors.ValidationError{
					Message: fmt.Sprintf("filter operator %q is not allowed; permitted operators: %s", key, allowedOperatorList()),
				}
			}
		}
		// Recurse into nested documents (e.g., field-level operators like {"age": {"$gt": 5}}).
		if nested, ok := val.(map[string]interface{}); ok {
			if err := validateFilter(nested); err != nil {
				return err
			}
		}
		// Recurse into arrays (e.g., $and / $or clauses).
		if arr, ok := val.([]interface{}); ok {
			for _, item := range arr {
				if nested, ok := item.(map[string]interface{}); ok {
					if err := validateFilter(nested); err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}

// allowedOperatorList returns a comma-separated list of allowed operators for error messages.
func allowedOperatorList() string {
	ops := make([]string, 0, len(allowedFilterOperators))
	for op := range allowedFilterOperators {
		ops = append(ops, op)
	}
	return fmt.Sprintf("%v", ops)
}
