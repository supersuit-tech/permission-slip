package redis

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestDelete_Success(t *testing.T) {
	t.Parallel()

	mock := newMockRedis()
	mock.data["mykey"] = "myvalue"

	conn := New()
	action := &deleteAction{conn: conn, doer: mock}

	params, _ := json.Marshal(deleteParams{Key: "mykey"})
	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "redis.delete",
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
	if data["deleted"] != float64(1) {
		t.Errorf("expected deleted=1, got %v", data["deleted"])
	}
	if _, ok := mock.data["mykey"]; ok {
		t.Error("expected key to be deleted from mock")
	}
}

func TestDelete_KeyNotFound(t *testing.T) {
	t.Parallel()

	mock := newMockRedis()
	conn := New()
	action := &deleteAction{conn: conn, doer: mock}

	params, _ := json.Marshal(deleteParams{Key: "missing"})
	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "redis.delete",
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
	if data["deleted"] != float64(0) {
		t.Errorf("expected deleted=0, got %v", data["deleted"])
	}
}

func TestDelete_MissingKey(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &deleteAction{conn: conn, doer: newMockRedis()}

	params, _ := json.Marshal(map[string]string{})
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "redis.delete",
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

func TestDelete_RedisError(t *testing.T) {
	t.Parallel()

	mock := newMockRedis()
	mock.delErr = errors.New("WRONGPASS invalid password")

	conn := New()
	action := &deleteAction{conn: conn, doer: mock}

	params, _ := json.Marshal(deleteParams{Key: "mykey"})
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "redis.delete",
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
