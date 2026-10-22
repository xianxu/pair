package remotesession

import (
	"fmt"
	"net/netip"
	"strings"
	"testing"
	"time"
	"unicode"
)

func validTailnetObservation() tailnetObservation {
	now := time.Date(2026, 10, 22, 12, 0, 0, 0, time.UTC)
	return tailnetObservation{login: "Operator", userID: 12, nodeID: "n-stable", peerIP: netip.MustParseAddr("100.64.0.1"), epoch: 3, observedAt: now, expires: now.Add(time.Hour), state: tailnetObservationCurrent}
}

func TestTailnetPrincipalRejectsIncompleteOrUntrustedObservations(t *testing.T) {
	now := validTailnetObservation().observedAt
	cases := []struct {
		name   string
		mutate func(*tailnetObservation)
	}{
		{"missing login", func(o *tailnetObservation) { o.login = "" }},
		{"header delimiter", func(o *tailnetObservation) { o.login = "op]" }},
		{"control", func(o *tailnetObservation) { o.login = "op\r" }},
		{"missing user", func(o *tailnetObservation) { o.userID = 0 }},
		{"negative user", func(o *tailnetObservation) { o.userID = -1 }},
		{"missing node", func(o *tailnetObservation) { o.nodeID = "" }},
		{"long node", func(o *tailnetObservation) { o.nodeID = strings.Repeat("n", 257) }},
		{"missing epoch", func(o *tailnetObservation) { o.epoch = 0 }},
		{"missing IP", func(o *tailnetObservation) { o.peerIP = netip.Addr{} }},
		{"public IP", func(o *tailnetObservation) { o.peerIP = netip.MustParseAddr("8.8.8.8") }},
		{"loopback", func(o *tailnetObservation) { o.peerIP = netip.MustParseAddr("127.0.0.1") }},
		{"below v4 range", func(o *tailnetObservation) { o.peerIP = netip.MustParseAddr("100.63.255.255") }},
		{"above v4 range", func(o *tailnetObservation) { o.peerIP = netip.MustParseAddr("100.128.0.0") }},
		{"other v6", func(o *tailnetObservation) { o.peerIP = netip.MustParseAddr("fd7a:115c:a1e1::1") }},
		{"zone", func(o *tailnetObservation) { o.peerIP = netip.MustParseAddr("fd7a:115c:a1e0::1%en0") }},
		{"unknown", func(o *tailnetObservation) { o.state = tailnetObservationUnknown }},
		{"revoked", func(o *tailnetObservation) { o.state = tailnetObservationRevoked }},
		{"invalid state", func(o *tailnetObservation) { o.state = 255 }},
		{"tagged", func(o *tailnetObservation) { o.tagged = true }},
		{"missing observation", func(o *tailnetObservation) { o.observedAt = time.Time{} }},
		{"future observation", func(o *tailnetObservation) { o.observedAt = now.Add(time.Second) }},
		{"ambiguous expiry", func(o *tailnetObservation) { o.nonExpiring = true }},
		{"unknown expiry", func(o *tailnetObservation) { o.expires = time.Time{} }},
		{"expired", func(o *tailnetObservation) { o.expires = now }},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			o := validTailnetObservation()
			tt.mutate(&o)
			p, err := newTailnetPrincipal(o, now)
			if err == nil || p != (TailnetPrincipal{}) {
				t.Fatalf("invalid observation admitted: %+v, %v", p, err)
			}
		})
	}
	if (TailnetPrincipal{}).validAt(now) {
		t.Fatal("zero principal admitted")
	}
}

func TestTailnetPrincipalCanonicalAddressAndLifetime(t *testing.T) {
	for _, ip := range []string{"100.64.0.0", "100.127.255.255", "::ffff:100.64.0.1", "fd7a:115c:a1e0::1", "fd7a:115c:a1e0:ffff:ffff:ffff:ffff:ffff"} {
		t.Run(ip, func(t *testing.T) {
			o := validTailnetObservation()
			o.peerIP = netip.MustParseAddr(ip)
			p, err := newTailnetPrincipal(o, o.observedAt)
			if err != nil {
				t.Fatal(err)
			}
			if p.peerIP != o.peerIP.Unmap() {
				t.Fatal("address not canonical")
			}
			if !p.validAt(o.expires.Add(-time.Nanosecond)) || p.validAt(o.expires) || p.validAt(o.observedAt.Add(-time.Nanosecond)) {
				t.Fatal("incorrect validity interval")
			}
		})
	}
}

func TestTailnetPrincipalBindsEveryIdentityDimension(t *testing.T) {
	o := validTailnetObservation()
	base, err := newTailnetPrincipal(o, o.observedAt)
	if err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func(*tailnetObservation){func(o *tailnetObservation) { o.login = "operator" }, func(o *tailnetObservation) { o.userID++ }, func(o *tailnetObservation) { o.nodeID = "other" }, func(o *tailnetObservation) { o.peerIP = netip.MustParseAddr("100.64.0.2") }, func(o *tailnetObservation) { o.epoch++ }} {
		changed := o
		mutate(&changed)
		p, err := newTailnetPrincipal(changed, o.observedAt)
		if err != nil {
			t.Fatal(err)
		}
		if base.samePrincipal(p) {
			t.Fatal("identity change ignored")
		}
	}
	refreshed := o
	refreshed.expires = o.expires.Add(time.Hour)
	p, err := newTailnetPrincipal(refreshed, o.observedAt)
	if err != nil || !base.samePrincipal(p) {
		t.Fatal("refresh changed principal")
	}
	if base.samePrincipal(TailnetPrincipal{}) || (TailnetPrincipal{}).samePrincipal(TailnetPrincipal{}) {
		t.Fatal("empty identity compared equal")
	}
}

