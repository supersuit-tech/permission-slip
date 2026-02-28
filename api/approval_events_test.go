package api

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestApprovalEventBroker_SubscribeNotifyUnsubscribe(t *testing.T) {
	broker := NewApprovalEventBroker()
	userID := "user-123"

	ch := broker.Subscribe(userID)
	defer broker.Unsubscribe(userID, ch)

	// Send an event
	broker.Notify(userID, "event: test\ndata: {}")

	select {
	case event := <-ch:
		if event != "event: test\ndata: {}" {
			t.Errorf("unexpected event: %q", event)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
	}
}

func TestApprovalEventBroker_NotifyWrongUser(t *testing.T) {
	broker := NewApprovalEventBroker()

	ch := broker.Subscribe("user-1")
	defer broker.Unsubscribe("user-1", ch)

	broker.Notify("user-2", "event: test\ndata: {}")

	select {
	case event := <-ch:
		t.Fatalf("should not receive event for different user, got: %q", event)
	case <-time.After(50 * time.Millisecond):
		// Expected: no event
	}
}

func TestApprovalEventBroker_MultipleSubscribers(t *testing.T) {
	broker := NewApprovalEventBroker()
	userID := "user-123"

	ch1 := broker.Subscribe(userID)
	ch2 := broker.Subscribe(userID)
	defer broker.Unsubscribe(userID, ch1)
	defer broker.Unsubscribe(userID, ch2)

	broker.Notify(userID, "event: test\ndata: {}")

	for i, ch := range []chan string{ch1, ch2} {
		select {
		case event := <-ch:
			if event != "event: test\ndata: {}" {
				t.Errorf("subscriber %d: unexpected event: %q", i, event)
			}
		case <-time.After(time.Second):
			t.Fatalf("subscriber %d: timed out", i)
		}
	}
}

func TestApprovalEventBroker_UnsubscribeRemovesChannel(t *testing.T) {
	broker := NewApprovalEventBroker()
	userID := "user-123"

	ch := broker.Subscribe(userID)
	broker.Unsubscribe(userID, ch)

	// Channel should be closed
	_, ok := <-ch
	if ok {
		t.Fatal("channel should be closed after unsubscribe")
	}

	// Notify should not panic
	broker.Notify(userID, "event: test\ndata: {}")
}

func TestApprovalEventBroker_UnsubscribeCleansUpEmptyUserMap(t *testing.T) {
	broker := NewApprovalEventBroker()
	userID := "user-123"

	ch := broker.Subscribe(userID)
	broker.Unsubscribe(userID, ch)

	broker.mu.RLock()
	_, exists := broker.subscribers[userID]
	broker.mu.RUnlock()

	if exists {
		t.Error("user entry should be removed when last subscriber unsubscribes")
	}
}

func TestApprovalEventBroker_NotifyAll(t *testing.T) {
	broker := NewApprovalEventBroker()

	ch1 := broker.Subscribe("user-1")
	ch2 := broker.Subscribe("user-2")
	defer broker.Unsubscribe("user-1", ch1)
	defer broker.Unsubscribe("user-2", ch2)

	broker.NotifyAll("event: broadcast\ndata: {}")

	for _, tc := range []struct {
		name string
		ch   chan string
	}{
		{"user-1", ch1},
		{"user-2", ch2},
	} {
		select {
		case event := <-tc.ch:
			if event != "event: broadcast\ndata: {}" {
				t.Errorf("%s: unexpected event: %q", tc.name, event)
			}
		case <-time.After(time.Second):
			t.Fatalf("%s: timed out", tc.name)
		}
	}
}

func TestApprovalEventBroker_DropEventOnFullBuffer(t *testing.T) {
	broker := NewApprovalEventBroker()
	userID := "user-123"

	ch := broker.Subscribe(userID)
	defer broker.Unsubscribe(userID, ch)

	// Fill the buffer (16 capacity)
	for i := 0; i < 16; i++ {
		broker.Notify(userID, "event: fill\ndata: {}")
	}

	// This should not block — event is dropped
	done := make(chan struct{})
	go func() {
		broker.Notify(userID, "event: overflow\ndata: {}")
		close(done)
	}()

	select {
	case <-done:
		// Expected: Notify returned without blocking
	case <-time.After(time.Second):
		t.Fatal("Notify blocked on full buffer — should drop event")
	}
}

func TestApprovalEventBroker_ConcurrentAccess(t *testing.T) {
	broker := NewApprovalEventBroker()
	const goroutines = 20

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			userID := "user-concurrent"
			ch := broker.Subscribe(userID)
			broker.Notify(userID, "event: test\ndata: {}")
			broker.NotifyAll("event: broadcast\ndata: {}")
			broker.Unsubscribe(userID, ch)
		}(i)
	}

	wg.Wait()
}

