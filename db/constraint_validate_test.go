package db

import (
	"encoding/json"
	"testing"
)

func TestIsSemanticWildcard(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		raw  string
		want bool
	}{
		{name: "bare star", raw: `"*"`, want: true},
		{name: "pattern star", raw: `{"$pattern":"*"}`, want: true},
		{name: "pattern double star", raw: `{"$pattern":"**"}`, want: true},
		{name: "pattern triple star", raw: `{"$pattern":"***"}`, want: true},
		{name: "pattern prefix", raw: `{"$pattern":"a*"}`, want: false},
		{name: "pattern email", raw: `{"$pattern":"*@example.com"}`, want: false},
		{name: "fixed string", raw: `"foo"`, want: false},
		{name: "empty string", raw: `""`, want: false},
		{name: "number", raw: `10`, want: false},
		{name: "boolean", raw: `true`, want: false},
		{name: "null", raw: `null`, want: false},
		{name: "empty object", raw: `{}`, want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := IsSemanticWildcard(json.RawMessage(tc.raw))
			if got != tc.want {
				t.Fatalf("IsSemanticWildcard(%s) = %v, want %v", tc.raw, got, tc.want)
			}
		})
	}
}

func TestIsWildcardStillBareStarOnly(t *testing.T) {
	t.Parallel()
	if !IsWildcard(json.RawMessage(`"*"`)) {
		t.Fatal("expected bare * to be a wildcard")
	}
	if IsWildcard(json.RawMessage(`{"$pattern":"*"}`)) {
		t.Fatal("IsWildcard must not treat $pattern wrappers as the bare wildcard")
	}
	if IsWildcard(json.RawMessage(`"**"`)) {
		t.Fatal("IsWildcard must not treat ** as the bare wildcard")
	}
}
