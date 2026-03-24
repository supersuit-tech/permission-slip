package dynamodb

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func (c *DynamoDBConnector) withTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if c.timeout <= 0 {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, c.timeout)
}

func validateAllowedTables(table string, allowed []string) error {
	if len(allowed) == 0 {
		return &connectors.ValidationError{Message: "allowed_tables must not be empty"}
	}
	if len(allowed) > maxAllowedTables {
		return &connectors.ValidationError{Message: fmt.Sprintf("allowed_tables must not exceed %d entries", maxAllowedTables)}
	}
	if !isValidTableName(table) {
		return &connectors.ValidationError{Message: "invalid table name"}
	}
	allowedSet := make(map[string]struct{}, len(allowed))
	for _, t := range allowed {
		t = strings.TrimSpace(t)
		if t == "" {
			return &connectors.ValidationError{Message: "allowed_tables must not contain empty strings"}
		}
		if !isValidTableName(t) {
			return &connectors.ValidationError{Message: fmt.Sprintf("invalid allowlist table name: %q", t)}
		}
		allowedSet[t] = struct{}{}
	}
	if _, ok := allowedSet[table]; !ok {
		return &connectors.ValidationError{Message: fmt.Sprintf("table %q is not in allowed_tables", table)}
	}
	return nil
}

// isValidTableName allows typical DynamoDB table names (alphanumeric, dash, underscore, dot).
func isValidTableName(s string) bool {
	if len(s) == 0 || len(s) > 255 {
		return false
	}
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
			continue
		}
		return false
	}
	return true
}

func validateAttrAllowlist(names []string) error {
	if len(names) > maxAttrAllowlist {
		return &connectors.ValidationError{Message: fmt.Sprintf("attribute allowlist must not exceed %d names", maxAttrAllowlist)}
	}
	seen := make(map[string]struct{}, len(names))
	for _, n := range names {
		if n == "" {
			return &connectors.ValidationError{Message: "attribute allowlist must not contain empty names"}
		}
		if !isValidAttrName(n) {
			return &connectors.ValidationError{Message: fmt.Sprintf("invalid attribute name in allowlist: %q", n)}
		}
		if _, dup := seen[n]; dup {
			return &connectors.ValidationError{Message: fmt.Sprintf("duplicate attribute in allowlist: %q", n)}
		}
		seen[n] = struct{}{}
	}
	return nil
}

func isValidAttrName(s string) bool {
	if len(s) == 0 || len(s) > 255 {
		return false
	}
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
			continue
		}
		return false
	}
	return true
}

func allowedReadSet(allowed []string) map[string]struct{} {
	if len(allowed) == 0 {
		return nil
	}
	m := make(map[string]struct{}, len(allowed))
	for _, a := range allowed {
		m[a] = struct{}{}
	}
	return m
}

// filterItemAttrs keeps only keys present in allowed (nil allowed = no filtering).
func filterItemAttrs(item map[string]types.AttributeValue, allowed map[string]struct{}) map[string]types.AttributeValue {
	if allowed == nil {
		return item
	}
	out := make(map[string]types.AttributeValue, len(allowed))
	for k, v := range item {
		if _, ok := allowed[k]; ok {
			out[k] = v
		}
	}
	return out
}

// validateProjectionSubset ensures projection keys are in allowed when allowed is set.
func validateProjectionSubset(projection map[string]bool, allowed map[string]struct{}) error {
	if allowed == nil || len(projection) == 0 {
		return nil
	}
	for k, include := range projection {
		if !include {
			return &connectors.ValidationError{Message: "projection values must be true only (DynamoDB-style)"}
		}
		if _, ok := allowed[k]; !ok {
			return &connectors.ValidationError{Message: fmt.Sprintf("projection attribute %q is not in allowed_read_attributes", k)}
		}
	}
	return nil
}

// validateItemWriteSubset ensures item keys are subset of allowed when allowed is set.
func validateItemWriteSubset(item map[string]types.AttributeValue, allowed map[string]struct{}) error {
	if allowed == nil {
		return nil
	}
	for k := range item {
		if _, ok := allowed[k]; !ok {
			return &connectors.ValidationError{Message: fmt.Sprintf("item attribute %q is not in allowed_write_attributes", k)}
		}
	}
	return nil
}

func cloneStringMap(m map[string]string) map[string]string {
	if len(m) == 0 {
		return nil
	}
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}
