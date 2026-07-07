package db

import (
	"encoding/json"
	"testing"
)

func TestValidateParametersAgainstStructuredConfig_ComparisonOps(t *testing.T) {
	t.Parallel()

	baseConfig := func(op string, threshold any) json.RawMessage {
		thresholdJSON, _ := json.Marshal(threshold)
		return mustJSON(`{
			"$version": 2,
			"match": "any",
			"groups": [{
				"match": "all",
				"conditions": [
					{"field":"limit","op":"` + op + `","value":` + string(thresholdJSON) + `}
				]
			}]
		}`)
	}

	cases := []struct {
		name   string
		config json.RawMessage
		exec   string
		wantOK bool
	}{
		{
			name:   "lte accepts equal",
			config: baseConfig("lte", 20),
			exec:   `{"limit":20}`,
			wantOK: true,
		},
		{
			name:   "lte accepts below",
			config: baseConfig("lte", 20),
			exec:   `{"limit":10}`,
			wantOK: true,
		},
		{
			name:   "lte rejects above",
			config: baseConfig("lte", 20),
			exec:   `{"limit":21}`,
			wantOK: false,
		},
		{
			name:   "gte accepts equal",
			config: baseConfig("gte", 5),
			exec:   `{"limit":5}`,
			wantOK: true,
		},
		{
			name:   "gte rejects below",
			config: baseConfig("gte", 5),
			exec:   `{"limit":4}`,
			wantOK: false,
		},
		{
			name:   "lt rejects equal",
			config: baseConfig("lt", 20),
			exec:   `{"limit":20}`,
			wantOK: false,
		},
		{
			name:   "gt accepts above",
			config: baseConfig("gt", 0),
			exec:   `{"limit":1}`,
			wantOK: true,
		},
		{
			name:   "datetime lte",
			config: baseConfig("lte", "2026-07-07T00:00:00Z"),
			exec:   `{"since":"2026-06-01T00:00:00Z"}`,
			wantOK: true,
		},
		{
			name:   "datetime gt rejects equal",
			config: baseConfig("gt", "2026-07-07T00:00:00Z"),
			exec:   `{"since":"2026-07-07T00:00:00Z"}`,
			wantOK: false,
		},
		{
			name:   "missing param rejected",
			config: baseConfig("lte", 20),
			exec:   `{}`,
			wantOK: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateParametersAgainstConfig(tc.config, mustJSON(tc.exec), nil)
			if tc.wantOK && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !tc.wantOK && err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestValidateStructuredConstraintShape_ComparisonOps(t *testing.T) {
	t.Parallel()

	valid := mustJSON(`{
		"$version": 2,
		"match": "any",
		"groups": [{
			"match": "all",
			"conditions": [{"field":"limit","op":"lte","value":20}]
		}]
	}`)
	if err := ValidateStructuredConstraintShape(valid); err != nil {
		t.Fatalf("valid comparison constraint rejected: %v", err)
	}

	invalidWildcard := mustJSON(`{
		"$version": 2,
		"match": "any",
		"groups": [{
			"match": "all",
			"conditions": [{"field":"limit","op":"lte","value":"*"}]
		}]
	}`)
	if err := ValidateStructuredConstraintShape(invalidWildcard); err == nil {
		t.Fatal("expected wildcard threshold to be rejected")
	}
}
