package context

import (
	"context"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// SlackAPI is the minimal surface the context helpers need from the Slack connector.
type SlackAPI interface {
	Post(ctx context.Context, method string, creds connectors.Credentials, body any, dest any) error
	Get(ctx context.Context, method string, creds connectors.Credentials, params map[string]string, dest any) error
}
