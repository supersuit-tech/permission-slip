package google

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// googleDriveWriteMetadataActions are Drive writes whose destination folder
// can be resolved to a verified Shared Drive ID for $meta.drive_id matching.
var googleDriveWriteMetadataActions = map[string]struct{}{
	"google.upload_drive_file":   {},
	"google.create_drive_folder": {},
}

// googleDriveMetaConstraintFields are valid $meta keys for Drive write actions.
var googleDriveMetaConstraintFields = []string{"drive_id"}

// ResolveConstraintMetadata returns verified Drive membership for standing
// approval matching. drive_id is the Shared Drive the destination folder lives
// on (the drive root ID when uploading/creating at the Shared Drive root, or
// the ancestor drive for any descendant folder). My Drive targets omit drive_id
// so a Shared Drive constraint does not match.
func (c *GoogleConnector) ResolveConstraintMetadata(ctx context.Context, actionType string, params json.RawMessage, creds connectors.Credentials) (map[string]any, error) {
	if _, ok := googleDriveWriteMetadataActions[actionType]; !ok {
		return nil, connectors.ErrConstraintMetadataUnavailable
	}

	var p struct {
		FolderID string `json:"folder_id"`
		ParentID string `json:"parent_id"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("%w: invalid drive folder params: %v", connectors.ErrConstraintMetadataUnavailable, err)
	}

	id := driveFolderTargetID(p.FolderID, p.ParentID)
	if id == "" {
		// Omitted destination is My Drive root — not a Shared Drive member.
		return map[string]any{}, nil
	}
	if !isValidDriveID(id) {
		return nil, fmt.Errorf("%w: invalid folder id", connectors.ErrConstraintMetadataUnavailable)
	}

	folder, err := c.lookupDriveFolder(ctx, creds, id)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", connectors.ErrConstraintMetadataUnavailable, err)
	}

	meta := map[string]any{}
	if folder.DriveID != "" {
		meta["drive_id"] = folder.DriveID
	}
	return meta, nil
}

// ConstraintMetadataActionSupport reports which $meta fields are valid per action.
func (c *GoogleConnector) ConstraintMetadataActionSupport(actionType string) ([]string, bool) {
	if _, ok := googleDriveWriteMetadataActions[actionType]; !ok {
		return nil, false
	}
	return googleDriveMetaConstraintFields, true
}

var _ connectors.ConstraintMetadataResolver = (*GoogleConnector)(nil)
var _ connectors.ConstraintMetadataCapabilities = (*GoogleConnector)(nil)
