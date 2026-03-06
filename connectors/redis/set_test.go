package redis

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestSet_Success(t *testing.T) {
	t.Parallel()

	mock := newMockRedis()
	conn := New()
	action := &setAction{conn: conn, doer: mock}

	params, _ := json.Marshal(setParams{Key: "mykey", Value: "myvalue"})
	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "redis.set",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data["ok"] != true {
		t.Errorf("expected ok=true, got %v", data["ok"])
	}
	if mock.data["mykey"] != "myvalue" {
		t.Errorf("expected mock to contain key 'mykey' with value 'myvalue', got %q", mock.data["mykey"])
	}
}

func TestSet_WithTTL(t *testing.T) {
	t.Parallel()

	mock := newMockRedis()
	conn := New()
	action := &setAction{conn: conn, doer: mock}

	ttl := 300
	params, _ := json.Marshal(setParams{Key: "mykey", Value: "myvalue", TTLSeconds: &ttl})
	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "redis.set",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data["ttl_seconds"] != float64(300) {
		t.Errorf("expected ttl_seconds=300, got %v", data["ttl_seconds"])
	}
}

func TestSet_TTLExceedsMax(t *testing.T) {
	t.Parallel()

	mock := newMockRedis()
	conn := New()
	action := &setAction{conn: conn, doer: mock, maxTTL: 3600}

	ttl := 7200
	params, _ := json.Marshal(setParams{Key: "mykey", Value: "myvalue", TTLSeconds: &ttl})
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "redis.set",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for TTL exceeding max")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestSet_NegativeTTL(t *testing.T) {
	t.Parallel()

	mock := newMockRedis()
	conn := New()
	action := &setAction{conn: conn, doer: mock}

	ttl := -1
	params, _ := json.Marshal(setParams{Key: "mykey", Value: "myvalue", TTLSeconds: &ttl})
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "redis.set",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for negative TTL")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestSet_MissingKey(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &setAction{conn: conn, doer: newMockRedis()}

	params, _ := json.Marshal(map[string]string{"value": "hello"})
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "redis.set",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing key")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestSet_RedisError(t *testing.T) {
	t.Parallel()

	mock := newMockRedis()
	mock.setErr = errors.New("READONLY You can't write against a read only replica")

	conn := New()
	action := &setAction{conn: conn, doer: mock}

	params, _ := json.Marshal(setParams{Key: "mykey", Value: "myvalue"})
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "redis.set",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for Redis error")
	}
	if !connectors.IsExternalError(err) {
		t.Errorf("expected ExternalError, got: %T (%v)", err, err)
	}
}

func TestSet_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &setAction{conn: conn, doer: newMockRedis()}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "redis.set",
		Parameters:  []byte(`{invalid`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}
