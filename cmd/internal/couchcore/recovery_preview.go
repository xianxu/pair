package couchcore

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/storagegc"
	"github.com/xianxu/pair/cmd/internal/threadrecord"
)

type RecoveryChoice struct {
	SourceKind   sessioninventory.RecoverySourceKind `json:"source_kind,omitempty"`
	CWD          string                              `json:"cwd,omitempty"`
	Key          string                              `json:"key"`
	Label        string                              `json:"label"`
	Agent        string                              `json:"agent"`
	NativeID     string                              `json:"native_id,omitempty"`
	Excerpt      string                              `json:"excerpt,omitempty"`
	LastActivity time.Time                           `json:"last_activity"`
	TailMatch    float64                             `json:"tail_match"`
	Matched      int                                 `json:"matched"`
	Total        int                                 `json:"total"`
}

// RecoveryPreview is shared by the menu and authenticated read-only CLI.
// Automatic remains false until a separate reviewed calibration policy exists.
type RecoveryPreview struct {
	Screen     ScreenObservation             `json:"screen"`
	Address    ThreadAddress                 `json:"address"`
	Revision   uint64                        `json:"revision"`
	Candidates []RecoveryChoice              `json:"candidates"`
	Proven     *RecoveryChoice               `json:"proven,omitempty"`
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
	Source                           *threadrecord.NativeRecoverySource
	Screen                           ScreenObservation
	BytesRead                        int64
	producingLaunch                  *time.Time
	renderedTail                     []string
}

type ScreenAvailability string

const (
	ScreenAvailable   ScreenAvailability = "available"
	ScreenUnavailable ScreenAvailability = "unavailable"
)

type ScreenObservation struct {
	Availability   ScreenAvailability `json:"availability"`
	Reason         string             `json:"reason,omitempty"`
	LastActivity   time.Time          `json:"last_activity,omitzero"`
	ActivitySource string             `json:"activity_source,omitempty"`
}

type RecoveryObservationRequest struct {
	ObservedAt       time.Time
	Source           *threadrecord.NativeRecoverySource
	BytesAlreadyRead int64
}

type RecoveryEvidenceReader func(context.Context, ThreadRecord, RecoveryObservationRequest) (RecoveryObservation, error)

type RecoveryResumeResult struct {
	// Mode records the completed operation, independently of the requested evidence kind.
	Mode   ContinuityAction
	Start  StartResult
	Notice string
}

func (r RecoveryResumeResult) Started() (StartResult, bool) { return r.Start, r.Start.Handle != nil }

func (r RecoveryResumeResult) Warning() string {
	parts := []string{}
	for _, note := range []string{r.Notice, r.Start.Warning()} {
		if note != "" {
			parts = append(parts, note)
		}
	}
	return strings.Join(parts, "; ")
}

