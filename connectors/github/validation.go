package github

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// ptrValidatable constrains PT to be a pointer to T that implements validate().
// This allows parseAndValidate to work with pointer-receiver validate() methods.
type ptrValidatable[T any] interface {
	*T
	validate() error
}

// parseAndValidate unmarshals req.Parameters into T and calls validate().
// This eliminates the repeated unmarshal→validate boilerplate in every action.
// Usage: parseAndValidate[createIssueParams](req.Parameters)
// Go infers the pointer type automatically from the constraint.
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

// requireOwnerRepo validates the common owner+repo pair present in every action.
func requireOwnerRepo(owner, repo string) error {
	if owner == "" {
		return &connectors.ValidationError{Message: "missing required parameter: owner"}
	}
	if repo == "" {
		return &connectors.ValidationError{Message: "missing required parameter: repo"}
	}
	return nil
}

// requirePositiveInt validates that an integer field is > 0.
func requirePositiveInt(val int, field string) error {
	if val <= 0 {
		return &connectors.ValidationError{Message: fmt.Sprintf("missing or invalid required parameter: %s", field)}
	}
	return nil
}

// requireNonEmptyStrings validates a string slice has at least one element
// and no element is empty.
func requireNonEmptyStrings(vals []string, field string) error {
	if len(vals) == 0 {
		return &connectors.ValidationError{Message: fmt.Sprintf("missing required parameter: %s", field)}
	}
	for _, v := range vals {
		if v == "" {
			return &connectors.ValidationError{Message: fmt.Sprintf("%s must not contain empty strings", field)}
		}
	}
	return nil
}

// validateFilePath rejects file paths that would corrupt URL construction.
// Paths starting with "/" are absolute and invalid for the Contents API.
// Paths containing "?" or "#" would break URL parsing by injecting a query
// string or fragment into the request path.
func validateFilePath(path string) error {
	if strings.HasPrefix(path, "/") {
		return &connectors.ValidationError{Message: "invalid path: must be a relative path (must not start with '/')"}
	}
	if strings.ContainsAny(path, "?#") {
		return &connectors.ValidationError{Message: "invalid path: must not contain '?' or '#'"}
	}
	return nil
}

// escapeFilePath URL-encodes each segment of a repository file path while
// preserving the "/" separator. This prevents special characters in file
// names (spaces, brackets, etc.) from corrupting the request URL.
func escapeFilePath(path string) string {
	segments := strings.Split(path, "/")
	for i, s := range segments {
		segments[i] = url.PathEscape(s)
	}
	return strings.Join(segments, "/")
}

// validatePerPage returns a ValidationError if perPage exceeds GitHub's maximum.
func validatePerPage(perPage int) error {
	if perPage > 100 {
		return &connectors.ValidationError{Message: fmt.Sprintf("per_page must not exceed 100 (got %d)", perPage)}
	}
	return nil
}

// invalidRefChars contains characters forbidden in git ref names.
// See: https://git-scm.com/docs/git-check-ref-format
var invalidRefChars = strings.NewReplacer(
	"..", "",
	"~", "",
	"^", "",
	":", "",
	"?", "",
	"*", "",
	"[", "",
	"\\", "",
)

// validateRefName checks that a git ref name conforms to git-check-ref-format rules.
func validateRefName(name, field string) error {
	if name == "" {
		return &connectors.ValidationError{Message: fmt.Sprintf("missing required parameter: %s", field)}
	}
	if strings.HasPrefix(name, "/") || strings.HasPrefix(name, ".") {
		return &connectors.ValidationError{Message: fmt.Sprintf("invalid %s: must not start with '/' or '.'", field)}
	}
	if strings.HasSuffix(name, ".lock") || strings.HasSuffix(name, ".") {
		return &connectors.ValidationError{Message: fmt.Sprintf("invalid %s: must not end with '.lock' or '.'", field)}
	}
	if invalidRefChars.Replace(name) != name {
		return &connectors.ValidationError{Message: fmt.Sprintf("invalid %s: contains forbidden characters (.. ~ ^ : ? * [ \\)", field)}
	}
	return nil
}
