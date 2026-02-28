// Package connectors defines the interfaces and types for the connector
// execution layer. Connectors are in-process Go implementations that execute
// actions against external services (e.g., GitHub, Slack) on behalf of users.
//
// This package has no dependency on api/ or db/ — only standard library.
// The API layer depends on connectors/, not the reverse.
package connectors

import (
	"context"
	"encoding/json"
)

// Action executes a single action type against an external service.
type Action interface {
	Execute(ctx context.Context, req ActionRequest) (*ActionResult, error)
}

// Connector represents an integration with an external service.
// It owns shared configuration (HTTP clients, base URLs, auth helpers)
// and registers the actions it supports.
type Connector interface {
	// ID returns the connector identifier (e.g., "github", "slack").
	// Must match the connectors.id value in the database.
	ID() string

	// Actions returns a map of action_type -> Action handler.
	// Keys must match connector_actions.action_type in the database.
	Actions() map[string]Action

	// ValidateCredentials checks that the provided credentials are
	// sufficient and well-formed for this connector (e.g., API key
	// format, required scopes). Called before first execution.
	ValidateCredentials(ctx context.Context, creds Credentials) error
}

// ActionRequest is passed to every Action.Execute call.
type ActionRequest struct {
	ActionType  string          // e.g., "github.create_issue"
	Parameters  json.RawMessage // validated against schema before reaching here
	Credentials Credentials     // decrypted at execution time; redacted in logs and JSON
}

// ActionResult is returned from a successful execution.
type ActionResult struct {
	Data json.RawMessage // service-specific response payload
}

// ManifestProvider is optionally implemented by connectors that can
// describe their metadata declaratively. All built-in and external
// connectors implement this. The server uses it to auto-seed DB rows
// on startup, replacing manual seed.go files.
type ManifestProvider interface {
	Manifest() *ConnectorManifest
}

