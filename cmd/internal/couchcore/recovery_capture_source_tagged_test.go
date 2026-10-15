package couchcore

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/threadrecord"
)

func TestTaggedRecoveryCaptureSourceUsesOrientationWithoutLegacyContext(t *testing.T) {
	paths, err := artifactpath.Resolve(artifactpath.Address{DataDir: t.TempDir(), RepoScope: "abcdef1234567890", Tag: "source"})
	if err != nil {
		t.Fatal(err)
	}
	archive, err := paths.ParkedScrollbackArtifacts("20261015T080000")
	if err != nil {
		t.Fatal(err)
	}
	capture := threadrecord.RecoveryCapture{Raw: archive.Raw, Events: archive.Events, Token: "20261015T080000"}
	for _, location := range []string{"claim", "record"} {
		t.Run(location, func(t *testing.T) {
			intent := &threadrecord.RecoveryIntent{Mode: threadrecord.RecoveryFresh, Orientation: &threadrecord.RecoveryOrientation{Capture: &capture}}
			record := ThreadRecord{RecoveryIntent: intent}
			if location == "claim" {
				record.RecoveryIntent = nil
				record.Incarnations = []ThreadIncarnation{{Start: &ThreadStartClaim{RecoveryIntent: intent}}}
			}
			got, err := recoveryCaptureSource(record, paths, "codex")
			if err != nil || got != capture {
				t.Fatalf("lost tagged capture: %+v err=%v", got, err)
			}
		})
	}
}

func TestTaggedRecoveryCaptureSourceNeverInfersFromAnotherTagOrMissingOrientation(t *testing.T) {
	root := t.TempDir()
	paths, _ := artifactpath.Resolve(artifactpath.Address{DataDir: root, RepoScope: "abcdef1234567890", Tag: "new-target"})
	old, _ := artifactpath.Resolve(artifactpath.Address{DataDir: root, RepoScope: "abcdef1234567890", Tag: "old-source"})
	archived, _ := old.ParkedScrollbackArtifacts("20261015T080000")
	for _, mode := range []string{"old-tag", "missing-capture", "arbitrary-path"} {
		t.Run(mode, func(t *testing.T) {
			capture := &threadrecord.RecoveryCapture{Raw: archived.Raw, Events: archived.Events, Token: "20261015T080000"}
			if mode == "missing-capture" {
				capture = nil
			}
			if mode == "arbitrary-path" {
				capture.Raw = filepath.Join(root, "outside.raw")
			}
			intent := &threadrecord.RecoveryIntent{Mode: threadrecord.RecoveryFresh, Orientation: &threadrecord.RecoveryOrientation{Capture: capture}}
			record := ThreadRecord{RecoveryIntent: intent, Incarnations: []ThreadIncarnation{{Start: &ThreadStartClaim{RecoveryIntent: intent}}}}
			got, err := recoveryCaptureSource(record, paths, "codex")
			current, _ := paths.ScrollbackArtifacts("codex")
			if err != nil || got.Raw != current.Raw || got.Events != current.Events || got.Token != "" {
				t.Fatalf("old-tag/empty legacy became inference authority: %+v err=%v", got, err)
			}
		})
	}
}

func TestTaggedRecoveryBranchIssueKeepsRetainedOrientation(t *testing.T) {
	for _, location := range []string{"claim", "record"} {
		t.Run(location, func(t *testing.T) {
			git := &recoveryEvidenceGit{}
			reader := OSRecoveryEvidenceReader{Git: git}
			intent := &threadrecord.RecoveryIntent{Mode: threadrecord.RecoveryFresh, Orientation: &threadrecord.RecoveryOrientation{Branch: "original-source-branch", Issue: "#556"}, Context: threadrecord.RecoveryContext{Branch: "legacy-context-is-not-tagged"}}
			record := ThreadRecord{RecoveryIntent: intent, RecoveryContext: &threadrecord.RecoveryContext{Branch: "unrelated-old-context"}}
			if location == "claim" {
				record.RecoveryIntent = nil
				record.Incarnations = []ThreadIncarnation{{Start: &ThreadStartClaim{RecoveryIntent: intent}}}
			}
			branch, issue := reader.recoveryBranchIssue(context.Background(), record)
			if branch != "original-source-branch" || issue != "#556" || git.calls != 0 {
				t.Fatalf("branch=%q issue=%q git=%d", branch, issue, git.calls)
			}
		})
	}
}

func TestTaggedRecoveryBranchIssueNeverUsesLegacyContext(t *testing.T) {
	git := &recoveryEvidenceGit{}
	reader := OSRecoveryEvidenceReader{Git: git}
	intent := &threadrecord.RecoveryIntent{Mode: threadrecord.RecoveryFresh, Context: threadrecord.RecoveryContext{Branch: "legacy-context-is-not-tagged"}}
	record := ThreadRecord{Incarnations: []ThreadIncarnation{{Start: &ThreadStartClaim{RecoveryIntent: intent}}}}
	branch, issue := reader.recoveryBranchIssue(context.Background(), record)
	if branch != "000528-recovery" || issue != "#528" || git.calls != 1 {
		t.Fatalf("branch=%q issue=%q git=%d", branch, issue, git.calls)
	}
}
