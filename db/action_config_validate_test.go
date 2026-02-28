package db

import (
	"encoding/json"
	"strings"
	"testing"
)

// ── Wildcard Syntax Tests ───────────────────────────────────────────────────

func TestWildcardValue_Constant(t *testing.T) {
	t.Parallel()
	if WildcardValue != "*" {
		t.Errorf("expected WildcardValue to be %q, got %q", "*", WildcardValue)
	}
}

func TestIsWildcard(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		raw  json.RawMessage
		want bool
	}{
		{"wildcard string", json.RawMessage(`"*"`), true},
		{"regular string", json.RawMessage(`"hello"`), false},
		{"empty string", json.RawMessage(`""`), false},
		{"number", json.RawMessage(`42`), false},
		{"boolean", json.RawMessage(`true`), false},
		{"null", json.RawMessage(`null`), false},
		{"array", json.RawMessage(`["*"]`), false},
		{"object", json.RawMessage(`{"key":"*"}`), false},
		{"pattern (not bare wildcard)", json.RawMessage(`"*@mycompany.com"`), false},
		{"prefix pattern", json.RawMessage(`"supersuit-tech/*"`), false},
		{"infix pattern", json.RawMessage(`"test-*-prod"`), false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := IsWildcard(tc.raw)
			if got != tc.want {
				t.Errorf("IsWildcard(%s) = %v, want %v", string(tc.raw), got, tc.want)
			}
		})
	}
}

// ── Pattern Syntax Tests ────────────────────────────────────────────────────

func TestIsPattern(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		raw  json.RawMessage
		want bool
	}{
		// Valid $pattern wrapper objects
		{"suffix pattern", json.RawMessage(`{"$pattern":"*@mycompany.com"}`), true},
		{"prefix pattern", json.RawMessage(`{"$pattern":"supersuit-tech/*"}`), true},
		{"infix pattern", json.RawMessage(`{"$pattern":"test-*-prod"}`), true},
		{"prefix only", json.RawMessage(`{"$pattern":"v*"}`), true},
		{"multiple wildcards", json.RawMessage(`{"$pattern":"*-*-*"}`), true},
		// Invalid: $pattern value without a star
		{"pattern without star", json.RawMessage(`{"$pattern":"hello"}`), false},
		{"pattern empty string", json.RawMessage(`{"$pattern":""}`), false},
		// Not patterns: plain strings (backward-compat — never auto-detected)
		{"plain string with star", json.RawMessage(`"*@mycompany.com"`), false},
		{"bare wildcard string", json.RawMessage(`"*"`), false},
		{"regular string", json.RawMessage(`"hello"`), false},
		// Not patterns: wrong types / shapes
		{"number", json.RawMessage(`42`), false},
		{"boolean", json.RawMessage(`true`), false},
		{"null", json.RawMessage(`null`), false},
		{"array", json.RawMessage(`["*"]`), false},
		{"object with wrong key", json.RawMessage(`{"key":"*"}`), false},
		{"object with extra keys", json.RawMessage(`{"$pattern":"v*","extra":"x"}`), false},
		{"$pattern value is number", json.RawMessage(`{"$pattern":42}`), false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := IsPattern(tc.raw)
			if got != tc.want {
				t.Errorf("IsPattern(%s) = %v, want %v", string(tc.raw), got, tc.want)
			}
		})
	}
}

