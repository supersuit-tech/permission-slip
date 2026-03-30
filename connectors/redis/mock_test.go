package redis

import (
	"context"
	"time"

	goredis "github.com/redis/go-redis/v9"
	"github.com/supersuit-tech/permission-slip/connectors"
)

// validCreds returns a Credentials value with a valid Redis URL for tests.
func validCreds() connectors.Credentials {
	return connectors.NewCredentials(map[string]string{
		"url": "redis://localhost:6379/0",
	})
}

// mockRedis implements the redisDoer interface for unit tests without
// a real Redis server.
type mockRedis struct {
	data   map[string]string
	lists  map[string][]string
	closed bool

	// Error overrides — set these to simulate failures.
	getErr   error
	setErr   error
	delErr   error
	lpushErr error
	rpushErr error
	lpopErr  error
	rpopErr  error
}

func newMockRedis() *mockRedis {
	return &mockRedis{
		data:  make(map[string]string),
		lists: make(map[string][]string),
	}
}

func (m *mockRedis) Get(_ context.Context, key string) *goredis.StringCmd {
	cmd := goredis.NewStringCmd(context.Background())
	if m.getErr != nil {
		cmd.SetErr(m.getErr)
		return cmd
	}
	val, ok := m.data[key]
	if !ok {
		cmd.SetErr(goredis.Nil)
		return cmd
	}
	cmd.SetVal(val)
	return cmd
}

func (m *mockRedis) Set(_ context.Context, key string, value any, _ time.Duration) *goredis.StatusCmd {
	cmd := goredis.NewStatusCmd(context.Background())
	if m.setErr != nil {
		cmd.SetErr(m.setErr)
		return cmd
	}
	m.data[key] = value.(string)
	cmd.SetVal("OK")
	return cmd
}

func (m *mockRedis) Del(_ context.Context, keys ...string) *goredis.IntCmd {
	cmd := goredis.NewIntCmd(context.Background())
	if m.delErr != nil {
		cmd.SetErr(m.delErr)
		return cmd
	}
	var deleted int64
	for _, key := range keys {
		if _, ok := m.data[key]; ok {
			delete(m.data, key)
			deleted++
		}
		if _, ok := m.lists[key]; ok {
			delete(m.lists, key)
			deleted++
		}
	}
	cmd.SetVal(deleted)
	return cmd
}

func (m *mockRedis) LPush(_ context.Context, key string, values ...any) *goredis.IntCmd {
	cmd := goredis.NewIntCmd(context.Background())
	if m.lpushErr != nil {
		cmd.SetErr(m.lpushErr)
		return cmd
	}
	list := m.lists[key]
	for _, v := range values {
		list = append([]string{v.(string)}, list...)
	}
	m.lists[key] = list
	cmd.SetVal(int64(len(list)))
	return cmd
}

func (m *mockRedis) RPush(_ context.Context, key string, values ...any) *goredis.IntCmd {
	cmd := goredis.NewIntCmd(context.Background())
	if m.rpushErr != nil {
		cmd.SetErr(m.rpushErr)
		return cmd
	}
	list := m.lists[key]
	for _, v := range values {
		list = append(list, v.(string))
	}
	m.lists[key] = list
	cmd.SetVal(int64(len(list)))
	return cmd
}

func (m *mockRedis) LPop(_ context.Context, key string) *goredis.StringCmd {
	cmd := goredis.NewStringCmd(context.Background())
	if m.lpopErr != nil {
		cmd.SetErr(m.lpopErr)
		return cmd
	}
	list := m.lists[key]
	if len(list) == 0 {
		cmd.SetErr(goredis.Nil)
		return cmd
	}
	val := list[0]
	m.lists[key] = list[1:]
	cmd.SetVal(val)
	return cmd
}

func (m *mockRedis) RPop(_ context.Context, key string) *goredis.StringCmd {
	cmd := goredis.NewStringCmd(context.Background())
	if m.rpopErr != nil {
		cmd.SetErr(m.rpopErr)
		return cmd
	}
	list := m.lists[key]
	if len(list) == 0 {
		cmd.SetErr(goredis.Nil)
		return cmd
	}
	val := list[len(list)-1]
	m.lists[key] = list[:len(list)-1]
	cmd.SetVal(val)
	return cmd
}

func (m *mockRedis) Close() error {
	m.closed = true
	return nil
}
