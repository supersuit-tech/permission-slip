package redis

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestGet_Success(t *testing.T) {
	t.Parallel()

	mock := newMockRedis()
	mock.data["mykey"] = "myvalue"

	conn := New()
	action := &getAction{conn: conn, doer: mock}

	params, _ := json.Marshal(getParams{Key: "mykey"})
	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "redis.get",
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
	if data["key"] != "mykey" {
		t.Errorf("expected key 'mykey', got %q", data["key"])
	}
	if data["value"] != "myvalue" {
		t.Errorf("expected value 'myvalue', got %q", data["value"])
	}
	if data["found"] != true {
		t.Errorf("expected found=true, got %v", data["found"])
	}
}

func TestGet_KeyNotFound(t *testing.T) {
	t.Parallel()

	mock := newMockRedis()
	conn := New()
	action := &getAction{conn: conn, doer: mock}

	params, _ := json.Marshal(getParams{Key: "missing"})
	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "redis.get",
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
	if data["found"] != false {
		t.Errorf("expected found=false, got %v", data["found"])
	}
	if data["value"] != nil {
		t.Errorf("expected value=nil, got %v", data["value"])
	}
}

func TestGet_MissingKey(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &getAction{conn: conn, doer: newMockRedis()}

	params, _ := json.Marshal(map[string]string{})
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "redis.get",
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

func TestGet_RedisError(t *testing.T) {
	t.Parallel()

	mock := newMockRedis()
	mock.getErr = errors.New("NOAUTH Authentication required")

	conn := New()
	action := &getAction{conn: conn, doer: mock}

	params, _ := json.Marshal(getParams{Key: "mykey"})
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "redis.get",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for Redis error")
	}
	if !connectors.IsAuthError(err) {
		t.Errorf("expected AuthError, got: %T (%v)", err, err)
	}
}

func TestGet_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &getAction{conn: conn, doer: newMockRedis()}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "redis.get",
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
