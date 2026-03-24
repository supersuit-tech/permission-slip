package bigquery

import (
	"errors"
	"fmt"
	"strings"

	"google.golang.org/api/googleapi"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func mapBQError(err error, action string) error {
	if err == nil {
		return nil
	}
	if connectors.IsTimeout(err) {
		return &connectors.TimeoutError{Message: fmt.Sprintf("BigQuery %s timed out: %v", action, err)}
	}
	var gerr *googleapi.Error
	if errors.As(err, &gerr) && gerr != nil {
		if gerr.Code == 401 || gerr.Code == 403 {
			return &connectors.AuthError{Message: fmt.Sprintf("BigQuery auth error: %v", err)}
		}
		if gerr.Code == 400 {
			return &connectors.ValidationError{Message: fmt.Sprintf("BigQuery request error: %v", err)}
		}
	}
	msg := err.Error()
	if strings.Contains(msg, "accessDenied") || strings.Contains(msg, "invalid_grant") {
		return &connectors.AuthError{Message: fmt.Sprintf("BigQuery authentication failed: %v", err)}
	}
	return &connectors.ExternalError{Message: fmt.Sprintf("BigQuery %s failed: %v", action, err)}
}
