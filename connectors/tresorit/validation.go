package tresorit

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/supersuit-tech/permission-slip/connectors"
)

var validTresorName = regexp.MustCompile(`^[a-z0-9][a-z0-9.-]{1,61}[a-z0-9]$`)

type ptrValidatable[T any] interface {
	*T
	validate() error
}

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

func validateTresorName(tresor string) error {
	if strings.TrimSpace(tresor) == "" {
		return &connectors.ValidationError{Message: "missing required parameter: tresor"}
	}
	if !validTresorName.MatchString(tresor) {
		return &connectors.ValidationError{
			Message: fmt.Sprintf("invalid tresor name %q: must be 3-63 lowercase alphanumeric characters, hyphens, or periods", tresor),
		}
	}
	return nil
}

func validateObjectKey(key, field string) error {
	if strings.TrimSpace(key) == "" {
		return &connectors.ValidationError{Message: fmt.Sprintf("missing required parameter: %s", field)}
	}
	return nil
}

func stringsHasSuffix(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}
