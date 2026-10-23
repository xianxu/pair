package couchcore

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"testing"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/sessioninventorytest"
	"github.com/xianxu/pair/cmd/internal/threadrecord"
)

func editedRestartMetadataFixture(agent string, unrelated int) (*sessioninventorytest.FakeRuntime, ThreadRecord, *threadrecord.TranscriptBinding) {
	rt := sessioninventorytest.NewFakeRuntime()
	located := editedRestartObserved(agent, bindID(3))
	rt.AddRoot(sessioninventory.StorageRoot{Agent: sessioninventory.Agent(agent), Name: located.StorageRoot, Path: "/native"})
	rt.PutFile(sessioninventory.FileEntry{Artifact: sessioninventory.Artifact{StorageRoot: located.StorageRoot, RelativePath: located.RelativePath}}, []byte("opaque native body: metadata completion must never read this"))
	for i := 0; i < unrelated; i++ {
		other := editedRestartObserved(agent, bindID(100+i))
		rt.PutFile(sessioninventory.FileEntry{Artifact: sessioninventory.Artifact{StorageRoot: other.StorageRoot, RelativePath: other.RelativePath}}, []byte("unrelated native body"))
	}
	thread := ThreadRecord{LatestLaunchProfile: &LaunchProfile{Agent: agent}, Transcript: &threadrecord.TranscriptBinding{Agent: agent, NativeID: located.NativeID, Source: threadrecord.TranscriptSourceLedger}}
	return rt, thread, located
}

func assertEditedRestartMetadataNoBodyReads(t *testing.T, rt *sessioninventorytest.FakeRuntime, root string) {
	t.Helper()
	for _, operation := range []sessioninventorytest.Operation{sessioninventorytest.OperationReadFile, sessioninventorytest.OperationReadAt, sessioninventorytest.OperationSQLite} {
		if n := rt.OperationCountForRoot(operation, root); n != 0 {
			t.Fatalf("metadata observation read native body: operation=%s count=%d", operation, n)
		}
	}
}

func TestEditedRestartMetadataFindsExactTrustedIDWithoutBodyReads(t *testing.T) {
	for _, agent := range []string{"claude", "codex"} {
		for _, origin := range []string{"established", "requested-resume"} {
			for _, unrelated := range []int{8, 16} {
				t.Run(fmt.Sprintf("%s/%s/N=%d", agent, origin, unrelated), func(t *testing.T) {
					rt, thread, located := editedRestartMetadataFixture(agent, unrelated)
					if origin == "established" {
						thread.Transcript = threadrecord.CloneTranscriptBinding(located)
					}
					got, err := observeEditedRestartNativeMetadata(context.Background(), rt, thread, located.NativeID)
					if err != nil || got.Kind != EditedRestartNativeFound || got.NativeID != located.NativeID || got.Transcript == nil || got.Transcript.StorageRoot != located.StorageRoot || got.Transcript.RelativePath != located.RelativePath || got.Transcript.Source != threadrecord.TranscriptSourceLedger {
						t.Fatalf("exact metadata: %+v %v", got, err)
					}
					if err := got.Transcript.ValidateLocation(); err != nil {
						t.Fatal(err)
					}
					got.Transcript.NativeID = "changed returned locator"
					if thread.Transcript.NativeID != located.NativeID {
						t.Fatal("returned locator aliases accepted transcript")
					}
					if n := rt.OperationCountForRoot(sessioninventorytest.OperationListFiles, located.StorageRoot); n != 1 {
						t.Fatalf("metadata observations=%d want1", n)
					}
					if n := rt.OperationCountForRoot(sessioninventorytest.OperationStatFile, located.StorageRoot); n != 1 {
						t.Fatalf("exact stats=%d want1", n)
					}
					assertEditedRestartMetadataNoBodyReads(t, rt, located.StorageRoot)
				})
			}
		}
	}
}

