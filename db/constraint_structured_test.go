package db

import (
	"encoding/json"
	"testing"
)

func mustJSON(s string) json.RawMessage {
	return json.RawMessage(s)
}

func TestParseStructuredConstraints_FlatMapAdapter(t *testing.T) {
	t.Parallel()
	raw := mustJSON(`{"repo":"supersuit-tech/ci-actions","title":"*"}`)
	sc, err := ParseStructuredConstraints(raw)
	if err != nil {
		t.Fatal(err)
	}
	if sc.Version != ConstraintVersionV2 {
		t.Fatalf("version = %d, want %d", sc.Version, ConstraintVersionV2)
	}
	if len(sc.Groups) != 1 || len(sc.Groups[0].Conditions) != 2 {
		t.Fatalf("groups: %+v", sc.Groups)
	}
}

func TestValidateParametersAgainstConfig_FlatBackwardCompat(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		config string
		exec   string
		wantOK bool
	}{
		{
			name:   "fixed match",
			config: `{"repo":"supersuit-tech/ci-actions"}`,
			exec:   `{"repo":"supersuit-tech/ci-actions"}`,
			wantOK: true,
		},
		{
			name:   "fixed mismatch",
			config: `{"repo":"supersuit-tech/ci-actions"}`,
			exec:   `{"repo":"other/repo"}`,
			wantOK: false,
		},
		{
			name:   "wildcard title",
			config: `{"repo":"supersuit-tech/permission-slip","title":"*"}`,
			exec:   `{"repo":"supersuit-tech/permission-slip","title":"anything"}`,
			wantOK: true,
		},
		{
			name:   "pattern match",
			config: `{"repo":{"$pattern":"supersuit-tech/*"}}`,
			exec:   `{"repo":"supersuit-tech/foo"}`,
			wantOK: true,
		},
		{
			name:   "extra param rejected",
			config: `{"repo":"foo"}`,
			exec:   `{"repo":"foo","extra":"bar"}`,
			wantOK: false,
		},
		{
			name:   "missing fixed rejected",
			config: `{"repo":"foo"}`,
			exec:   `{}`,
			wantOK: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateParametersAgainstConfig(mustJSON(tc.config), mustJSON(tc.exec), nil)
			if tc.wantOK && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !tc.wantOK && err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestValidateParametersAgainstStructuredConfig_Negation(t *testing.T) {
	t.Parallel()
	config := mustJSON(`{
		"$version": 2,
		"match": "any",
		"groups": [{
			"match": "all",
			"conditions": [
				{"field":"recipient","op":"any_of","values":[{"$pattern":"*@acme.com"},"boss@partner.com"]},
				{"field":"recipient","op":"none_of","values":["ceo@acme.com"]}
			]
		}]
	}`)
	sc, err := ParseStructuredConstraints(config)
	if err != nil {
		t.Fatal(err)
	}

	okExec := mustJSON(`{"recipient":"team@acme.com"}`)
	if err := ValidateParametersAgainstStructuredConfig(sc, okExec, nil); err != nil {
		t.Fatalf("expected match: %v", err)
	}

	deniedExec := mustJSON(`{"recipient":"ceo@acme.com"}`)
	if err := ValidateParametersAgainstStructuredConfig(sc, deniedExec, nil); err == nil {
		t.Fatal("expected deny for ceo")
	}

	wrongDomain := mustJSON(`{"recipient":"x@other.com"}`)
	if err := ValidateParametersAgainstStructuredConfig(sc, wrongDomain, nil); err == nil {
		t.Fatal("expected allow-list miss")
	}
}

func TestValidateParametersAgainstStructuredConfig_OrGroups(t *testing.T) {
	t.Parallel()
	config := mustJSON(`{
		"$version": 2,
		"match": "any",
		"groups": [
			{
				"match": "all",
				"conditions": [
					{"field":"repo","op":"matches","value":"supersuit-tech/webapp"},
					{"field":"title","op":"matches","value":"bug"}
				]
			},
			{
				"match": "all",
				"conditions": [
					{"field":"channel","op":"matches","value":"#incidents"}
				]
			}
		]
	}`)
	sc, err := ParseStructuredConstraints(config)
	if err != nil {
		t.Fatal(err)
	}

	if err := ValidateParametersAgainstStructuredConfig(sc, mustJSON(`{"repo":"supersuit-tech/webapp","title":"bug"}`), nil); err != nil {
		t.Fatalf("group1: %v", err)
	}
	if err := ValidateParametersAgainstStructuredConfig(sc, mustJSON(`{"channel":"#incidents"}`), nil); err != nil {
		t.Fatalf("group2: %v", err)
	}
	if err := ValidateParametersAgainstStructuredConfig(sc, mustJSON(`{"repo":"other","title":"x"}`), nil); err == nil {
		t.Fatal("expected no match")
	}
}

func TestValidateParametersAgainstStructuredConfig_RecipientNoneOf(t *testing.T) {
	t.Parallel()
	config := mustJSON(`{
		"$version": 2,
		"match": "any",
		"groups": [{
			"match": "all",
			"conditions": [
				{"field":"$meta.to","op":"none_of","values":["ceo@acme.com"]},
				{"field":"$meta.to","op":"matches","value":"*"}
			]
		}]
	}`)
	sc, err := ParseStructuredConstraints(config)
	if err != nil {
		t.Fatal(err)
	}
	meta := mustJSON(`{"to":["team@acme.com","boss@acme.com"]}`)
	if err := ValidateParametersAgainstStructuredConfig(sc, mustJSON(`{}`), meta); err != nil {
		t.Fatalf("expected pass: %v", err)
	}
	metaDenied := mustJSON(`{"to":["team@acme.com","ceo@acme.com"]}`)
	if err := ValidateParametersAgainstStructuredConfig(sc, mustJSON(`{}`), metaDenied); err == nil {
		t.Fatal("expected ceo in recipients to fail none_of")
	}
}

func TestStructuredConstraintsHasNonWildcard(t *testing.T) {
	t.Parallel()
	allWild := StructuredConstraints{
		Groups: []ConstraintGroup{{
			Conditions: []ConstraintCondition{
				{Field: "x", Op: OpMatches, Value: mustJSON(`"*"`)},
			},
		}},
	}
	if StructuredConstraintsHasNonWildcard(allWild) {
		t.Fatal("expected all wildcard")
	}
	fixed := StructuredConstraints{
		Groups: []ConstraintGroup{{
			Conditions: []ConstraintCondition{
				{Field: "x", Op: OpMatches, Value: mustJSON(`"foo"`)},
			},
		}},
	}
	if !StructuredConstraintsHasNonWildcard(fixed) {
		t.Fatal("expected non-wildcard")
	}
}

func TestValidateStructuredConstraintShape_RejectsEmptyGroup(t *testing.T) {
	t.Parallel()
	raw := mustJSON(`{"$version":2,"match":"any","groups":[{"match":"all","conditions":[]}]}`)
	if err := ValidateStructuredConstraintShape(raw); err == nil {
		t.Fatal("expected empty group rejection")
	}
}

func TestFlatAndStructuredEquivalence(t *testing.T) {
	t.Parallel()
	flat := mustJSON(`{"repo":"supersuit-tech/ci-actions","title":"*"}`)
	structured := mustJSON(`{
		"$version": 2,
		"match": "any",
		"groups": [{
			"match": "all",
			"conditions": [
				{"field":"repo","op":"matches","value":"supersuit-tech/ci-actions"},
				{"field":"title","op":"matches","value":"*"}
			]
		}]
	}`)
	execCases := []string{
		`{"repo":"supersuit-tech/ci-actions","title":"hello"}`,
		`{"repo":"supersuit-tech/ci-actions"}`,
		`{"repo":"wrong","title":"x"}`,
	}
	for _, exec := range execCases {
		flatErr := ValidateParametersAgainstConfig(flat, mustJSON(exec), nil)
		structErr := ValidateParametersAgainstConfig(structured, mustJSON(exec), nil)
		if (flatErr == nil) != (structErr == nil) {
			t.Fatalf("exec %s: flat err=%v structured err=%v", exec, flatErr, structErr)
		}
	}
}
