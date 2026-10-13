package couchcore

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/xianxu/pair/cmd/internal/orientation"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/storagegc"
	"github.com/xianxu/pair/cmd/internal/threadrecord"
)

type RecoveryChoice struct {
	Key          string    `json:"key"`
	Label        string    `json:"label"`
	Agent        string    `json:"agent"`
	NativeID     string    `json:"native_id,omitempty"`
	Excerpt      string    `json:"excerpt,omitempty"`
	LastActivity time.Time `json:"last_activity"`
	TailMatch    float64   `json:"tail_match"`
	Matched      int       `json:"matched"`
	Total        int       `json:"total"`
}

// RecoveryPreview is shared by the menu and authenticated read-only CLI.
// Automatic remains false until a separate reviewed calibration policy exists.
type RecoveryPreview struct {
	Address    ThreadAddress                 `json:"address"`
	Revision   uint64                        `json:"revision"`
	Candidates []RecoveryChoice              `json:"candidates"`
	Fresh      *RecoveryChoice               `json:"fresh,omitempty"`
	Incomplete bool                          `json:"incomplete"`
	Reasons    []string                      `json:"reasons,omitempty"`
	Automatic  bool                          `json:"automatic"`
	Truncated  bool                          `json:"truncated"`
	Snapshot   *threadrecord.RecoveryContext `json:"-"`
}

// RecoveryObservation contains bounded IO facts, not permission to stop/start.
type RecoveryObservation struct {
	Capture                          threadrecord.RecoveryCapture
	Generation, Agent, Branch, Issue string
	Ranked                           []sessioninventory.RankedRecoveryCandidate
	Incomplete                       bool
	Reasons                          []string
}

type RecoveryEvidenceReader func(context.Context, ThreadRecord) (RecoveryObservation, error)

type RecoveryResumeResult struct {
	Start       StartResult
	Notice      string
	Orientation orientation.DeliveryState
}

func (r RecoveryResumeResult) Started() (StartResult, bool) { return r.Start, r.Start.Handle != nil }

