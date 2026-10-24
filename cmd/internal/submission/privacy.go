package submission

import (
	"bytes"
	"encoding/json"
	"errors"
	"net"
	"net/url"
	"strconv"
	"strings"

	"github.com/xianxu/pair/internal/jsonstrict"
)

const MaxOriginPolicyBytes = 64 << 10
const MaxOriginRules = 256

type Privacy string

const (
	Private   Privacy = "private"
	Shareable Privacy = "shareable"
)

type OriginRule struct {
	Origin  string  `json:"origin"`
	Privacy Privacy `json:"privacy"`
}

// PolicyEvidence is either an exact matched rule, an invalid-file explanation,
// or null (the default). Only the operator's exact rule can permit sharing.
type PolicyEvidence struct {
	Rule    *OriginRule
	Invalid string
}

func (e PolicyEvidence) MarshalJSON() ([]byte, error) {
	if e.Invalid != "" {
		return json.Marshal("invalid policy file: " + e.Invalid)
	}
	return json.Marshal(e.Rule)
}
func (e *PolicyEvidence) UnmarshalJSON(b []byte) error {
	*e = PolicyEvidence{}
	if bytes.Equal(bytes.TrimSpace(b), []byte("null")) {
		return nil
	}
	if len(b) > MaxOriginPolicyBytes {
		return errors.New("privacy rule too large")
	}
	var reason string
	if json.Unmarshal(b, &reason) == nil {
		const prefix = "invalid policy file: "
		if !strings.HasPrefix(reason, prefix) {
			return errors.New("invalid privacy reason")
		}
		e.Invalid = strings.TrimPrefix(reason, prefix)
		if !safePolicyReason(e.Invalid) {
			return errors.New("invalid privacy reason")
		}
		return nil
	}
	var r OriginRule
	if err := strictJSON(b, &r); err != nil {
		return errors.New("invalid privacy rule")
	}
	e.Rule = &r
	return nil
}

type PrivacyDecision struct {
	Privacy       Privacy        `json:"privacy,omitempty"`
	PrivacyRule   PolicyEvidence `json:"privacy_rule"`
	PrivacyOrigin string         `json:"privacy_origin,omitempty"`
}

func (d PrivacyDecision) Validate() error {
	if d.Privacy == "" && d.PrivacyOrigin == "" && d.PrivacyRule.Rule == nil && d.PrivacyRule.Invalid == "" {
		return nil
	} // older bundles are private
	if d.Privacy != Private && d.Privacy != Shareable {
		return errors.New("invalid submission privacy")
	}
	if d.PrivacyOrigin != "" && OriginOf(d.PrivacyOrigin) != d.PrivacyOrigin {
		return errors.New("invalid privacy origin")
	}
	e := d.PrivacyRule
	if e.Invalid != "" {
		if d.Privacy != Private || e.Rule != nil || !safePolicyReason(e.Invalid) {
			return errors.New("invalid policy must be private")
		}
		return nil
	}
	if e.Rule == nil {
		if d.Privacy == Shareable {
			return errors.New("shareable requires an origin rule")
		}
		return nil
	}
	if e.Rule.Origin == "" || e.Rule.Origin != d.PrivacyOrigin || e.Rule.Privacy != d.Privacy {
		return errors.New("privacy rule does not match capture")
	}
	return nil
}

// OriginPolicy is immutable after parsing; callers cannot add grants directly.
type OriginPolicy struct {
	rules   []OriginRule
	invalid string
}

func InvalidOriginPolicy(reason string) OriginPolicy {
	if !safePolicyReason(reason) {
		reason = "invalid policy"
	}
	return OriginPolicy{invalid: reason}
}
func safePolicyReason(reason string) bool {
	switch reason {
	case "invalid JSON", "invalid origin rule", "duplicate origin", "too many rules", "file exceeds 64 KiB", "cannot read file", "not a regular file", "invalid policy":
		return true
	}
	return false
}
func ParseOriginPolicy(data []byte) (OriginPolicy, error) {
	if len(data) > MaxOriginPolicyBytes {
		return OriginPolicy{}, errors.New("file exceeds 64 KiB")
	}
	var wire struct {
		Origins []OriginRule `json:"origins"`
	}
	if strictJSON(data, &wire) != nil || wire.Origins == nil {
		return OriginPolicy{}, errors.New("invalid JSON")
	}
	if len(wire.Origins) > MaxOriginRules {
		return OriginPolicy{}, errors.New("too many rules")
	}
	seen := map[string]bool{}
	for i, r := range wire.Origins {
		u, err := url.Parse(r.Origin)
		origin := OriginOf(r.Origin)
		if err != nil || origin == "" || (u.Path != "" && u.Path != "/") || u.RawQuery != "" || u.ForceQuery || strings.Contains(r.Origin, "#") || (r.Privacy != Private && r.Privacy != Shareable) {
			return OriginPolicy{}, errors.New("invalid origin rule")
		}
		if seen[origin] {
			return OriginPolicy{}, errors.New("duplicate origin")
		}
		seen[origin] = true
		wire.Origins[i].Origin = origin
	}
	return OriginPolicy{rules: wire.Origins}, nil
}
func (p OriginPolicy) Decide(pageURL string) PrivacyDecision {
	d := PrivacyDecision{Privacy: Private, PrivacyOrigin: OriginOf(pageURL)}
	if p.invalid != "" {
		d.PrivacyRule.Invalid = p.invalid
		return d
	}
	if d.PrivacyOrigin == "" {
		return d
	}
	for _, r := range p.rules {
		if r.Origin == d.PrivacyOrigin {
			d.Privacy = r.Privacy
			d.PrivacyRule.Rule = &r
			break
		}
	}
	return d
}

// OriginOf deliberately grants nothing to opaque or credentialed URLs.
// Paths and queries never participate in a page's origin identity.
func OriginOf(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.User != nil || u.Opaque != "" {
		return ""
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return ""
	}
	host := strings.ToLower(u.Hostname())
	if host == "" || strings.ContainsAny(host, "*%\\ \t\r\n") || strings.HasSuffix(u.Host, ":") {
		return ""
	}
	if ip := net.ParseIP(host); ip != nil {
		host = ip.String()
	} else {
		for _, r := range host {
			if !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '.' || r == '-' || r == '_') {
				return ""
			}
		}
	}
	port := u.Port()
	if port != "" {
		n, e := strconv.Atoi(port)
		if e != nil || n < 1 || n > 65535 {
			return ""
		}
		port = strconv.Itoa(n)
		if scheme == "http" && port == "80" || scheme == "https" && port == "443" {
			port = ""
		}
	}
	if strings.Contains(host, ":") {
		host = "[" + host + "]"
	}
	if port != "" {
		host += ":" + port
	}
	return scheme + "://" + host
}

func checkJSONKeys(data []byte, check func(depth int, key string) error) error {
	return jsonstrict.Scan(data, jsonstrict.Options{PreserveNumbers: true, VisitKey: check})
}

func strictJSON(data []byte, out any) error {
	if err := checkJSONKeys(data, func(_ int, key string) error {
		switch key {
		case "origins", "origin", "privacy":
			return nil
		}
		return errors.New("unknown privacy JSON key")
	}); err != nil {
		return err
	}
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	return d.Decode(out)
}

// UnmarshalJSON rejects ambiguous manifest permissions before struct decoding
// can discard duplicates or accept case-folded field aliases. Keep this on the
// complete manifest, not its embedded decision, so other fields still decode.
func (s *Submission) UnmarshalJSON(data []byte) error {
	decoded, err := DecodePreserving(data)
	if err != nil {
		return err
	}
	*s = decoded
	return nil
}
