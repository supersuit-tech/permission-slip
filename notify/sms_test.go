package notify

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/sns"
)

// mockSNSPublisher records Publish calls and returns a configurable error.
type mockSNSPublisher struct {
	input *sns.PublishInput
	err   error
}

func (m *mockSNSPublisher) Publish(ctx context.Context, params *sns.PublishInput, _ ...func(*sns.Options)) (*sns.PublishOutput, error) {
	m.input = params
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if m.err != nil {
		return nil, m.err
	}
	return &sns.PublishOutput{}, nil
}

func TestSMSSender_Name(t *testing.T) {
	t.Parallel()
	s := NewSMSSender(SMSConfig{}, &mockSNSPublisher{})
	if s.Name() != "sms" {
		t.Fatalf("expected name %q, got %q", "sms", s.Name())
	}
}

func TestSMSSender_Send_Success(t *testing.T) {
	t.Parallel()

	mock := &mockSNSPublisher{}
	sender := NewSMSSender(SMSConfig{
		Region:            "us-east-1",
		OriginationNumber: "+15550001111",
	}, mock)

	phone := "+15559998888"
	err := sender.Send(context.Background(), testApproval(), Recipient{
		UserID:   "user-001",
		Username: "alice",
		Phone:    &phone,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if mock.input == nil {
		t.Fatal("expected Publish to be called")
	}

	// Verify phone number.
	if *mock.input.PhoneNumber != "+15559998888" {
		t.Errorf("expected PhoneNumber %q, got %q", "+15559998888", *mock.input.PhoneNumber)
	}

	// Verify message body is present.
	if mock.input.Message == nil || *mock.input.Message == "" {
		t.Error("expected non-empty Message")
	}

	// Verify Transactional SMS type.
	attr, ok := mock.input.MessageAttributes["AWS.SNS.SMS.SMSType"]
	if !ok {
		t.Error("expected AWS.SNS.SMS.SMSType attribute")
	} else if *attr.StringValue != "Transactional" {
		t.Errorf("expected SMSType %q, got %q", "Transactional", *attr.StringValue)
	}

	// Verify origination number attribute.
	origAttr, ok := mock.input.MessageAttributes["AWS.MM.SMS.OriginationNumber"]
	if !ok {
		t.Error("expected AWS.MM.SMS.OriginationNumber attribute")
	} else if *origAttr.StringValue != "+15550001111" {
		t.Errorf("expected origination number %q, got %q", "+15550001111", *origAttr.StringValue)
	}
}

func TestSMSSender_Send_WithSenderID(t *testing.T) {
	t.Parallel()

	mock := &mockSNSPublisher{}
	sender := NewSMSSender(SMSConfig{
		Region:   "eu-west-1",
		SenderID: "PermSlip",
	}, mock)

	phone := "+447700900000"
	err := sender.Send(context.Background(), testApproval(), Recipient{
		UserID: "user-001",
		Phone:  &phone,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	attr, ok := mock.input.MessageAttributes["AWS.SNS.SMS.SenderID"]
	if !ok {
		t.Error("expected AWS.SNS.SMS.SenderID attribute")
	} else if *attr.StringValue != "PermSlip" {
		t.Errorf("expected SenderID %q, got %q", "PermSlip", *attr.StringValue)
	}

	// No origination number — attribute should be absent.
	if _, ok := mock.input.MessageAttributes["AWS.MM.SMS.OriginationNumber"]; ok {
		t.Error("expected no origination number attribute when not configured")
	}
}

func TestSMSSender_Send_NoPhone(t *testing.T) {
	t.Parallel()

	sender := NewSMSSender(SMSConfig{Region: "us-east-1"}, &mockSNSPublisher{})

	// nil phone — should skip without error.
	err := sender.Send(context.Background(), testApproval(), Recipient{
		UserID:   "user-001",
		Username: "alice",
		Phone:    nil,
	})
	if err != nil {
		t.Fatalf("expected nil error for nil phone, got: %v", err)
	}

	// Empty phone — should skip without error.
	empty := ""
	err = sender.Send(context.Background(), testApproval(), Recipient{
		UserID:   "user-001",
		Username: "alice",
		Phone:    &empty,
	})
	if err != nil {
		t.Fatalf("expected nil error for empty phone, got: %v", err)
	}
}

func TestSMSSender_Send_SNSError(t *testing.T) {
	t.Parallel()

	mock := &mockSNSPublisher{err: errors.New("SNS: throttling exception")}
	sender := NewSMSSender(SMSConfig{Region: "us-east-1"}, mock)

	phone := "+15559998888"
	err := sender.Send(context.Background(), testApproval(), Recipient{
		UserID: "user-001",
		Phone:  &phone,
	})

	if err == nil {
		t.Fatal("expected error for SNS failure")
	}
	if !strings.Contains(err.Error(), "SNS publish failed") {
		t.Errorf("expected error to mention SNS publish, got: %v", err)
	}
}

func TestSMSSender_ContextCancellation(t *testing.T) {
	t.Parallel()

	// No pre-set error — the mock checks ctx.Err() to verify context propagation.
	mock := &mockSNSPublisher{}
	sender := NewSMSSender(SMSConfig{Region: "us-east-1"}, mock)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	phone := "+15559998888"
	err := sender.Send(ctx, testApproval(), Recipient{
		UserID: "user-001",
		Phone:  &phone,
	})

	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled error, got: %v", err)
	}
}

// ── FormatSMSBody tests ─────────────────────────────────────────────────────

func TestFormatSMSBody_WithAgentName(t *testing.T) {
	t.Parallel()
	a := Approval{
		AgentID:     42,
		AgentName:   "My AI Assistant",
		Action:      json.RawMessage(`{"type":"email.send"}`),
		ApprovalURL: "https://app.example.com/approve/appr_xyz",
	}
	body := formatSMSBody(a)

	if !strings.Contains(body, "My AI Assistant") {
		t.Errorf("expected agent name in body, got: %s", body)
	}
	if !strings.Contains(body, "email.send") {
		t.Errorf("expected action type in body, got: %s", body)
	}
	if !strings.Contains(body, "https://app.example.com/approve/appr_xyz") {
		t.Errorf("expected approval URL in body, got: %s", body)
	}
}

func TestFormatSMSBody_NoAgentName(t *testing.T) {
	t.Parallel()
	a := Approval{
		AgentID:     99,
		AgentName:   "",
		Action:      json.RawMessage(`{"type":"slack.post"}`),
		ApprovalURL: "https://example.com/approve/appr_123",
	}
	body := formatSMSBody(a)

	if !strings.Contains(body, "Agent #99") {
		t.Errorf("expected fallback agent name, got: %s", body)
	}
}

func TestFormatSMSBody_NoApprovalURL(t *testing.T) {
	t.Parallel()
	a := Approval{
		AgentID:   42,
		AgentName: "Bot",
		Action:    json.RawMessage(`{"type":"test"}`),
	}
	body := formatSMSBody(a)

	if strings.Contains(body, "Approve:") {
		t.Errorf("expected no Approve URL in body, got: %s", body)
	}
}

func TestFormatSMSBody_CardExpiring(t *testing.T) {
	t.Parallel()
	a := Approval{
		Type:        NotificationTypeCardExpiring,
		Context:     json.RawMessage(`{"brand":"Visa","last4":"4242","exp_month":3,"exp_year":2026,"expired":false}`),
		ApprovalURL: "https://app.example.com/settings?tab=billing",
	}
	body := formatSMSBody(a)

	for _, check := range []string{"Visa", "4242", "03/26", "replacement", "settings?tab=billing"} {
		if !strings.Contains(body, check) {
			t.Errorf("expected SMS body to contain %q, got: %s", check, body)
		}
	}
}

func TestFormatSMSBody_CardExpired(t *testing.T) {
	t.Parallel()
	a := Approval{
		Type:        NotificationTypeCardExpiring,
		Context:     json.RawMessage(`{"brand":"Amex","last4":"3333","exp_month":1,"exp_year":2024,"expired":true}`),
		ApprovalURL: "https://app.example.com/settings?tab=billing",
	}
	body := formatSMSBody(a)

	for _, check := range []string{"Amex", "3333", "has expired"} {
		if !strings.Contains(body, check) {
			t.Errorf("expected SMS body to contain %q, got: %s", check, body)
		}
	}
}

// ── BuildSenders config tests ───────────────────────────────────────────────

func TestBuildSenders_SMSEnabled(t *testing.T) {
	t.Parallel()
	cfg := Config{
		AWSRegion:          "us-east-1",
		AWSAccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
		AWSSecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
	}
	senders := cfg.BuildSenders()
	if len(senders) != 1 {
		t.Fatalf("expected 1 sender, got %d", len(senders))
	}
	if senders[0].Name() != "sms" {
		t.Errorf("expected sender name %q, got %q", "sms", senders[0].Name())
	}
}

func TestBuildSenders_SMSDisabledNoRegion(t *testing.T) {
	t.Parallel()
	cfg := Config{
		// No AWSRegion — SMS should be disabled.
	}
	senders := cfg.BuildSenders()
	if len(senders) != 0 {
		t.Fatalf("expected 0 senders when AWS_REGION not set, got %d", len(senders))
	}
}

func TestBuildSenders_SMSPartialCredentials(t *testing.T) {
	t.Parallel()
	// Only AWSAccessKeyID set without AWSSecretAccessKey — falls through to
	// the default credential chain. With region set, SMS is still configured
	// (the SDK may resolve credentials from IAM role or shared config).
	cfg := Config{
		AWSRegion:      "us-east-1",
		AWSAccessKeyID: "AKIAIOSFODNN7EXAMPLE",
		// Missing AWSSecretAccessKey — static credentials not used.
	}
	senders := cfg.BuildSenders()
	// SMS should still be enabled because the region is set (credentials
	// resolved via default chain, not static).
	if len(senders) != 1 {
		t.Fatalf("expected 1 sender with partial static credentials (default chain fallback), got %d", len(senders))
	}
	if senders[0].Name() != "sms" {
		t.Errorf("expected sender name %q, got %q", "sms", senders[0].Name())
	}
}
