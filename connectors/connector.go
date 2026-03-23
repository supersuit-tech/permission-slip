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

// ParameterAliaser is an optional interface implemented by actions that
// accept common aliases for their parameter names. The API layer checks for
// this interface at approval ingestion time and rewrites aliases to canonical
// keys before storing, so stored parameters are always canonical.
//
// Example: an action that expects "start_time" may return
// map[string]string{"start": "start_time"} so that agents sending "start"
// have their parameter automatically rewritten before storage and execution.
type ParameterAliaser interface {
	ParameterAliases() map[string]string // alias key → canonical key
}

// Normalizer is an optional interface for actions that need arbitrary
// parameter transformation beyond flat key aliasing. The API layer calls
// Normalize() after ParameterAliaser (if both are implemented), before
// storing the approval or executing the action.
//
// Use this for nested structures (e.g., rewriting snake_case field names
// inside arrays of objects) where ParameterAliaser's flat key→key map
// is insufficient.
type Normalizer interface {
	Normalize(params json.RawMessage) json.RawMessage
}

// RequestValidator is an optional interface for actions that can validate
// parameters at approval request time. Called after ParameterAliaser and
// Normalizer but before the approval is stored. Return a ValidationError
// to reject the request immediately — the agent receives HTTP 400 with the
// error message and can fix the parameters before the user ever sees it.
type RequestValidator interface {
	ValidateRequest(params json.RawMessage) error
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
	Payment     *PaymentInfo    // non-nil when the action requires a payment method
	UserEmail   string          // email of the Permission Slip user executing the action (may be empty)
}

// PaymentInfo contains the resolved payment method details passed to connectors
// that declare requires_payment_method: true. Raw card data is never included —
// only the Stripe payment method token and safe display metadata.
type PaymentInfo struct {
	StripePaymentMethodID string // Stripe payment method token (e.g. "pm_...")
	Brand                 string // card brand (e.g. "visa")
	Last4                 string // last 4 digits of the card
	AmountCents           int    // authorized transaction amount in cents
}

// ActionResult is returned from a successful execution.
type ActionResult struct {
	Data json.RawMessage // service-specific response payload
}

// ResourceDetailResolver is optionally implemented by connectors that can
// look up human-readable details for resources referenced by opaque IDs.
// Called when an approval request is created, before the approval is stored.
// Errors are non-fatal — the approval is stored without details on failure.
type ResourceDetailResolver interface {
	// ResolveResourceDetails fetches human-readable metadata for the resources
	// referenced by the action parameters. Returns a map of display-friendly
	// fields (e.g., {"title": "Q1 Planning", "start_time": "2026-03-15T14:00:00Z"}).
	ResolveResourceDetails(ctx context.Context, actionType string, params json.RawMessage, creds Credentials) (map[string]any, error)
}

// ManifestProvider is optionally implemented by connectors that can
// describe their metadata declaratively. All built-in and external
// connectors implement this. The server uses it to auto-seed DB rows
// on startup, replacing manual seed.go files.
type ManifestProvider interface {
	Manifest() *ConnectorManifest
}

// CalendarListItem is one calendar row for dashboard selectors (Google Calendar
// and Microsoft Graph). Fields are populated depending on the provider.
type CalendarListItem struct {
	ID                string `json:"id"`
	Summary           string `json:"summary,omitempty"`
	Name              string `json:"name,omitempty"`
	Primary           bool   `json:"primary,omitempty"`
	IsDefaultCalendar bool   `json:"isDefaultCalendar,omitempty"`
}

// CalendarLister is optionally implemented by connectors that can list the
// user's calendars for UI configuration (session-authenticated proxy endpoints).
type CalendarLister interface {
	ListCalendars(ctx context.Context, creds Credentials) ([]CalendarListItem, error)
	// CalendarListCredentialActionType is the connector_actions.action_type used
	// to look up required credentials for this list call (must exist in the DB).
	CalendarListCredentialActionType() string
}

