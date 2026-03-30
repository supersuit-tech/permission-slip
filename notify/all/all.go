// Package all blank-imports every built-in notification sender package so that
// their init() functions run and self-register with the notify sender registry.
//
// Usage in main.go:
//
//	import _ "github.com/supersuit-tech/permission-slip/notify/all"
//
// To add a new sender, create notify/<channel>/register.go with an init() that
// calls notify.RegisterSenderFactory(...), then add a blank import here.
//
// Note: email and SMS senders register themselves via init() in the notify
// package itself, so no additional imports are needed for those channels.
package all

import (
	_ "github.com/supersuit-tech/permission-slip/notify/mobilepush"
	_ "github.com/supersuit-tech/permission-slip/notify/webpush"
)
