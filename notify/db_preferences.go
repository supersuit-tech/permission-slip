package notify

import (
	"context"

	"github.com/supersuit-tech/permission-slip/db"
)

// DBPreferenceChecker adapts db.IsNotificationChannelEnabled to the
// PreferenceChecker interface so the Dispatcher can query the database
// without importing the db package directly in the hot path.
type DBPreferenceChecker struct {
	DB db.DBTX
}

// IsChannelEnabled returns true when the channel is enabled for the user.
// Missing preference rows default to enabled.
func (c *DBPreferenceChecker) IsChannelEnabled(ctx context.Context, userID, channel string) (bool, error) {
	return db.IsNotificationChannelEnabled(ctx, c.DB, userID, channel)
}
