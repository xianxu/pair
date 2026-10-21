package remotesession

import (
	"net/http"
	"testing"
	"time"
)

func TestRemotePolicyOriginAndClientIP(t *testing.T) {
	for _, tc := range []struct {
		name, origin, host, ip  string
		duplicate, worker, want bool
	}{
		{name: "trusted IPv4", origin: "https://remote.example.com", host: "remote.example.com", ip: "192.0.2.1", want: true},
		{name: "trusted IPv6", origin: "https://remote.example.com", host: "remote.example.com", ip: "2001:db8::1", want: true},
		{name: "foreign origin", origin: "https://evil.example", host: "remote.example.com", ip: "192.0.2.1"},
		{name: "missing origin", host: "remote.example.com", ip: "192.0.2.1"},
		{name: "foreign host", origin: "https://remote.example.com", host: "evil.example", ip: "192.0.2.1"},
		{name: "IP list", origin: "https://remote.example.com", host: "remote.example.com", ip: "192.0.2.1, 192.0.2.2"},
		{name: "duplicate IP", origin: "https://remote.example.com", host: "remote.example.com", ip: "192.0.2.1", duplicate: true},
		{name: "worker", origin: "https://remote.example.com", host: "remote.example.com", ip: "192.0.2.1", worker: true},
		{name: "missing IP", origin: "https://remote.example.com", host: "remote.example.com"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r, _ := http.NewRequest(http.MethodPost, "https://remote.example.com/remote/handshake", nil)
			r.Host = tc.host
			r.Header.Set("Origin", tc.origin)
			r.Header.Set("Cf-Connecting-IP", tc.ip)
			r.Header.Set("X-Forwarded-For", "198.51.100.2")
			if tc.duplicate {
				r.Header.Add("Cf-Connecting-IP", tc.ip)
			}
			if tc.worker {
				r.Header.Set("Cf-Worker", "worker.example")
			}
			p, err := newRemotePolicy("https://remote.example.com")
			if err != nil {
				t.Fatal(err)
			}
			ip, err := p.requestIP(r)
			if (err == nil) != tc.want {
				t.Fatalf("ip=%q error=%v want=%v", ip, err, tc.want)
			}
		})
	}
}
func TestRemotePolicyIdentityExpiry(t *testing.T) {
	now := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
	id := AccessIdentity{issuer: "https://team.cloudflareaccess.com", subject: "sub", audience: "aud", email: "user@example.com", expires: now.Add(time.Hour)}
	if !id.validAt(now) || id.validAt(id.expires) || id.validAt(id.expires.Add(time.Second)) {
		t.Fatal("Access expiry is exact upper boundary")
	}
	id.email = "user@example.com\nforged"
	if id.validAt(now) {
		t.Fatal("unsafe identity admitted into remote prefix")
	}
}
func TestRemotePolicyRequiresHTTPSOrigin(t *testing.T) {
	for _, value := range []string{"http://remote.example.com", "https://remote.example.com/extra", "https://user@remote.example.com", "https://remote.example.com?token=x", "https://remote.example.com/", "https://REMOTE.example.com"} {
		if _, err := newRemotePolicy(value); err == nil {
			t.Fatalf("unsafe/canonicalization-ambiguous origin %q", value)
		}
	}
}
