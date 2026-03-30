package square

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestCreateBooking_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/bookings" {
			t.Errorf("path = %s, want /bookings", r.URL.Path)
		}

		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]json.RawMessage
		if err := json.Unmarshal(body, &reqBody); err != nil {
			t.Fatalf("unmarshaling request body: %v", err)
		}

		if _, ok := reqBody["idempotency_key"]; !ok {
			t.Error("missing idempotency_key in request body")
		}

		var booking map[string]json.RawMessage
		json.Unmarshal(reqBody["booking"], &booking)

		var locationID string
		json.Unmarshal(booking["location_id"], &locationID)
		if locationID != "L123" {
			t.Errorf("location_id = %q, want %q", locationID, "L123")
		}

		var startAt string
		json.Unmarshal(booking["start_at"], &startAt)
		if startAt != "2024-03-15T14:30:00Z" {
			t.Errorf("start_at = %q, want %q", startAt, "2024-03-15T14:30:00Z")
		}

		// Verify appointment_segments structure.
		var segments []map[string]json.RawMessage
		json.Unmarshal(booking["appointment_segments"], &segments)
		if len(segments) != 1 {
			t.Fatalf("appointment_segments length = %d, want 1", len(segments))
		}

		var svcID string
		json.Unmarshal(segments[0]["service_variation_id"], &svcID)
		if svcID != "SVC123" {
			t.Errorf("service_variation_id = %q, want %q", svcID, "SVC123")
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"booking": map[string]any{
				"id":       "BKG123",
				"status":   "ACCEPTED",
				"start_at": "2024-03-15T14:30:00Z",
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["square.create_booking"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType: "square.create_booking",
		Parameters: json.RawMessage(`{
			"location_id": "L123",
			"start_at": "2024-03-15T14:30:00Z",
			"service_variation_id": "SVC123"
		}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["id"] != "BKG123" {
		t.Errorf("booking id = %v, want BKG123", data["id"])
	}
	if data["status"] != "ACCEPTED" {
		t.Errorf("booking status = %v, want ACCEPTED", data["status"])
	}
}

func TestCreateBooking_WithOptionalFields(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]json.RawMessage
		json.Unmarshal(body, &reqBody)

		var booking map[string]json.RawMessage
		json.Unmarshal(reqBody["booking"], &booking)

		var customerID string
		json.Unmarshal(booking["customer_id"], &customerID)
		if customerID != "CUST123" {
			t.Errorf("customer_id = %q, want %q", customerID, "CUST123")
		}

		var customerNote string
		json.Unmarshal(booking["customer_note"], &customerNote)
		if customerNote != "Allergic to latex" {
			t.Errorf("customer_note = %q, want %q", customerNote, "Allergic to latex")
		}

		var segments []map[string]json.RawMessage
		json.Unmarshal(booking["appointment_segments"], &segments)

		var teamID string
		json.Unmarshal(segments[0]["team_member_id"], &teamID)
		if teamID != "TM123" {
			t.Errorf("team_member_id = %q, want %q", teamID, "TM123")
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"booking": map[string]any{"id": "BKG456"},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["square.create_booking"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType: "square.create_booking",
		Parameters: json.RawMessage(`{
			"location_id": "L123",
			"customer_id": "CUST123",
			"start_at": "2024-03-15T14:30:00Z",
			"service_variation_id": "SVC123",
			"team_member_id": "TM123",
			"customer_note": "Allergic to latex"
		}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}

func TestCreateBooking_MissingParams(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["square.create_booking"]

	tests := []struct {
		name   string
		params string
	}{
		{name: "missing location_id", params: `{"start_at": "2024-03-15T14:30:00Z", "service_variation_id": "SVC123"}`},
		{name: "missing start_at", params: `{"location_id": "L123", "service_variation_id": "SVC123"}`},
		{name: "invalid start_at format", params: `{"location_id": "L123", "start_at": "next tuesday", "service_variation_id": "SVC123"}`},
		{name: "missing service_variation_id", params: `{"location_id": "L123", "start_at": "2024-03-15T14:30:00Z"}`},
		{name: "invalid JSON", params: `{invalid}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "square.create_booking",
				Parameters:  json.RawMessage(tt.params),
				Credentials: validCreds(),
			})
			if err == nil {
				t.Fatal("Execute() expected error, got nil")
			}
			if !connectors.IsValidationError(err) {
				t.Errorf("expected ValidationError, got %T: %v", err, err)
			}
		})
	}
}

func TestCreateBooking_APIError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{
			"errors": []map[string]string{
				{"category": "INVALID_REQUEST_ERROR", "code": "INVALID_VALUE", "detail": "Invalid start_at time"},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["square.create_booking"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType: "square.create_booking",
		Parameters: json.RawMessage(`{
			"location_id": "L123",
			"start_at": "2024-03-15T14:30:00Z",
			"service_variation_id": "SVC123"
		}`),
		Credentials: validCreds(),
	})

	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestCreateBooking_Timeout(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(100 * time.Millisecond)
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(t.Context(), 1*time.Millisecond)
	defer cancel()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["square.create_booking"]

	_, err := action.Execute(ctx, connectors.ActionRequest{
		ActionType: "square.create_booking",
		Parameters: json.RawMessage(`{
			"location_id": "L123",
			"start_at": "2024-03-15T14:30:00Z",
			"service_variation_id": "SVC123"
		}`),
		Credentials: validCreds(),
	})

	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsTimeoutError(err) {
		t.Errorf("expected TimeoutError, got %T: %v", err, err)
	}
}
