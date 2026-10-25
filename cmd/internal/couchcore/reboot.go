package couchcore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
)

// RebootTarget selects explicit Fresh recovery at a slot or exact thread.
// Agent optionally chooses another agent's known configuration.
// Argv, with Agent, is an explicit launch profile (pair#847): that exact agent
// and argv, not remembered for the repository.
type RebootTarget struct {
	Path    string
	Address ThreadAddress
	Agent   string
	Argv    *[]string
}

type RebootResult struct {
	Start             StartResult
	Archived          ThreadAddress
	ArchiveOnly       bool
	Reason            string
	SessionNotStopped bool
	Notice            string
}

func (r RebootResult) Started() (StartResult, bool) { return r.Start, r.Start.Handle != nil }
func (r RebootResult) Warning() string {
	parts := []string{}
	if note := (RecoveryResumeResult{Start: r.Start, Notice: r.Notice}).Warning(); note != "" {
		parts = append(parts, note)
	}
	if note := (ArchiveResult{Record: ThreadRecord{Address: r.Archived}, SessionNotStopped: r.SessionNotStopped}).Warning(); note != "" {
		parts = append(parts, note)
	}
	return strings.Join(parts, "; ")
}

// Reboot is the compatibility facade for shared Fresh recovery. The only
// separate effect is filing primary bytes that cannot supply a usable launch
// path; that escape never observes, stops or launches a process.
func (c *Couch) Reboot(ctx context.Context, t RebootTarget) (RebootResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return RebootResult{}, err
	}
	if c == nil || c.Threads == nil {
		return RebootResult{}, errors.New("reboot requires a thread store")
	}
	if (t.Path == "") == (t.Address == ThreadAddress{}) {
		return RebootResult{}, errors.New("reboot needs exactly one slot path or thread address")
	}
	if t.Argv != nil {
		if _, err := ParseBenchProfile(t.Agent, *t.Argv); err != nil {
			return RebootResult{}, err
		}
	}
	target := ResumeTarget{Address: t.Address, Path: t.Path}
	if t.Path != "" {
		_, record, err := c.recoveryTarget(ctx, target)
		if err != nil {
			// Recordless and malformed numbered slots retain the shared transaction's
			// version, ownership and workspace admission; the facade has no stop path.
			recovered, err := c.RecoverSelection(ctx, RecoverySelection{Target: target, Kind: RecoveryFresh, Agent: t.Agent, Argv: t.Argv})
			return RebootResult{Start: recovered.Start, Notice: recovered.Notice}, err
		}
		target = ResumeTarget{Address: record.Address}
	}
	address := target.Address
	if err := validateThreadAddress(address); err != nil {
		return RebootResult{}, err
	}
	ctx, release, err := c.hold(ctx, address, "reboot")
	if err != nil {
		return RebootResult{}, err
	}
	defer release()
	view := *c.Threads
	view.readOnly = true
	backend, err := view.storeForAddressContext(ctx, address)
	if err != nil {
		return RebootResult{}, err
	}
	var raw []byte
	err = backend.withReadLockContext(ctx, func() error {
		var e error
		raw, e = backend.readPayloadContext(ctx, backend.recordPath(address))
		return e
	})
	if errors.Is(err, os.ErrNotExist) {
		return RebootResult{}, fmt.Errorf("reboot %s: %s", address.Tag, rebootNothing)
	}
	if err != nil {
		return RebootResult{}, err
	}
	// A complete future envelope is an explicit version refusal. Truncated JSON
	// can still leave through the raw archive escape without schema invention.
	var envelope struct {
		SchemaVersion int `json:"schema_version"`
	}
	if json.Unmarshal(raw, &envelope) == nil && envelope.SchemaVersion > ThreadSchemaVersion {
		return RebootResult{}, fmt.Errorf("unsupported thread schema version %d; use a compatible Couch binary", envelope.SchemaVersion)
	}
	record, decodeErr := backend.decodeThreadRaw(address, raw)
	if backend.layout.Local {
		recovered, err := c.RecoverSelection(ctx, RecoverySelection{Target: target, Kind: RecoveryFresh, Agent: t.Agent, Argv: t.Argv})
		return RebootResult{Start: recovered.Start, Archived: address, Notice: recovered.Notice}, err
	}
	facts := RebootFacts{Record: RebootRecordReadable}
	if decodeErr != nil {
		facts.Record = RebootRecordUnreadable
	} else {
		facts.DirectoryPresent = c.directoryPresent(record.StartingPath) && c.workingPathExists(record)
	}
	plan, reason := DecideReboot(facts)
	if plan == RebootArchiveOnly {
		if err := ctx.Err(); err != nil {
			return RebootResult{}, err
		}
		if decodeErr != nil {
			escape, err := backend.unreadableEscape(address, raw)
			if err != nil {
				return RebootResult{}, err
			}
			if err := c.Threads.ArchiveThreadExpectedBytes(address, raw, escape); err != nil {
				return RebootResult{}, err
			}
			return RebootResult{Archived: address, ArchiveOnly: true, Reason: reason, SessionNotStopped: true}, nil
		}
		// Readable with its folder gone: the escape still stops nothing, but
		// the record names a tree, so it leaves only once that tree is
		// proven down (pair#695); a survivor is named and refused.
		proof, err := c.ProveDown(ctx, record)
		if err != nil {
			return RebootResult{}, fmt.Errorf("reboot %s: %w", address.Tag, err)
		}
		if err := c.Threads.ArchiveThreadExpected(address, proof.Revision(), proof); err != nil {
			return RebootResult{}, err
		}
		return RebootResult{Archived: address, ArchiveOnly: true, Reason: reason, SessionNotStopped: true}, nil
	}
	recovered, err := c.RecoverSelection(ctx, RecoverySelection{Target: target, Kind: RecoveryFresh, Agent: t.Agent, Argv: t.Argv})
	return RebootResult{Start: recovered.Start, Archived: address, Notice: recovered.Notice}, err
}

func (c *Couch) directoryPresent(path string) bool {
	if path == "" || c.Path == nil {
		return false
	}
	_, err := c.Path.Physical(path)
	return err == nil
}
