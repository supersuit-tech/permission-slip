package protonmail

import (
	"encoding/json"
	"fmt"

	"github.com/emersion/go-imap/v2"
	"github.com/supersuit-tech/permission-slip/connectors"
)

// uidMessageRaw handles flexible JSON input: accepts either a single integer
// for message_id or an array for message_ids.
type uidMessageRaw struct {
	MessageID  *uint32         `json:"message_id,omitempty"`
	MessageIDs json.RawMessage `json:"message_ids,omitempty"`
	Folder     string          `json:"folder"`
}

// uidMessageParams normalizes UID-scoped message actions.
type uidMessageParams struct {
	MessageIDs []uint32 `json:"-"`
	Folder     string   `json:"folder"`
}

func parseUIDMessageParams(raw []byte) (*uidMessageParams, error) {
	var r uidMessageRaw
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}

	params := &uidMessageParams{Folder: r.Folder}

	if len(r.MessageIDs) > 0 && string(r.MessageIDs) != "null" {
		if err := json.Unmarshal(r.MessageIDs, &params.MessageIDs); err != nil {
			return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid message_ids: %v", err)}
		}
	}

	if r.MessageID != nil {
		params.MessageIDs = append(params.MessageIDs, *r.MessageID)
	}

	return params, nil
}

func (p *uidMessageParams) validate() error {
	if len(p.MessageIDs) == 0 {
		return &connectors.ValidationError{Message: "missing required parameter: provide message_id (single) or message_ids (array)"}
	}

	p.MessageIDs = deduplicateUint32(p.MessageIDs)

	if len(p.MessageIDs) > maxLimit {
		return &connectors.ValidationError{Message: fmt.Sprintf("too many message_ids: maximum is %d", maxLimit)}
	}
	for _, id := range p.MessageIDs {
		if id == 0 {
			return &connectors.ValidationError{Message: "message_ids must not contain zero values"}
		}
	}
	if p.Folder == "" {
		p.Folder = "INBOX"
	}
	return nil
}

func uidSetFromMessageIDs(ids []uint32) imap.UIDSet {
	var uidSet imap.UIDSet
	for _, id := range ids {
		uidSet.AddNum(imap.UID(id))
	}
	return uidSet
}

// deduplicateUint32 returns a new slice with duplicate values removed,
// preserving the original order.
func deduplicateUint32(ids []uint32) []uint32 {
	seen := make(map[uint32]struct{}, len(ids))
	out := make([]uint32, 0, len(ids))
	for _, id := range ids {
		if _, ok := seen[id]; !ok {
			seen[id] = struct{}{}
			out = append(out, id)
		}
	}
	return out
}
