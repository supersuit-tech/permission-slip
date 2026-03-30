// Package redis implements the Redis connector for the Permission Slip
// connector execution layer. It uses github.com/redis/go-redis/v9 to execute
// key-value and list operations against a Redis instance.
package redis

import (
	_ "embed"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	goredis "github.com/redis/go-redis/v9"
	"github.com/supersuit-tech/permission-slip/connectors"
)

const (
	defaultTimeout       = 10 * time.Second
	credKeyURL           = "url"
	defaultMaxTTLSeconds = 86400 // 24 hours
)

// RedisConnector owns the shared configuration for all Redis actions.
// A new go-redis client is created per-execution from credentials (the
// connection string is stored in the vault, not on the connector struct).
type RedisConnector struct {
	timeout time.Duration
}

// New creates a RedisConnector with sensible defaults.
func New() *RedisConnector {
	return &RedisConnector{
		timeout: defaultTimeout,
	}
}

// ID returns "redis", matching the connectors.id in the database.
func (c *RedisConnector) ID() string { return "redis" }

//go:embed logo.svg
var logoSVG string

// Manifest returns the connector's metadata manifest.
func (c *RedisConnector) Manifest() *connectors.ConnectorManifest {
	return &connectors.ConnectorManifest{
		ID:          "redis",
		Name:        "Redis",
		Description: "Redis integration for cache, session, and queue data management",
		LogoSVG:     logoSVG,
		Actions: []connectors.ManifestAction{
			{
				ActionType:  "redis.get",
				Name:        "Get",
				Description: "Get a value by key",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["key"],
					"properties": {
						"key": {
							"type": "string",
							"description": "The Redis key to retrieve"
						}
					}
				}`)),
			},
			{
				ActionType:  "redis.set",
				Name:        "Set",
				Description: "Set a key-value pair with optional TTL",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["key", "value"],
					"properties": {
						"key": {
							"type": "string",
							"description": "The Redis key to set"
						},
						"value": {
							"type": "string",
							"description": "The value to store"
						},
						"ttl_seconds": {
							"type": "integer",
							"minimum": 0,
							"description": "Time-to-live in seconds (0 means no expiry, subject to max TTL enforcement)"
						}
					}
				}`)),
			},
			{
				ActionType:  "redis.delete",
				Name:        "Delete",
				Description: "Delete a key",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["key"],
					"properties": {
						"key": {
							"type": "string",
							"description": "The Redis key to delete"
						}
					}
				}`)),
			},
			{
				ActionType:  "redis.lpush",
				Name:        "Left Push",
				Description: "Push one or more values to the head of a list",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["key", "values"],
					"properties": {
						"key": {
							"type": "string",
							"description": "The Redis list key"
						},
						"values": {
							"type": "array",
							"items": {"type": "string"},
							"minItems": 1,
							"description": "Values to push to the head of the list"
						}
					}
				}`)),
			},
			{
				ActionType:  "redis.rpush",
				Name:        "Right Push",
				Description: "Push one or more values to the tail of a list",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["key", "values"],
					"properties": {
						"key": {
							"type": "string",
							"description": "The Redis list key"
						},
						"values": {
							"type": "array",
							"items": {"type": "string"},
							"minItems": 1,
							"description": "Values to push to the tail of the list"
						}
					}
				}`)),
			},
			{
				ActionType:  "redis.lpop",
				Name:        "Left Pop",
				Description: "Pop a value from the head of a list",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["key"],
					"properties": {
						"key": {
							"type": "string",
							"description": "The Redis list key"
						}
					}
				}`)),
			},
			{
				ActionType:  "redis.rpop",
				Name:        "Right Pop",
				Description: "Pop a value from the tail of a list",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["key"],
					"properties": {
						"key": {
							"type": "string",
							"description": "The Redis list key"
						}
					}
				}`)),
			},
		},
		RequiredCredentials: []connectors.ManifestCredential{
			{Service: "redis", AuthType: "custom", InstructionsURL: "https://redis.io/docs/latest/operate/oss_and_stack/management/security/acl/"},
		},
		Templates: []connectors.ManifestTemplate{
			{
				ID:          "tpl_redis_cache_read",
				ActionType:  "redis.get",
				Name:        "Read cache keys",
				Description: "Agent can read values from keys matching a prefix.",
				Parameters:  json.RawMessage(`{"key":"cache:*"}`),
			},
			{
				ID:          "tpl_redis_cache_write",
				ActionType:  "redis.set",
				Name:        "Write cache keys with TTL",
				Description: "Agent can write values to cache keys with a required TTL.",
				Parameters:  json.RawMessage(`{"key":"cache:*","value":"*","ttl_seconds":3600}`),
			},
			{
				ID:          "tpl_redis_session_manage",
				ActionType:  "redis.delete",
				Name:        "Manage session keys",
				Description: "Agent can delete session keys.",
				Parameters:  json.RawMessage(`{"key":"session:*"}`),
			},
			{
				ID:          "tpl_redis_queue_push",
				ActionType:  "redis.rpush",
				Name:        "Push to a queue",
				Description: "Agent can push items to a queue list.",
				Parameters:  json.RawMessage(`{"key":"queue:*","values":["*"]}`),
			},
			{
				ID:          "tpl_redis_queue_pop",
				ActionType:  "redis.lpop",
				Name:        "Pop from a queue",
				Description: "Agent can pop items from a queue list.",
				Parameters:  json.RawMessage(`{"key":"queue:*"}`),
			},
		},
	}
}

// Actions returns the registered action handlers keyed by action_type.
func (c *RedisConnector) Actions() map[string]connectors.Action {
	return map[string]connectors.Action{
		"redis.get":    &getAction{conn: c},
		"redis.set":    &setAction{conn: c},
		"redis.delete": &deleteAction{conn: c},
		"redis.lpush":  &lpushAction{conn: c},
		"redis.rpush":  &rpushAction{conn: c},
		"redis.lpop":   &lpopAction{conn: c},
		"redis.rpop":   &rpopAction{conn: c},
	}
}

// ValidateCredentials checks that the provided credentials contain a
// parseable Redis connection URL.
func (c *RedisConnector) ValidateCredentials(_ context.Context, creds connectors.Credentials) error {
	rawURL, ok := creds.Get(credKeyURL)
	if !ok || rawURL == "" {
		return &connectors.ValidationError{Message: "missing required credential: url"}
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return &connectors.ValidationError{Message: fmt.Sprintf("invalid Redis URL: %v", err)}
	}
	if u.Scheme != "redis" && u.Scheme != "rediss" {
		return &connectors.ValidationError{Message: "Redis URL must use redis:// or rediss:// scheme"}
	}
	return nil
}

// clientFromCreds creates a go-redis client from the credentials URL.
// The caller must close the client when done.
func (c *RedisConnector) clientFromCreds(creds connectors.Credentials) (*goredis.Client, error) {
	rawURL, ok := creds.Get(credKeyURL)
	if !ok || rawURL == "" {
		return nil, &connectors.ValidationError{Message: "missing required credential: url"}
	}
	opts, err := goredis.ParseURL(rawURL)
	if err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid Redis URL: %v", err)}
	}
	opts.DialTimeout = c.timeout
	opts.ReadTimeout = c.timeout
	opts.WriteTimeout = c.timeout
	return goredis.NewClient(opts), nil
}

// mapRedisError converts a go-redis error to the appropriate connector error type.
func mapRedisError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, goredis.Nil) {
		return nil // key not found is not an error
	}
	if errors.Is(err, context.DeadlineExceeded) || connectors.IsTimeout(err) {
		return &connectors.TimeoutError{Message: fmt.Sprintf("Redis operation timed out: %v", err)}
	}
	if errors.Is(err, context.Canceled) {
		return &connectors.CanceledError{Message: "Redis operation canceled"}
	}
	errMsg := err.Error()
	if strings.Contains(errMsg, "NOAUTH") || strings.Contains(errMsg, "WRONGPASS") ||
		strings.Contains(errMsg, "AUTH") {
		return &connectors.AuthError{Message: fmt.Sprintf("Redis auth error: %v", err)}
	}
	return &connectors.ExternalError{Message: fmt.Sprintf("Redis error: %v", err)}
}

// redisDoer abstracts the go-redis commands used by actions, enabling tests
// to inject a mock without a real Redis server.
type redisDoer interface {
	Get(ctx context.Context, key string) *goredis.StringCmd
	Set(ctx context.Context, key string, value any, expiration time.Duration) *goredis.StatusCmd
	Del(ctx context.Context, keys ...string) *goredis.IntCmd
	LPush(ctx context.Context, key string, values ...any) *goredis.IntCmd
	RPush(ctx context.Context, key string, values ...any) *goredis.IntCmd
	LPop(ctx context.Context, key string) *goredis.StringCmd
	RPop(ctx context.Context, key string) *goredis.StringCmd
	Close() error
}
