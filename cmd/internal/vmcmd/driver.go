package vmcmd

import (
	"context"
	"errors"
	"io"
)

// ErrUnconfirmedProcessCleanup means an owned command group was not proved
// gone. Callers must retain its durable claim and inputs.
var ErrUnconfirmedProcessCleanup = errors.New("VM subprocess cleanup unconfirmed")

type VMStatus struct {
	Exists bool
	State  VMObservation
}

type VMChild interface {
	PID() int
	Identity() string
}

// VMDriver owns transport and observations, while the host owns lane claims,
// snapshots and decisions about when cloning/deletion are permitted.
type VMDriver interface {
	Inspect(context.Context, string) (VMStatus, error)
	Clone(context.Context, string, string) error
	Launch(context.Context, string, string, string, io.Writer) (VMChild, error)
	AwaitGuest(context.Context, string) error
	Execute(context.Context, string, []string, io.Writer, io.Writer) (int, error)
	Stop(context.Context, string, VMChild) error
	Delete(context.Context, string) error
	Recover(context.Context, LaunchClaim) (RecoveryEvidence, error)
}
