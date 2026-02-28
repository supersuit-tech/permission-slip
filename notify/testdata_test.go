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
