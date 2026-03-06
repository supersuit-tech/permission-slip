package redis

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

type deleteAction struct {
	conn *RedisConnector
	doer redisDoer // non-nil in tests
}

type deleteParams struct {
	Key string `json:"key"`
}

func (p *deleteParams) validate() error {
	if p.Key == "" {
		return &connectors.ValidationError{Message: "missing required parameter: key"}
	}
	return nil
}

func (a *deleteAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params deleteParams
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

	deleted, err := client.Del(ctx, params.Key).Result()
	if err != nil {
		return nil, mapRedisError(err)
	}

	return connectors.JSONResult(map[string]any{
		"key":     params.Key,
		"deleted": deleted,
	})
}
