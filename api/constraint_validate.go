package api

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"strings"
	"time"

	"github.com/supersuit-tech/permission-slip/connectors"
	"github.com/supersuit-tech/permission-slip/db"
)

// constraintMetadataResolveTimeout bounds verified metadata lookup on the
// constraint-matching path (standing approvals, action configs, bulk).
const constraintMetadataResolveTimeout = 30 * time.Second

// validateActionConstraints checks execution parameters and optional $meta
// constraints. When the configuration includes $meta constraints, verified
// metadata is resolved via the connector before matching.
func validateActionConstraints(
	ctx context.Context,
	deps *Deps,
	agentID int64,
	userID string,
	actionType string,
	connectorInstanceID string,
	configConstraints json.RawMessage,
	execParams json.RawMessage,
) error {
	if !configHasMetaConstraints(configConstraints) {
		return db.ValidateParametersAgainstConfig(configConstraints, execParams, nil)
	}

	resolvedMeta, err := resolveConstraintMetadata(ctx, deps, agentID, userID, actionType, connectorInstanceID, execParams)
	if err != nil {
		return err
	}

	var metaBytes json.RawMessage
	if resolvedMeta != nil {
		metaBytes, err = json.Marshal(resolvedMeta)
		if err != nil {
			return err
		}
	}

	return db.ValidateParametersAgainstConfig(configConstraints, execParams, metaBytes)
}

func configHasMetaConstraints(configConstraints json.RawMessage) bool {
	if len(configConstraints) == 0 || string(configConstraints) == "null" {
		return false
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(configConstraints, &obj); err != nil {
		return false
	}
	meta, ok := obj[db.MetaNamespaceKey]
	return ok && len(meta) > 0 && string(meta) != "null" && string(meta) != "{}"
}

func resolveConstraintMetadata(
	ctx context.Context,
	deps *Deps,
	agentID int64,
	userID string,
	actionType string,
	connectorInstanceID string,
	execParams json.RawMessage,
) (map[string]any, error) {
	if deps == nil || deps.Connectors == nil {
		return nil, db.ErrMetadataUnresolved
	}

	parts := strings.SplitN(actionType, ".", 2)
	if len(parts) != 2 {
		return nil, db.ErrMetadataUnresolved
	}
	connectorID := parts[0]

	conn, ok := deps.Connectors.Get(connectorID)
	if !ok {
		return nil, db.ErrMetadataUnresolved
	}

	resolver, ok := conn.(connectors.ConstraintMetadataResolver)
	if !ok {
		return nil, db.ErrMetadataUnresolved
	}

	resolveCtx, cancel := context.WithTimeout(ctx, constraintMetadataResolveTimeout)
	defer cancel()
	resolveCtx, creds := resolveResourceDetailsContext(resolveCtx, deps, agentID, userID, actionType, connectorID, connectorInstanceID)

	meta, err := resolver.ResolveConstraintMetadata(resolveCtx, actionType, execParams, creds)
	if err != nil {
		if errors.Is(err, connectors.ErrConstraintMetadataUnavailable) {
			log.Printf("[%s] ResolveConstraintMetadata unavailable for %s: %v", TraceID(ctx), actionType, err)
			return nil, db.ErrMetadataUnresolved
		}
		log.Printf("[%s] ResolveConstraintMetadata failed for %s: %v", TraceID(ctx), actionType, err)
		return nil, db.ErrMetadataUnresolved
	}
	if meta == nil {
		return nil, db.ErrMetadataUnresolved
	}
	return meta, nil
}

// constraintMatchErr reports whether a constraint validation error should be
// treated as a non-match (fall through) rather than a hard failure.
func constraintMatchErr(err error) (configErr *db.ConfigValidationError, unresolved bool) {
	if errors.Is(err, db.ErrMetadataUnresolved) {
		return nil, true
	}
	var cve *db.ConfigValidationError
	if errors.As(err, &cve) {
		return cve, false
	}
	return nil, false
}
