package protonmail

import (
	"context"

	"github.com/emersion/go-imap/v2"
	"github.com/supersuit-tech/permission-slip/connectors"
)

type storeMessageFlagsAction struct {
	conn       *ProtonMailConnector
	actionType string
	op         imap.StoreFlagsOp
	flags      []imap.Flag
	status     string
}

func (a *storeMessageFlagsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	params, err := parseUIDMessageParams(req.Parameters)
	if err != nil {
		return nil, err
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	err = executeUIDMessageAction(ctx, a.conn, req, params, func(session *imapSession, uidSet imap.UIDSet) error {
		if err := storeMessageFlags(session, uidSet, a.op, a.flags); err != nil {
			return mapUIDNotFoundError(err, params.Folder)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]any{
		"status":      a.status,
		"folder":      params.Folder,
		"updated":     len(params.MessageIDs),
		"message_ids": params.MessageIDs,
	})
}

func newMarkReadAction(conn *ProtonMailConnector) connectors.Action {
	return &storeMessageFlagsAction{
		conn:       conn,
		actionType: "protonmail.mark_read",
		op:         imap.StoreFlagsAdd,
		flags:      []imap.Flag{imap.FlagSeen},
		status:     "marked_read",
	}
}

func newMarkUnreadAction(conn *ProtonMailConnector) connectors.Action {
	return &storeMessageFlagsAction{
		conn:       conn,
		actionType: "protonmail.mark_unread",
		op:         imap.StoreFlagsDel,
		flags:      []imap.Flag{imap.FlagSeen},
		status:     "marked_unread",
	}
}

func newFlagAction(conn *ProtonMailConnector) connectors.Action {
	return &storeMessageFlagsAction{
		conn:       conn,
		actionType: "protonmail.flag",
		op:         imap.StoreFlagsAdd,
		flags:      []imap.Flag{imap.FlagFlagged},
		status:     "flagged",
	}
}

func newUnflagAction(conn *ProtonMailConnector) connectors.Action {
	return &storeMessageFlagsAction{
		conn:       conn,
		actionType: "protonmail.unflag",
		op:         imap.StoreFlagsDel,
		flags:      []imap.Flag{imap.FlagFlagged},
		status:     "unflagged",
	}
}
