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

// cleanRegistry clears the factory registry for the duration of a test and
// restores the original state when the test ends via t.Cleanup.
//
// Registry tests must NOT call t.Parallel() because they temporarily modify
// global state. Non-parallel tests run before the parallel pool starts,
// preventing interference with config_test.go tests that call
// Config.BuildSenders() (which reads the same global registry).
func cleanRegistry(t *testing.T) {
	t.Helper()
	senderMu.Lock()
	saved := senderFactories
	senderFactories = nil
	senderMu.Unlock()

	t.Cleanup(func() {
		senderMu.Lock()
		senderFactories = saved
		senderMu.Unlock()
	})
}

func TestBuildSenders_MockFactory(t *testing.T) {
	cleanRegistry(t)
	RegisterSenderFactory("test", func(_ context.Context, _ BuildContext) ([]Sender, error) {
		return []Sender{&mockSender{name: "test-channel"}}, nil
	})

	senders := BuildSenders(context.Background(), BuildContext{})
	if len(senders) != 1 {
		t.Fatalf("expected 1 sender, got %d", len(senders))
	}
	if senders[0].Name() != "test-channel" {
		t.Errorf("expected name=test-channel, got %q", senders[0].Name())
	}
}

func TestBuildSenders_MultipleFactories(t *testing.T) {
	cleanRegistry(t)
	for _, name := range []string{"alpha", "beta", "gamma"} {
		n := name // capture loop variable
		RegisterSenderFactory(n, func(_ context.Context, _ BuildContext) ([]Sender, error) {
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
}

func TestBuildSenders_FactoryError_OthersStillRun(t *testing.T) {
	cleanRegistry(t)
	RegisterSenderFactory("broken", func(_ context.Context, _ BuildContext) ([]Sender, error) {
		return nil, errors.New("intentional factory error")
	})
	RegisterSenderFactory("resilient", func(_ context.Context, _ BuildContext) ([]Sender, error) {
		return []Sender{&mockSender{name: "resilient"}}, nil
	})

	senders := BuildSenders(context.Background(), BuildContext{})
	if len(senders) != 1 {
		t.Fatalf("expected 1 sender (erroring factory skipped), got %d", len(senders))
	}
	if senders[0].Name() != "resilient" {
		t.Errorf("expected name=resilient, got %q", senders[0].Name())
	}
}

func TestBuildSenders_FactoryPanic_OthersStillRun(t *testing.T) {
	cleanRegistry(t)
	RegisterSenderFactory("panicky", func(_ context.Context, _ BuildContext) ([]Sender, error) {
		panic("factory meltdown")
	})
	RegisterSenderFactory("survivor", func(_ context.Context, _ BuildContext) ([]Sender, error) {
		return []Sender{&mockSender{name: "survived"}}, nil
	})

	senders := BuildSenders(context.Background(), BuildContext{})
	if len(senders) != 1 {
		t.Fatalf("expected 1 sender (panicking factory skipped), got %d", len(senders))
	}
	if senders[0].Name() != "survived" {
		t.Errorf("expected name=survived, got %q", senders[0].Name())
	}
}

func TestBuildSenders_FactoryDisabledChannel_ReturnsNil(t *testing.T) {
	cleanRegistry(t)
	RegisterSenderFactory("disabled", func(_ context.Context, _ BuildContext) ([]Sender, error) {
		// Simulates a channel whose required env vars are missing.
		return nil, nil
	})

	senders := BuildSenders(context.Background(), BuildContext{})
	if len(senders) != 0 {
		t.Errorf("expected 0 senders when factory returns nil, got %d", len(senders))
	}
}

func TestBuildSenders_BuildContextPassedToFactory(t *testing.T) {
	cleanRegistry(t)
	var gotDevMode bool
	RegisterSenderFactory("ctx-check", func(_ context.Context, bc BuildContext) ([]Sender, error) {
		gotDevMode = bc.DevMode
		return nil, nil
	})

	BuildSenders(context.Background(), BuildContext{DevMode: true})
	if !gotDevMode {
		t.Error("expected DevMode=true to be passed to factory")
	}
}

func TestBuildSenders_OnVAPIDPublicKey_Callback(t *testing.T) {
	cleanRegistry(t)
	var captured string
	RegisterSenderFactory("web-push", func(_ context.Context, bc BuildContext) ([]Sender, error) {
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
}

func TestSenderFactoryCount(t *testing.T) {
	cleanRegistry(t)
	if SenderFactoryCount() != 0 {
		t.Errorf("expected 0 factories in clean registry, got %d", SenderFactoryCount())
	}
	RegisterSenderFactory("a", func(_ context.Context, _ BuildContext) ([]Sender, error) { return nil, nil })
	RegisterSenderFactory("b", func(_ context.Context, _ BuildContext) ([]Sender, error) { return nil, nil })
	if SenderFactoryCount() != 2 {
		t.Errorf("expected 2 factories, got %d", SenderFactoryCount())
	}
}

func TestRegisterSenderFactory_NilPanics(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Error("expected panic when registering nil factory, got none")
		}
	}()
	RegisterSenderFactory("bad", nil)
}
