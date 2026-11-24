package api

import (
	"testing"

	"github.com/supersuit-tech/permission-slip/db"
)

func inst(id string, isDefault bool) db.AgentConnectorInstance {
	return db.AgentConnectorInstance{ConnectorInstanceID: id, IsDefault: isDefault, DisplayName: id}
}

func elig(id string, ready, dangling, live bool) db.ConnectorInstanceEligibility {
	return db.ConnectorInstanceEligibility{
		ConnectorInstanceID: id,
		CredentialsReady:    ready,
		Dangling:            dangling,
		HasLiveBinding:      live,
	}
}

func TestSelectDefaultConnectorInstance_ReadyDefaultWins(t *testing.T) {
	instances := []db.AgentConnectorInstance{inst("def", true), inst("other", false)}
	eligibility := []db.ConnectorInstanceEligibility{
		elig("def", true, false, true),
		elig("other", true, false, true),
	}
	chosen, require := selectDefaultConnectorInstance(instances, eligibility)
	if len(require) != 0 {
		t.Fatalf("expected no required list, got %+v", require)
	}
	if chosen == nil || chosen.ConnectorInstanceID != "def" {
		t.Fatalf("expected default, got %+v", chosen)
	}
}

func TestSelectDefaultConnectorInstance_SkipsDanglingDefault(t *testing.T) {
	instances := []db.AgentConnectorInstance{inst("stale", true), inst("ready", false)}
	eligibility := []db.ConnectorInstanceEligibility{
		elig("stale", false, true, false),
		elig("ready", true, false, true),
	}
	chosen, require := selectDefaultConnectorInstance(instances, eligibility)
	if len(require) != 0 {
		t.Fatalf("expected no required list, got %+v", require)
	}
	if chosen == nil || chosen.ConnectorInstanceID != "ready" {
		t.Fatalf("expected ready instance, got %+v", chosen)
	}
}

func TestSelectDefaultConnectorInstance_SkipsUnreadyDefaultWhenOneReady(t *testing.T) {
	instances := []db.AgentConnectorInstance{inst("revoked", true), inst("ready", false)}
	eligibility := []db.ConnectorInstanceEligibility{
		elig("revoked", false, false, true),
		elig("ready", true, false, true),
	}
	chosen, require := selectDefaultConnectorInstance(instances, eligibility)
	if len(require) != 0 {
		t.Fatalf("expected no required list, got %+v", require)
	}
	if chosen == nil || chosen.ConnectorInstanceID != "ready" {
		t.Fatalf("expected ready instance, got %+v", chosen)
	}
}

func TestSelectDefaultConnectorInstance_MultipleReadyRequiresChoice(t *testing.T) {
	instances := []db.AgentConnectorInstance{inst("revoked", true), inst("a", false), inst("b", false)}
	eligibility := []db.ConnectorInstanceEligibility{
		elig("revoked", false, false, true),
		elig("a", true, false, true),
		elig("b", true, false, true),
	}
	chosen, require := selectDefaultConnectorInstance(instances, eligibility)
	if chosen != nil {
		t.Fatalf("expected no automatic choice, got %+v", chosen)
	}
	if len(require) != 2 {
		t.Fatalf("expected 2 ready instances, got %+v", require)
	}
	got := map[string]bool{}
	for _, inst := range require {
		got[inst.ConnectorInstanceID] = true
	}
	if !got["a"] || !got["b"] || got["revoked"] {
		t.Fatalf("required list = %+v", require)
	}
}

func TestSelectDefaultConnectorInstance_NoDefaultRequiresChoice(t *testing.T) {
	instances := []db.AgentConnectorInstance{inst("a", false), inst("b", false)}
	eligibility := []db.ConnectorInstanceEligibility{
		elig("a", true, false, true),
		elig("b", true, false, true),
	}
	chosen, require := selectDefaultConnectorInstance(instances, eligibility)
	if chosen != nil {
		t.Fatalf("expected no automatic choice, got %+v", chosen)
	}
	if len(require) != 2 {
		t.Fatalf("expected both instances, got %+v", require)
	}
	if msg := instanceRequiredMessage(instances); msg != "connector_instance is required when multiple instances exist and no default is set" {
		t.Fatalf("message = %q", msg)
	}
}

func TestSelectDefaultConnectorInstance_UnreadyDefaultKeptWhenNothingElseWorks(t *testing.T) {
	instances := []db.AgentConnectorInstance{inst("revoked", true), inst("also", false)}
	eligibility := []db.ConnectorInstanceEligibility{
		elig("revoked", false, false, true),
		elig("also", false, false, true),
	}
	chosen, require := selectDefaultConnectorInstance(instances, eligibility)
	if len(require) != 0 {
		t.Fatalf("expected no required list, got %+v", require)
	}
	if chosen == nil || chosen.ConnectorInstanceID != "revoked" {
		t.Fatalf("expected revoked default, got %+v", chosen)
	}
}
