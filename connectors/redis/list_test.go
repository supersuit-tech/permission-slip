package redis

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestLPush_Success(t *testing.T) {
	t.Parallel()

	mock := newMockRedis()
	conn := New()
	action := &lpushAction{conn: conn, doer: mock}

	params, _ := json.Marshal(pushParams{Key: "mylist", Values: []string{"a", "b"}})
	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "redis.lpush",
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
	if data["length"] != float64(2) {
		t.Errorf("expected length=2, got %v", data["length"])
	}
	// LPUSH inserts at head, so "b" pushed last ends up first.
	if mock.lists["mylist"][0] != "b" {
		t.Errorf("expected first element 'b', got %q", mock.lists["mylist"][0])
	}
}

func TestRPush_Success(t *testing.T) {
	t.Parallel()

	mock := newMockRedis()
	conn := New()
	action := &rpushAction{conn: conn, doer: mock}

	params, _ := json.Marshal(pushParams{Key: "mylist", Values: []string{"a", "b"}})
	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "redis.rpush",
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
	if data["length"] != float64(2) {
		t.Errorf("expected length=2, got %v", data["length"])
	}
	if mock.lists["mylist"][0] != "a" {
		t.Errorf("expected first element 'a', got %q", mock.lists["mylist"][0])
	}
}

func TestLPush_MissingValues(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &lpushAction{conn: conn, doer: newMockRedis()}

	params, _ := json.Marshal(map[string]any{"key": "mylist"})
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "redis.lpush",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing values")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestLPush_MissingKey(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &lpushAction{conn: conn, doer: newMockRedis()}

	params, _ := json.Marshal(map[string]any{"values": []string{"a"}})
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "redis.lpush",
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

func TestLPush_RedisError(t *testing.T) {
	t.Parallel()

	mock := newMockRedis()
	mock.lpushErr = errors.New("WRONGTYPE Operation against a key holding the wrong kind of value")

	conn := New()
	action := &lpushAction{conn: conn, doer: mock}

	params, _ := json.Marshal(pushParams{Key: "mylist", Values: []string{"a"}})
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "redis.lpush",
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

func TestLPop_Success(t *testing.T) {
	t.Parallel()

	mock := newMockRedis()
	mock.lists["mylist"] = []string{"a", "b", "c"}

	conn := New()
	action := &lpopAction{conn: conn, doer: mock}

	params, _ := json.Marshal(popParams{Key: "mylist"})
	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "redis.lpop",
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
	if data["value"] != "a" {
		t.Errorf("expected value='a', got %v", data["value"])
	}
	if data["found"] != true {
		t.Errorf("expected found=true, got %v", data["found"])
	}
	if len(mock.lists["mylist"]) != 2 {
		t.Errorf("expected 2 remaining items, got %d", len(mock.lists["mylist"]))
	}
}

func TestRPop_Success(t *testing.T) {
	t.Parallel()

	mock := newMockRedis()
	mock.lists["mylist"] = []string{"a", "b", "c"}

	conn := New()
	action := &rpopAction{conn: conn, doer: mock}

	params, _ := json.Marshal(popParams{Key: "mylist"})
	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "redis.rpop",
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
	if data["value"] != "c" {
		t.Errorf("expected value='c', got %v", data["value"])
	}
	if data["found"] != true {
		t.Errorf("expected found=true, got %v", data["found"])
	}
}

func TestLPop_EmptyList(t *testing.T) {
	t.Parallel()

	mock := newMockRedis()
	conn := New()
	action := &lpopAction{conn: conn, doer: mock}

	params, _ := json.Marshal(popParams{Key: "empty"})
	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "redis.lpop",
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

func TestRPop_EmptyList(t *testing.T) {
	t.Parallel()

	mock := newMockRedis()
	conn := New()
	action := &rpopAction{conn: conn, doer: mock}

	params, _ := json.Marshal(popParams{Key: "empty"})
	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "redis.rpop",
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
}

func TestLPop_MissingKey(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &lpopAction{conn: conn, doer: newMockRedis()}

	params, _ := json.Marshal(map[string]string{})
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "redis.lpop",
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

func TestRPop_RedisError(t *testing.T) {
	t.Parallel()

	mock := newMockRedis()
	mock.rpopErr = errors.New("connection refused")

	conn := New()
	action := &rpopAction{conn: conn, doer: mock}

	params, _ := json.Marshal(popParams{Key: "mylist"})
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "redis.rpop",
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
