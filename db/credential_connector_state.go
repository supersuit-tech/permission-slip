package db

import (
	"context"
	"encoding/json"
	"fmt"
)

// ProtonmailConnectorState holds non-secret Proton Mail sync metadata for a credential.
type ProtonmailConnectorState struct {
	Folders map[string]ProtonmailFolderState `json:"folders,omitempty"`
}

// ProtonmailFolderState records IMAP state for a single mailbox folder.
type ProtonmailFolderState struct {
	UIDValidity uint32 `json:"uidvalidity"`
}

type credentialConnectorState struct {
	Protonmail *ProtonmailConnectorState `json:"protonmail,omitempty"`
}

// GetProtonmailUIDValidity returns the stored UIDVALIDITY for a folder, or false
// if none has been recorded yet.
func GetProtonmailUIDValidity(ctx context.Context, db DBTX, credentialID, folder string) (uint32, bool, error) {
	state, err := loadCredentialConnectorState(ctx, db, credentialID)
	if err != nil {
		return 0, false, err
	}
	if state.Protonmail == nil || state.Protonmail.Folders == nil {
		return 0, false, nil
	}
	folderState, ok := state.Protonmail.Folders[folder]
	if !ok {
		return 0, false, nil
	}
	return folderState.UIDValidity, true, nil
}

// SetProtonmailUIDValidity records UIDVALIDITY for a folder on the credential.
func SetProtonmailUIDValidity(ctx context.Context, db DBTX, credentialID, folder string, uidValidity uint32) error {
	state, err := loadCredentialConnectorState(ctx, db, credentialID)
	if err != nil {
		return err
	}
	if state.Protonmail == nil {
		state.Protonmail = &ProtonmailConnectorState{}
	}
	if state.Protonmail.Folders == nil {
		state.Protonmail.Folders = make(map[string]ProtonmailFolderState)
	}
	state.Protonmail.Folders[folder] = ProtonmailFolderState{UIDValidity: uidValidity}

	raw, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("marshal connector_state: %w", err)
	}

	_, err = db.Exec(ctx, `
		UPDATE credentials
		SET connector_state = $2
		WHERE id = $1`, credentialID, raw)
	return err
}

func loadCredentialConnectorState(ctx context.Context, db DBTX, credentialID string) (*credentialConnectorState, error) {
	var raw []byte
	err := db.QueryRow(ctx, `
		SELECT connector_state
		FROM credentials
		WHERE id = $1`, credentialID).Scan(&raw)
	if err != nil {
		return nil, err
	}
	if len(raw) == 0 {
		return &credentialConnectorState{}, nil
	}
	var state credentialConnectorState
	if err := json.Unmarshal(raw, &state); err != nil {
		return nil, fmt.Errorf("unmarshal connector_state: %w", err)
	}
	return &state, nil
}
