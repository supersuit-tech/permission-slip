package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/supersuit-tech/permission-slip/db"
)

// connectorInstanceResolutionError carries an HTTP error for applyConnectorInstanceToAction.
type connectorInstanceResolutionError struct {
	status int
	resp   ErrorResponse
}

func (e *connectorInstanceResolutionError) Error() string {
	return e.resp.Error.Message
}

// applyConnectorInstanceToAction removes parameters.connector_instance, resolves the target
// instance for the action's connector, and sets _connector_instance_id and _connector_instance_display
// on the action object. Returns the resolved instance UUID for credential routing (empty if the
// connector has no enabled instances on the agent — downstream "no credential" handling applies).
func applyConnectorInstanceToAction(ctx context.Context, d db.DBTX, agent *db.Agent, actionType string, actionObj map[string]json.RawMessage) (connectorInstanceID string, err error) {
	parts := strings.SplitN(actionType, ".", 2)
	if len(parts) != 2 {
		return "", nil
	}
	connectorID := parts[0]

	var paramsObj map[string]json.RawMessage
	if raw, ok := actionObj["parameters"]; ok && len(raw) > 0 && string(raw) != "null" {
		if err := json.Unmarshal(raw, &paramsObj); err != nil {
			return "", err
		}
	}

	var selector string
	if paramsObj != nil {
		if raw, ok := paramsObj["connector_instance"]; ok {
			_ = json.Unmarshal(raw, &selector)
			selector = strings.TrimSpace(selector)
			delete(paramsObj, "connector_instance")
			if len(paramsObj) == 0 {
				delete(actionObj, "parameters")
			} else {
				b, mErr := json.Marshal(paramsObj)
				if mErr != nil {
					return "", mErr
				}
				actionObj["parameters"] = b
			}
		}
	}

	instances, err := db.ListAgentConnectorInstances(ctx, d, agent.AgentID, agent.ApproverID, connectorID)
	if err != nil {
		return "", err
	}
	if len(instances) == 0 {
		return "", nil
	}

	var inst *db.AgentConnectorInstance
	if len(instances) == 1 {
		only := &instances[0]
		if selector == "" {
			inst = only
		} else {
			resolved, err := db.ResolveAgentConnectorInstance(ctx, d, agent.AgentID, agent.ApproverID, connectorID, selector)
			if err != nil {
				if amb := ambiguousConnectorInstanceErr(err); amb != nil {
					return "", amb
				}
				return "", err
			}
			if resolved == nil || resolved.ConnectorInstanceID != only.ConnectorInstanceID {
				return "", &connectorInstanceResolutionError{
					status: http.StatusBadRequest,
					resp:   BadRequest(ErrConnectorInstanceNotFound, "no connector instance matches the given connector_instance"),
				}
			}
			inst = resolved
		}
	} else {
		if selector == "" {
			nicknames := make([]string, 0, len(instances))
			for _, i := range instances {
				nicknames = append(nicknames, i.DisplayName)
			}
			resp := BadRequest(ErrConnectorInstanceRequired, "connector_instance is required when multiple instances exist")
			resp.Error.Details = map[string]any{"available_instances": nicknames}
			return "", &connectorInstanceResolutionError{status: http.StatusBadRequest, resp: resp}
		}
		resolved, err := db.ResolveAgentConnectorInstance(ctx, d, agent.AgentID, agent.ApproverID, connectorID, selector)
		if err != nil {
			if amb := ambiguousConnectorInstanceErr(err); amb != nil {
				return "", amb
			}
			return "", err
		}
		if resolved == nil {
			return "", &connectorInstanceResolutionError{
				status: http.StatusBadRequest,
				resp:   BadRequest(ErrConnectorInstanceNotFound, "no connector instance matches the given connector_instance"),
			}
		}
		inst = resolved
	}

	idRaw, _ := json.Marshal(inst.ConnectorInstanceID)
	actionObj["_connector_instance_id"] = idRaw
	displayRaw, _ := json.Marshal(inst.DisplayName)
	actionObj["_connector_instance_display"] = displayRaw
	return inst.ConnectorInstanceID, nil
}

func ambiguousConnectorInstanceErr(err error) *connectorInstanceResolutionError {
	var instErr *db.AgentConnectorInstanceError
	if !errors.As(err, &instErr) || instErr.Code != db.AgentConnectorInstanceErrAmbiguousDisplay {
		return nil
	}
	resp := BadRequest(ErrConnectorInstanceAmbiguous,
		"multiple connector instances match this display name; use connector_instance with a UUID from GET /agents/{id}/capabilities")
	resp.Error.Details = map[string]any{"matching_instance_ids": instErr.AmbiguousInstanceIDs}
	return &connectorInstanceResolutionError{status: http.StatusBadRequest, resp: resp}
}
