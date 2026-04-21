package context

import (
	"strings"
	"testing"
)

func TestTruncateBody(t *testing.T) {
	t.Parallel()
	short := "hello"
	got, tr := TruncateBody(short)
	if got != short || tr {
		t.Fatalf("short text: got %q truncated=%v", got, tr)
	}
	long := strings.Repeat("x", maxBodyRunes+50)
	got, tr = TruncateBody(long)
	if !tr || len([]rune(got)) != maxBodyRunes {
		t.Fatalf("truncate: rune len=%d truncated=%v", len([]rune(got)), tr)
	}
}
