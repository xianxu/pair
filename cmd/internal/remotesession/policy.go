package remotesession

import (
	"errors"
	"net/http"
	"net/mail"
	"net/netip"
	"net/url"
	"strings"
	"time"
)

// AccessIdentity is produced only by the signature verifier. Private fields
// prevent HTTP callers and document adapters from minting verified identities.
type AccessIdentity struct {
	issuer   string
	subject  string
	audience string
	email    string
	expires  time.Time
}

func (i AccessIdentity) validAt(now time.Time) bool {
	if i.issuer == "" || i.subject == "" || i.audience == "" || !now.Before(i.expires) || len(i.email) > 254 || strings.ContainsAny(i.email, "\r\n\t []<>") {
		return false
	}
	addr, err := mail.ParseAddress(i.email)
	return err == nil && addr.Address == i.email
}
func (i AccessIdentity) samePrincipal(other AccessIdentity) bool {
	return i.issuer == other.issuer && i.subject == other.subject && i.audience == other.audience && i.email == other.email
}

type remotePolicy struct {
	origin string
	host   string
}

func newRemotePolicy(origin string) (remotePolicy, error) {
	u, err := url.Parse(origin)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" || u.Opaque != "" || u.Host != strings.ToLower(u.Host) || u.Port() != "" || strings.HasSuffix(u.Host, ".") {
		return remotePolicy{}, errors.New("remote requires a canonical HTTPS origin")
	}
	if _, err := netip.ParseAddr(u.Host); err == nil {
		return remotePolicy{}, errors.New("remote requires a named hostname")
	}
	for _, label := range strings.Split(u.Host, ".") {
		if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return remotePolicy{}, errors.New("invalid remote hostname")
		}
		for _, c := range label {
			if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-') {
				return remotePolicy{}, errors.New("invalid remote hostname")
			}
		}
	}
	if len(u.Host) > 253 || !strings.Contains(u.Host, ".") {
		return remotePolicy{}, errors.New("invalid remote hostname")
	}
	return remotePolicy{origin: origin, host: u.Host}, nil
}

// requestIP is meaningful only behind the protected named connector's private
// Unix origin. JWT does not sign this header. Do not substitute XFF/RemoteAddr.
func (p remotePolicy) requestIP(r *http.Request) (string, error) {
	origins := r.Header.Values("Origin")
	read := (r.Method == http.MethodGet || r.Method == http.MethodHead) && r.Header.Get("Upgrade") == ""
	validOrigin := len(origins) == 1 && origins[0] == p.origin
	if r.Host != p.host || (!validOrigin && !(read && len(origins) == 0)) {
		return "", errors.New("remote origin refused")
	}
	for _, name := range []string{"Cf-Worker", "Cf-Ew-Via"} {
		if _, present := r.Header[http.CanonicalHeaderKey(name)]; present {
			return "", errors.New("remote Worker ingress refused")
		}
	}
	values := r.Header.Values("Cf-Connecting-IP")
	if len(values) != 1 || strings.TrimSpace(values[0]) != values[0] {
		return "", errors.New("remote client IP missing or ambiguous")
	}
	ip, err := netip.ParseAddr(values[0])
	if err != nil || ip.Zone() != "" || ip.IsUnspecified() {
		return "", errors.New("remote client IP invalid")
	}
	return ip.Unmap().String(), nil
}
