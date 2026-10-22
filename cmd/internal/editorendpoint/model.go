// Package editorendpoint owns the pure, bounded endpoint authority model.
package editorendpoint

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
)

const MaxStateBytes = 8 << 20
const MaxRecords = 4096
const MaxProcessActive = 256

type EndpointBinding struct {
	Namespace         string                     `json:"Namespace"`
	DocumentID        string                     `json:"DocumentID"`
	EditorIncarnation string                     `json:"EditorIncarnation"`
	AttachID          string                     `json:"AttachID"`
	PID               int                        `json:"PID"`
	ProcessIdentity   string                     `json:"ProcessIdentity"`
	Socket            string                     `json:"Socket"`
	Buffer            int                        `json:"Buffer"`
	Mode              string                     `json:"Mode"`
	BindingGeneration string                     `json:"BindingGeneration"`
	File              string                     `json:"File"`
	Unknown           map[string]json.RawMessage `json:"-"`
}
type RegisterRequest struct {
	Version           int    `json:"Version"`
	Namespace         string `json:"Namespace"`
	DocumentID        string `json:"DocumentID"`
	EditorIncarnation string `json:"EditorIncarnation"`
	AttachID          string `json:"AttachID"`
	PID               int    `json:"PID"`
	Socket            string `json:"Socket"`
	Buffer            int    `json:"Buffer"`
	Mode              string `json:"Mode"`
}
type UnregisterEntry struct {
	DocumentID        string `json:"DocumentID"`
	AttachID          string `json:"AttachID"`
	BindingGeneration string `json:"BindingGeneration"`
}
type UnregisterRequest struct {
	Version           int               `json:"Version"`
	Namespace         string            `json:"Namespace"`
	EditorIncarnation string            `json:"EditorIncarnation"`
	PID               int               `json:"PID"`
	ProcessIdentity   string            `json:"ProcessIdentity"`
	Entries           []UnregisterEntry `json:"Entries"`
}
type UnregisterResult struct {
	DocumentID        string `json:"DocumentID"`
	AttachID          string `json:"AttachID"`
	BindingGeneration string `json:"BindingGeneration"`
	Status            string `json:"Status"`
	Changed           bool   `json:"-"`
}
type EndpointReply struct {
	Version        int                `json:"Version"`
	Code           string             `json:"Code"`
	Error          string             `json:"Error"`
	Changed        bool               `json:"Changed"`
	EditorEndpoint *EndpointBinding   `json:"EditorEndpoint"`
	Results        []UnregisterResult `json:"Results"`
}
type EndpointRecord struct {
	// Current retains an older desktop while a newer refused headless attach
	// occupies this pair's terminal high-water. It is never a fallback candidate.
	Current *EndpointBinding           `json:"Current"`
	Binding EndpointBinding            `json:"Binding"`
	Status  string                     `json:"Status"`
	Unknown map[string]json.RawMessage `json:"-"`
}
type EndpointState struct {
	Version   int                        `json:"Version"`
	Namespace string                     `json:"Namespace"`
	Revision  uint64                     `json:"Revision"`
	Records   []EndpointRecord           `json:"Records"`
	Unknown   map[string]json.RawMessage `json:"-"`
}
type ProcessEvidence struct {
	EditorIncarnation string
	PID               int
	ProcessIdentity   string
}

// EndpointEvent receives entropy and native identity evidence from its caller.
// Collect means positively dead, exact PID+birth evidence; Retire is generation fenced.
type EndpointEvent struct {
	Register          *RegisterRequest
	Unregister        *UnregisterRequest
	Retire            *EndpointBinding
	Collect           *ProcessEvidence
	ProcessIdentity   string
	BindingGeneration string
	File              string
}

func validHex(s string) bool {
	if len(s) != 32 {
		return false
	}
	for _, c := range s {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}
func canonicalPath(s string) bool {
	return len(s) > 0 && len(s) <= 4096 && filepath.IsAbs(s) && filepath.Clean(s) == s && !strings.ContainsRune(s, 0)
}
func attach(s string) (uint64, error) {
	n, e := strconv.ParseUint(s, 10, 64)
	if e != nil || n == 0 || strconv.FormatUint(n, 10) != s {
		return 0, errors.New("attach_id must be canonical positive uint64 decimal string")
	}
	return n, nil
}
func base(namespace, inc string, pid int) error {
	if !canonicalPath(namespace) || !validHex(inc) || pid <= 0 {
		return errors.New("invalid namespace, editor_incarnation, or pid")
	}
	return nil
}
func ValidateEndpointRequest(request any) error {
	switch r := request.(type) {
	case *RegisterRequest:
		if r == nil {
			return errors.New("nil request")
		}
		return ValidateEndpointRequest(*r)
	case RegisterRequest:
		if r.Version != 1 {
			return errors.New("unsupported version")
		}
		if e := base(r.Namespace, r.EditorIncarnation, r.PID); e != nil {
			return e
		}
		if !validHex(r.DocumentID) || !canonicalPath(r.Socket) || r.Buffer <= 0 || (r.Mode != "desktop" && r.Mode != "headless") {
			return errors.New("invalid register payload")
		}
		_, e := attach(r.AttachID)
		return e
	case *UnregisterRequest:
		if r == nil {
			return errors.New("nil request")
		}
		return ValidateEndpointRequest(*r)
	case UnregisterRequest:
		if r.Version != 1 {
			return errors.New("unsupported version")
		}
		if e := base(r.Namespace, r.EditorIncarnation, r.PID); e != nil {
			return e
		}
		if len(r.Entries) == 0 || len(r.Entries) > 256 || len(r.ProcessIdentity) > 4096 {
			return errors.New("invalid unregister bounds")
		}
		seen := map[string]bool{}
		for _, x := range r.Entries {
			if seen[x.DocumentID] || !validHex(x.DocumentID) || !validHex(x.BindingGeneration) {
				return errors.New("invalid or duplicate unregister entry")
			}
			seen[x.DocumentID] = true
			if _, e := attach(x.AttachID); e != nil {
				return e
			}
		}
		return nil
	default:
		return fmt.Errorf("unsupported request %T", request)
	}
}
