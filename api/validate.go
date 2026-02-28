package api

import (
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"sync"

	"github.com/go-playground/validator/v10"
)

// structValidator is the package-level, concurrency-safe validator instance.
var (
	structValidator     *validator.Validate
	structValidatorOnce sync.Once
)

// getValidator returns the singleton validator instance, initializing it on
// first call. The instance uses JSON tag names for field identification in
// error messages and is safe for concurrent use.
func getValidator() *validator.Validate {
	structValidatorOnce.Do(func() {
		structValidator = validator.New(validator.WithRequiredStructEnabled())

		// Use JSON tag names in error messages so they match the API field names
		// callers see (e.g. "agent_id" instead of "AgentID").
		structValidator.RegisterTagNameFunc(func(fld reflect.StructField) string {
			name, _, _ := strings.Cut(fld.Tag.Get("json"), ",")
			if name == "-" || name == "" {
				return fld.Name
			}
			return name
		})
	})
	return structValidator
}

// ValidateStruct validates a struct using its `validate` tags and returns an
// error with a user-friendly message describing the first failure, or nil.
func ValidateStruct(v any) error {
	err := getValidator().Struct(v)
	if err == nil {
		return nil
	}
	var ve validator.ValidationErrors
	if !errors.As(err, &ve) {
		return fmt.Errorf("validation failed")
	}
	if len(ve) == 0 {
		return nil
	}
	return fmt.Errorf("%s", formatValidationError(ve[0]))
}

// ValidateRequest validates the struct v and, if invalid, writes a 400
// BadRequest error response. Returns true if valid, false if an error
// response was written (caller should return).
func ValidateRequest(w http.ResponseWriter, r *http.Request, v any) bool {
	if err := ValidateStruct(v); err != nil {
		RespondError(w, r, http.StatusBadRequest, BadRequest(ErrInvalidRequest, err.Error()))
		return false
	}
	return true
}

// formatValidationError converts a single FieldError into a human-readable
// message using the JSON field name (set via RegisterTagNameFunc).
func formatValidationError(fe validator.FieldError) string {
	field := fe.Field()
	switch fe.Tag() {
	case "required":
		return field + " is required"
	case "gt":
		if fe.Param() == "0" {
			return field + " must be a positive integer"
		}
		return fmt.Sprintf("%s must be greater than %s", field, fe.Param())
	case "gte":
		return fmt.Sprintf("%s must be at least %s", field, fe.Param())
	case "lte":
		return fmt.Sprintf("%s must be at most %s", field, fe.Param())
	case "max":
		return field + " exceeds maximum length"
	case "min":
		kind := fe.Kind()
		if kind == reflect.Map || kind == reflect.Slice {
			return field + " is required and must be non-empty"
		}
		return fmt.Sprintf("%s must be at least %s characters", field, fe.Param())
	case "oneof":
		values := strings.Split(fe.Param(), " ")
		quoted := make([]string, len(values))
		for i, v := range values {
			quoted[i] = "'" + v + "'"
		}
		return fmt.Sprintf("%s must be one of: %s", field, strings.Join(quoted, ", "))
	default:
		return field + " is invalid"
	}
}