// DecideSlotRecovery offers explicit choices even for a sole perfect match.
// Every key binds the original capture and complete ordered evidence snapshot;
// changed rank, root identity or capture requires a new explicit selection.
func DecideSlotRecovery(record ThreadRecord, observation RecoveryObservation, now time.Time) (RecoveryPreview, error) {
	p := RecoveryPreview{Address: record.Address, Revision: record.Revision, Candidates: []RecoveryChoice{}, Incomplete: observation.Incomplete, Reasons: append([]string(nil), observation.Reasons...)}
	if observation.Incomplete {
		return p, nil
	}
	capture := observation.Capture
	expiry := capture.CapturedAt.Add(storagegc.CaptureRetentionPeriod)
	if capture.CapturedAt.IsZero() || now.Before(capture.CapturedAt) || !now.Before(expiry) {
		return p, errors.New("recovery screen evidence expired or has no capture time; refresh resume candidates")
	}
	snapshot := threadrecord.RecoveryContext{Version: 1, Address: threadrecord.Address{RepoScope: record.Address.RepoScope, Tag: string(record.Address.Tag)}, Capture: capture, Generation: observation.Generation, CreatedAt: now, ExpiresAt: expiry, Branch: observation.Branch, Issue: observation.Issue}
	for _, ranked := range observation.Ranked {
		candidate := ranked.Candidate
		if candidate.Incomplete || !candidate.RootEligible || candidate.Interactive.State != sessioninventory.InteractiveYes || candidate.CWD != record.WorkingPath {
			continue
		}
		if len(snapshot.Candidates) == 5 {
			p.Truncated = true
			break
		}
		entry := candidate.Entry
		if entry.ModTime == nil || candidate.SemanticLast.IsZero() {
			return p, errors.New("recovery candidate has incomplete activity evidence")
		}
		snapshot.Candidates = append(snapshot.Candidates, threadrecord.RecoveryCandidate{Agent: string(candidate.Agent), NativeID: candidate.NativeID, ArtifactRoot: entry.Artifact.StorageRoot, ArtifactRelativePath: entry.Artifact.RelativePath, StableFileID: string(entry.StableFileID), GenerationToken: string(entry.GenerationToken), MutationToken: string(entry.MutationToken), Size: entry.Size, ModTime: *entry.ModTime, Score: ranked.Score.Percent, Matched: ranked.Score.Matched, Total: ranked.Score.Total, LastActivity: candidate.SemanticLast, Excerpt: ranked.Score.Excerpt, Eligible: true})
	}
	// Time of listing and chosen key are not source evidence. Omitting them
	// keeps an unchanged listing stable without creating a durable preview store.
	evidence := snapshot
	evidence.CreatedAt = time.Time{}
	encoded, err := json.Marshal(struct {
		Revision uint64
		Evidence threadrecord.RecoveryContext
	}{record.Revision, evidence})
	if err != nil {
		return p, err
	}
	sum := sha256.Sum256(encoded)
	snapshot.Digest = hex.EncodeToString(sum[:])
	for i := range snapshot.Candidates {
		candidate := &snapshot.Candidates[i]
		key, err := EncodeRecoveryCandidateKey(RecoveryCandidateSelector{Version: 1, Address: record.Address, Agent: candidate.Agent, NativeID: candidate.NativeID, Digest: snapshot.Digest})
		if err != nil {
			return p, err
		}
		candidate.Key = key
		label := "resume (best candidate)"
		if i > 0 {
			label = fmt.Sprintf("resume (option %d)", i)
		}
		p.Candidates = append(p.Candidates, RecoveryChoice{Key: key, Label: label, Agent: candidate.Agent, NativeID: candidate.NativeID, Excerpt: candidate.Excerpt, LastActivity: candidate.LastActivity, TailMatch: candidate.Score, Matched: candidate.Matched, Total: candidate.Total})
	}
	key, err := EncodeRecoveryCandidateKey(RecoveryCandidateSelector{Version: 1, Address: record.Address, Agent: observation.Agent, Digest: snapshot.Digest})
	if err != nil {
		return p, err
	}
	p.Fresh = &RecoveryChoice{Key: key, Label: "start fresh from previous screen", Agent: observation.Agent, LastActivity: capture.LastActivity}
	if err := threadrecord.ValidateRecoveryContext(snapshot, snapshot.Address); err != nil {
		return p, err
	}
	p.Snapshot = &snapshot
	return p, nil
}

func (c *Couch) recoveryTarget(ctx context.Context, t ResumeTarget) (*ThreadStore, ThreadRecord, error) {
	if c == nil || c.Threads == nil || (t.Path == "") == (t.Address == ThreadAddress{}) {
		return nil, ThreadRecord{}, errors.New("recovery needs exactly one slot path or thread address")
	}
	if t.Path == "" {
		record, err := c.Threads.GetThread(t.Address)
		return c.Threads, record, err
	}
	local, _, err := c.selectedSlot(ctx, t.Path)
	if err != nil {
		return nil, ThreadRecord{}, err
	}
	observed, err := local.observeSlotCurrent()
	if err != nil {
		return nil, ThreadRecord{}, err
	}
	if observed.Unsupported {
		return nil, ThreadRecord{}, observed.Err
	}
	if observed.Record == nil {
		return nil, ThreadRecord{}, errors.New("slot has no retained conversation record")
	}
	return local, *observed.Record, nil
}

// ResumeCandidates performs no lifecycle writes and never launches an agent.
func (c *Couch) ResumeCandidates(ctx context.Context, t ResumeTarget) (RecoveryPreview, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	_, record, err := c.recoveryTarget(ctx, t)
	if err != nil {
		return RecoveryPreview{}, err
	}
	if c.RecoveryEvidence == nil {
		return RecoveryPreview{}, errors.New("recovery evidence reader unavailable")
	}
	evidence, err := c.RecoveryEvidence(ctx, record)
	if err != nil {
		return RecoveryPreview{}, err
	}
	if err := ctx.Err(); err != nil {
		return RecoveryPreview{}, err
	}
	return DecideSlotRecovery(record, evidence, c.Clock.Now())
}