func TestMatchPattern(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		pattern string
		value   string
		want    bool
	}{
		// Suffix patterns
		{"suffix match", "*@mycompany.com", "alice@mycompany.com", true},
		{"suffix match with dots", "*@mycompany.com", "alice.bob@mycompany.com", true},
		{"suffix no match", "*@mycompany.com", "alice@other.com", false},
		{"suffix empty prefix", "*@mycompany.com", "@mycompany.com", true},
		// Prefix patterns
		{"prefix match", "supersuit-tech/*", "supersuit-tech/webapp", true},
		{"prefix match with slash", "supersuit-tech/*", "supersuit-tech/deep/nested", true},
		{"prefix no match", "supersuit-tech/*", "other-org/webapp", false},
		{"prefix empty suffix", "supersuit-tech/*", "supersuit-tech/", true},
		{"version prefix", "v*", "v1.2.3", true},
		{"version prefix no match", "v*", "1.2.3", false},
		// Infix patterns
		{"infix match", "test-*-prod", "test-alpha-prod", true},
		{"infix match with dashes", "test-*-prod", "test-alpha-beta-prod", true},
		{"infix no match suffix", "test-*-prod", "test-alpha-staging", false},
		{"infix no match prefix", "test-*-prod", "dev-alpha-prod", false},
		{"infix empty middle", "test-*-prod", "test--prod", true},
		// Multiple wildcards
		{"multi wildcard", "*-*-*", "a-b-c", true},
		{"multi wildcard complex", "*-*-*", "alpha-beta-gamma-delta", true},
		{"multi wildcard no match", "*-*-*", "abc", false},
		// Regex metacharacters in pattern (should be treated literally)
		{"regex dot escaped", "*.json", "data.json", true},
		{"regex dot not special", "*.json", "dataxjson", false},
		{"regex paren escaped", "func(*)", "func(x)", true},
		{"regex bracket escaped", "arr[*]", "arr[0]", true},
		// Edge cases
		{"exact match (no star)", "hello", "hello", true},
		{"exact no match", "hello", "world", false},
		{"empty pattern star", "*", "anything", true},
		{"empty pattern star empty value", "*", "", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := MatchPattern(tc.pattern, tc.value)
			if got != tc.want {
				t.Errorf("MatchPattern(%q, %q) = %v, want %v", tc.pattern, tc.value, got, tc.want)
			}
		})
	}
}

// ── Parameter Validation Tests ──────────────────────────────────────────────

