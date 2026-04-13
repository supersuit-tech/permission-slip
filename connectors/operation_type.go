package connectors

import "strings"

// OperationType classifies connector actions for bulk template UX (read / write / delete).
type OperationType string

const (
	OperationRead   OperationType = "read"
	OperationWrite  OperationType = "write"
	OperationDelete OperationType = "delete"
)

// InferOperationType derives read/write/delete from the action_type suffix (after the first '.').
// Manifests may override via ManifestAction.OperationType; this is used when unset.
func InferOperationType(actionType string) OperationType {
	dot := strings.IndexByte(actionType, '.')
	if dot < 0 || dot == len(actionType)-1 {
		return OperationWrite
	}
	suffix := actionType[dot+1:]
	if suffix == "" {
		return OperationWrite
	}
	for _, seg := range strings.Split(suffix, "_") {
		if seg == "" {
			continue
		}
		if deleteVerbs[seg] {
			return OperationDelete
		}
		if readVerbs[seg] {
			return OperationRead
		}
		if writeVerbs[seg] {
			return OperationWrite
		}
	}
	return OperationWrite
}

// Multi-word or domain-specific segments that imply read operations.
var readVerbs = map[string]bool{
	"get":         true,
	"list":        true,
	"read":        true,
	"search":      true,
	"describe":    true,
	"query":       true,
	"check":       true,
	"download":    true,
	"export":      true,
	"fetch":       true,
	"view":        true,
	"price":       true, // price_check → split gives price? Actually "price_check" is one segment — see below
	"price_check": true,
}

var deleteVerbs = map[string]bool{
	"delete":    true,
	"remove":    true,
	"cancel":    true,
	"close":     true,
	"archive":   true,
	"purge":     true,
	"void":      true,
	"unfollow":  true,
	"unlike":    true,
	"unretweet": true,
}

var writeVerbs = map[string]bool{
	"create":     true,
	"update":     true,
	"send":       true,
	"post":       true,
	"put":        true,
	"patch":      true,
	"add":        true,
	"merge":      true,
	"trigger":    true,
	"upload":     true,
	"share":      true,
	"move":       true,
	"complete":   true,
	"transition": true,
	"assign":     true,
	"invite":     true,
	"set":        true,
	"schedule":   true,
	"fulfill":    true,
	"enroll":     true,
	"reply":      true,
	"retweet":    true,
	"like":       true,
	"follow":     true,
	"swap":       true,
	"book":       true,
	"issue":      true,
	"adjust":     true,
	"record":     true,
	"convert":    true,
	"run":        true,
	"append":     true,
	"write":      true,
	"tag":        true,
	"reconcile":  true,
}
