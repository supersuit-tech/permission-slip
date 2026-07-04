package api

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/supersuit-tech/permission-slip/db"
)

// applyStandingApprovalDataWindow resolves $data_window on a matched standing
// approval and injects/clamps the connector's window parameters. Returns the
// original params when no $data_window is present.
func applyStandingApprovalDataWindow(
	ctx context.Context,
	d db.DBTX,
	actionType string,
	constraints json.RawMessage,
	params json.RawMessage,
	now time.Time,
) (json.RawMessage, error) {
	dwRaw, ok := db.ExtractDataWindowConstraint(constraints)
	if !ok {
		return params, nil
	}

	pair, err := db.GetActionDataWindowParams(ctx, d, actionType)
	if err != nil {
		return nil, err
	}
	if pair == nil {
		return nil, db.ErrDataWindowUnsupported
	}

	window, err := db.ResolveDataWindowConstraint(dwRaw, now)
	if err != nil {
		return nil, err
	}

	return db.ApplyDataWindowToParams(params, window, pair.StartParam, pair.EndParam)
}

// isDataWindowUnsupported reports whether err means the standing approval should
// fail closed (fall through to pending approval).
func isDataWindowUnsupported(err error) bool {
	return errors.Is(err, db.ErrDataWindowUnsupported)
}