func TestValidateParametersAgainstConfig_AllFixed(t *testing.T) {
	t.Parallel()
	config := json.RawMessage(`{"repo":"supersuit-tech/webapp","label":"bug"}`)
	exec := json.RawMessage(`{"repo":"supersuit-tech/webapp","label":"bug"}`)

	if err := ValidateParametersAgainstConfig(config, exec); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestValidateParametersAgainstConfig_FixedMismatch(t *testing.T) {
	t.Parallel()
	config := json.RawMessage(`{"repo":"supersuit-tech/webapp"}`)
	exec := json.RawMessage(`{"repo":"supersuit-tech/api"}`)

	err := ValidateParametersAgainstConfig(config, exec)
	if err == nil {
		t.Fatal("expected error for mismatched fixed param")
	}
	cve, ok := err.(*ConfigValidationError)
	if !ok {
		t.Fatalf("expected ConfigValidationError, got %T", err)
	}
	if cve.Parameter != "repo" {
		t.Errorf("expected parameter 'repo', got %q", cve.Parameter)
	}
}

func TestValidateParametersAgainstConfig_WildcardAcceptsAny(t *testing.T) {
	t.Parallel()
	config := json.RawMessage(`{"repo":"supersuit-tech/webapp","title":"*","body":"*"}`)
	exec := json.RawMessage(`{"repo":"supersuit-tech/webapp","title":"Fix login bug","body":"The login page crashes"}`)

	if err := ValidateParametersAgainstConfig(config, exec); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestValidateParametersAgainstConfig_WildcardOmittedIsOK(t *testing.T) {
	t.Parallel()
	// Wildcard parameters can be omitted — the agent chose not to provide them.
	config := json.RawMessage(`{"repo":"supersuit-tech/webapp","title":"*","body":"*"}`)
	exec := json.RawMessage(`{"repo":"supersuit-tech/webapp","title":"My issue"}`)

	if err := ValidateParametersAgainstConfig(config, exec); err != nil {
		t.Errorf("expected no error for omitted wildcard param, got: %v", err)
	}
}

func TestValidateParametersAgainstConfig_MissingFixedParam(t *testing.T) {
	t.Parallel()
	config := json.RawMessage(`{"repo":"supersuit-tech/webapp","label":"bug"}`)
	exec := json.RawMessage(`{"repo":"supersuit-tech/webapp"}`)

	err := ValidateParametersAgainstConfig(config, exec)
	if err == nil {
		t.Fatal("expected error for missing fixed param")
	}
	cve, ok := err.(*ConfigValidationError)
	if !ok {
		t.Fatalf("expected ConfigValidationError, got %T", err)
	}
	if cve.Parameter != "label" {
		t.Errorf("expected parameter 'label', got %q", cve.Parameter)
	}
}

func TestValidateParametersAgainstConfig_ExtraParam(t *testing.T) {
	t.Parallel()
	config := json.RawMessage(`{"repo":"supersuit-tech/webapp"}`)
	exec := json.RawMessage(`{"repo":"supersuit-tech/webapp","extra":"value"}`)

	err := ValidateParametersAgainstConfig(config, exec)
	if err == nil {
		t.Fatal("expected error for extra param")
	}
	cve, ok := err.(*ConfigValidationError)
	if !ok {
		t.Fatalf("expected ConfigValidationError, got %T", err)
	}
	if cve.Parameter != "extra" {
		t.Errorf("expected parameter 'extra', got %q", cve.Parameter)
	}
}

func TestValidateParametersAgainstConfig_EmptyConfig(t *testing.T) {
	t.Parallel()
	// Empty config means no parameters allowed.
	config := json.RawMessage(`{}`)
	exec := json.RawMessage(`{}`)

	if err := ValidateParametersAgainstConfig(config, exec); err != nil {
		t.Errorf("expected no error for empty config and exec, got: %v", err)
	}
}

func TestValidateParametersAgainstConfig_EmptyConfigRejectsExec(t *testing.T) {
	t.Parallel()
	config := json.RawMessage(`{}`)
	exec := json.RawMessage(`{"foo":"bar"}`)

	err := ValidateParametersAgainstConfig(config, exec)
	if err == nil {
		t.Fatal("expected error for param not in empty config")
	}
}

func TestValidateParametersAgainstConfig_ComplexValues(t *testing.T) {
	t.Parallel()
	// Fixed parameter with an array value.
	config := json.RawMessage(`{"to":["alice@example.com","bob@example.com"],"subject":"*"}`)
	exec := json.RawMessage(`{"to":["alice@example.com","bob@example.com"],"subject":"Hello"}`)

	if err := ValidateParametersAgainstConfig(config, exec); err != nil {
		t.Errorf("expected no error for matching complex value, got: %v", err)
	}
}

func TestValidateParametersAgainstConfig_ComplexValueMismatch(t *testing.T) {
	t.Parallel()
	config := json.RawMessage(`{"to":["alice@example.com"],"subject":"*"}`)
	exec := json.RawMessage(`{"to":["bob@example.com"],"subject":"Hello"}`)

	err := ValidateParametersAgainstConfig(config, exec)
	if err == nil {
		t.Fatal("expected error for mismatched array value")
	}
}

func TestValidateParametersAgainstConfig_NumericValues(t *testing.T) {
	t.Parallel()
	config := json.RawMessage(`{"amount":9900,"currency":"USD"}`)
	exec := json.RawMessage(`{"amount":9900,"currency":"USD"}`)

	if err := ValidateParametersAgainstConfig(config, exec); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestValidateParametersAgainstConfig_NumericMismatch(t *testing.T) {
	t.Parallel()
	config := json.RawMessage(`{"amount":9900}`)
	exec := json.RawMessage(`{"amount":5000}`)

	err := ValidateParametersAgainstConfig(config, exec)
	if err == nil {
		t.Fatal("expected error for mismatched numeric value")
	}
}

func TestValidateParametersAgainstConfig_NullConfig(t *testing.T) {
	t.Parallel()
	// Nil/empty config should be treated as empty object.
	if err := ValidateParametersAgainstConfig(nil, json.RawMessage(`{}`)); err != nil {
		t.Errorf("expected no error for nil config, got: %v", err)
	}
}

func TestValidateParametersAgainstConfig_BooleanValues(t *testing.T) {
	t.Parallel()
	config := json.RawMessage(`{"draft":true,"title":"*"}`)
	exec := json.RawMessage(`{"draft":true,"title":"My PR"}`)

	if err := ValidateParametersAgainstConfig(config, exec); err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestValidateParametersAgainstConfig_WildcardWithAnyType(t *testing.T) {
	t.Parallel()
	// Wildcard should accept any type: numbers, arrays, objects.
	config := json.RawMessage(`{"data":"*"}`)

	tests := []struct {
		name string
		exec json.RawMessage
	}{
		{"string", json.RawMessage(`{"data":"hello"}`)},
		{"number", json.RawMessage(`{"data":42}`)},
		{"array", json.RawMessage(`{"data":[1,2,3]}`)},
		{"object", json.RawMessage(`{"data":{"nested":"value"}}`)},
		{"boolean", json.RawMessage(`{"data":true}`)},
		{"null", json.RawMessage(`{"data":null}`)},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if err := ValidateParametersAgainstConfig(config, tc.exec); err != nil {
				t.Errorf("expected wildcard to accept %s, got: %v", tc.name, err)
			}
		})
	}
}

// ── Pattern Validation Tests ────────────────────────────────────────────────

func TestValidateParametersAgainstConfig_PatternSuffix(t *testing.T) {
	t.Parallel()
	// Pattern: email must end with @mycompany.com
	config := json.RawMessage(`{"to":{"$pattern":"*@mycompany.com"},"subject":"*"}`)
	exec := json.RawMessage(`{"to":"alice@mycompany.com","subject":"Hello"}`)

	if err := ValidateParametersAgainstConfig(config, exec); err != nil {
		t.Errorf("expected pattern suffix to match, got: %v", err)
	}
}

func TestValidateParametersAgainstConfig_PatternPrefix(t *testing.T) {
	t.Parallel()
	// Pattern: repo must start with supersuit-tech/
	config := json.RawMessage(`{"repo":{"$pattern":"supersuit-tech/*"},"title":"*"}`)
	exec := json.RawMessage(`{"repo":"supersuit-tech/webapp","title":"Fix bug"}`)

	if err := ValidateParametersAgainstConfig(config, exec); err != nil {
		t.Errorf("expected pattern prefix to match, got: %v", err)
	}
}

func TestValidateParametersAgainstConfig_PatternInfix(t *testing.T) {
	t.Parallel()
	// Pattern: environment must match test-*-prod
	config := json.RawMessage(`{"env":{"$pattern":"test-*-prod"},"cmd":"*"}`)
	exec := json.RawMessage(`{"env":"test-us-east-prod","cmd":"deploy"}`)

	if err := ValidateParametersAgainstConfig(config, exec); err != nil {
		t.Errorf("expected pattern infix to match, got: %v", err)
	}
}

func TestValidateParametersAgainstConfig_PatternMismatch(t *testing.T) {
	t.Parallel()
	config := json.RawMessage(`{"to":{"$pattern":"*@mycompany.com"}}`)
	exec := json.RawMessage(`{"to":"alice@other.com"}`)

	err := ValidateParametersAgainstConfig(config, exec)
	if err == nil {
		t.Fatal("expected error for pattern mismatch")
	}
	cve, ok := err.(*ConfigValidationError)
	if !ok {
		t.Fatalf("expected ConfigValidationError, got %T", err)
	}
	if cve.Parameter != "to" {
		t.Errorf("expected parameter 'to', got %q", cve.Parameter)
	}
	if !strings.Contains(cve.Reason, "does not match pattern") {
		t.Errorf("expected reason to mention pattern mismatch, got: %s", cve.Reason)
	}
}

func TestValidateParametersAgainstConfig_PatternRequiresString(t *testing.T) {
	t.Parallel()
	// Pattern parameters must receive a string value — numbers/booleans/etc are rejected.
	config := json.RawMessage(`{"tag":{"$pattern":"v*"}}`)

	tests := []struct {
		name string
		exec json.RawMessage
	}{
		{"number", json.RawMessage(`{"tag":42}`)},
		{"boolean", json.RawMessage(`{"tag":true}`)},
		{"array", json.RawMessage(`{"tag":["v1"]}`)},
		{"object", json.RawMessage(`{"tag":{"v":"1"}}`)},
		{"null", json.RawMessage(`{"tag":null}`)},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateParametersAgainstConfig(config, tc.exec)
			if err == nil {
				t.Errorf("expected pattern to reject %s value", tc.name)
			}
			cve, ok := err.(*ConfigValidationError)
			if !ok {
				t.Fatalf("expected ConfigValidationError, got %T", err)
			}
			if cve.Parameter != "tag" {
				t.Errorf("expected parameter 'tag', got %q", cve.Parameter)
			}
		})
	}
}

