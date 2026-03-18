package notify

import (
	"context"
	"log"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sns"
)

func init() {
	RegisterSenderFactory("sms", func(ctx context.Context, bc BuildContext) ([]Sender, error) {
		cfg := bc.Config
		if cfg.AWSRegion == "" {
			if cfg.AWSAccessKeyID != "" || cfg.AWSSecretAccessKey != "" {
				log.Println("notify: SMS (SNS) partially configured — AWS_REGION is required")
			}
			return nil, nil
		}

		// Build AWS config. When explicit credentials are provided, use them;
		// otherwise fall back to the SDK's default credential chain (IAM roles,
		// env vars, shared config, etc.).
		opts := []func(*awsconfig.LoadOptions) error{
			awsconfig.WithRegion(cfg.AWSRegion),
		}
		if cfg.AWSAccessKeyID != "" && cfg.AWSSecretAccessKey != "" {
			opts = append(opts, awsconfig.WithCredentialsProvider(
				credentials.NewStaticCredentialsProvider(
					cfg.AWSAccessKeyID,
					cfg.AWSSecretAccessKey,
					"", // session token — empty for long-lived keys
				),
			))
		}

		awsCfg, err := awsconfig.LoadDefaultConfig(ctx, opts...)
		if err != nil {
			log.Printf("notify: failed to load AWS config for SMS: %v", err)
			return nil, nil
		}

		snsClient := sns.NewFromConfig(awsCfg)

		var s Sender = NewSMSSender(SMSConfig{
			Region:            cfg.AWSRegion,
			SenderID:          cfg.SNSSenderID,
			OriginationNumber: cfg.SNSOriginationNumber,
		}, snsClient)

		// Gate SMS behind paid tier when the database is available.
		// Free-tier users are blocked; paid-tier usage is tracked in usage_periods.
		//
		// Security note: when bc.DB is nil (no database configured), plan gating
		// is skipped and SMS is sent to all recipients without billing checks.
		// This matches the behaviour of the previous inline main.go setup and only
		// occurs in configurations that explicitly disable the database.
		if bc.DB != nil {
			s = NewPlanGatedSender(s, &DBSMSGate{DB: bc.DB})
		} else {
			log.Println("notify: SMS plan gating disabled — no database configured; all recipients will receive SMS regardless of subscription tier")
		}
		return []Sender{s}, nil
	})
}
