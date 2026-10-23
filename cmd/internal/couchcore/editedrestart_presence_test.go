package couchcore

import (
	"context"
	"errors"
	"testing"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/sessioninventorytest"
)

type editedRestartPresenceArtifacts struct {
	*FakeThreadArtifactCollisionChecker
	bound       TranscriptBind
	bindErr     error
	presence    editedRestartNativeObservation
	presenceErr error
	calls       int
}

type editedRestartRuntimeArtifacts struct {
	*FakeThreadArtifactCollisionChecker
	runtime       sessioninventory.Runtime
	metadataCalls int
	t             *testing.T
}

func (f *editedRestartRuntimeArtifacts) BindTranscript(ctx context.Context, thread ThreadRecord) (TranscriptBind, error) {
	return bindTranscriptWith(ctx, f.runtime, listingResolver{t: f.t}, thread)
}

func (f *editedRestartRuntimeArtifacts) ObserveEditedRestartNativeMetadata(ctx context.Context, thread ThreadRecord, id string) (editedRestartNativeObservation, error) {
	f.metadataCalls++
	return observeEditedRestartNativeMetadata(ctx, f.runtime, thread, id)
}

func TestResolveEditedRestartLocatedEvidenceKeepsOneStatAndNoList(t *testing.T) {
	for _, agent := range []string{"claude", "codex"} {
		for _, n := range []int{8, 16} {
			t.Run(agent+map[int]string{8: "/8", 16: "/16"}[n], func(t *testing.T) {
				env := newTestEnv(t, "/repo")
				rt := sessioninventorytest.NewFakeRuntime()
				rt.SetPairDataRoot(sessioninventory.StorageRoot{Name: "pair-data", Path: "/data/repos/scope"})
				stored := editedRestartObserved(agent, bindID(3))
				rt.AddRoot(sessioninventory.StorageRoot{Agent: sessioninventory.Agent(agent), Name: stored.StorageRoot, Path: "/native"})
				for i := 0; i < n; i++ {
					b := editedRestartObserved(agent, bindID(i+10))
					rt.PutFile(sessioninventory.FileEntry{Artifact: sessioninventory.Artifact{StorageRoot: b.StorageRoot, RelativePath: b.RelativePath}}, []byte("{}\n"))
				}
				rt.PutFile(sessioninventory.FileEntry{Artifact: sessioninventory.Artifact{StorageRoot: stored.StorageRoot, RelativePath: stored.RelativePath}}, []byte("{}\n"))
				artifacts := &editedRestartRuntimeArtifacts{FakeThreadArtifactCollisionChecker: env.Artifacts, runtime: rt, t: t}
				env.Couch.Artifacts = artifacts
				thread := ThreadRecord{Address: ThreadAddress{RepoScope: "scope", Tag: "work"}, LatestLaunchProfile: &LaunchProfile{Agent: agent}, Transcript: stored}
				got, err := env.Couch.resolveEditedRestartEvidence(context.Background(), thread, &stored.NativeID)
				if err != nil || got.Kind != EditedRestartNativeFound || got.Transcript == nil || artifacts.metadataCalls != 0 {
					t.Fatalf("located binding took metadata fallback: %+v calls=%d err=%v", got, artifacts.metadataCalls, err)
				}
				if lists, stats := rt.OperationCountForRoot(sessioninventorytest.OperationListFiles, stored.StorageRoot), rt.OperationCountForRoot(sessioninventorytest.OperationStatFile, stored.StorageRoot); lists != 0 || stats != 1 {
					t.Fatalf("N=%d native lists=%d stats=%d, want0/1", n, lists, stats)
				}
				if rt.OperationCountForRoot(sessioninventorytest.OperationReadAt, stored.StorageRoot)+rt.OperationCountForRoot(sessioninventorytest.OperationReadFile, stored.StorageRoot) != 0 {
					t.Fatal("located observation read native body")
				}
			})
		}
	}
}

