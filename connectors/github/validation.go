package github

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/supersuit-tech/permission-slip/connectors"
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

// setPagination adds per_page and page query parameters to q.
// perPage defaults to 30 if <= 0; page is omitted when <= 1.
// This centralises the repeated pagination-building pattern across list/search actions.
func setPagination(q url.Values, perPage, page int) {
	if perPage <= 0 {
		perPage = 30
	}
	q.Set("per_page", fmt.Sprintf("%d", perPage))
	if page > 1 {
		q.Set("page", fmt.Sprintf("%d", page))
	}
}

// repoNameRe matches valid GitHub repository names: alphanumeric, hyphen, underscore, dot.
var repoNameRe = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)

// orgNameRe matches valid GitHub organization/user names: alphanumeric and hyphens,
// must start and end with an alphanumeric character.
var orgNameRe = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?$`)

// validateRepoName checks that a repository name is valid per GitHub's naming rules.
// Names must be non-empty and contain only alphanumeric characters, hyphens,
// underscores, and dots. Callers should TrimSpace before calling.
func validateRepoName(name string) error {
	if name == "" {
		return &connectors.ValidationError{Message: "missing required parameter: name"}
	}
	if name == "." || name == ".." {
		return &connectors.ValidationError{Message: fmt.Sprintf("invalid repository name %q: '.' and '..' are reserved", name)}
	}
	if strings.HasSuffix(name, ".git") {
		return &connectors.ValidationError{Message: fmt.Sprintf("invalid repository name %q: names ending with '.git' are reserved", name)}
	}
	if len(name) > 100 {
		return &connectors.ValidationError{Message: fmt.Sprintf("invalid repository name: must not exceed 100 characters (got %d)", len(name))}
	}
	if !repoNameRe.MatchString(name) {
		return &connectors.ValidationError{Message: fmt.Sprintf("invalid repository name %q: must contain only alphanumeric characters, hyphens, underscores, and dots", name)}
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
