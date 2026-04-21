package context

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/supersuit-tech/permission-slip/connectors"
)

type mockAPI struct {
	mu sync.Mutex

	authTest authTestResponse

	postHandlers map[string]func(body json.RawMessage) (any, error)
	getHandlers  map[string]func(params map[string]string) (any, error)
}

func newMockAPI() *mockAPI {
	return &mockAPI{
		postHandlers: make(map[string]func(body json.RawMessage) (any, error)),
		getHandlers:  make(map[string]func(params map[string]string) (any, error)),
		authTest: authTestResponse{
			slackResponse: slackResponse{OK: true},
			URL:           "https://acme.slack.com/",
			UserID:        "U_SELF",
		},
	}
}

func (m *mockAPI) Post(ctx context.Context, method string, creds connectors.Credentials, body any, dest any) error {
	_ = ctx
	_ = creds
	if method == "auth.test" {
		return copyInto(dest, m.authTest)
	}
	raw, _ := json.Marshal(body)
	m.mu.Lock()
	h, ok := m.postHandlers[method]
	m.mu.Unlock()
	if !ok {
		return mapSlackErr("unknown_method")
	}
	resp, err := h(raw)
	if err != nil {
		return err
	}
	return copyInto(dest, resp)
}

func (m *mockAPI) Get(ctx context.Context, method string, creds connectors.Credentials, params map[string]string, dest any) error {
	_ = ctx
	_ = creds
	m.mu.Lock()
	h, ok := m.getHandlers[method]
	m.mu.Unlock()
	if !ok {
		return mapSlackErr("unknown_method")
	}
	resp, err := h(params)
	if err != nil {
		return err
	}
	return copyInto(dest, resp)
}

func copyInto(dest any, src any) error {
	b, err := json.Marshal(src)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, dest)
}

func testSlackCredentials() connectors.Credentials {
	return connectors.NewCredentials(map[string]string{"access_token": "xoxp-test"})
}
