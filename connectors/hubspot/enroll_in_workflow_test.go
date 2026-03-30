package hubspot

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestEnrollInWorkflow_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/automation/v4/flows/100/enrollments" {
			t.Errorf("expected path /automation/v4/flows/100/enrollments, got %s", r.URL.Path)
		}

		var body enrollmentRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if body.ObjectID != "501" {
			t.Errorf("expected objectId 501, got %q", body.ObjectID)
		}
		if body.ObjectType != "CONTACT" {
			t.Errorf("expected objectType CONTACT, got %q", body.ObjectType)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(enrollmentResponse{
			ID:         "enroll-1",
			Status:     "ENROLLED",
			ObjectID:   "501",
			ObjectType: "CONTACT",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &enrollInWorkflowAction{conn: conn}

	params, _ := json.Marshal(enrollInWorkflowParams{
		FlowID:    "100",
		ContactID: "501",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "hubspot.enroll_in_workflow",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data enrollmentResponse
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data.Status != "ENROLLED" {
		t.Errorf("expected status ENROLLED, got %q", data.Status)
	}
}

func TestEnrollInWorkflow_MissingFlowID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &enrollInWorkflowAction{conn: conn}

	params, _ := json.Marshal(map[string]any{"contact_id": "501"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "hubspot.enroll_in_workflow",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing flow_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestEnrollInWorkflow_MissingContactID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &enrollInWorkflowAction{conn: conn}

	params, _ := json.Marshal(map[string]any{"flow_id": "100"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "hubspot.enroll_in_workflow",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing contact_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestEnrollInWorkflow_NonNumericFlowID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &enrollInWorkflowAction{conn: conn}

	params, _ := json.Marshal(enrollInWorkflowParams{
		FlowID:    "../../admin",
		ContactID: "501",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "hubspot.enroll_in_workflow",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for non-numeric flow_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestEnrollInWorkflow_NonNumericContactID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &enrollInWorkflowAction{conn: conn}

	params, _ := json.Marshal(enrollInWorkflowParams{
		FlowID:    "100",
		ContactID: "abc",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "hubspot.enroll_in_workflow",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for non-numeric contact_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}
