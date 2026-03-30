package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/supersuit-tech/permission-slip/connectors"
)

type setAction struct {
	conn      *RedisConnector
	doer      redisDoer // non-nil in tests
	maxTTL    int       // max TTL in seconds; 0 means use defaultMaxTTLSeconds
}

type setParams struct {
	Key        string `json:"key"`
	Value      string `json:"value"`
	TTLSeconds *int   `json:"ttl_seconds,omitempty"`
}

func (p *setParams) validate(maxTTL int) error {
	if p.Key == "" {
		return &connectors.ValidationError{Message: "missing required parameter: key"}
	}
	if p.TTLSeconds != nil {
		if *p.TTLSeconds < 0 {
			return &connectors.ValidationError{Message: "ttl_seconds must be non-negative"}
		}
		if maxTTL > 0 && *p.TTLSeconds > maxTTL {
			return &connectors.ValidationError{
				Message: fmt.Sprintf("ttl_seconds %d exceeds maximum allowed TTL of %d seconds", *p.TTLSeconds, maxTTL),
			}
		}
	}
	return nil
}

func (a *setAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params setParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}

	maxTTL := a.maxTTL
	if maxTTL == 0 {
		maxTTL = defaultMaxTTLSeconds
	}

	if err := params.validate(maxTTL); err != nil {
		return nil, err
	}

	client := a.doer
	if client == nil {
		c, err := a.conn.clientFromCreds(req.Credentials)
		if err != nil {
			return nil, err
		}
		defer c.Close()
		client = c
	}

	var ttl time.Duration
	if params.TTLSeconds != nil && *params.TTLSeconds > 0 {
		ttl = time.Duration(*params.TTLSeconds) * time.Second
	}

	if err := client.Set(ctx, params.Key, params.Value, ttl).Err(); err != nil {
		return nil, mapRedisError(err)
	}

	result := map[string]any{
		"key": params.Key,
		"ok":  true,
	}
	if params.TTLSeconds != nil {
		result["ttl_seconds"] = *params.TTLSeconds
	}
	return connectors.JSONResult(result)
}
