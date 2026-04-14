package connectors

import "strings"

// OperationType classifies connector actions for bulk template UX (read / write / edit / delete).
type OperationType string

const (
	OperationRead   OperationType = "read"
	OperationWrite  OperationType = "write"
	OperationEdit   OperationType = "edit"
	OperationDelete OperationType = "delete"
)

// InferOperationType derives read/write/edit/delete from the action_type suffix (after the first '.').
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
		if editVerbs[seg] {
			return OperationEdit
		}
		if writeVerbs[seg] {
			return OperationWrite
		}
	}
	return OperationWrite
}

// Multi-word or domain-specific segments that imply read operations.
var readVerbs = map[string]bool{
	"get":      true,
	"list":     true,
	"read":     true,
	"search":   true,
	"describe": true,
	"query":    true,
	"check":    true,
	"download": true,
	"export":   true,
	"fetch":    true,
	"view":     true,
	"price":    true, // e.g. expedia.price_check → segment "price" matches before "check"
}

// deleteVerbs includes verbs that remove or irreversibly change external state. Note that
// connectors often use "close", "archive", or "cancel" for non-destructive workflow moves;
// we still classify those as delete so bulk "Delete actions" approval covers them — users
// who need finer control can approve per template.
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
	"unpin":     true,
}

// editVerbs covers modifying existing resources without necessarily creating new ones.
var editVerbs = map[string]bool{
	"update": true,
	"set":    true,
	"rename": true,
	"edit":   true,
	"modify": true,
	"patch":  true,
	"put":    true,
}

var writeVerbs = map[string]bool{
	"create":     true,
	"send":       true,
	"post":       true,
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
