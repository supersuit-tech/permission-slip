package ticketmaster

import (
	"net/http"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestCheckResponse_Success(t *testing.T) {
	t.Parallel()
	if err := checkResponse(200, nil, nil); err != nil {
		t.Errorf("checkResponse(200) = %v, want nil", err)
	}
}

func TestCheckResponse_RateLimit(t *testing.T) {
	t.Parallel()
	h := http.Header{"Retry-After": []string{"5"}}
	err := checkResponse(429, h, []byte(`{"fault":{"faultstring":"limit"}}`))
	if err == nil {
		t.Fatal("expected error")
	}
	if !connectors.IsRateLimitError(err) {
		t.Fatalf("got %T, want RateLimitError", err)
	}
}

func TestCheckResponse_Auth(t *testing.T) {
	t.Parallel()
	err := checkResponse(401, nil, []byte(`{"fault":{"faultstring":"bad key"}}`))
	if !connectors.IsAuthError(err) {
		t.Fatalf("got %T, want AuthError", err)
	}
}

func TestCheckResponse_Validation(t *testing.T) {
	t.Parallel()
	err := checkResponse(400, nil, []byte(`{}`))
	if !connectors.IsValidationError(err) {
		t.Fatalf("got %T, want ValidationError", err)
	}
}

func TestCheckResponse_NotFound(t *testing.T) {
	t.Parallel()
	err := checkResponse(404, nil, []byte(`{}`))
	if !connectors.IsValidationError(err) {
		t.Fatalf("got %T, want ValidationError", err)
	}
}

func TestCheckResponse_External(t *testing.T) {
	t.Parallel()
	err := checkResponse(500, nil, []byte(`err`))
	if !connectors.IsExternalError(err) {
		t.Fatalf("got %T, want ExternalError", err)
	}
}
