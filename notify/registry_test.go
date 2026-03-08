package notify

import (
	"context"
	"errors"
	"testing"
)

// mockSender is a minimal Sender for testing the registry.
type mockSender struct{ name string }

func (m *mockSender) Name() string { return m.name }
func (m *mockSender) Send(_ context.Context, _ Approval, _ Recipient) error {
	return nil
}

// withCleanRegistry runs fn with a temporarily empty factory registry, then
// restores the original state. Registry tests must NOT call t.Parallel()
// because they temporarily modify global state shared with Config.BuildSenders.
// Non-parallel tests run before the parallel pool starts, so there is no
// interference with the config_test.go tests.
func withCleanRegistry(fn func()) {
	senderMu.Lock()
	saved := senderFactories
	senderFactories = nil
	senderMu.Unlock()

	defer func() {
		senderMu.Lock()
		senderFactories = saved
		senderMu.Unlock()
	}()

	fn()
}

func TestBuildSenders_MockFactory(t *testing.T) {
	withCleanRegistry(func() {
		RegisterSenderFactory(func(_ context.Context, _ BuildContext) ([]Sender, error) {
			return []Sender{&mockSender{name: "test-channel"}}, nil
		})

		senders := BuildSenders(context.Background(), BuildContext{})
		if len(senders) != 1 {
			t.Fatalf("expected 1 sender, got %d", len(senders))
		}
		if senders[0].Name() != "test-channel" {
			t.Errorf("expected name=test-channel, got %q", senders[0].Name())
		}
	})
}

func TestBuildSenders_MultipleFactories(t *testing.T) {
	withCleanRegistry(func() {
		for _, name := range []string{"alpha", "beta", "gamma"} {
			n := name // capture loop variable
			RegisterSenderFactory(func(_ context.Context, _ BuildContext) ([]Sender, error) {
				return []Sender{&mockSender{name: n}}, nil
			})
		}

		senders := BuildSenders(context.Background(), BuildContext{})
		if len(senders) != 3 {
			t.Fatalf("expected 3 senders, got %d", len(senders))
		}
		if senders[0].Name() != "alpha" || senders[1].Name() != "beta" || senders[2].Name() != "gamma" {
			t.Errorf("unexpected sender order: %v", senders)
		}
	})
}

func TestBuildSenders_FactoryError_OthersStillRun(t *testing.T) {
	withCleanRegistry(func() {
		RegisterSenderFactory(func(_ context.Context, _ BuildContext) ([]Sender, error) {
			return nil, errors.New("intentional factory error")
		})
		RegisterSenderFactory(func(_ context.Context, _ BuildContext) ([]Sender, error) {
			return []Sender{&mockSender{name: "resilient"}}, nil
		})

		senders := BuildSenders(context.Background(), BuildContext{})
		if len(senders) != 1 {
			t.Fatalf("expected 1 sender (erroring factory skipped), got %d", len(senders))
		}
		if senders[0].Name() != "resilient" {
			t.Errorf("expected name=resilient, got %q", senders[0].Name())
		}
	})
}

func TestBuildSenders_FactoryPanic_OthersStillRun(t *testing.T) {
	withCleanRegistry(func() {
		RegisterSenderFactory(func(_ context.Context, _ BuildContext) ([]Sender, error) {
			panic("factory meltdown")
		})
		RegisterSenderFactory(func(_ context.Context, _ BuildContext) ([]Sender, error) {
			return []Sender{&mockSender{name: "survived"}}, nil
		})

		senders := BuildSenders(context.Background(), BuildContext{})
		if len(senders) != 1 {
			t.Fatalf("expected 1 sender (panicking factory skipped), got %d", len(senders))
		}
		if senders[0].Name() != "survived" {
			t.Errorf("expected name=survived, got %q", senders[0].Name())
		}
	})
}

func TestBuildSenders_FactoryDisabledChannel_ReturnsNil(t *testing.T) {
	withCleanRegistry(func() {
		RegisterSenderFactory(func(_ context.Context, _ BuildContext) ([]Sender, error) {
			// Simulates a channel whose required env vars are missing.
			return nil, nil
		})

		senders := BuildSenders(context.Background(), BuildContext{})
		if len(senders) != 0 {
			t.Errorf("expected 0 senders when factory returns nil, got %d", len(senders))
		}
	})
}

func TestBuildSenders_BuildContextPassedToFactory(t *testing.T) {
	withCleanRegistry(func() {
		var gotDevMode bool
		RegisterSenderFactory(func(_ context.Context, bc BuildContext) ([]Sender, error) {
			gotDevMode = bc.DevMode
			return nil, nil
		})

		BuildSenders(context.Background(), BuildContext{DevMode: true})
		if !gotDevMode {
			t.Error("expected DevMode=true to be passed to factory")
		}
	})
}

func TestBuildSenders_OnVAPIDPublicKey_Callback(t *testing.T) {
	withCleanRegistry(func() {
		var captured string
		RegisterSenderFactory(func(_ context.Context, bc BuildContext) ([]Sender, error) {
			if bc.OnVAPIDPublicKey != nil {
				bc.OnVAPIDPublicKey("test-vapid-key")
			}
			return []Sender{&mockSender{name: "web-push"}}, nil
		})

		BuildSenders(context.Background(), BuildContext{
			OnVAPIDPublicKey: func(key string) { captured = key },
		})

		if captured != "test-vapid-key" {
			t.Errorf("expected VAPID key to be captured, got %q", captured)
		}
	})
}

func TestSenderFactoryCount(t *testing.T) {
	withCleanRegistry(func() {
		if SenderFactoryCount() != 0 {
			t.Errorf("expected 0 factories in clean registry, got %d", SenderFactoryCount())
		}
		RegisterSenderFactory(func(_ context.Context, _ BuildContext) ([]Sender, error) { return nil, nil })
		RegisterSenderFactory(func(_ context.Context, _ BuildContext) ([]Sender, error) { return nil, nil })
		if SenderFactoryCount() != 2 {
			t.Errorf("expected 2 factories, got %d", SenderFactoryCount())
		}
	})
}
