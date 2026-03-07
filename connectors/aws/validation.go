package aws

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// ptrValidatable constrains PT to be a pointer to T that implements validate().
type ptrValidatable[T any] interface {
	*T
	validate() error
}

// parseAndValidate unmarshals raw JSON into T and calls validate().
// Usage: parseAndValidate[describeInstancesParams](req.Parameters)
func parseAndValidate[T any, PT ptrValidatable[T]](raw json.RawMessage) (PT, error) {
	params := PT(new(T))
	if err := json.Unmarshal(raw, params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}
	return params, nil
}

// validateRegion checks that region is non-empty and looks like an AWS region
// (e.g. "us-east-1", "eu-west-2", "ap-southeast-1").
func validateRegion(region string) error {
	if region == "" {
		return &connectors.ValidationError{Message: "missing required parameter: region"}
	}
	// AWS regions follow the pattern: 2-letter area, dash, direction, dash, digit(s).
	// Rather than maintaining a hard-coded list that goes stale, we do a basic
	// format check — the AWS API itself will reject truly invalid regions.
	if !strings.Contains(region, "-") {
		return &connectors.ValidationError{
			Message: fmt.Sprintf("invalid region %q: expected format like us-east-1, eu-west-2", region),
		}
	}
	return nil
}

// validateInstanceID checks that an EC2 instance ID is non-empty and starts
// with "i-". Provides a helpful error if a common mistake is detected.
func validateInstanceID(id string) error {
	if id == "" {
		return &connectors.ValidationError{Message: "missing required parameter: instance_id"}
	}
	if !strings.HasPrefix(id, "i-") {
		return &connectors.ValidationError{
			Message: fmt.Sprintf("invalid instance_id %q: must start with 'i-' (e.g. i-1234567890abcdef0)", id),
		}
	}
	return nil
}

// uriEncodePath encodes a path string per AWS SigV4 rules: each segment is
// percent-encoded but "/" separators are preserved. This is necessary because
// S3 object keys can contain spaces, +, ?, # and other reserved characters.
func uriEncodePath(path string) string {
	segments := strings.Split(path, "/")
	for i, seg := range segments {
		segments[i] = url.PathEscape(seg)
	}
	return strings.Join(segments, "/")
}

// validateRFC3339 checks that a timestamp string is valid RFC 3339 format.
func validateRFC3339(value, field string) error {
	if value == "" {
		return &connectors.ValidationError{Message: fmt.Sprintf("missing required parameter: %s", field)}
	}
	if _, err := time.Parse(time.RFC3339, value); err != nil {
		return &connectors.ValidationError{
			Message: fmt.Sprintf("invalid %s: must be RFC 3339 format (e.g. 2024-01-15T00:00:00Z), got %q", field, value),
		}
	}
	return nil
}
