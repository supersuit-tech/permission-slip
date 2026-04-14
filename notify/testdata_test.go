package notify

import (
	"encoding/json"
	"time"
)

// testApproval returns a standard Approval fixture for use across all
// notify tests. Individual tests that need specific Action/Context JSON
// should create their own inline fixtures.
func testApproval() Approval {
	return Approval{
		ApprovalID:  "appr_test123",
		AgentID:     42,
		AgentName:   "Test Agent",
		Action:      json.RawMessage(`{"type":"email.send","description":"Send welcome email"}`),
		Context:     json.RawMessage(`{"description":"test"}`),
		ApprovalURL: "https://app.example.com/approve/appr_test123",
		ExpiresAt:   time.Now().Add(5 * time.Minute),
		CreatedAt:   time.Now(),
	}
}

// testStandingExecutionApproval returns a standard standing execution
// Approval fixture for use across notify tests.
func testStandingExecutionApproval() Approval {
	return Approval{
		ApprovalID:  "appr_standing_001",
		AgentID:     42,
		AgentName:   "Deploy Bot",
		Action:      json.RawMessage(`{"type":"github.issues.create","parameters":{"repo":"acme/app","title":"Deploy v2.1"}}`),
		Context:     json.RawMessage(`{}`),
		ApprovalURL: "https://app.example.com/activity",
		CreatedAt:   time.Date(2026, 3, 14, 12, 0, 0, 0, time.UTC),
		Type:        NotificationTypeStandingExecution,
	}
}

// testRecipient returns a standard Recipient fixture with email and phone.
func testRecipient() Recipient {
	email := "alice@example.com"
	phone := "+15559998888"
	return Recipient{
		UserID:   "user-001",
		Username: "alice",
		Email:    &email,
		Phone:    &phone,
	}
}
