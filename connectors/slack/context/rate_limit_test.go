package context

import (
	"errors"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestHandleRateLimit(t *testing.T) {
	t.Parallel()
	sc, ok := HandleRateLimit(&connectors.RateLimitError{Message: "slow"})
	if !ok || sc == nil || sc.ContextScope != ScopeMetadataOnly {
		t.Fatalf("HandleRateLimit = %v, %v", sc, ok)
	}
	sc, ok = HandleRateLimit(errors.New("other"))
	if ok || sc != nil {
		t.Fatalf("expected no degrade: %v %v", sc, ok)
	}
}

func TestIsRateLimit(t *testing.T) {
	t.Parallel()
	if !IsRateLimit(&connectors.RateLimitError{}) {
		t.Fatal("expected rate limit")
	}
	if IsRateLimit(errors.New("x")) {
		t.Fatal("expected false")
	}
}
