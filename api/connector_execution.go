package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/supersuit-tech/permission-slip-web/connectors"
	"github.com/supersuit-tech/permission-slip-web/db"
)

// executeConnectorAction looks up the action in the connector registry,
// fetches and decrypts the user's credentials, and executes the action.
// Returns (nil, nil) if no connector is registered for the action type
// (graceful degradation during the transition period).
func executeConnectorAction(ctx context.Context, deps *Deps, userID, actionType string, parameters json.RawMessage) (*connectors.ActionResult, error) {
	if deps.Connectors == nil {
		return nil, nil
	}

	action, ok := deps.Connectors.GetAction(actionType)
	if !ok {
		return nil, nil
	}

	// Look up which credential services this connector requires.
	services, err := db.GetRequiredServicesByActionType(ctx, deps.DB, actionType)
	if err != nil {
		return nil, fmt.Errorf("look up required services: %w", err)
	}

	// Fetch and decrypt credentials for each required service.
	credMap := make(map[string]string, len(services))
	for _, service := range services {
		if deps.Vault == nil {
			return nil, fmt.Errorf("credential vault is not configured but connector requires service %q", service)
		}
		decrypted, err := db.GetDecryptedCredentials(ctx, deps.DB, deps.Vault.ReadSecret, userID, service, nil)
		if err != nil {
			var credErr *db.CredentialError
			if errors.As(err, &credErr) && credErr.Code == db.CredentialErrNotFound {
				return nil, &connectors.ValidationError{
					Message: fmt.Sprintf("no credentials stored for service %q", service),
				}
			}
			return nil, fmt.Errorf("decrypt credentials for service %q: %w", service, err)
		}
		// Flatten decrypted JSON map into string values for the Credentials type.
		for k, v := range decrypted {
			switch vv := v.(type) {
			case string:
				credMap[k] = vv
			default:
				b, err := json.Marshal(v)
				if err != nil {
					return nil, fmt.Errorf("marshal credential %q for service %q: %w", k, service, err)
				}
				credMap[k] = string(b)
			}
		}
	}

	creds := connectors.NewCredentials(credMap)

	// Validate credentials before executing the action.
	connectorID := strings.SplitN(actionType, ".", 2)[0]
	if conn, ok := deps.Connectors.Get(connectorID); ok {
		if err := conn.ValidateCredentials(ctx, creds); err != nil {
			return nil, err
		}
	}

	result, err := action.Execute(ctx, connectors.ActionRequest{
		ActionType:  actionType,
		Parameters:  parameters,
		Credentials: creds,
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}
