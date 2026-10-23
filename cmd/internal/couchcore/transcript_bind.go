package couchcore

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/sessionledger"
	"github.com/xianxu/pair/cmd/internal/threadrecord"
)

const (
	// ResumeTranscriptMissing: the stored conversation's file is gone. The
	// answer is Start fresh, never a search for another conversation.
	ResumeTranscriptMissing ResumeDiagnosticCode = "resume-transcript-missing"
	// ResumeTranscriptInvalid: the stored binding fails its structural check.
	ResumeTranscriptInvalid ResumeDiagnosticCode = "resume-transcript-invalid"
)

// TranscriptBinder binds a cold resume to its exact native conversation
// (pair#695 P2): the binding stored on the record, refreshed by one read of
// the tag's own ledger, and opened by one stat. It never lists a directory
// once a binding names its file.
type TranscriptBinder interface {
	BindTranscript(ctx context.Context, thread ThreadRecord) (TranscriptBind, error)
}

// TranscriptBind is a binding decision. Resolution is what the resume rules
// read. Store, when set, is written onto the record by the resume's start
// claim, in the same revision-checked write, which stamps its BoundAt with
// the claim's start time.
type TranscriptBind struct {
	Resolution NativeBindingResolution
	Store      *threadrecord.TranscriptBinding
	// Observed names the validated artifact selected for StatFile. Its presence
	// proves neither existence nor resume authority; the result/error does that.
	// It is independent of the source record and the success-only Store value.
	Observed *threadrecord.TranscriptBinding
}

// ledgerTranscript is what one exact read of the tag's ledger said.
type ledgerTranscript struct {
	Err      error // the ledger could not be read; nil when it was read
	Conflict bool
	NativeID string // the current launch's established root, if any
	// The root transcript named by the binding's authorization proof.
	StorageRoot, RelativePath string
}

// transcriptPlan is planTranscriptBind's answer: bind Use (stat it), or ask
// the listing resolver once (Fallback) because no file is known yet.
type transcriptPlan struct {
	Use      *threadrecord.TranscriptBinding
	Store    bool
	Fallback bool
	Refuse   ResumeDiagnosticCode
}

// planTranscriptBind is the pure binding rule.
//
//   - A stored binding for another agent (a switch-agent since) is ignored.
//   - The ledger's current established root wins when it differs from the
//     record: that is the conversation the operator moved to after a context
//     reset the record never saw.
//   - An unreadable or missing ledger never blocks a stored, located
//     binding: the binding is the authority, the ledger only a refresh.
//   - Only when no file is known (a session before its agent's first turn, a
//     record from before the field, a binding row without a proof) does the
//     listing resolver run, as it did before the field existed.
func planTranscriptBind(stored *threadrecord.TranscriptBinding, agent string, ledger ledgerTranscript) transcriptPlan {
	source := transcriptSource(stored)
	if stored != nil && stored.Agent != agent {
		stored = nil
	}
	// A stored binding this build cannot accept (hand-edited, or a newer
	// build's root) is replaced by a located ledger binding, else refused.
	invalid := stored != nil && stored.ValidateLocation() != nil
	if ledger.Err == nil && ledger.Conflict {
		return transcriptPlan{Refuse: ResumeBindingAmbiguous}
	}
	if ledger.Err == nil && ledger.NativeID != "" {
		candidate := &threadrecord.TranscriptBinding{Agent: agent, NativeID: ledger.NativeID,
			StorageRoot: ledger.StorageRoot, RelativePath: ledger.RelativePath, Source: source}
		if invalid && candidate.Established() {
			return transcriptPlan{Use: candidate, Store: true}
		}
		if invalid {
			return transcriptPlan{Refuse: ResumeTranscriptInvalid}
		}
		if stored != nil && stored.NativeID == ledger.NativeID && stored.Established() {
			if !candidate.Established() || (stored.StorageRoot == candidate.StorageRoot && stored.RelativePath == candidate.RelativePath) {
				return transcriptPlan{Use: stored}
			}
		}
		if candidate.Established() {
			return transcriptPlan{Use: candidate, Store: true}
		}
		return transcriptPlan{Fallback: true}
	}
	if invalid {
		return transcriptPlan{Refuse: ResumeTranscriptInvalid}
	}
	if stored != nil && stored.Established() {
		return transcriptPlan{Use: stored}
	}
	return transcriptPlan{Fallback: true}
}

