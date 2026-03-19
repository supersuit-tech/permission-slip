package microsoft

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func graphErrorBody(code, message string) []byte {
	b, _ := json.Marshal(map[string]any{
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
	})
	return b
}

func TestMapGraphError_401_ReturnsAuthError(t *testing.T) {
	t.Parallel()
	err := mapGraphError(http.StatusUnauthorized, graphErrorBody("InvalidAuthenticationToken", "token expired"))
	if !connectors.IsAuthError(err) {
		t.Errorf("expected AuthError, got %T: %v", err, err)
	}
}

func TestMapGraphError_403_ReturnsAuthError(t *testing.T) {
	t.Parallel()
	err := mapGraphError(http.StatusForbidden, graphErrorBody("accessDenied", "insufficient scopes"))
	if !connectors.IsAuthError(err) {
		t.Errorf("expected AuthError, got %T: %v", err, err)
	}
}

func TestMapGraphError_404_ReturnsExternalError(t *testing.T) {
	t.Parallel()
	err := mapGraphError(http.StatusNotFound, graphErrorBody("itemNotFound", "resource not found"))
	if !connectors.IsExternalError(err) {
		t.Errorf("expected ExternalError, got %T: %v", err, err)
	}
}

func TestMapGraphError_400_ClientError_ReturnsValidationError(t *testing.T) {
	t.Parallel()
	err := mapGraphError(http.StatusBadRequest, graphErrorBody("invalidRequest", "bad field"))
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

// TestMapGraphError_400_ServerSideCodes ensures that all Graph error codes
// indicating server-side or transient infrastructure issues are mapped to
// ExternalError (not ValidationError) even when the HTTP status is 400.
// Covers all codes in serverSideGraphErrorCodes, including variant
// capitalisations observed in the wild.
func TestMapGraphError_400_ServerSideCodes_ReturnExternalError(t *testing.T) {
	t.Parallel()
	codes := []string{
		// Both capitalisations observed in the wild.
		"fileCorruptTryRepair",
		"FileCorruptTryRepair",
		// Standard codes.
		"serviceNotAvailable",
		"generalException",
		"notSupported",
		"resourceModified",
		"lockMismatch",
		"editModeRequired",
	}
	for _, code := range codes {
		code := code
		t.Run(code, func(t *testing.T) {
			t.Parallel()
			err := mapGraphError(http.StatusBadRequest, graphErrorBody(code, "server-side error"))
			if !connectors.IsExternalError(err) {
				t.Errorf("expected ExternalError for %q, got %T: %v", code, err, err)
			}
			if connectors.IsValidationError(err) {
				t.Errorf("must NOT be ValidationError for %q (would be swallowed by Sentry)", code)
			}
		})
	}
}

func TestMapGraphError_500_ReturnsExternalError(t *testing.T) {
	t.Parallel()
	err := mapGraphError(http.StatusInternalServerError, graphErrorBody("generalException", "internal error"))
	if !connectors.IsExternalError(err) {
		t.Errorf("expected ExternalError, got %T: %v", err, err)
	}
}

func TestMapGraphError_ErrorMessageContainsCode(t *testing.T) {
	t.Parallel()
	err := mapGraphError(http.StatusBadRequest, graphErrorBody("fileCorruptTryRepair", "the workbook is locked"))
	if err == nil {
		t.Fatal("expected non-nil error")
	}
	msg := err.Error()
	if msg == "" {
		t.Error("expected non-empty error message")
	}
	// The error message should surface the Graph error code so operators
	// can identify the root cause without digging into raw logs.
	if !strings.Contains(msg, "fileCorruptTryRepair") {
		t.Errorf("expected error message to contain %q, got: %q", "fileCorruptTryRepair", msg)
	}
}

func TestMapGraphError_MalformedBody_FallsBackToRawBody(t *testing.T) {
	t.Parallel()
	err := mapGraphError(http.StatusInternalServerError, []byte("not valid json"))
	if err == nil {
		t.Fatal("expected non-nil error")
	}
	if !connectors.IsExternalError(err) {
		t.Errorf("expected ExternalError, got %T: %v", err, err)
	}
}

// TestServerSideGraphErrorCodesCompleteness ensures the serverSideGraphErrorCodes
// map contains all expected entries (regression guard for future refactors).
// Keys are stored lowercase; the lookup normalises via strings.ToLower.
func TestServerSideGraphErrorCodesCompleteness(t *testing.T) {
	t.Parallel()
	required := []string{
		"filecorrupttryrepair",
		"servicenotavailable",
		"generalexception",
		"notsupported",
		"resourcemodified",
		"lockmismatch",
		"editmoderequired",
	}
	for _, code := range required {
		if !serverSideGraphErrorCodes[code] {
			t.Errorf("serverSideGraphErrorCodes is missing required entry %q", code)
		}
	}
}
