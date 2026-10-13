package artifactpath

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	SessionLaunchPolicyVersion  = 1
	NonSerializingPolicyVersion = 1
	MaxSessionLaunchPolicyBytes = 8192
)

// SessionLaunchPolicy attests the policy of one exact terminal generation.
// It does not prove that generation is still present; the launcher reobserves it.
// pair:m5-concept pure
type SessionLaunchPolicy struct {
	Version        int    `json:"version"`
	Socket         string `json:"socket"`
	SocketDevice   uint64 `json:"socket_device"`
	SocketInode    uint64 `json:"socket_inode"`
	Name           string `json:"name"`
	ServerPID      int    `json:"server_pid"`
	ServerIdentity string `json:"server_identity"`
	Scope          string `json:"scope"`
	Tag            string `json:"tag"`
	Agent          string `json:"agent"`
	Nonce          string `json:"nonce"`
	ConfigDigest   string `json:"config_digest"`
	PolicyVersion  int    `json:"policy_version"`
}

func ValidateSessionLaunchPolicy(p SessionLaunchPolicy) error {
	if p.Version != SessionLaunchPolicyVersion || p.PolicyVersion != NonSerializingPolicyVersion {
		return errors.New("unsupported session launch policy version")
	}
	if err := validatePolicySocket(p.Socket); err != nil {
		return err
	}
	if p.Name != filepath.Base(p.Socket) {
		return errors.New("session policy name does not match socket")
	}
	if p.ServerPID <= 0 || p.SocketInode == 0 {
		return errors.New("session policy requires exact process and socket identities")
	}
	for name, value := range map[string]string{"server identity": p.ServerIdentity, "session name": p.Name, "agent": p.Agent, "nonce": p.Nonce} {
		if !policyText(value, 256) {
			return fmt.Errorf("invalid session policy %s", name)
		}
	}
	if err := validateComponent("policy agent", p.Agent); err != nil {
		return err
	}
	if p.Scope != "" {
		if err := validateComponent("policy scope", p.Scope); err != nil {
			return err
		}
	}
	if err := validateComponent("policy tag", p.Tag); err != nil {
		return err
	}
	if !policyText(p.Tag, 256) || !policyText(p.Scope, 256) && p.Scope != "" {
		return errors.New("session policy owner exceeds limits")
	}
	if len(p.ConfigDigest) != 64 {
		return errors.New("session policy config digest must be SHA-256")
	}
	if _, err := hex.DecodeString(p.ConfigDigest); err != nil || strings.ToLower(p.ConfigDigest) != p.ConfigDigest {
		return errors.New("invalid session policy config digest")
	}
	return nil
}

func policyText(value string, max int) bool {
	return value != "" && len(value) <= max && utf8.ValidString(value) && strings.IndexFunc(value, unicode.IsControl) < 0
}
func validatePolicySocket(socket string) error {
	if !policyText(socket, 4096) || !filepath.IsAbs(socket) || filepath.Clean(socket) != socket || socket == string(filepath.Separator) {
		return errors.New("session socket must be a canonical absolute path")
	}
	return nil
}

// DecodeSessionLaunchPolicy rejects extensions, duplicate keys and trailing JSON.
func DecodeSessionLaunchPolicy(raw []byte) (SessionLaunchPolicy, error) {
	var p SessionLaunchPolicy
	if !utf8.Valid(raw) {
		return p, errors.New("session launch policy contains invalid UTF-8")
	}
	if len(raw) == 0 || len(raw) > MaxSessionLaunchPolicyBytes {
		return p, errors.New("session launch policy exceeds size limit or is empty")
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	token, err := d.Token()
	if err != nil || token != json.Delim('{') {
		return p, errors.New("session launch policy must be an object")
	}
	seen := map[string]bool{}
	for d.More() {
		key, err := d.Token()
		if err != nil {
			return p, err
		}
		name, ok := key.(string)
		if !ok || seen[name] {
			return p, errors.New("duplicate or invalid session policy field")
		}
		switch name {
		case "version", "socket", "socket_device", "socket_inode", "name", "server_pid", "server_identity", "scope", "tag", "agent", "nonce", "config_digest", "policy_version":
		default:
			return p, fmt.Errorf("unknown session policy field %q", name)
		}
		seen[name] = true
		var value json.RawMessage
		if err := d.Decode(&value); err != nil {
			return p, err
		}
		if bytes.Equal(value, []byte("null")) {
			return p, fmt.Errorf("null session policy field %q", name)
		}
	}
	if _, err = d.Token(); err != nil {
		return p, err
	}
	if _, err = d.Token(); err != io.EOF {
		return p, errors.New("trailing session policy data")
	}
	if len(seen) != 13 {
		return p, errors.New("session policy fields are missing")
	}
	if err = json.Unmarshal(raw, &p); err != nil {
		return SessionLaunchPolicy{}, err
	}
	if err = ValidateSessionLaunchPolicy(p); err != nil {
		return SessionLaunchPolicy{}, err
	}
	return p, nil
}

// The OS caller canonicalizes socket directory symlinks before using these pure
// constructors. Both artifacts share the same key, independent of scope aliases.
func (p PairCachePaths) TerminalLease(canonicalSocket string) (string, error) {
	return p.terminalArtifact(canonicalSocket, "session-locks", ".lock")
}
func (p PairCachePaths) SessionPolicy(canonicalSocket string) (string, error) {
	return p.terminalArtifact(canonicalSocket, "session-policy", ".json")
}
func (p PairCachePaths) terminalArtifact(socket, dir, ext string) (string, error) {
	if p.dir == "" {
		return "", errors.New("unresolved Pair cache")
	}
	if err := validatePolicySocket(socket); err != nil {
		return "", err
	}
	key := sha256.Sum256([]byte(socket))
	return filepath.Join(p.dir, dir, hex.EncodeToString(key[:])+ext), nil
}
