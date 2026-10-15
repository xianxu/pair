package sessionsync

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"
)

const (
	registryMaxBytes     = 16 << 20
	registryMaxDocuments = 4096
	registryMaxPaths     = 256
	documentMaxBytes     = 4 << 20
)

type PathInterval struct {
	File     string
	Spelling string `json:",omitempty"`
	From     time.Time
	Until    *time.Time `json:",omitempty"`
}
type DocumentRegistration struct {
	ID              string
	File            string
	RegisteredOwner string
	RegisteredAt    time.Time
	Paths           []PathInterval
}
type RegistryState struct {
	Version   int
	Documents []DocumentRegistration
}
type RegistryEvent struct {
	Kind         string
	File         string
	Old          string
	Owner        string
	ID           string
	Spelling     string
	At           time.Time
	InitialPaths []PathInterval
}
type RegistryChange struct {
	Registration *DocumentRegistration
	Changed      bool
}

func registryRefusal(code, reason string) error { return &Refusal{Exit: 1, Code: code, Reason: reason} }
func registryInvalid(reason string) error       { return registryRefusal("registry-invalid", reason) }
func registryLimit() error {
	return registryRefusal("registry-limit", "session-sync registry limit reached; explicitly remove an unused registration with couch --session-sync forget FILE")
}
func registryPath(path string) bool {
	return filepath.IsAbs(path) && filepath.Clean(path) == path && !strings.ContainsAny(path, "\n\x00")
}
func registryID(id string) bool {
	if len(id) != 32 {
		return false
	}
	for _, c := range id {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}

// Imported path history can precede RegisteredAt. Its own ordered intervals,
// rather than registration time, are the authority for attribution. A zero
// first From represents legacy history with no recorded lower bound.
func (r RegistryState) validate() error {
	if r.Version != 1 {
		return registryInvalid("unsupported registry version")
	}
	if len(r.Documents) > registryMaxDocuments {
		return registryInvalid("too many registry documents")
	}
	ids, files := map[string]bool{}, map[string]bool{}
	for _, d := range r.Documents {
		if !registryID(d.ID) || ids[d.ID] || !registryPath(d.File) || files[d.File] || !ValidSlot(d.RegisteredOwner) || d.RegisteredAt.IsZero() {
			return registryInvalid("invalid or duplicate document registration")
		}
		ids[d.ID], files[d.File] = true, true
		if len(d.Paths) == 0 || len(d.Paths) > registryMaxPaths {
			return registryInvalid("invalid path history size")
		}
		for i, p := range d.Paths {
			if !registryPath(p.File) || (p.Spelling != "" && !registryPath(p.Spelling)) || (i > 0 && p.From.IsZero()) {
				return registryInvalid("invalid path interval")
			}
			if i > 0 && (d.Paths[i-1].Until == nil || p.From.Before(*d.Paths[i-1].Until)) {
				return registryInvalid("overlapping path intervals")
			}
			if i == len(d.Paths)-1 {
				if p.Until != nil || p.File != d.File {
					return registryInvalid("invalid current path interval")
				}
			} else if p.Until == nil || p.Until.Before(p.From) {
				return registryInvalid("invalid closed path interval")
			}
		}
	}
	return nil
}

func decodeRegistry(raw []byte) (RegistryState, error) {
	if len(raw) > registryMaxBytes {
		return RegistryState{}, registryInvalid("registry exceeds byte limit")
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	var r RegistryState
	if err := dec.Decode(&r); err != nil {
		return RegistryState{}, registryInvalid(fmt.Sprintf("invalid registry JSON: %v", err))
	}
	var trailing any
	if err := dec.Decode(&trailing); err != io.EOF {
		return RegistryState{}, registryInvalid("trailing registry data")
	}
	if err := r.validate(); err != nil {
		return RegistryState{}, err
	}
	return r, nil
}

func cloneRegistryPaths(paths []PathInterval) []PathInterval {
	if paths == nil {
		return nil
	}
	out := append([]PathInterval{}, paths...)
	for i := range out {
		if out[i].Until != nil {
			v := *out[i].Until
			out[i].Until = &v
		}
	}
	return out
}
func (r RegistryState) clone() RegistryState {
	out := RegistryState{Version: r.Version}
	if r.Documents != nil {
		out.Documents = append([]DocumentRegistration{}, r.Documents...)
	}
	for i := range out.Documents {
		out.Documents[i].Paths = cloneRegistryPaths(out.Documents[i].Paths)
	}
	return out
}
func registryChange(d DocumentRegistration, changed bool) RegistryChange {
	d.Paths = cloneRegistryPaths(d.Paths)
	return RegistryChange{Registration: &d, Changed: changed}
}

// transition is a pure transaction: neither a returned state nor a change
// shares mutable history with its input, including on idempotent retries.
func (r RegistryState) transition(e RegistryEvent) (RegistryState, RegistryChange, error) {
	fail := func(err error) (RegistryState, RegistryChange, error) { return r, RegistryChange{}, err }
	if err := r.validate(); err != nil {
		return fail(err)
	}
	if !registryPath(e.File) || (e.Spelling != "" && !registryPath(e.Spelling)) {
		return fail(registryInvalid("invalid event path"))
	}
	out := r.clone()
	at := -1
	for i, d := range out.Documents {
		if d.File == e.File {
			at = i
			break
		}
	}
	var change RegistryChange
	switch e.Kind {
	case "register":
		if at >= 0 {
			d := out.Documents[at]
			if d.RegisteredOwner != e.Owner {
				return fail(registryRefusal("owner-conflict", "document already registered to a different owner"))
			}
			return out, registryChange(d, false), nil
		}
		if len(out.Documents) >= registryMaxDocuments {
			return fail(registryLimit())
		}
		d := DocumentRegistration{ID: e.ID, File: e.File, RegisteredOwner: e.Owner, RegisteredAt: e.At, Paths: cloneRegistryPaths(e.InitialPaths)}
		if len(d.Paths) == 0 {
			d.Paths = []PathInterval{{File: e.File, Spelling: e.Spelling, From: e.At}}
		} else {
			last := &d.Paths[len(d.Paths)-1]
			if last.Until != nil || last.File != e.File {
				if last.Until == nil {
					end := e.At
					last.Until = &end
				}
				d.Paths = append(d.Paths, PathInterval{File: e.File, Spelling: e.Spelling, From: e.At})
			}
		}
		if len(d.Paths) > registryMaxPaths {
			return fail(registryLimit())
		}
		out.Documents = append(out.Documents, d)
		change = registryChange(d, true)
	case "rename":
		if !registryPath(e.Old) {
			return fail(registryInvalid("invalid old path"))
		}
		source := -1
		for i, d := range out.Documents {
			if d.File == e.Old {
				source = i
				break
			}
		}
		if source < 0 {
			if at >= 0 {
				d := out.Documents[at]
				if len(d.Paths) > 1 {
					prev := d.Paths[len(d.Paths)-2]
					if prev.File == e.Old || prev.Spelling == e.Old {
						return out, registryChange(d, false), nil
					}
				}
			}
			return fail(registryRefusal("not-registered", "rename source is not registered"))
		}
		if e.Old == e.File {
			current := out.Documents[source]
			if e.Spelling == "" || e.Spelling == current.Paths[len(current.Paths)-1].Spelling {
				return out, registryChange(current, false), nil
			}
		}
		if at >= 0 && at != source {
			return fail(registryRefusal("target-conflict", "rename target is already registered"))
		}
		d := &out.Documents[source]
		if len(d.Paths) >= registryMaxPaths {
			return fail(registryLimit())
		}
		last := &d.Paths[len(d.Paths)-1]
		if e.At.IsZero() || e.At.Before(last.From) {
			return fail(registryInvalid("rename time precedes current path interval"))
		}
		end := e.At
		last.Until = &end
		d.Paths = append(d.Paths, PathInterval{File: e.File, Spelling: e.Spelling, From: e.At})
		d.File = e.File
		change = registryChange(*d, true)
	case "forget":
		if at < 0 {
			return out, RegistryChange{}, nil
		}
		change = registryChange(out.Documents[at], true)
		out.Documents = append(out.Documents[:at], out.Documents[at+1:]...)
	default:
		return fail(registryInvalid("unknown registry event"))
	}
	if _, err := encodeRegistry(out); err != nil {
		return fail(err)
	}
	return out, change, nil
}

func encodeRegistry(r RegistryState) ([]byte, error) {
	if err := r.validate(); err != nil {
		return nil, err
	}
	raw, err := json.Marshal(r)
	if err != nil {
		return nil, registryInvalid(err.Error())
	}
	if len(raw) > registryMaxBytes {
		return nil, registryLimit()
	}
	return raw, nil
}
