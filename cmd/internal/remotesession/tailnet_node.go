package remotesession

import (
	"context"
	"net/netip"
)

// tailnetNode is the IO boundary for one explicit, owned node lifetime. Merely
// constructing an adapter must not initialize a node. Start begins that work;
// Close must join it. A Close error means cleanup remains unproved and callers
// must retain ownership rather than start a replacement.
//
// This partial seam has no production adapter yet. In particular it does not
// establish listener/TLS or accepted-connection provenance. The native adapter
// must add its owned ListenTLS boundary before serving any remote requests.
type tailnetNode interface {
	Start(context.Context) error
	Status(context.Context) (tailnetNodeStatus, error)
	WhoIs(context.Context, netip.AddrPort) (tailnetObservation, error)
	Watch(context.Context) (tailnetWatch, error)
	Close() error
}

// tailnetWatch owns its read lifetime. Close cancels and joins outstanding Next
// calls. Any read error is uncertain health, not permission to retain authority.
type tailnetWatch interface {
	Next() (tailnetNodeStatus, error)
	Close() error
}

type tailnetNodePhase uint8

const (
	tailnetNodePending tailnetNodePhase = iota
	tailnetNodeRunning
	tailnetNodeNeedsLogin
	tailnetNodeRevoked
	tailnetNodeClosed
)

// These are raw backend observations, not authorization. fqdn is the backend's
// actual assigned name. loginURL is sensitive console-only enrollment material:
// never publish it through remote status, logs, transcript or audit records.
type tailnetNodeStatus struct {
	phase      tailnetNodePhase
	fqdn       string
	selfNodeID string
	selfUserID int64
	loginURL   string
}
