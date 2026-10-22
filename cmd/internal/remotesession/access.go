package remotesession

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
)

// AccessConfig pins one Cloudflare team and application. AllowedEmails is an
// exact identity allowlist, never a domain or suffix match.
type AccessConfig struct {
	Issuer        string
	Audience      string
	AllowedEmails []string
}

// AccessVerifier owns its bounded team key cache. It performs no discovery and
// never uses key URLs or identity headers supplied by the remote caller.
type AccessVerifier struct {
	verifier         *oidc.IDTokenVerifier
	issuer, audience string
	allowed          map[string]struct{}
	now              func() time.Time
}

var errAccessRefused = errors.New("Access assertion refused")

func NewAccessVerifier(c AccessConfig) (*AccessVerifier, error) {
	return newAccessVerifier(c, http.DefaultTransport, time.Now)
}

// The transport seam preserves the configured production URL even in fixtures.
func newAccessVerifier(c AccessConfig, transport http.RoundTripper, now func() time.Time) (*AccessVerifier, error) {
	u, err := url.Parse(c.Issuer)
	if err != nil || u.Scheme != "https" || u.User != nil || u.Host != u.Hostname() || u.Path != "" || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || u.Opaque != "" {
		return nil, errors.New("invalid Access team issuer")
	}
	const suffix = ".cloudflareaccess.com"
	team := strings.TrimSuffix(u.Host, suffix)
	if !strings.HasSuffix(u.Host, suffix) || len(team) < 1 || len(team) > 63 || strings.Contains(team, ".") || team[0] == '-' || team[len(team)-1] == '-' {
		return nil, errors.New("invalid Access team issuer")
	}
	for _, r := range team {
		if !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-') {
			return nil, errors.New("invalid Access team issuer")
		}
	}
	if c.Audience == "" || len(c.Audience) > 256 || strings.TrimSpace(c.Audience) != c.Audience || len(c.AllowedEmails) == 0 || len(c.AllowedEmails) > 64 {
		return nil, errors.New("invalid Access application or identities")
	}
	if now == nil || transport == nil {
		return nil, errors.New("invalid Access verifier dependencies")
	}
	allowed := make(map[string]struct{}, len(c.AllowedEmails))
	for _, email := range c.AllowedEmails {
		id := AccessIdentity{issuer: c.Issuer, subject: "config", audience: c.Audience, email: email, expires: now().Add(time.Hour)}
		if !id.validAt(now()) {
			return nil, errors.New("invalid Access allowed identity")
		}
		allowed[email] = struct{}{}
	}
	keys := &accessKeys{url: c.Issuer + "/cdn-cgi/access/certs", client: &http.Client{Transport: transport, Timeout: 5 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return errors.New("Access key redirect refused") }}, now: now}
	return &AccessVerifier{verifier: oidc.NewVerifier(c.Issuer, keys, &oidc.Config{ClientID: c.Audience, SupportedSigningAlgs: []string{"RS256"}, Now: now}), issuer: c.Issuer, audience: c.Audience, allowed: allowed, now: now}, nil
}

func (v *AccessVerifier) Verify(ctx context.Context, assertion string) (AccessIdentity, error) {
	if ctx.Err() != nil || len(assertion) == 0 || len(assertion) > 8*1024 {
		return AccessIdentity{}, errAccessRefused
	}
	token, err := v.verifier.Verify(ctx, assertion)
	if err != nil {
		return AccessIdentity{}, errAccessRefused
	}
	var claims struct {
		Email          string `json:"email"`
		Type           string `json:"type"`
		NotBefore      *int64 `json:"nbf"`
		CommonName     string `json:"common_name"`
		ServiceTokenID string `json:"service_token_id"`
	}
	if token.Claims(&claims) != nil || claims.Type != "app" || claims.CommonName != "" || claims.ServiceTokenID != "" || len(token.Subject) > 256 || strings.TrimSpace(token.Subject) != token.Subject || strings.ContainsAny(token.Subject, "\r\n\t") {
		return AccessIdentity{}, errAccessRefused
	}
	now := v.now()
	if claims.NotBefore != nil && time.Unix(*claims.NotBefore, 0).After(now) {
		return AccessIdentity{}, errAccessRefused
	}
	id := AccessIdentity{issuer: v.issuer, subject: token.Subject, audience: v.audience, email: claims.Email, expires: token.Expiry}
	if _, ok := v.allowed[id.email]; !ok || !id.validAt(now) || ctx.Err() != nil {
		return AccessIdentity{}, errAccessRefused
	}
	return id, nil
}
