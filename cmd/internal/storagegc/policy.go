// Package storagegc implements Pair's explicit storage-retention policy.
package storagegc

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
)

const RetentionPeriod = 60 * 24 * time.Hour

// ActivityRecord is bounded evidence for one owner incarnation, never a log.
type ActivityRecord struct {
	Version       int                       `json:"version"`
	Owner         artifactpath.StorageOwner `json:"owner"`
	Incarnation   string                    `json:"incarnation"`
	InitializedAt time.Time                 `json:"initialized_at"`
	LastUse       time.Time                 `json:"last_use"`
}

func (a ActivityRecord) validate(owner artifactpath.StorageOwner) error {
	resolved, err := artifactpath.NewStorageOwner(owner.DataDir, owner.RepoScope, owner.Tag)
	if err != nil || resolved != owner {
		return errors.New("invalid storage owner")
	}
	if a.Version != 1 || a.Owner != owner || a.Incarnation == "" || a.InitializedAt.IsZero() {
		return errors.New("invalid activity evidence")
	}
	return nil
}

func DecodeActivity(raw []byte, owner artifactpath.StorageOwner) (ActivityRecord, error) {
	var a ActivityRecord
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&a); err != nil {
		return a, fmt.Errorf("decode activity: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return a, errors.New("activity has trailing data")
	}
	return a, a.validate(owner)
}

type RetentionState string

const (
	Protected RetentionState = "protected"
	Live      RetentionState = "live"
	Grace     RetentionState = "grace"
	Eligible  RetentionState = "eligible"
	Untracked RetentionState = "untracked"
	Blocked   RetentionState = "blocked"
)

type RetentionDecision struct {
	State      RetentionState `json:"state"`
	Reason     string         `json:"reason"`
	EligibleAt time.Time      `json:"eligible_at,omitempty"`
}

// Evidence is a complete snapshot acquired by the IO shell. Zero values fail
// closed. Missing archive grace is represented explicitly by a zero timestamp.
type Evidence struct {
	Owner        artifactpath.StorageOwner
	Activity     *ActivityRecord
	Protected    bool
	Live         Liveness
	Complete     bool
	BlockReason  string
	ArchiveTimes []time.Time
}

func Decide(now time.Time, e Evidence) RetentionDecision {
	result := func(s RetentionState, r string) RetentionDecision { return RetentionDecision{State: s, Reason: r} }
	if e.Protected {
		return result(Protected, "visible Couch thread")
	}
	if e.Live == ProcessAlive {
		return result(Live, "active process or reader")
	}
	if !e.Complete {
		return result(Blocked, "incomplete ownership inventory")
	}
	if e.BlockReason != "" {
		return result(Blocked, e.BlockReason)
	}
	if e.Live != ProcessDead {
		return result(Blocked, "process liveness unknown")
	}
	if e.Activity == nil {
		return result(Untracked, "initial 60-day grace has not started")
	}
	if err := e.Activity.validate(e.Owner); err != nil {
		return result(Blocked, err.Error())
	}
	if now.IsZero() {
		return result(Blocked, "invalid current time")
	}
	last := e.Activity.InitializedAt
	if e.Activity.LastUse.After(last) {
		last = e.Activity.LastUse
	}
	for _, archive := range e.ArchiveTimes {
		if archive.IsZero() {
			return result(Untracked, "archive grace has not started")
		}
		if archive.After(last) {
			last = archive
		}
	}
	if last.After(now) {
		return result(Blocked, "retention clock is in the future")
	}
	expiry := last.Add(RetentionPeriod)
	if now.Before(expiry) {
		return RetentionDecision{State: Grace, Reason: "within 60 days of meaningful use or initialization", EligibleAt: expiry}
	}
	return RetentionDecision{State: Eligible, Reason: "60 days without meaningful use", EligibleAt: expiry}
}