func TestTailnetLoginPolicyExactBoundedMembership(t *testing.T) {
	for _, size := range []int{1, 8, 16, 64} {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			logins := make([]string, size)
			for i := range logins {
				logins[i] = fmt.Sprintf("Operator%d", i)
			}
			policy, err := NewTailnetLoginPolicy(logins)
			if err != nil {
				t.Fatal(err)
			}
			logins[0] = "mutated"
			for i := 0; i < size; i++ {
				o := validTailnetObservation()
				o.login = fmt.Sprintf("Operator%d", i)
				p, err := newTailnetPrincipal(o, o.observedAt)
				if err != nil || !policy.Allows(p, o.observedAt) {
					t.Fatalf("member %d denied: %v", i, err)
				}
			}
			for _, login := range []string{"operator0", "Operator0.attacker", "Operator0@other", "mutated", "Operator64"} {
				o := validTailnetObservation()
				o.login = login
				p, _ := newTailnetPrincipal(o, o.observedAt)
				if policy.Allows(p, o.observedAt) {
					t.Fatalf("nonmember %q admitted", login)
				}
			}
		})
	}
	policy, _ := NewTailnetLoginPolicy([]string{"Operator"})
	o := validTailnetObservation()
	p, _ := newTailnetPrincipal(o, o.observedAt)
	if policy.Allows(p, o.expires) || policy.Allows(TailnetPrincipal{}, o.observedAt) || (TailnetLoginPolicy{}).Allows(p, o.observedAt) {
		t.Fatal("invalid principal/policy admitted")
	}
}

func TestTailnetLoginPolicyRejectsMalformedConfiguration(t *testing.T) {
	for _, logins := range [][]string{nil, {}, make([]string, 65), {""}, {" op"}, {"op "}, {"op\nadmin"}, {"op\x00"}, {"op]"}, {"[op"}, {"op<"}, {"op>"}, {"op\u0085"}, {"op\u00a0"}, {string([]byte{0xff})}, {strings.Repeat("a", 257)}} {
		if _, err := NewTailnetLoginPolicy(logins); err == nil {
			t.Fatalf("invalid config admitted: %q", logins)
		}
	}
	for _, login := range []string{"Operator", "operator@example.com", "identity/provider", "身份", strings.Repeat("a", 256)} {
		if _, err := NewTailnetLoginPolicy([]string{login, login}); err != nil {
			t.Fatalf("valid exact login %q rejected: %v", login, err)
		}
	}
}

func FuzzTailnetLoginPolicy(f *testing.F) {
	for _, s := range []string{"Operator", "op\r\n", "", "身份", "other]", "op\u202e", "op\u2066", "op\u200b"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, login string) {
		policy, err := NewTailnetLoginPolicy([]string{login})
		o := validTailnetObservation()
		o.login = login
		p, principalErr := newTailnetPrincipal(o, o.observedAt)
		if err != nil {
			if principalErr == nil {
				t.Fatal("constructor accepts login rejected by policy")
			}
			return
		}
		if principalErr != nil || !policy.Allows(p, o.observedAt) {
			t.Fatalf("configured valid login rejected: %v", principalErr)
		}
		o.login = login + "x"
		other, otherErr := newTailnetPrincipal(o, o.observedAt)
		if otherErr == nil && policy.Allows(other, o.observedAt) {
			t.Fatal("suffix login admitted")
		}
	})
}

func FuzzTailnetPrincipalPeerAddress(f *testing.F) {
	for _, ip := range []string{"100.64.0.1", "::ffff:100.64.0.1", "fd7a:115c:a1e0::1%en0", "8.8.8.8", ""} {
		f.Add(ip)
	}
	f.Fuzz(func(t *testing.T, input string) {
		o := validTailnetObservation()
		o.peerIP, _ = netip.ParseAddr(input)
		p, err := newTailnetPrincipal(o, o.observedAt)
		if err != nil {
			return
		}
		if !p.peerIP.IsValid() || p.peerIP.Zone() != "" || p.peerIP.Is4In6() {
			t.Fatal("noncanonical admitted address")
		}
		if p.peerIP != o.peerIP.Unmap() {
			t.Fatal("address substitution")
		}
		if !p.validAt(o.observedAt) {
			t.Fatal("constructor produced invalid principal")
		}
	})
}

func TestTailnetPrincipalExplicitNonExpiringObservation(t *testing.T) {
	o := validTailnetObservation()
	o.expires = time.Time{}
	o.nonExpiring = true
	p, err := newTailnetPrincipal(o, o.observedAt)
	if err != nil {
		t.Fatal(err)
	}
	if !p.validAt(o.observedAt.AddDate(10, 0, 0)) || p.validAt(o.observedAt.Add(-time.Nanosecond)) {
		t.Fatal("incorrect explicit non-expiring validity")
	}
}

func TestTailnetIdentityRejectsUnicodeFormatControls(t *testing.T) {
	for r := rune(0); r <= unicode.MaxRune; r++ {
		if !unicode.Is(unicode.Cf, r) {
			continue
		}
		login := "Operator" + string(r)
		if _, err := NewTailnetLoginPolicy([]string{login}); err == nil {
			t.Errorf("configuration accepted format control U+%04X", r)
		}
		o := validTailnetObservation()
		o.login = login
		if p, err := newTailnetPrincipal(o, o.observedAt); err == nil || p != (TailnetPrincipal{}) {
			t.Errorf("principal accepted format control U+%04X", r)
		}
	}
}