func TestValidateParametersAgainstConfig_PatternMissing(t *testing.T) {
	t.Parallel()
	// Pattern parameters are required — they cannot be omitted.
	config := json.RawMessage(`{"repo":{"$pattern":"supersuit-tech/*"},"title":"*"}`)
	exec := json.RawMessage(`{"title":"Bug fix"}`)

	err := ValidateParametersAgainstConfig(config, exec)
	if err == nil {
		t.Fatal("expected error for missing pattern parameter")
	}
	cve, ok := err.(*ConfigValidationError)
	if !ok {
		t.Fatalf("expected ConfigValidationError, got %T", err)
	}
	if cve.Parameter != "repo" {
		t.Errorf("expected parameter 'repo', got %q", cve.Parameter)
	}
	if !strings.Contains(cve.Reason, "required pattern parameter") {
		t.Errorf("expected reason to mention required pattern, got: %s", cve.Reason)
	}
}

func TestValidateParametersAgainstConfig_PatternWithFixedAndWildcard(t *testing.T) {
	t.Parallel()
	// Mix of all three types: fixed, pattern, and wildcard.
	config := json.RawMessage(`{"repo":{"$pattern":"supersuit-tech/*"},"label":"bug","title":"*"}`)
	exec := json.RawMessage(`{"repo":"supersuit-tech/webapp","label":"bug","title":"Fix login"}`)

	if err := ValidateParametersAgainstConfig(config, exec); err != nil {
		t.Errorf("expected mixed config to pass, got: %v", err)
	}
}

