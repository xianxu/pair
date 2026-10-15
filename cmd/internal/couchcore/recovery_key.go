package couchcore

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/xianxu/pair/cmd/internal/strictjson"
)

// RecoveryCandidateSelector is an untrusted reference to evidence, not launch
// authority. An empty NativeID selects the explicit fresh-from-screen action.
type RecoveryCandidateSelector struct {
	Version  int           `json:"version"`
	Address  ThreadAddress `json:"address"`
	Agent    string        `json:"agent"`
	NativeID string        `json:"native_id"`
	Digest   string        `json:"digest"`
}

const maxRecoveryKeyBytes = 2048

func validateRecoverySelector(s RecoveryCandidateSelector) error {
	if s.Version != 1 || s.Address.RepoScope == "" || s.Address.Tag == "" || len(s.Address.RepoScope) > 256 || len(s.Address.Tag) > 256 {
		return errors.New("invalid recovery selector address or version")
	}
	for _, s := range []string{s.Address.RepoScope, string(s.Address.Tag), s.Agent, s.NativeID} {
		if len(s) > 256 || !utf8.ValidString(s) || strings.ContainsAny(s, "\x00\r\n") {
			return errors.New("invalid recovery selector value")
		}
	}
	if s.Agent != "codex" && s.Agent != "claude" && s.Agent != "grok" {
		return errors.New("unsupported recovery selector agent")
	}
	b, err := hex.DecodeString(s.Digest)
	if err != nil || len(b) != 32 || strings.ToLower(s.Digest) != s.Digest {
		return errors.New("invalid recovery evidence digest")
	}
	return nil
}

// EncodeRecoveryCandidateKey never encodes a display rank: rescoring cannot
// reinterpret a previously displayed option number as a different conversation.
func EncodeRecoveryCandidateKey(s RecoveryCandidateSelector) (string, error) {
	if err := validateRecoverySelector(s); err != nil {
		return "", err
	}
	b, err := json.Marshal(s)
	if err != nil {
		return "", err
	}
	key := base64.RawURLEncoding.EncodeToString(b)
	if len(key) > maxRecoveryKeyBytes {
		return "", errors.New("recovery selector exceeds bound")
	}
	return key, nil
}

func decodeRecoveryCandidateKey(key string) (RecoveryCandidateSelector, error) {
	var s RecoveryCandidateSelector
	if len(key) == 0 || len(key) > maxRecoveryKeyBytes {
		return s, errors.New("invalid recovery selector size")
	}
	b, err := base64.RawURLEncoding.Strict().DecodeString(key)
	if err != nil {
		return s, err
	}
	if err := strictjson.Decode(b, &s); err != nil {
		return s, err
	}
	if err := validateRecoverySelector(s); err != nil {
		return s, err
	}
	return s, nil
}

// ValidateRecoveryCandidateKey compares with freshly collected exact evidence.
// A copied key is never sufficient permission to bypass lifecycle admission.
func ValidateRecoveryCandidateKey(key string, expected RecoveryCandidateSelector) (RecoveryCandidateSelector, error) {
	s, err := decodeRecoveryCandidateKey(key)
	if err != nil {
		return s, err
	}
	if err := validateRecoverySelector(expected); err != nil {
		return s, err
	}
	if s != expected {
		return s, errors.New("recovery evidence changed; refresh resume candidates")
	}
	return s, nil
}
