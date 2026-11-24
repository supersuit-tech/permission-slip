package api

import "github.com/supersuit-tech/permission-slip/db"

// selectDefaultConnectorInstance chooses which instance to use when the caller
// did not pass connector_instance and more than one instance exists.
//
// A credentials-ready default wins. Otherwise the only ready instance is used,
// so a stale default (deleted OAuth connection, revoked token) does not block
// the account that still works. When several instances are ready and the
// default is not, require is the list the caller must pick from.
func selectDefaultConnectorInstance(instances []db.AgentConnectorInstance, elig []db.ConnectorInstanceEligibility) (chosen *db.AgentConnectorInstance, require []db.AgentConnectorInstance) {
	if len(instances) == 0 {
		return nil, nil
	}
	if len(instances) == 1 {
		only := instances[0]
		return &only, nil
	}

	byID := make(map[string]db.ConnectorInstanceEligibility, len(elig))
	anyLive := false
	for _, e := range elig {
		byID[e.ConnectorInstanceID] = e
		if e.HasLiveBinding {
			anyLive = true
		}
	}

	var def *db.AgentConnectorInstance
	var ready []db.AgentConnectorInstance
	var eligible []db.AgentConnectorInstance
	for i := range instances {
		inst := instances[i]
		e := byID[inst.ConnectorInstanceID]
		visible := db.ConnectorInstanceVisible(e.Dangling, e.HasLiveBinding, e.CredentialsReady, anyLive)
		if inst.IsDefault {
			cp := inst
			def = &cp
		}
		if !visible {
			continue
		}
		eligible = append(eligible, inst)
		if e.CredentialsReady {
			ready = append(ready, inst)
		}
	}

	if def != nil {
		e := byID[def.ConnectorInstanceID]
		if e.CredentialsReady && db.ConnectorInstanceVisible(e.Dangling, e.HasLiveBinding, e.CredentialsReady, anyLive) {
			return def, nil
		}
	}
	if len(ready) == 1 {
		cp := ready[0]
		return &cp, nil
	}
	if len(ready) > 1 {
		return nil, ready
	}
	// Nothing is credential-ready. Keep a visible default so execution reports
	// the existing credential error instead of asking the caller to pick.
	if def != nil {
		e := byID[def.ConnectorInstanceID]
		if db.ConnectorInstanceVisible(e.Dangling, e.HasLiveBinding, e.CredentialsReady, anyLive) {
			return def, nil
		}
	}
	if len(eligible) == 1 {
		cp := eligible[0]
		return &cp, nil
	}
	if len(eligible) > 1 {
		return nil, eligible
	}
	if def != nil {
		return def, nil
	}
	return nil, instances
}

func instanceRequiredMessage(instances []db.AgentConnectorInstance) string {
	for _, inst := range instances {
		if inst.IsDefault {
			return "connector_instance is required when the default instance is not ready and more than one ready instance exists"
		}
	}
	return "connector_instance is required when multiple instances exist and no default is set"
}
