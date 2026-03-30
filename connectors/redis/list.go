package redis

import (
	"context"
	"encoding/json"
	"fmt"

	goredis "github.com/redis/go-redis/v9"
	"github.com/supersuit-tech/permission-slip/connectors"
)

// lpushAction implements connectors.Action for redis.lpush.
type lpushAction struct {
	conn *RedisConnector
	doer redisDoer
}

// rpushAction implements connectors.Action for redis.rpush.
type rpushAction struct {
	conn *RedisConnector
	doer redisDoer
}

// lpopAction implements connectors.Action for redis.lpop.
type lpopAction struct {
	conn *RedisConnector
	doer redisDoer
}

// rpopAction implements connectors.Action for redis.rpop.
type rpopAction struct {
	conn *RedisConnector
	doer redisDoer
}

type pushParams struct {
	Key    string   `json:"key"`
	Values []string `json:"values"`
}

func (p *pushParams) validate() error {
	if p.Key == "" {
		return &connectors.ValidationError{Message: "missing required parameter: key"}
	}
	if len(p.Values) == 0 {
		return &connectors.ValidationError{Message: "missing required parameter: values"}
	}
	return nil
}

type popParams struct {
	Key string `json:"key"`
}

func (p *popParams) validate() error {
	if p.Key == "" {
		return &connectors.ValidationError{Message: "missing required parameter: key"}
	}
	return nil
}

func (a *lpushAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	return executePush(ctx, req, a.conn, a.doer, "lpush")
}

func (a *rpushAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	return executePush(ctx, req, a.conn, a.doer, "rpush")
}

func executePush(ctx context.Context, req connectors.ActionRequest, conn *RedisConnector, doer redisDoer, direction string) (*connectors.ActionResult, error) {
	var params pushParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	client := doer
	if client == nil {
		c, err := conn.clientFromCreds(req.Credentials)
		if err != nil {
			return nil, err
		}
		defer c.Close()
		client = c
	}

	// Convert []string to []any for the go-redis API.
	vals := make([]any, len(params.Values))
	for i, v := range params.Values {
		vals[i] = v
	}

	var length int64
	var err error
	if direction == "lpush" {
		length, err = client.LPush(ctx, params.Key, vals...).Result()
	} else {
		length, err = client.RPush(ctx, params.Key, vals...).Result()
	}
	if err != nil {
		return nil, mapRedisError(err)
	}

	return connectors.JSONResult(map[string]any{
		"key":    params.Key,
		"length": length,
	})
}

func (a *lpopAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	return executePop(ctx, req, a.conn, a.doer, "lpop")
}

func (a *rpopAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	return executePop(ctx, req, a.conn, a.doer, "rpop")
}

func executePop(ctx context.Context, req connectors.ActionRequest, conn *RedisConnector, doer redisDoer, direction string) (*connectors.ActionResult, error) {
	var params popParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	client := doer
	if client == nil {
		c, err := conn.clientFromCreds(req.Credentials)
		if err != nil {
			return nil, err
		}
		defer c.Close()
		client = c
	}

	var val string
	var err error
	if direction == "lpop" {
		val, err = client.LPop(ctx, params.Key).Result()
	} else {
		val, err = client.RPop(ctx, params.Key).Result()
	}
	if err != nil {
		if err == goredis.Nil {
			return connectors.JSONResult(map[string]any{
				"key":   params.Key,
				"value": nil,
				"found": false,
			})
		}
		return nil, mapRedisError(err)
	}

	return connectors.JSONResult(map[string]any{
		"key":   params.Key,
		"value": val,
		"found": true,
	})
}
