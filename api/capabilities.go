package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/supersuit-tech/permission-slip-web/db"
)

// ── Response types ──────────────────────────────────────────────────────────

type capabilitiesResponse struct {
	AgentID    int64                    `json:"agent_id"`
	Approver   *approverInfo            `json:"approver,omitempty"`
	Connectors []connectorCapability    `json:"connectors"`
}

type connectorCapability struct {
	ID                  string              `json:"id"`
	Name                string              `json:"name"`
	Description         *string             `json:"description,omitempty"`
	CredentialsReady    bool                `json:"credentials_ready"`
	CredentialsSetupURL *string             `json:"credentials_setup_url,omitempty"`
	Actions             []actionCapability  `json:"actions"`
}

type actionCapability struct {
	ActionType           string                        `json:"action_type"`
	Name                 string                        `json:"name"`
	Description          *string                       `json:"description,omitempty"`
	RiskLevel            *string                       `json:"risk_level,omitempty"`
	ParametersSchema     json.RawMessage               `json:"parameters_schema,omitempty"`
	StandingApprovals    []standingApprovalCapability   `json:"standing_approvals"`
	ActionConfigurations []actionConfigCapability       `json:"action_configurations"`
}

type actionConfigCapability struct {
	ConfigurationID string          `json:"configuration_id"`
	ActionType      string          `json:"action_type"`
	Name            string          `json:"name"`
	Description     *string         `json:"description,omitempty"`
	Parameters      json.RawMessage `json:"parameters"`
}

type standingApprovalCapability struct {
	StandingApprovalID  string          `json:"standing_approval_id"`
	Constraints         json.RawMessage `json:"constraints"`
	MaxExecutions       *int            `json:"max_executions"`
	ExecutionsRemaining *int            `json:"executions_remaining"`
	ExpiresAt           *time.Time      `json:"expires_at"`
}

// ── Route registration ──────────────────────────────────────────────────────

func init() {
	RegisterRouteGroup(RegisterCapabilityRoutes)
}

// RegisterCapabilityRoutes adds the agent capabilities endpoint to the mux.
func RegisterCapabilityRoutes(mux *http.ServeMux, deps *Deps) {
	mux.HandleFunc("GET /agents/{agent_id}/capabilities", handleGetCapabilities(deps))
}

// ── GET /agents/{agent_id}/capabilities ─────────────────────────────────────

func handleGetCapabilities(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if deps.DB == nil {
			RespondError(w, r, http.StatusServiceUnavailable, ServiceUnavailable("Database not available"))
			return
		}

		// Parse agent_id from path.
		agentID, ok := parsePathAgentID(w, r)
		if !ok {
			return
		}

		// Authenticate before any status/authorization checks so that
		// unauthenticated callers cannot probe agent status.
		agent, _, ok := requireAgentSignature(w, r, deps, agentID, nil)
		if !ok {
			return
		}

		// Only registered agents can query capabilities.
		if agent.Status != "registered" {
			RespondError(w, r, http.StatusNotFound, NotFound(ErrAgentNotFound, "Agent not found"))
			return
		}

		// Fetch capabilities from the database.
		caps, err := db.GetAgentCapabilities(r.Context(), deps.DB, agentID, agent.ApproverID)
		if err != nil {
			log.Printf("[%s] GetCapabilities: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
			RespondError(w, r, http.StatusInternalServerError, InternalError("Failed to retrieve capabilities"))
			return
		}

		resp := buildCapabilitiesResponse(agentID, caps, deps.BaseURL)

		// Look up the approver's profile to include their username.
		profile, err := db.GetProfileByUserID(r.Context(), deps.DB, agent.ApproverID)
		if err != nil {
			log.Printf("[%s] GetCapabilities: lookup approver profile: %v", TraceID(r.Context()), err)
			CaptureError(r.Context(), err)
		} else if profile != nil {
			resp.Approver = &approverInfo{Username: profile.Username}
		}

		RespondJSON(w, http.StatusOK, resp)
	}
}

// buildCapabilitiesResponse transforms the DB result into the API response,
// nesting actions under their connectors, standing approvals under their actions,
// and action configurations under their actions.
func buildCapabilitiesResponse(agentID int64, caps *db.AgentCapabilities, baseURL string) capabilitiesResponse {
	// Index actions by connector ID.
	actionsByConnector := make(map[string][]db.CapabilityAction)
	for _, a := range caps.Actions {
		actionsByConnector[a.ConnectorID] = append(actionsByConnector[a.ConnectorID], a)
	}

	// Index standing approvals by action type.
	saByAction := make(map[string][]db.CapabilityStandingApproval)
	for _, sa := range caps.StandingApprovals {
		saByAction[sa.ActionType] = append(saByAction[sa.ActionType], sa)
	}

	// Index action configurations by connector+action_type composite key.
	// Unlike standing approvals (which lack ConnectorID), action configs
	// are connector-scoped and must not bleed across connectors that share
	// the same action type.
	type connAction struct{ connectorID, actionType string }
	acByConnAction := make(map[connAction][]db.CapabilityActionConfig)
	for _, ac := range caps.ActionConfigs {
		key := connAction{ac.ConnectorID, ac.ActionType}
		acByConnAction[key] = append(acByConnAction[key], ac)
	}

	connectors := make([]connectorCapability, 0, len(caps.Connectors))
	for _, c := range caps.Connectors {
		cc := connectorCapability{
			ID:               c.ID,
			Name:             c.Name,
			Description:      c.Description,
			CredentialsReady: c.CredentialsReady,
		}

		if !c.CredentialsReady && baseURL != "" {
			url := fmt.Sprintf("%s/connect/%s", baseURL, c.ID)
			cc.CredentialsSetupURL = &url
		}

		// Build actions for this connector.
		dbActions := actionsByConnector[c.ID]
		actions := make([]actionCapability, 0, len(dbActions))
		for _, a := range dbActions {
			acap := actionCapability{
				ActionType:       a.ActionType,
				Name:             a.Name,
				Description:      a.Description,
				RiskLevel:        a.RiskLevel,
				ParametersSchema: a.ParametersSchema,
			}

			// Attach standing approvals for this action type.
			dbSAs := saByAction[a.ActionType]
			sas := make([]standingApprovalCapability, 0, len(dbSAs))
			for _, sa := range dbSAs {
				sas = append(sas, standingApprovalCapability{
					StandingApprovalID:  sa.StandingApprovalID,
					Constraints:         sa.Constraints,
					MaxExecutions:       sa.MaxExecutions,
					ExecutionsRemaining: sa.ExecutionsRemaining,
					ExpiresAt:           sa.ExpiresAt,
				})
			}
			acap.StandingApprovals = sas

			// Attach action configurations for this connector + action type.
			dbACs := acByConnAction[connAction{c.ID, a.ActionType}]
			acs := make([]actionConfigCapability, 0, len(dbACs))
			for _, cfg := range dbACs {
				acs = append(acs, actionConfigCapability{
					ConfigurationID: cfg.ConfigurationID,
					ActionType:      cfg.ActionType,
					Name:            cfg.Name,
					Description:     cfg.Description,
					Parameters:      cfg.Parameters,
				})
			}
			acap.ActionConfigurations = acs

			actions = append(actions, acap)
		}
		cc.Actions = actions

		connectors = append(connectors, cc)
	}

	return capabilitiesResponse{
		AgentID:    agentID,
		Connectors: connectors,
	}
}