func TestEditedRestartMetadataMissingAndUnknownStayDistinct(t *testing.T) {
	for _, agent := range []string{"claude", "codex"} {
		for _, fault := range []string{"complete-absent", "root-absent", "duplicate-path", "duplicate-root", "partial-listing", "permission-listing", "permission-stat", "vanished-at-stat", "cancelled-before", "cancelled-list", "cancelled-stat", "root-unavailable", "empty-id", "unsupported-agent"} {
			t.Run(agent+"/"+fault, func(t *testing.T) {
				rt, thread, located := editedRestartMetadataFixture(agent, 8)
				artifact := sessioninventory.Artifact{StorageRoot: located.StorageRoot, RelativePath: located.RelativePath}
				wrapped := &editedRestartMetadataRuntime{FakeRuntime: rt}
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				id := located.NativeID
				want, locator := EditedRestartNativeUnknown, false
				switch fault {
				case "complete-absent":
					rt.DeleteFile(artifact)
					want = EditedRestartNativeMissing
				case "root-absent":
					rt.SetError(sessioninventorytest.OperationListFiles, located.StorageRoot, sessioninventory.ErrStorageAbsent)
					want = EditedRestartNativeMissing
				case "duplicate-path":
					duplicate := "duplicate/" + filepath.Base(located.RelativePath)
					if agent == "codex" {
						duplicate = "2026/08/29/" + filepath.Base(located.RelativePath)
					}
					rt.PutFile(sessioninventory.FileEntry{Artifact: sessioninventory.Artifact{StorageRoot: located.StorageRoot, RelativePath: duplicate}}, []byte("other root-shaped artifact with same ID"))
				case "duplicate-root":
					rt.AddRoot(sessioninventory.StorageRoot{Agent: sessioninventory.Agent(agent), Name: located.StorageRoot, Path: "/another-native"})
				case "partial-listing":
					rt.SetError(sessioninventorytest.OperationListFiles, located.StorageRoot, &sessioninventory.ListingIssuesError{Artifacts: []sessioninventory.Artifact{{StorageRoot: located.StorageRoot, RelativePath: "unrelated-rejected-entry"}}})
				case "permission-listing":
					rt.SetError(sessioninventorytest.OperationListFiles, located.StorageRoot, fs.ErrPermission)
				case "permission-stat":
					rt.SetError(sessioninventorytest.OperationStatFile, located.StorageRoot+":"+located.RelativePath, fs.ErrPermission)
				case "vanished-at-stat":
					wrapped.beforeStat = func() { rt.DeleteFile(artifact) }
					want, locator = EditedRestartNativeMissing, true
				case "cancelled-before":
					cancel()
				case "cancelled-list":
					wrapped.afterList = cancel
				case "cancelled-stat":
					wrapped.afterStat = cancel
				case "root-unavailable":
					wrapped.rootsUnavailable = true
				case "empty-id":
					id = ""
				case "unsupported-agent":
					thread.LatestLaunchProfile.Agent = "unsupported"
				}
				got, err := observeEditedRestartNativeMetadata(ctx, wrapped, thread, id)
				if got.Kind != want || (err == nil) != (want != EditedRestartNativeUnknown) {
					t.Fatalf("metadata %s: %+v %v want=%v", fault, got, err, want)
				}
				if want == EditedRestartNativeMissing && (got.NativeID != located.NativeID || (got.Transcript != nil) != locator) {
					t.Fatalf("lost exact absence identity/location: %+v", got)
				}
				if fault == "permission-stat" && !errors.Is(err, fs.ErrPermission) {
					t.Fatal("lost stat permission cause")
				}
				if fault == "cancelled-before" || fault == "cancelled-list" || fault == "cancelled-stat" {
					if !errors.Is(err, context.Canceled) {
						t.Fatal("lost cancellation cause")
					}
				}
				if fault == "partial-listing" || fault == "permission-listing" || fault == "duplicate-path" || fault == "duplicate-root" {
					if n := rt.OperationCountForRoot(sessioninventorytest.OperationStatFile, located.StorageRoot); n != 0 {
						t.Fatalf("uncertain candidates were statted: %d", n)
					}
				}
				assertEditedRestartMetadataNoBodyReads(t, rt, located.StorageRoot)
			})
		}
	}
}

type editedRestartMetadataRuntime struct {
	*sessioninventorytest.FakeRuntime
	beforeStat, afterStat, afterList func()
	rootsUnavailable                 bool
}

func (r *editedRestartMetadataRuntime) NativeRoots(agent sessioninventory.Agent) []sessioninventory.StorageRoot {
	if r.rootsUnavailable {
		return nil
	}
	return r.FakeRuntime.NativeRoots(agent)
}
func (r *editedRestartMetadataRuntime) ListFiles(root sessioninventory.StorageRoot) ([]sessioninventory.FileEntry, error) {
	files, err := r.FakeRuntime.ListFiles(root)
	if r.afterList != nil {
		r.afterList()
	}
	return files, err
}
func (r *editedRestartMetadataRuntime) StatFile(artifact sessioninventory.Artifact) (sessioninventory.FileEntry, error) {
	if r.beforeStat != nil {
		r.beforeStat()
	}
	entry, err := r.FakeRuntime.StatFile(artifact)
	if r.afterStat != nil {
		r.afterStat()
	}
	return entry, err
}
