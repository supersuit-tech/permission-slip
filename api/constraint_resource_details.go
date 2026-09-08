package api

import (
	"context"
	"encoding/json"
	"log"
	"strings"

	"github.com/supersuit-tech/permission-slip/connectors"
	"github.com/supersuit-tech/permission-slip/db"
)

// maxConstraintResourceLookups bounds how many ResolveResourceDetails calls a
// single standing-approval write will make (one per extra any_of ID).
const maxConstraintResourceLookups = 20

// resolveConstraintResourceDetails walks standing-approval constraints, collects
// fixed string IDs, and asks the connector resolver for human-readable names
// and URLs. Lookup failures are non-fatal and return nil.
func resolveConstraintResourceDetails(
	ctx context.Context,
	deps *Deps,
	agentID int64,
	userID string,
	actionType string,
	connectorInstanceID *string,
	constraints []byte,
) []byte {
	if deps == nil || deps.Connectors == nil || len(constraints) == 0 {
		return nil
	}
	idsByField, err := collectFixedConstraintIDs(constraints)
	if err != nil || len(idsByField) == 0 {
		return nil
	}

	resolver, connectorID, ok := resourceDetailResolverForAction(deps, actionType)
	if !ok {
		return nil
	}

	instanceID := ""
	if connectorInstanceID != nil {
		instanceID = *connectorInstanceID
	}

	params := firstIDsParams(idsByField)
	merged := lookupResourceDetails(ctx, deps, resolver, connectorID, agentID, userID, actionType, instanceID, params)

	lookups := 1
	for field, ids := range idsByField {
		for _, extraID := range ids[1:] {
			if lookups >= maxConstraintResourceLookups {
				break
			}
			extra := cloneStringMap(params)
			extra[field] = extraID
			part := lookupResourceDetails(ctx, deps, resolver, connectorID, agentID, userID, actionType, instanceID, extra)
			merged = connectors.MergeResourceDetails(merged, part)
			lookups++
		}
	}
	return marshalResourceDetails(merged)
}

func resourceDetailResolverForAction(deps *Deps, actionType string) (connectors.ResourceDetailResolver, string, bool) {
	cid := strings.SplitN(actionType, ".", 2)
	if len(cid) != 2 {
		return nil, "", false
	}
	conn, ok := deps.Connectors.Get(cid[0])
	if !ok {
		return nil, "", false
	}
	resolver, ok := conn.(connectors.ResourceDetailResolver)
	if !ok {
		return nil, "", false
	}
	return resolver, cid[0], true
}

func lookupResourceDetails(
	ctx context.Context,
	deps *Deps,
	resolver connectors.ResourceDetailResolver,
	connectorID string,
	agentID int64,
	userID, actionType, connectorInstanceID string,
	params map[string]string,
) map[string]any {
	if len(params) == 0 {
		return nil
	}
	encoded, err := json.Marshal(params)
	if err != nil {
		return nil
	}
	resolveCtx, cancel := context.WithTimeout(ctx, resourceDetailsResolveTimeout)
	defer cancel()
	resolveCtx, creds := resolveResourceDetailsContext(resolveCtx, deps, agentID, userID, actionType, connectorID, connectorInstanceID)
	details, err := retryResourceDetails(resolveCtx, func(callCtx context.Context) (map[string]any, error) {
		return resolver.ResolveResourceDetails(callCtx, actionType, encoded, creds)
	})
	if err != nil {
		log.Printf("[%s] ResolveResourceDetails (constraints): %v", TraceID(ctx), err)
		return nil
	}
	return details
}

func marshalResourceDetails(details map[string]any) []byte {
	if len(details) == 0 {
		return nil
	}
	encoded, err := json.Marshal(details)
	if err != nil {
		return nil
	}
	return encoded
}

func firstIDsParams(idsByField map[string][]string) map[string]string {
	params := make(map[string]string, len(idsByField))
	for field, ids := range idsByField {
		if len(ids) > 0 {
			params[field] = ids[0]
		}
	}
	return params
}

func cloneStringMap(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

// collectFixedConstraintIDs returns unique fixed (non-wildcard, non-pattern)
// string or numeric IDs keyed by constraint field.
func collectFixedConstraintIDs(raw []byte) (map[string][]string, error) {
	sc, err := db.ParseStructuredConstraints(raw)
	if err != nil {
		return nil, err
	}
	seen := map[string]map[string]struct{}{}
	for _, group := range sc.Groups {
		for _, cond := range group.Conditions {
			if cond.Field == "" || strings.HasPrefix(cond.Field, db.MetaNamespaceKey) || cond.Field == db.DataWindowNamespaceKey {
				continue
			}
			switch cond.Op {
			case db.OpNoneOf, db.OpDoesNotMatch, db.OpLte, db.OpGte, db.OpLt, db.OpGt:
				continue
			}
			values := cond.Values
			if len(values) == 0 && len(cond.Value) > 0 {
				values = []json.RawMessage{cond.Value}
			}
			for _, v := range values {
				id, ok := decodeFixedConstraintID(v)
				if !ok {
					continue
				}
				if seen[cond.Field] == nil {
					seen[cond.Field] = map[string]struct{}{}
				}
				seen[cond.Field][id] = struct{}{}
			}
		}
	}
	out := make(map[string][]string, len(seen))
	for field, ids := range seen {
		list := make([]string, 0, len(ids))
		for id := range ids {
			list = append(list, id)
		}
		out[field] = list
	}
	return out, nil
}

func decodeFixedConstraintID(raw json.RawMessage) (string, bool) {
	if len(raw) == 0 || string(raw) == "null" {
		return "", false
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		if s == "" || s == "*" {
			return "", false
		}
		return s, true
	}
	var n json.Number
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.UseNumber()
	if err := dec.Decode(&n); err == nil {
		if n.String() == "" {
			return "", false
		}
		return n.String(), true
	}
	return "", false
}

func unmarshalResourceDetails(raw []byte) any {
	if len(raw) == 0 {
		return nil
	}
	var details any
	if err := json.Unmarshal(raw, &details); err != nil {
		return nil
	}
	return details
}
