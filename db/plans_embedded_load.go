package db

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sync"
)

//go:embed plans_embedded.json
var plansEmbeddedJSON []byte

// planFile matches plans_embedded.json entries (IDs are map keys).
type planFile struct {
	Name                      string `json:"name"`
	MaxRequestsPerMonth       *int   `json:"max_requests_per_month"`
	MaxAgents                 *int   `json:"max_agents"`
	MaxStandingApprovals      *int   `json:"max_standing_approvals"`
	MaxCredentials            *int   `json:"max_credentials"`
	AuditRetentionDays        int    `json:"audit_retention_days"`
	PricePerRequestMillicents int    `json:"price_per_request_millicents"`
}

var validEmbeddedPlanID = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

var (
	embeddedOnce   sync.Once
	embeddedPlans  map[string]*Plan
	embeddedList   []*Plan
	embeddedLoadErr error
)

func loadEmbeddedPlans() {
	embeddedOnce.Do(func() {
		var raw map[string]*planFile
		if err := json.Unmarshal(plansEmbeddedJSON, &raw); err != nil {
			embeddedLoadErr = fmt.Errorf("parse plans_embedded.json: %w", err)
			return
		}
		out := make(map[string]*Plan, len(raw))
		list := make([]*Plan, 0, len(raw))
		for id, p := range raw {
			if !validEmbeddedPlanID.MatchString(id) {
				embeddedLoadErr = fmt.Errorf("invalid plan ID %q: must be alphanumeric/underscore", id)
				return
			}
			pl := &Plan{
				ID:                        id,
				Name:                      p.Name,
				MaxRequestsPerMonth:       p.MaxRequestsPerMonth,
				MaxAgents:                 p.MaxAgents,
				MaxStandingApprovals:      p.MaxStandingApprovals,
				MaxCredentials:            p.MaxCredentials,
				AuditRetentionDays:        p.AuditRetentionDays,
				PricePerRequestMillicents: p.PricePerRequestMillicents,
			}
			out[id] = pl
			list = append(list, pl)
		}
		embeddedPlans = out
		embeddedList = list
	})
}

// GetPlan returns the plan with the given ID from embedded config, or nil if not found.
func GetPlan(id string) *Plan {
	loadEmbeddedPlans()
	if embeddedLoadErr != nil {
		return nil
	}
	p := embeddedPlans[id]
	if p == nil {
		return nil
	}
	cp := *p
	return &cp
}

// AllPlans returns copies of all embedded plans.
func AllPlans() []*Plan {
	loadEmbeddedPlans()
	if embeddedLoadErr != nil {
		return nil
	}
	out := make([]*Plan, len(embeddedList))
	for i, p := range embeddedList {
		cp := *p
		out[i] = &cp
	}
	return out
}

// MustGetPlan returns the plan with the given ID, panicking if not found.
func MustGetPlan(id string) *Plan {
	p := GetPlan(id)
	if p == nil {
		panic(fmt.Sprintf("plan %q not found in plans_embedded.json", id))
	}
	return p
}
