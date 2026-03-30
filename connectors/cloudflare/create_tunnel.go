package cloudflare

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip/connectors"
)

type createTunnelAction struct {
	conn *CloudflareConnector
}

type createTunnelParams struct {
	AccountID    string `json:"account_id"`
	Name         string `json:"name"`
	TunnelSecret string `json:"tunnel_secret"`
}

func (p *createTunnelParams) validate() error {
	if err := requirePathParam("account_id", p.AccountID); err != nil {
		return err
	}
	if err := requireParam("name", p.Name); err != nil {
		return err
	}
	return nil
}

func (a *createTunnelAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params createTunnelParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	// Auto-generate a cryptographically random tunnel secret if the caller
	// didn't supply one. This avoids requiring users to pass a sensitive
	// secret through the approval system as a plain-text parameter.
	tunnelSecret := params.TunnelSecret
	if tunnelSecret != "" {
		decoded, err := base64.StdEncoding.DecodeString(tunnelSecret)
		if err != nil {
			return nil, &connectors.ValidationError{Message: "tunnel_secret must be valid base64"}
		}
		if len(decoded) != 32 {
			return nil, &connectors.ValidationError{Message: fmt.Sprintf("tunnel_secret must be exactly 32 bytes (got %d)", len(decoded))}
		}
	} else {
		secret := make([]byte, 32)
		if _, err := rand.Read(secret); err != nil {
			return nil, &connectors.ExternalError{Message: fmt.Sprintf("generating tunnel secret: %v", err)}
		}
		tunnelSecret = base64.StdEncoding.EncodeToString(secret)
	}

	body := map[string]any{
		"name":          params.Name,
		"tunnel_secret": tunnelSecret,
	}

	var tunnel json.RawMessage
	path := fmt.Sprintf("/accounts/%s/cfd_tunnel", params.AccountID)
	if err := a.conn.doJSON(ctx, req.Credentials, http.MethodPost, path, body, &tunnel); err != nil {
		return nil, err
	}

	return connectors.JSONResult(tunnel)
}
