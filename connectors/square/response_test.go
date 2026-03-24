package square

import (
	"net/http"
	"strings"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestFormatErrorMessage_FieldAndCode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		body string
		want string
	}{
		{
			name: "detail with code",
			body: `{"errors":[{"category":"INVALID_REQUEST_ERROR","code":"MISSING_REQUIRED_PARAMETER","detail":"Missing required parameter: location_id"}]}`,
			want: "Missing required parameter: location_id (MISSING_REQUIRED_PARAMETER)",
		},
		{
			name: "detail with field and code",
			body: `{"errors":[{"category":"INVALID_REQUEST_ERROR","code":"INVALID_VALUE","detail":"Invalid value","field":"amount_money.amount"}]}`,
			want: "amount_money.amount: Invalid value (INVALID_VALUE)",
		},
		{
			name: "code only no detail",
			body: `{"errors":[{"category":"API_ERROR","code":"INTERNAL_SERVER_ERROR"}]}`,
			want: "INTERNAL_SERVER_ERROR",
		},
		{
			name: "multiple errors joined",
			body: `{"errors":[{"category":"INVALID_REQUEST_ERROR","code":"MISSING_REQUIRED_PARAMETER","detail":"Missing: location_id"},{"category":"INVALID_REQUEST_ERROR","code":"MISSING_REQUIRED_PARAMETER","detail":"Missing: line_items"}]}`,
			want: "Missing: location_id (MISSING_REQUIRED_PARAMETER); Missing: line_items (MISSING_REQUIRED_PARAMETER)",
		},
		{
			name: "unparseable body falls back to raw",
			body: `not json`,
			want: "not json",
		},
		{
			name: "empty errors array falls back to raw",
			body: `{"errors":[]}`,
			want: `{"errors":[]}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw := []byte(tt.body)
			parsed := parseSquareErrors(raw)
			got := formatErrorMessage(parsed, raw)
			if got != tt.want {
				t.Errorf("formatErrorMessage() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTruncateBody(t *testing.T) {
	t.Parallel()

	t.Run("short body unchanged", func(t *testing.T) {
		body := []byte("short error")
		got := truncateBody(body)
		if got != "short error" {
			t.Errorf("truncateBody() = %q, want %q", got, "short error")
		}
	})

	t.Run("long body truncated", func(t *testing.T) {
		body := []byte(strings.Repeat("x", 1000))
		got := truncateBody(body)
		if len(got) > maxErrorBodyLen+20 {
			t.Errorf("truncateBody() returned %d chars, want at most %d", len(got), maxErrorBodyLen+20)
		}
		if !strings.HasSuffix(got, "...(truncated)") {
			t.Errorf("truncateBody() should end with truncation marker, got %q", got[len(got)-20:])
		}
	})

	t.Run("exactly at limit unchanged", func(t *testing.T) {
		body := []byte(strings.Repeat("x", maxErrorBodyLen))
		got := truncateBody(body)
		if strings.Contains(got, "truncated") {
			t.Error("truncateBody() should not truncate body at exactly the limit")
		}
	})
}

func TestParsedHasCategory(t *testing.T) {
	t.Parallel()

	errs := []squareError{
		{Category: "AUTHENTICATION_ERROR", Code: "UNAUTHORIZED"},
		{Category: "API_ERROR", Code: "INTERNAL_SERVER_ERROR"},
	}

	if !parsedHasCategory(errs, "AUTHENTICATION_ERROR") {
		t.Error("parsedHasCategory(AUTHENTICATION_ERROR) = false, want true")
	}
	if parsedHasCategory(errs, "RATE_LIMIT_ERROR") {
		t.Error("parsedHasCategory(RATE_LIMIT_ERROR) = true, want false")
	}
	if parsedHasCategory(nil, "AUTHENTICATION_ERROR") {
		t.Error("parsedHasCategory(nil, ...) = true, want false")
	}
}

func TestCheckResponse_Success(t *testing.T) {
	t.Parallel()
	for _, code := range []int{200, 201, 204} {
		if err := checkResponse(code, http.Header{}, nil); err != nil {
			t.Errorf("checkResponse(%d) = %v, want nil", code, err)
		}
	}
}

func TestCheckResponse_ServerError(t *testing.T) {
	t.Parallel()
	body := []byte(`{"errors":[{"category":"API_ERROR","code":"INTERNAL_SERVER_ERROR","detail":"An unexpected error occurred"}]}`)
	err := checkResponse(500, http.Header{}, body)
	if err == nil {
		t.Fatal("checkResponse(500) = nil, want error")
	}
	if !connectors.IsExternalError(err) {
		t.Errorf("expected ExternalError, got %T: %v", err, err)
	}
}