func TestValidateParametersAgainstConfig_PatternMultipleWildcards(t *testing.T) {
	t.Parallel()
	// Pattern with multiple * characters.
	config := json.RawMessage(`{"env":{"$pattern":"*-*-prod"}}`)
	exec := json.RawMessage(`{"env":"us-east-prod"}`)

	if err := ValidateParametersAgainstConfig(config, exec); err != nil {
		t.Errorf("expected multi-wildcard pattern to match, got: %v", err)
	}
}

// ── ValidateConfigParameters (input validation) ────────────────────────────

func TestValidateConfigParameters_ValidPatterns(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		params json.RawMessage
	}{
		{"empty", json.RawMessage(`{}`)},
		{"nil", nil},
		{"fixed only", json.RawMessage(`{"repo":"myrepo"}`)},
		{"wildcard only", json.RawMessage(`{"title":"*"}`)},
		{"valid pattern", json.RawMessage(`{"repo":{"$pattern":"supersuit-tech/*"}}`)},
		{"mixed", json.RawMessage(`{"repo":{"$pattern":"supersuit-tech/*"},"title":"*","label":"bug"}`)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if err := ValidateConfigParameters(tc.params); err != nil {
				t.Errorf("expected no error, got: %v", err)
			}
		})
	}
}

func TestValidateConfigParameters_RejectsPatternWithoutStar(t *testing.T) {
	t.Parallel()
	params := json.RawMessage(`{"tag":{"$pattern":"hello"}}`)
	err := ValidateConfigParameters(params)
	if err == nil {
		t.Fatal("expected error for $pattern without *")
	}
	cve, ok := err.(*ConfigValidationError)
	if !ok {
		t.Fatalf("expected ConfigValidationError, got %T", err)
	}
	if cve.Parameter != "tag" {
		t.Errorf("expected parameter 'tag', got %q", cve.Parameter)
	}
	if !strings.Contains(cve.Reason, "must contain at least one '*'") {
		t.Errorf("expected reason to mention missing wildcard, got: %s", cve.Reason)
	}
}