// DecideSlotRecovery offers explicit choices even for a sole perfect match.
// Every key binds the original capture and complete ordered evidence snapshot;
// changed rank, root identity or capture requires a new explicit selection.
func DecideSlotRecovery(record ThreadRecord, observation RecoveryObservation, now time.Time) (RecoveryPreview, error) {
	p := RecoveryPreview{Address: record.Address, Revision: record.Revision, Screen: observation.Screen, Candidates: []RecoveryChoice{}, Incomplete: observation.Incomplete, Reasons: append([]string(nil), observation.Reasons...)}
	// Always project Fresh. An empty key is deliberately not native-resume
	// authority: the UI uses its existing confirmed fresh-start action instead.
	p.Fresh = &RecoveryChoice{Label: "Start fresh", Agent: observation.Agent, LastActivity: observation.Capture.LastActivity}
	if observation.Source != nil {
		// Native alternatives have no producing capture. Its readable file time
		// stays in Screen with its provenance instead of labelling Fresh activity.
		p.Fresh.LastActivity = time.Time{}
	}

	if observation.Incomplete {
		return p, nil
	}
	capture := observation.Capture
	expiry := capture.CapturedAt.Add(storagegc.CaptureRetentionPeriod)
	if observation.Source != nil {
		if err := threadrecord.ValidateNativeRecoverySourceAt(observation.Source, now, record.Revision); err != nil {
			return p, err
		}
		expiry = observation.Source.Cutoff.Add(threadrecord.NativeRecoverySelectionLifetime)
	} else if capture.CapturedAt.IsZero() || now.Before(capture.CapturedAt) || !now.Before(expiry) {
		p.Reasons = append(p.Reasons, "Previous screen is unavailable or expired; a fresh conversation may lose its working context.")
		return p, nil
	}
	snapshot := threadrecord.RecoveryContext{Version: 1, Address: threadrecord.Address{RepoScope: record.Address.RepoScope, Tag: string(record.Address.Tag)}, Capture: capture, Generation: observation.Generation, CreatedAt: now, ExpiresAt: expiry, Branch: observation.Branch, Issue: observation.Issue}
	if observation.Source != nil {
		snapshot.Version, snapshot.Source = 2, observation.Source
		snapshot.CreatedAt = observation.Source.Cutoff
		snapshot.Capture, snapshot.Generation = threadrecord.RecoveryCapture{}, ""
	}
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
		if snapshot.Source != nil {
			last := &snapshot.Candidates[len(snapshot.Candidates)-1]
			last.CWD = candidate.CWD
			last.Score, last.Matched, last.Total, last.Excerpt = 0, 0, 0, ""
		}
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
		selector := RecoveryCandidateSelector{Version: snapshot.Version, Address: record.Address, Agent: candidate.Agent, NativeID: candidate.NativeID, Digest: snapshot.Digest, Source: snapshot.Source}
		if snapshot.Source != nil {
			selector.ExpiresAt = snapshot.ExpiresAt
		}
		key, err := EncodeRecoveryCandidateKey(selector)
		if err != nil {
			return p, err
		}
		candidate.Key = key
		label := "resume (best candidate)"
		if i > 0 {
			label = fmt.Sprintf("resume (option %d)", i)
		}
		if snapshot.Source != nil {
			label = "resume (unconfirmed native candidate)"
		}
		choice := RecoveryChoice{Key: key, Label: label, Agent: candidate.Agent, NativeID: candidate.NativeID, Excerpt: candidate.Excerpt, LastActivity: candidate.LastActivity, TailMatch: candidate.Score, Matched: candidate.Matched, Total: candidate.Total}
		if snapshot.Source != nil {
			choice.SourceKind, choice.CWD = snapshot.Source.Kind, candidate.CWD
		}
		p.Candidates = append(p.Candidates, choice)
	}
	if snapshot.Source != nil {
		if len(snapshot.Candidates) == 0 {
			return p, nil
		}
		if err := threadrecord.ValidateRecoveryContext(snapshot, snapshot.Address); err != nil {
			return p, err
		}
		p.Snapshot = &snapshot
		return p, nil
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
		view := *c.Threads
		view.readOnly = true
		record, err := view.PreviewThreadContext(ctx, t.Address)
		return c.Threads, record, err
	}
	if c.Slots == nil {
		return nil, ThreadRecord{}, errors.New("slot services unavailable")
	}
	target, _, err := c.resolveSlotInput(ctx, t.Path)
	if err != nil {
		return nil, ThreadRecord{}, err
	}
	if target == nil || target.Kind != ThreadTargetSlot {
		return nil, ThreadRecord{}, errors.New("recovery path does not identify an existing slot")
	}
	local := newSlotThreadStore(c.Namespace, target.Slot)
	local.coordinator = c.Threads.coordinator
	view := *local
	view.readOnly = true
	var observed slotCurrentObservation
	err = view.withPreviewLockContext(ctx, func() error {
		var readErr error
		observed, readErr = view.readSlotCurrentLockedContext(ctx)
		return readErr
	})
	if errors.Is(err, os.ErrNotExist) {
		return nil, ThreadRecord{}, ErrThreadNotFound
	}
	if err != nil {
		return nil, ThreadRecord{}, err
	}
	if observed.Unsupported {
		return nil, ThreadRecord{}, observed.Err
	}
	if observed.Record == nil {
		return nil, ThreadRecord{}, fmt.Errorf("slot has no retained conversation record: %w", ErrThreadNotFound)
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
		return RecoveryPreview{Address: t.Address, Incomplete: true, Reasons: []string{err.Error()}, Fresh: &RecoveryChoice{Label: "Start fresh"}}, nil
	}
	observation := c.recoveryObservation(ctx, record)
	preview, err := DecideSlotRecovery(record, observation, c.Clock.Now())
	if err != nil {
		preview.Candidates = nil
		preview.Incomplete = true
		preview.Reasons = append(preview.Reasons, err.Error())
	}
	if binding, err := c.resumeEvidence(ctx, record); err == nil && coldResumeAuthorized(binding) {
		preview.Proven = &RecoveryChoice{Label: "Resume from disk (confirmed)", Agent: RecordAgent(record), NativeID: binding.NativeID, Excerpt: "Established native binding"}
		if len(preview.Candidates) > 4 {
			preview.Candidates = preview.Candidates[:4]
			preview.Truncated = true
		}
	}
	return preview, nil
}
