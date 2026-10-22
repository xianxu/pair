package remotesession

import (
	"errors"
	"net/netip"
	"time"
	"unicode"
	"unicode/utf8"
)

// TailnetPrincipal is value evidence from this epoch's owned connection/WhoIs
// adapter. Its private fields prevent callers outside this package from creating
// verified identities. Pure validation does not authenticate external IO.
type TailnetPrincipal struct {
	login               string
	userID              int64
	nodeID              string
	peerIP              netip.Addr
	epoch               uint64
	observedAt, expires time.Time
	nonExpiring         bool
}

type tailnetObservationState uint8

const (
	tailnetObservationUnknown tailnetObservationState = iota
	tailnetObservationCurrent
	tailnetObservationRevoked
)

// tailnetObservation is supplied only by the owned connection/WhoIs seam.
// Unknown health, tagged identity and unknown expiration all fail closed.
// nonExpiring must be explicit and must not accompany an expiry deadline.
type tailnetObservation struct {
	login               string
	userID              int64
	nodeID              string
	peerIP              netip.Addr
	epoch               uint64
	observedAt, expires time.Time
	nonExpiring         bool
	state               tailnetObservationState
	tagged              bool
}

var errTailnetIdentityRefused = errors.New("tailnet identity refused")

func newTailnetPrincipal(o tailnetObservation, now time.Time) (TailnetPrincipal, error) {
	if o.state != tailnetObservationCurrent || o.tagged || o.peerIP.Zone() != "" {
		return TailnetPrincipal{}, errTailnetIdentityRefused
	}
	p := TailnetPrincipal{login: o.login, userID: o.userID, nodeID: o.nodeID, peerIP: o.peerIP.Unmap(), epoch: o.epoch, observedAt: o.observedAt, expires: o.expires, nonExpiring: o.nonExpiring}
	if !p.validAt(now) {
		return TailnetPrincipal{}, errTailnetIdentityRefused
	}
	return p, nil
}
func (p TailnetPrincipal) validAt(now time.Time) bool {
	if !validTailnetIdentityText(p.login) || !validTailnetIdentityText(p.nodeID) || p.userID <= 0 || p.epoch == 0 || p.observedAt.IsZero() || now.IsZero() || now.Before(p.observedAt) {
		return false
	}
	if !tailnetIPv4.Contains(p.peerIP) && !tailnetIPv6.Contains(p.peerIP) || p.peerIP.Zone() != "" || p.peerIP.Is4In6() {
		return false
	}
	if p.nonExpiring {
		return p.expires.IsZero()
	}
	return !p.expires.IsZero() && now.Before(p.expires)
}

// samePrincipal compares stable authority, allowing an observation refresh to
// change its validity deadline without changing the session identity.
func (p TailnetPrincipal) samePrincipal(other TailnetPrincipal) bool {
	return p.validAt(p.observedAt) && other.validAt(other.observedAt) && p.login == other.login && p.userID == other.userID && p.nodeID == other.nodeID && p.peerIP == other.peerIP && p.epoch == other.epoch
}

var (
	tailnetIPv4 = netip.MustParsePrefix("100.64.0.0/10")
	tailnetIPv6 = netip.MustParsePrefix("fd7a:115c:a1e0::/48")
)

// Login is an opaque, exact user name, not necessarily an email address. The
// bound and rejected delimiters keep it safe for the remote attribution header.
func validTailnetIdentityText(value string) bool {
	if len(value) == 0 || len(value) > 256 || !utf8.ValidString(value) {
		return false
	}
	for _, r := range value {
		if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) || unicode.IsSpace(r) || r == '[' || r == ']' || r == '<' || r == '>' {
			return false
		}
	}
	return true
}

// TailnetLoginPolicy permits only explicitly configured, case-sensitive logins.
// The zero policy denies every principal.
type TailnetLoginPolicy struct{ allowed map[string]struct{} }

func NewTailnetLoginPolicy(logins []string) (TailnetLoginPolicy, error) {
	if len(logins) == 0 || len(logins) > 64 {
		return TailnetLoginPolicy{}, errTailnetIdentityRefused
	}
	allowed := make(map[string]struct{}, len(logins))
	for _, login := range logins {
		if !validTailnetIdentityText(login) {
			return TailnetLoginPolicy{}, errTailnetIdentityRefused
		}
		allowed[login] = struct{}{}
	}
	return TailnetLoginPolicy{allowed: allowed}, nil
}
func (p TailnetLoginPolicy) Allows(principal TailnetPrincipal, now time.Time) bool {
	if !principal.validAt(now) {
		return false
	}
	_, ok := p.allowed[principal.login]
	return ok
}
