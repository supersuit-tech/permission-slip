package notify

import (
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sns"
	snstypes "github.com/aws/aws-sdk-go-v2/service/sns/types"
)

// SNSPublisher is the subset of the SNS client API used by SMSSender.
// Defined as an interface so tests can substitute a mock.
type SNSPublisher interface {
	Publish(ctx context.Context, params *sns.PublishInput, optFns ...func(*sns.Options)) (*sns.PublishOutput, error)
}

// SMSConfig holds the AWS configuration needed to deliver SMS notifications
// via Amazon SNS.
type SMSConfig struct {
	// Region is the AWS region for the SNS client (e.g. "us-east-1").
	Region string

	// SenderID is an optional alphanumeric Sender ID displayed on the
	// recipient's device. Not supported in all countries (notably the US).
	// Leave empty to use SNS defaults.
	SenderID string

	// OriginationNumber is an optional origination phone number in E.164
	// format (e.g. "+15551234567"). Required for US destinations when a
	// short/long code or toll-free number is registered in SNS.
	OriginationNumber string
}

// SMSSender delivers approval notifications via Amazon SNS.
// It is safe for concurrent use.
type SMSSender struct {
	cfg    SMSConfig
	client SNSPublisher
}

// NewSMSSender creates a Sender that delivers SMS via Amazon SNS.
func NewSMSSender(cfg SMSConfig, client SNSPublisher) *SMSSender {
	return &SMSSender{
		cfg:    cfg,
		client: client,
	}
}

// Name returns the channel identifier used for preference matching.
func (s *SMSSender) Name() string { return "sms" }

// Send delivers an SMS notification for the given approval to the recipient.
// If the recipient has no phone number, it logs and returns nil (no error)
// so other channels can still deliver.
func (s *SMSSender) Send(ctx context.Context, approval Approval, recipient Recipient) error {
	if recipient.Phone == nil || *recipient.Phone == "" {
		log.Printf("notify/sms: recipient %s has no phone number, skipping", recipient.UserID)
		return nil
	}

	body := formatSMSBody(approval)
	return s.sendViaSNS(ctx, *recipient.Phone, body)
}

// formatSMSBody constructs a concise SMS message. We aim for ≤160 characters
// (single SMS segment) but allow overflow when the content requires it.
func formatSMSBody(a Approval) string {
	if a.Type == NotificationTypePaymentFailed {
		msg := "[Permission Slip] Payment failed — update your payment method to keep your subscription"
		if a.ApprovalURL != "" {
			return fmt.Sprintf("%s: %s", msg, a.ApprovalURL)
		}
		return msg
	}

	if a.Type == NotificationTypeStandingExecution {
		return formatStandingExecutionSMS(a)
	}

	if a.Type == NotificationTypeCardExpiring {
		info := extractCardExpiringInfo(a.Context)
		var msg string
		if info.Expired {
			msg = fmt.Sprintf("[Permission Slip] Your %s has expired — update your payment methods", info.CardIdentifier())
		} else {
			msg = fmt.Sprintf("[Permission Slip] Your %s expires %s — add a replacement", info.CardIdentifier(), formatCardExpiry(info.ExpMonth, info.ExpYear))
		}
		if a.ApprovalURL != "" {
			return fmt.Sprintf("%s: %s", msg, a.ApprovalURL)
		}
		return msg
	}

	agentName := AgentDisplayName(a.AgentName, a.AgentID)
	actionSummary := SummarizeAction(a.Action)

	msg := fmt.Sprintf("[Permission Slip] %s requests: %s", agentName, actionSummary)

	if a.ApprovalURL != "" {
		return fmt.Sprintf("%s Approve: %s", msg, a.ApprovalURL)
	}

	return msg
}

// sendViaSNS publishes an SMS message using Amazon SNS.
func (s *SMSSender) sendViaSNS(ctx context.Context, to, body string) error {
	input := &sns.PublishInput{
		Message:     aws.String(body),
		PhoneNumber: aws.String(to),
	}

	// Set SMS-specific message attributes.
	input.MessageAttributes = map[string]snstypes.MessageAttributeValue{
		"AWS.SNS.SMS.SMSType": {
			DataType:    aws.String("String"),
			StringValue: aws.String("Transactional"),
		},
	}

	if s.cfg.SenderID != "" {
		input.MessageAttributes["AWS.SNS.SMS.SenderID"] = snstypes.MessageAttributeValue{
			DataType:    aws.String("String"),
			StringValue: aws.String(s.cfg.SenderID),
		}
	}

	if s.cfg.OriginationNumber != "" {
		input.MessageAttributes["AWS.MM.SMS.OriginationNumber"] = snstypes.MessageAttributeValue{
			DataType:    aws.String("String"),
			StringValue: aws.String(s.cfg.OriginationNumber),
		}
	}

	_, err := s.client.Publish(ctx, input)
	if err != nil {
		return fmt.Errorf("notify/sms: SNS publish failed: %w", err)
	}

	return nil
}