// bindTranscriptWith is the production binder over a native runtime: one
// exact ledger read, then one stat of the bound file. fallback is the listing
// resolver, used only when planTranscriptBind finds no file to stat.
func bindTranscriptWith(ctx context.Context, rt sessioninventory.Runtime, fallback NativeBindingResolver, thread ThreadRecord) (TranscriptBind, error) {
	agent := threadAgent(thread)
	if agent == "" {
		resolution := NativeBindingResolution{Status: sessioninventory.BindingUnbound}
		return TranscriptBind{Resolution: resolution}, refuseBinding(ResumeBindingUnbound)
	}
	scope, tag := thread.Address.RepoScope, string(thread.Address.Tag)
	plan := planTranscriptBind(thread.Transcript, agent, readLedgerTranscript(ctx, rt, scope, tag, agent))
	if err := ctx.Err(); err != nil {
		return TranscriptBind{}, err
	}
	switch {
	case plan.Refuse == ResumeTranscriptInvalid:
		return TranscriptBind{}, refuseResume(ResumeTranscriptInvalid, "its transcript binding is not one this build can open; start it fresh")
	case plan.Refuse != "":
		resolution := NativeBindingResolution{Status: sessioninventory.BindingAmbiguous}
		return TranscriptBind{Resolution: resolution}, refuseBinding(plan.Refuse)
	case plan.Fallback:
		return bindByResolver(ctx, fallback, thread)
	}
	use := plan.Use
	if err := use.ValidateLocation(); err != nil {
		return TranscriptBind{}, refuseResume(ResumeTranscriptInvalid, fmt.Sprintf("its transcript binding is invalid (%v); start it fresh", err))
	}
	out := TranscriptBind{Observed: threadrecord.CloneTranscriptBinding(use)}
	if _, err := rt.StatFile(sessioninventory.Artifact{StorageRoot: use.StorageRoot, RelativePath: use.RelativePath}); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return out, refuseResume(ResumeTranscriptMissing, "the transcript of its stored conversation "+use.NativeID+" is missing; start it fresh")
		}
		return out, fmt.Errorf("stat stored transcript: %w", err)
	}
	out.Resolution = NativeBindingResolution{Status: sessioninventory.BindingEstablished, NativeID: use.NativeID}
	if plan.Store {
		out.Store = threadrecord.CloneTranscriptBinding(use)
	}
	return out, nil
}

// bindByResolver is the binder's fallback when no file is known: the listing
// resolver answers, as before the field existed, and an established answer is
// stored (ID only) so the record carries it from then on. It is also how a
// Couch whose artifacts only resolve (test doubles) binds, so both share it.
func bindByResolver(ctx context.Context, resolver NativeBindingResolver, thread ThreadRecord) (TranscriptBind, error) {
	agent := threadAgent(thread)
	resolution, err := resolver.ResolveEstablished(ctx, thread.Address.RepoScope, string(thread.Address.Tag), agent)
	out := TranscriptBind{Resolution: resolution}
	if err != nil || agent == "" || resolution.Status != sessioninventory.BindingEstablished || resolution.NativeID == "" {
		return out, err
	}
	stored := thread.Transcript
	if stored != nil && stored.Agent == agent && stored.NativeID == resolution.NativeID {
		return out, nil
	}
	out.Store = &threadrecord.TranscriptBinding{Agent: agent, NativeID: resolution.NativeID, Source: transcriptSource(stored)}
	return out, nil
}

// transcriptSource labels a binding about to be stored: legacy-ledger only
// for a record that had no binding at all, ledger for a refresh of one.
func transcriptSource(stored *threadrecord.TranscriptBinding) string {
	if stored == nil {
		return threadrecord.TranscriptSourceLegacyLedger
	}
	return threadrecord.TranscriptSourceLedger
}

func threadAgent(thread ThreadRecord) string {
	if thread.LatestLaunchProfile == nil {
		return ""
	}
	return thread.LatestLaunchProfile.Agent
}

// readLedgerTranscript reads the tag's ledger by its known name (no listing)
// and reports the current launch's established root and the root transcript
// its proof names.
func readLedgerTranscript(ctx context.Context, rt sessioninventory.Runtime, scope, tag, agent string) ledgerTranscript {
	root := rt.PairDataRoot()
	paths, err := artifactpath.ResolveScoped(root.Path, tag)
	if err != nil {
		return ledgerTranscript{Err: err}
	}
	ledger := sessioninventory.Artifact{StorageRoot: root.Name, RelativePath: filepath.Base(paths.Ledger())}
	current, ok, _, err := sessioninventory.ReadCurrentLaunchAt(ctx, rt, sessionledger.Owner{ScopeKey: scope, Tag: tag, Agent: agent}, ledger)
	if err != nil {
		return ledgerTranscript{Err: err}
	}
	if !ok {
		return ledgerTranscript{}
	}
	if current.Conflict {
		return ledgerTranscript{Conflict: true}
	}
	if current.Binding == nil || current.Binding.RootNativeID == "" {
		return ledgerTranscript{}
	}
	out := ledgerTranscript{NativeID: current.Binding.RootNativeID}
	if proof := current.Binding.AuthorizationProof; proof != nil {
		for _, artifact := range proof.Artifacts {
			if id, ok := sessioninventory.RootTranscriptID(sessioninventory.Agent(agent), artifact.RelativePath); ok && id == out.NativeID {
				out.StorageRoot, out.RelativePath = artifact.StorageRoot, artifact.RelativePath
				break
			}
		}
	}
	return out
}
