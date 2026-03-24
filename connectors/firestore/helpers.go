package firestore

import (
	"context"
	"fmt"
	"strings"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func (c *FirestoreConnector) withTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if c.timeout <= 0 {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, c.timeout)
}

const maxPathSegments = 32

// validateDocumentPath checks a Firestore document path: odd number of segments, each non-empty,
// no "..", reasonable length. Paths must not start with "projects/" (use project_id credential instead).
func validateDocumentPath(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return &connectors.ValidationError{Message: "document path must not be empty"}
	}
	if strings.HasPrefix(path, "projects/") {
		return &connectors.ValidationError{Message: "document path must be relative to the database (do not include projects/.../databases/...)"}
	}
	segs := splitFirestorePath(path)
	if len(segs) == 0 {
		return &connectors.ValidationError{Message: "invalid document path"}
	}
	if len(segs)%2 != 0 {
		return &connectors.ValidationError{Message: "document path must point to a document (even number of path segments: collection/doc[/collection/doc]...)"}
	}
	if len(segs) > maxPathSegments {
		return &connectors.ValidationError{Message: fmt.Sprintf("document path must not exceed %d segments", maxPathSegments)}
	}
	for _, s := range segs {
		if err := validatePathSegment(s); err != nil {
			return err
		}
	}
	return nil
}

// validateCollectionPath checks a collection path: even number of segments (collection under a document, ...).
func validateCollectionPath(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return &connectors.ValidationError{Message: "collection path must not be empty"}
	}
	if strings.HasPrefix(path, "projects/") {
		return &connectors.ValidationError{Message: "collection path must be relative to the database"}
	}
	segs := splitFirestorePath(path)
	if len(segs) == 0 {
		return &connectors.ValidationError{Message: "invalid collection path"}
	}
	if len(segs)%2 == 0 {
		return &connectors.ValidationError{Message: "collection path must end at a collection (odd number of path segments)"}
	}
	if len(segs) > maxPathSegments {
		return &connectors.ValidationError{Message: fmt.Sprintf("collection path must not exceed %d segments", maxPathSegments)}
	}
	for _, s := range segs {
		if err := validatePathSegment(s); err != nil {
			return err
		}
	}
	return nil
}

func validatePathSegment(s string) error {
	if s == "" {
		return &connectors.ValidationError{Message: "path segments must not be empty"}
	}
	if s == "." || s == ".." {
		return &connectors.ValidationError{Message: "path segments may not be '.' or '..'"}
	}
	if len(s) > 1500 {
		return &connectors.ValidationError{Message: "path segment too long"}
	}
	for _, r := range s {
		if r == '/' || r == '\\' {
			return &connectors.ValidationError{Message: "path segments may not contain slashes"}
		}
	}
	return nil
}

func splitFirestorePath(path string) []string {
	path = strings.Trim(path, "/")
	if path == "" {
		return nil
	}
	return strings.Split(path, "/")
}

func validateAllowedPaths(path string, allowed []string, kind string) error {
	if len(allowed) == 0 {
		return &connectors.ValidationError{Message: "allowed_paths must not be empty"}
	}
	if len(allowed) > maxAllowedPaths {
		return &connectors.ValidationError{Message: fmt.Sprintf("allowed_paths must not exceed %d entries", maxAllowedPaths)}
	}
	seen := make(map[string]struct{}, len(allowed))
	for _, p := range allowed {
		p = strings.TrimSpace(p)
		if p == "" {
			return &connectors.ValidationError{Message: "allowed_paths must not contain empty strings"}
		}
		if _, dup := seen[p]; dup {
			return &connectors.ValidationError{Message: fmt.Sprintf("duplicate entry in allowed_paths: %q", p)}
		}
		seen[p] = struct{}{}
		var err error
		if kind == "document" {
			err = validateDocumentPath(p)
		} else {
			err = validateCollectionPath(p)
		}
		if err != nil {
			return err
		}
	}
	if !pathAllowed(path, allowed, kind) {
		return &connectors.ValidationError{Message: fmt.Sprintf("path %q is not covered by allowed_paths", path)}
	}
	return nil
}

// pathAllowed returns true if path equals a prefix entry in allowed or equals / is under an allowed path
// (allowed path is a prefix followed by '/' or exact match).
func pathAllowed(path string, allowed []string, kind string) bool {
	path = strings.TrimSpace(path)
	for _, prefix := range allowed {
		prefix = strings.TrimSpace(prefix)
		if path == prefix {
			return true
		}
		if strings.HasPrefix(path, prefix+"/") {
			return true
		}
	}
	return false
}

func validateFieldAllowlist(names []string) error {
	if len(names) > maxFieldAllowlist {
		return &connectors.ValidationError{Message: fmt.Sprintf("field allowlist must not exceed %d names", maxFieldAllowlist)}
	}
	seen := make(map[string]struct{}, len(names))
	for _, n := range names {
		if n == "" {
			return &connectors.ValidationError{Message: "field allowlist must not contain empty names"}
		}
		if !isValidFieldName(n) {
			return &connectors.ValidationError{Message: fmt.Sprintf("invalid field name in allowlist: %q", n)}
		}
		if _, dup := seen[n]; dup {
			return &connectors.ValidationError{Message: fmt.Sprintf("duplicate field in allowlist: %q", n)}
		}
		seen[n] = struct{}{}
	}
	return nil
}

// isValidFieldName allows typical Firestore map keys; rejects control chars only.
func isValidFieldName(s string) bool {
	if len(s) == 0 || len(s) > 1500 {
		return false
	}
	for _, r := range s {
		if r < 0x20 {
			return false
		}
	}
	return true
}

func buildFieldSet(allowed []string) map[string]struct{} {
	if len(allowed) == 0 {
		return nil
	}
	m := make(map[string]struct{}, len(allowed))
	for _, a := range allowed {
		m[a] = struct{}{}
	}
	return m
}

func filterMapKeys(data map[string]interface{}, allowed map[string]struct{}) map[string]interface{} {
	if allowed == nil || data == nil {
		return data
	}
	out := make(map[string]interface{}, len(allowed))
	for k, v := range data {
		if _, ok := allowed[k]; ok {
			out[k] = v
		}
	}
	return out
}

func validateMapKeysSubset(data map[string]interface{}, allowed map[string]struct{}, label string) error {
	if allowed == nil || len(data) == 0 {
		return nil
	}
	for k := range data {
		if _, ok := allowed[k]; !ok {
			return &connectors.ValidationError{Message: fmt.Sprintf("%s field %q is not in allowed_write_fields", label, k)}
		}
	}
	return nil
}

func (c *FirestoreConnector) openRunner(ctx context.Context, creds connectors.Credentials) (fsRunner, error) {
	raw, ok := creds.Get("service_account_json")
	if !ok || strings.TrimSpace(raw) == "" {
		return nil, &connectors.ValidationError{Message: "missing service_account_json"}
	}
	credJSON := []byte(raw)
	projectID, err := resolveProjectID(creds, credJSON)
	if err != nil {
		return nil, err
	}
	emulatorHost, _ := creds.Get("emulator_host")
	return c.newRunner(ctx, projectID, credJSON, emulatorHost)
}
