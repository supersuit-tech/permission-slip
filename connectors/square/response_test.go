package square

import (
	"net/http"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestExtractErrorMessage_FieldAndCode(t *testing.T) {
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
			got := extractErrorMessage([]byte(tt.body))
			if got != tt.want {
				t.Errorf("extractErrorMessage() = %q, want %q", got, tt.want)
			}
		})
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
