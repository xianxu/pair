package ptychild

import (
	"errors"
	"os/exec"
)

// asExitError keeps the errors.As call in one place, mirroring couchcore's
// errors.go. Inlining it at the call site is how a second, subtly different
// unwrap gets written later.
func asExitError(err error, target **exec.ExitError) bool { return errors.As(err, target) }