func TestValidateConfigParameters_RejectsEmptyPattern(t *testing.T) {
	t.Parallel()
	params := json.RawMessage(`{"tag":{"$pattern":""}}`)
	err := ValidateConfigParameters(params)
	if err == nil {
		t.Fatal("expected error for empty $pattern")
	}
}

// ── Defensive: $pattern without * at validation time ────────────────────────

func TestValidateParametersAgainstConfig_PatternWithoutStarExactMatch(t *testing.T) {
	t.Parallel()
	// If a malformed $pattern without "*" exists in the DB, it should be
	// treated as an exact string comparison (not compared as raw JSON object).
	config := json.RawMessage(`{"tag":{"$pattern":"hello"}}`)
	exec := json.RawMessage(`{"tag":"hello"}`)

	if err := ValidateParametersAgainstConfig(config, exec); err != nil {
		t.Errorf("expected $pattern without * to match exact string, got: %v", err)
	}
}

func TestValidateParametersAgainstConfig_PatternWithoutStarMismatch(t *testing.T) {
	t.Parallel()
	config := json.RawMessage(`{"tag":{"$pattern":"hello"}}`)
	exec := json.RawMessage(`{"tag":"world"}`)

	err := ValidateParametersAgainstConfig(config, exec)
	if err == nil {
		t.Fatal("expected error for mismatched value against $pattern without *")
	}
	cve, ok := err.(*ConfigValidationError)
	if !ok {
		t.Fatalf("expected ConfigValidationError, got %T", err)
	}
	if cve.Parameter != "tag" {
		t.Errorf("expected parameter 'tag', got %q", cve.Parameter)
	}
}

// ── Backward Compatibility Tests ────────────────────────────────────────────

func TestValidateParametersAgainstConfig_PlainStringWithStarIsFixedValue(t *testing.T) {
	t.Parallel()
	// A plain string containing "*" (not wrapped in $pattern) is a fixed value.
	// It requires exact match — the "*" has no special meaning.
	config := json.RawMessage(`{"tag":"v*-release"}`)
	exec := json.RawMessage(`{"tag":"v*-release"}`)

	if err := ValidateParametersAgainstConfig(config, exec); err != nil {
		t.Errorf("expected exact match for plain string with star, got: %v", err)
	}
}

func TestValidateParametersAgainstConfig_PlainStringWithStarRejectsGlobMatch(t *testing.T) {
	t.Parallel()
	// A plain string "v*-release" must NOT be treated as a glob pattern.
	// "v1.0-release" does not equal "v*-release", so it must be rejected.
	config := json.RawMessage(`{"tag":"v*-release"}`)
	exec := json.RawMessage(`{"tag":"v1.0-release"}`)

	err := ValidateParametersAgainstConfig(config, exec)
	if err == nil {
		t.Fatal("expected error: plain string with * should require exact match, not glob")
	}
}
