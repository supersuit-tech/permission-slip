package snowflake

import (
	"fmt"
	"strings"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func mapSnowflakeError(err error, action string) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	if connectors.IsTimeout(err) {
		return &connectors.TimeoutError{Message: fmt.Sprintf("Snowflake %s timed out: %v", action, err)}
	}
	if strings.Contains(msg, "390100") || strings.Contains(msg, "390102") ||
		strings.Contains(msg, "authentication failed") || strings.Contains(msg, "Incorrect username or password") {
		return &connectors.AuthError{Message: fmt.Sprintf("Snowflake authentication failed: %v", err)}
	}
	if strings.Contains(msg, "syntax error") || strings.Contains(msg, "SQL compilation error") ||
		strings.Contains(msg, "Object does not exist") {
		return &connectors.ValidationError{Message: fmt.Sprintf("Snowflake SQL error: %v", err)}
	}
	if strings.Contains(msg, "timeout") || strings.Contains(msg, "canceled") {
		return &connectors.TimeoutError{Message: fmt.Sprintf("Snowflake %s: %v", action, err)}
	}
	return &connectors.ExternalError{Message: fmt.Sprintf("Snowflake %s failed: %v", action, err)}
}
