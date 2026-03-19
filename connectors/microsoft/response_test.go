package microsoft

import (
	"encoding/json"
	"fmt"
	"net/http"
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

// TestMapGraphError_400_FileCorruptTryRepair ensures that Graph errors indicating
// server-side or transient infrastructure issues are NOT silently treated as
// client validation errors. They must be mapped to ExternalError so they are
// captured in Sentry and visible to operators.
func TestMapGraphError_400_FileCorruptTryRepair_ReturnsExternalError(t *testing.T) {
	t.Parallel()
	// Both capitalizations observed in the wild.
	for _, code := range []string{"fileCorruptTryRepair", "FileCorruptTryRepair"} {
		t.Run(code, func(t *testing.T) {
			t.Parallel()
			err := mapGraphError(http.StatusBadRequest, graphErrorBody(code, "The file is corrupt. Please try to repair the file."))
			if !connectors.IsExternalError(err) {
				t.Errorf("expected ExternalError for %q, got %T: %v", code, err, err)
			}
			if connectors.IsValidationError(err) {
				t.Errorf("must NOT be ValidationError for %q (would be swallowed by Sentry)", code)
			}
		})
	}
}

func TestMapGraphError_400_ServiceNotAvailable_ReturnsExternalError(t *testing.T) {
	t.Parallel()
	err := mapGraphError(http.StatusBadRequest, graphErrorBody("serviceNotAvailable", "service is unavailable"))
	if !connectors.IsExternalError(err) {
		t.Errorf("expected ExternalError, got %T: %v", err, err)
	}
}

func TestMapGraphError_400_GeneralException_ReturnsExternalError(t *testing.T) {
	t.Parallel()
	err := mapGraphError(http.StatusBadRequest, graphErrorBody("generalException", "unexpected error"))
	if !connectors.IsExternalError(err) {
		t.Errorf("expected ExternalError, got %T: %v", err, err)
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
	expected := "fileCorruptTryRepair"
	if !containsString(msg, expected) {
		t.Errorf("expected error message to contain %q, got: %q", expected, msg)
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && func() bool {
		for i := 0; i+len(substr) <= len(s); i++ {
			if s[i:i+len(substr)] == substr {
				return true
			}
		}
		return false
	}())
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
// map is not accidentally empty (regression guard for future refactors).
func TestServerSideGraphErrorCodesCompleteness(t *testing.T) {
	t.Parallel()
	required := []string{
		"fileCorruptTryRepair",
		"FileCorruptTryRepair",
		"serviceNotAvailable",
		"generalException",
	}
	for _, code := range required {
		if !serverSideGraphErrorCodes[code] {
			t.Errorf("serverSideGraphErrorCodes is missing required entry %q", code)
		}
	}
}

// Ensure the error message includes the Graph error code for debuggability.
func TestMapGraphError_400_FileCorruptTryRepair_MessageIncludesCode(t *testing.T) {
	t.Parallel()
	err := mapGraphError(http.StatusBadRequest, graphErrorBody("fileCorruptTryRepair", "workbook locked"))
	if err == nil {
		t.Fatal("expected non-nil error")
	}
	msg := fmt.Sprintf("%v", err)
	if !containsString(msg, "fileCorruptTryRepair") {
		t.Errorf("error message should contain the Graph error code, got: %q", msg)
	}
}