func TestSSETokenFromQuery(t *testing.T) {
	t.Run("copies token to Authorization header", func(t *testing.T) {
		inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			got := r.Header.Get("Authorization")
			if got != "Bearer my-jwt-token" {
				t.Errorf("Authorization = %q, want %q", got, "Bearer my-jwt-token")
			}
		})

		handler := sseTokenFromQuery(inner)
		req := httptest.NewRequest("GET", "/approvals/events?token=my-jwt-token", nil)
		handler.ServeHTTP(httptest.NewRecorder(), req)
	})

	t.Run("strips token from URL to prevent log leaking", func(t *testing.T) {
		inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Get("token") != "" {
				t.Error("token query parameter should be removed after extraction")
			}
			if r.Header.Get("Authorization") != "Bearer secret" {
				t.Errorf("Authorization = %q, want %q", r.Header.Get("Authorization"), "Bearer secret")
			}
		})

		handler := sseTokenFromQuery(inner)
		req := httptest.NewRequest("GET", "/approvals/events?token=secret", nil)
		handler.ServeHTTP(httptest.NewRecorder(), req)
	})

	t.Run("does not overwrite existing Authorization header", func(t *testing.T) {
		inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			got := r.Header.Get("Authorization")
			if got != "Bearer existing-token" {
				t.Errorf("Authorization = %q, want %q", got, "Bearer existing-token")
			}
		})

		handler := sseTokenFromQuery(inner)
		req := httptest.NewRequest("GET", "/approvals/events?token=query-token", nil)
		req.Header.Set("Authorization", "Bearer existing-token")
		handler.ServeHTTP(httptest.NewRecorder(), req)
	})

	t.Run("passes through when no token", func(t *testing.T) {
		inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			got := r.Header.Get("Authorization")
			if got != "" {
				t.Errorf("Authorization = %q, want empty", got)
			}
		})

		handler := sseTokenFromQuery(inner)
		req := httptest.NewRequest("GET", "/approvals/events", nil)
		handler.ServeHTTP(httptest.NewRecorder(), req)
	})
}

func TestApprovalEventBroker_SubscriberCount(t *testing.T) {
	broker := NewApprovalEventBroker()
	userID := "user-123"

	if broker.SubscriberCount(userID) != 0 {
		t.Errorf("expected 0, got %d", broker.SubscriberCount(userID))
	}

	ch1 := broker.Subscribe(userID)
	ch2 := broker.Subscribe(userID)
	if broker.SubscriberCount(userID) != 2 {
		t.Errorf("expected 2, got %d", broker.SubscriberCount(userID))
	}

	broker.Unsubscribe(userID, ch1)
	if broker.SubscriberCount(userID) != 1 {
		t.Errorf("expected 1, got %d", broker.SubscriberCount(userID))
	}

	broker.Unsubscribe(userID, ch2)
	if broker.SubscriberCount(userID) != 0 {
		t.Errorf("expected 0, got %d", broker.SubscriberCount(userID))
	}
}

func TestApprovalEventBroker_DoubleUnsubscribeNoPanic(t *testing.T) {
	broker := NewApprovalEventBroker()
	userID := "user-123"

	ch := broker.Subscribe(userID)
	broker.Unsubscribe(userID, ch)

	// Second unsubscribe should not panic (close of closed channel)
	broker.Unsubscribe(userID, ch)
}

func TestApprovalEventBroker_SubscribeWithLimit(t *testing.T) {
	broker := NewApprovalEventBroker()
	userID := "user-limit"
	limit := 3

	// Subscribe up to the limit.
	var channels []chan string
	for i := 0; i < limit; i++ {
		ch, ok := broker.SubscribeWithLimit(userID, limit)
		if !ok {
			t.Fatalf("SubscribeWithLimit should succeed for connection %d", i+1)
		}
		channels = append(channels, ch)
	}

	if broker.SubscriberCount(userID) != limit {
		t.Errorf("expected %d subscribers, got %d", limit, broker.SubscriberCount(userID))
	}

	// The next subscription should be rejected.
	ch, ok := broker.SubscribeWithLimit(userID, limit)
	if ok {
		t.Error("SubscribeWithLimit should reject when at limit")
		broker.Unsubscribe(userID, ch)
	}

	// Unsubscribe one — the next attempt should succeed.
	broker.Unsubscribe(userID, channels[0])
	ch, ok = broker.SubscribeWithLimit(userID, limit)
	if !ok {
		t.Error("SubscribeWithLimit should succeed after freeing a slot")
	} else {
		broker.Unsubscribe(userID, ch)
	}

	// Clean up.
	for _, c := range channels[1:] {
		broker.Unsubscribe(userID, c)
	}
}

func TestNotifyApprovalChange_NilBroker(t *testing.T) {
	// notifyApprovalChange should not panic when ApprovalEvents is nil
	deps := &Deps{ApprovalEvents: nil}
	notifyApprovalChange(deps, "user-1", "approval_created", "appr_123")
	// No panic = pass
}