func (f *editedRestartPresenceArtifacts) BindTranscript(context.Context, ThreadRecord) (TranscriptBind, error) {
	return f.bound, f.bindErr
}
func (f *editedRestartPresenceArtifacts) ObserveEditedRestartNativeMetadata(_ context.Context, _ ThreadRecord, id string) (editedRestartNativeObservation, error) {
	f.calls++
	if id != bindID(3) {
		return editedRestartNativeObservation{}, errors.New("wrong metadata target")
	}
	return f.presence, f.presenceErr
}

func TestResolveEditedRestartEvidenceCompletesIDOnlyEvidence(t *testing.T) {
	for _, agent := range []string{"claude", "codex"} {
		for _, status := range []sessioninventory.BindingStatus{sessioninventory.BindingEstablished, sessioninventory.BindingProvisional} {
			for _, missing := range []bool{false, true} {
				t.Run(agent+"/"+string(status)+map[bool]string{false: "/found", true: "/missing"}[missing], func(t *testing.T) {
					env := newTestEnv(t, "/repo")
					thread := createParkedThreadInCouch(t, env, LaunchProfile{Agent: agent, Argv: []string{}})
					id := bindID(3)
					kind := EditedRestartNativeFound
					if missing {
						kind = EditedRestartNativeMissing
					}
					artifacts := &editedRestartPresenceArtifacts{FakeThreadArtifactCollisionChecker: env.Artifacts, bound: TranscriptBind{Resolution: NativeBindingResolution{Status: status, NativeID: id, RequestedNativeID: id}}, presence: editedRestartNativeObservation{Kind: kind, NativeID: id, Transcript: editedRestartObserved(agent, id)}}
					env.Couch.Artifacts = artifacts
					got, err := env.Couch.resolveEditedRestartEvidence(context.Background(), thread, &id)
					if err != nil || got.Kind != kind || got.NativeID != id || artifacts.calls != 1 || got.Transcript == nil || got.Transcript == artifacts.presence.Transcript {
						t.Fatalf("ID-only evidence not completed/cloned: %+v calls=%d err=%v", got, artifacts.calls, err)
					}
				})
			}
		}
	}
}

func TestResolveEditedRestartMetadataCannotOverridePriorRefusal(t *testing.T) {
	for _, fault := range []string{"located", "replacement", "ambiguous", "provisional-fresh", "incomplete", "permission"} {
		t.Run(fault, func(t *testing.T) {
			env := newTestEnv(t, "/repo")
			thread := createParkedThreadInCouch(t, env, LaunchProfile{Agent: "claude", Argv: []string{}})
			id := bindID(3)
			artifacts := &editedRestartPresenceArtifacts{FakeThreadArtifactCollisionChecker: env.Artifacts, bound: TranscriptBind{Resolution: NativeBindingResolution{Status: sessioninventory.BindingEstablished, NativeID: id}}, presence: editedRestartNativeObservation{Kind: EditedRestartNativeMissing, NativeID: id}}
			switch fault {
			case "located":
				artifacts.bound.Observed = editedRestartObserved("claude", id)
			case "replacement":
				artifacts.bound.Resolution.NativeID = bindID(5)
			case "ambiguous":
				artifacts.bound.Resolution.Status = sessioninventory.BindingAmbiguous
			case "provisional-fresh":
				artifacts.bound.Resolution.Status = sessioninventory.BindingProvisional
				artifacts.bound.Resolution.FreshRequired = true
			case "incomplete":
				artifacts.bound.Resolution.ObservationIncomplete = true
			case "permission":
				artifacts.bindErr = errors.New("permission denied")
			}
			env.Couch.Artifacts = artifacts
			got, err := env.Couch.resolveEditedRestartEvidence(context.Background(), thread, &id)
			if artifacts.calls != 0 || got.Kind == EditedRestartNativeMissing || fault != "located" && err == nil {
				t.Fatalf("prior evidence bypassed: %+v calls=%d err=%v", got, artifacts.calls, err)
			}
		})
	}
}
