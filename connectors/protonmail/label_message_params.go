package protonmail

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/supersuit-tech/permission-slip/connectors"
)

type labelMessageParams struct {
	MessageIDs    []uint32 `json:"-"`
	Folder        string   `json:"folder"`
	Label         string   `json:"label"`
	LabelMailbox  string   `json:"-"`
	IncludeThread *bool    `json:"include_thread"`
}

func parseLabelMessageParams(raw []byte) (*labelMessageParams, error) {
	var r struct {
		uidMessageRaw
		Label         string `json:"label"`
		IncludeThread *bool  `json:"include_thread"`
	}
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}

	base := &uidMessageParams{Folder: r.Folder}
	if len(r.MessageIDs) > 0 && string(r.MessageIDs) != "null" {
		if err := json.Unmarshal(r.MessageIDs, &base.MessageIDs); err != nil {
			return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid message_ids: %v", err)}
		}
	}
	if r.MessageID != nil {
		base.MessageIDs = append(base.MessageIDs, *r.MessageID)
	}

	return &labelMessageParams{
		MessageIDs:    base.MessageIDs,
		Folder:        base.Folder,
		Label:         r.Label,
		IncludeThread: r.IncludeThread,
	}, nil
}

func (p *labelMessageParams) validate() error {
	base := &uidMessageParams{MessageIDs: p.MessageIDs, Folder: p.Folder}
	if err := base.validate(); err != nil {
		return err
	}
	p.MessageIDs = base.MessageIDs
	p.Folder = base.Folder

	labelMailbox, err := validateLabelParam(p.Label)
	if err != nil {
		return err
	}
	p.LabelMailbox = labelMailbox

	if strings.EqualFold(p.Folder, labelMailbox) {
		return &connectors.ValidationError{Message: "source folder must differ from the label mailbox"}
	}
	return nil
}
