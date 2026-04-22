package context

import (
	"encoding/json"
	"testing"
)

func TestSlackNullableText_UnmarshalJSON(t *testing.T) {
	t.Parallel()
	var got slackNullableText
	if err := json.Unmarshal([]byte(`null`), &got); err != nil {
		t.Fatal(err)
	}
	if got.String() != "" {
		t.Errorf("null → %q, want empty", got.String())
	}
}
