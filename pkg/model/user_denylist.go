package model

import (
	"strings"
	"sync"
)

// disabledUsernames carries usernames whose sessions must be rejected
// without a database read. Sealos desktop sessions authenticate from JWT
// claims (no per-request user lookup), so an administrative disable is made
// effective for live sessions through this registry: SetUserEnabled records
// flips here, and the Sealos login and hydration paths register what the
// users table says. A user without a row cannot be disabled, so absence
// from the registry always means "not disabled".
var disabledUsernames sync.Map // username -> struct{}

// SetUsernameDisabled records whether username's sessions must be rejected.
func SetUsernameDisabled(username string, disabled bool) {
	username = strings.TrimSpace(username)
	if username == "" {
		return
	}
	if disabled {
		disabledUsernames.Store(username, struct{}{})
		return
	}
	disabledUsernames.Delete(username)
}

// IsUsernameDisabled reports whether username's sessions must be rejected.
func IsUsernameDisabled(username string) bool {
	_, ok := disabledUsernames.Load(username)
	return ok
}
