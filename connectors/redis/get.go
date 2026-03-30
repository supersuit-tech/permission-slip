package redis

import (
	"context"
	"encoding/json"
	"fmt"

	goredis "github.com/redis/go-redis/v9"
	"github.com/supersuit-tech/permission-slip/connectors"
)

type getAction struct {
	conn  *RedisConnector
	doer  redisDoer // non-nil in tests
}

type getParams struct {
	Key string `json:"key"`
}

func (p *getParams) validate() error {
	if p.Key == "" {
		return &connectors.ValidationError{Message: "missing required parameter: key"}
	}
	return nil
}

func (a *getAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params getParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
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

	val, err := client.Get(ctx, params.Key).Result()
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
