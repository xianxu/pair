package couchcore

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/storagegc"
	"github.com/xianxu/pair/cmd/internal/threadrecord"
)

func TestRecoveryCaptureSnapshotRejectsChangedMember(t *testing.T) {
	dir := t.TempDir()
	raw := filepath.Join(dir, "raw")
	events := filepath.Join(dir, "events")
	for _, p := range []string{raw, events} {
		if err := os.WriteFile(p, []byte("old"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	rawID, _ := storagegc.CaptureIdentity(raw)
	eventsID, _ := storagegc.CaptureIdentity(events)
	c := threadrecord.RecoveryCapture{Raw: raw, Events: events, RawIdentity: rawID, EventsIdentity: eventsID, CapturedAt: time.Now()}
	if err := validateRecoveryCaptureSnapshot(context.Background(), c); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(events, []byte("newer"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := validateRecoveryCaptureSnapshot(context.Background(), c); err == nil {
		t.Fatal("changed screen member accepted")
	}
}

func TestRecoveryCaptureSourceUsesRetainedClaimBeforeCurrent(t *testing.T) {
	paths, err := artifactpath.Resolve(artifactpath.Address{DataDir: t.TempDir(), RepoScope: "abcdef1234567890", Tag: "1-pair-42"})
	if err != nil {
		t.Fatal(err)
	}
	capture := threadrecord.RecoveryCapture{Raw: "/original/raw", Events: "/original/events"}
	record := ThreadRecord{LatestLaunchProfile: &LaunchProfile{Agent: "codex"}, RecoveryContext: &threadrecord.RecoveryContext{Capture: threadrecord.RecoveryCapture{Raw: "/older/raw", Events: "/older/events"}}, Incarnations: []ThreadIncarnation{{Start: &ThreadStartClaim{RecoveryIntent: &threadrecord.RecoveryIntent{Context: threadrecord.RecoveryContext{Capture: capture}}}}}}
	got, err := recoveryCaptureSource(record, paths, "codex")
	if err != nil || got.Raw != capture.Raw || got.Events != capture.Events {
		t.Fatalf("lost retained source %+v %v", got, err)
	}
}

type recoveryEvidenceGit struct{ calls int }

func (g *recoveryEvidenceGit) Run(dir string, args ...string) (string, error) {
	return g.RunContext(context.Background(), dir, args...)
}
func (g *recoveryEvidenceGit) RunContext(ctx context.Context, dir string, args ...string) (string, error) {
	g.calls++
	return "000528-recovery", ctx.Err()
}

func recoveryEvidenceFixture(t *testing.T, witnessed bool) (OSRecoveryEvidenceReader, ThreadRecord) {
	t.Helper()
	t.Setenv("CODEX_HOME", "")
	dir := t.TempDir()
	record := validThreadRecord(t)
	record.WorkingPath = dir
	record.StartingPath = dir
	record.Incarnations = nil
	record.LatestLaunchProfile = &LaunchProfile{Agent: "codex", Argv: []string{}}
	reader := OSRecoveryEvidenceReader{DataDir: filepath.Join(dir, "data"), HomeDir: filepath.Join(dir, "home"), Git: &recoveryEvidenceGit{}}
	paths, err := artifactpath.Resolve(artifactpath.Address{DataDir: reader.DataDir, RepoScope: record.Address.RepoScope, Tag: string(record.Address.Tag)})
	if err != nil {
		t.Fatal(err)
	}
	if err = os.MkdirAll(paths.ScopeDir(), 0700); err != nil {
		t.Fatal(err)
	}
	files, err := paths.ScrollbackArtifacts("codex")
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(files.Raw, []byte("previous objective and remaining work\r\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(files.Events, nil, 0600); err != nil {
		t.Fatal(err)
	}
	if witnessed {
		rawID, _ := storagegc.CaptureIdentity(files.Raw)
		eventsID, _ := storagegc.CaptureIdentity(files.Events)
		witness := artifactpath.CaptureWitness{Type: "capture_witness", Version: 2, OpenedAt: time.Now().UTC().Add(-time.Minute), Generation: "generation", ScopeKey: record.Address.RepoScope, Tag: string(record.Address.Tag), Agent: "codex", LaunchOrdinal: 5, Nonce: "producer", RawFile: artifactpath.CaptureFileID{Device: rawID.Device, Inode: rawID.Inode}, EventsFile: artifactpath.CaptureFileID{Device: eventsID.Device, Inode: eventsID.Inode}}
		raw, err := json.Marshal(witness)
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(files.Events, append(raw, '\n'), 0600); err != nil {
			t.Fatal(err)
		}
	}
	// Distinguish original activity from the archive publisher's new clock.
	old := time.Now().UTC().Add(-30 * time.Second)
	if err = os.Chtimes(files.Raw, old, old); err != nil {
		t.Fatal(err)
	}
	return reader, record
}
func TestRecoveryEvidenceReadPreserveReadExactOriginalScreen(t *testing.T) {
	for _, witnessed := range []bool{false, true} {
		t.Run(fmt.Sprint(witnessed), func(t *testing.T) {
			reader, record := recoveryEvidenceFixture(t, witnessed)
			ctx := context.Background()
			first, err := reader.Read(ctx, record)
			if err != nil {
				t.Fatal(err)
			}
			if first.Branch != "000528-recovery" || first.Issue != "#528" {
				t.Fatalf("git context lost: %+v", first)
			}
			if !witnessed && (len(first.Ranked) != 0 || len(first.Reasons) == 0) {
				t.Fatal("legacy capture inferred native authority")
			}
			archived, err := reader.Preserve(ctx, record, first)
			if err != nil {
				t.Fatal(err)
			}
			if archived.Capture.Raw == first.Capture.Raw || archived.Capture.Token == "" || !archived.Capture.LastActivity.Equal(first.Capture.LastActivity) || !archived.Capture.CapturedAt.After(first.Capture.CapturedAt) {
				t.Fatalf("original activity or archive identity lost: %+v", archived.Capture)
			}
			c := &threadrecord.RecoveryContext{Capture: archived.Capture, Branch: "original-branch", Issue: "original-issue"}
			record.Incarnations = []ThreadIncarnation{{Start: &ThreadStartClaim{RecoveryIntent: &threadrecord.RecoveryIntent{Context: *c}}}}
			record.RecoveryContext = &threadrecord.RecoveryContext{Branch: "wrong later branch", Capture: archived.Capture}
			replay, err := reader.Read(ctx, record)
			if err != nil {
				t.Fatal(err)
			}
			if replay.Generation != archived.Generation || replay.Capture != archived.Capture || replay.Branch != c.Branch || replay.Issue != c.Issue {
				t.Fatalf("recovery snapshot drifted: %+v / %+v", replay, archived)
			}
			again, err := reader.Preserve(ctx, record, replay)
			if err != nil || again.Capture != archived.Capture {
				t.Fatalf("archive not reused: %+v %v", again, err)
			}
		})
	}
}
func TestRecoveryPreserveRejectsSameSizeChangedArchive(t *testing.T) {
	reader, record := recoveryEvidenceFixture(t, false)
	ctx := context.Background()
	in, err := reader.Read(ctx, record)
	if err != nil {
		t.Fatal(err)
	}
	reader.parkScrollback = func(ctx context.Context, dir, tag, agent string) (string, bool) {
		base, ok := (launcher.OSRuntime{DataDir: dir}).ParkScrollbackContext(ctx, tag, agent, false)
		if !ok {
			return base, ok
		}
		owner, e := storagegc.SelectedOwner(dir, record.Address.RepoScope, tag)
		if e != nil {
			t.Fatal(e)
		}
		member, e := artifactpath.MatchArtifact(base, []artifactpath.StorageOwner{owner}, nil)
		if e != nil {
			t.Fatal(e)
		}
		capture, e := artifactpath.ParseParkedCapture(member, time.UTC)
		if e != nil {
			t.Fatal(e)
		}
		identity, e := storagegc.CaptureIdentity(capture.Raw)
		if e != nil {
			t.Fatal(e)
		}
		if e = os.WriteFile(capture.Raw, bytes.Repeat([]byte("x"), int(identity.Size)), 0600); e != nil {
			t.Fatal(e)
		}
		if e = os.Chtimes(capture.Raw, identity.ModifiedAt, identity.ModifiedAt); e != nil {
			t.Fatal(e)
		}
		return base, true
	}
	if _, err = reader.Preserve(ctx, record, in); err == nil || !strings.Contains(err.Error(), "bytes differ") {
		t.Fatalf("accepted same-size altered copy: %v", err)
	}
}
func TestRecoveryReadRejectsWrongWitnessFileAndForeignEvents(t *testing.T) {
	for _, foreign := range []bool{false, true} {
		reader, record := recoveryEvidenceFixture(t, true)
		paths, _ := artifactpath.Resolve(artifactpath.Address{DataDir: reader.DataDir, RepoScope: record.Address.RepoScope, Tag: string(record.Address.Tag)})
		files, _ := paths.ScrollbackArtifacts("codex")
		if foreign {
			rawID, _ := storagegc.CaptureIdentity(files.Raw)
			eventsID, _ := storagegc.CaptureIdentity(files.Events)
			record.RecoveryContext = &threadrecord.RecoveryContext{Capture: threadrecord.RecoveryCapture{Raw: files.Raw, Events: filepath.Join(t.TempDir(), "events"), RawIdentity: rawID, EventsIdentity: eventsID}}
		} else {
			raw, err := os.ReadFile(files.Events)
			if err != nil {
				t.Fatal(err)
			}
			w, err := artifactpath.DecodeCaptureWitness(raw)
			if err != nil {
				t.Fatal(err)
			}
			w.RawFile.Inode++
			raw, _ = json.Marshal(w)
			if err = os.WriteFile(files.Events, append(raw, '\n'), 0600); err != nil {
				t.Fatal(err)
			}
		}
		got, err := reader.Read(context.Background(), record)
		if err != nil || got.Screen.Availability != ScreenUnavailable || got.Source == nil || got.Source.LaunchHint != nil || len(got.Ranked) != 0 {
			t.Fatalf("mismatched screen gained source authority or hid native scan: %+v %v", got, err)
		}
	}
}

func TestRecoveryOldArchiveDoesNotInventActivityFromCopyClock(t *testing.T) {
	reader, record := recoveryEvidenceFixture(t, true)
	first, err := reader.Read(context.Background(), record)
	if err != nil {
		t.Fatal(err)
	}
	archived, err := reader.Preserve(context.Background(), record, first)
	if err != nil {
		t.Fatal(err)
	}
	archived.Capture.LastActivity = time.Time{}
	record.RecoveryContext = &threadrecord.RecoveryContext{Capture: archived.Capture}
	got, err := reader.Read(context.Background(), record)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Capture.LastActivity.IsZero() || len(got.Ranked) != 0 || len(got.Reasons) == 0 || !strings.Contains(got.Reasons[0], "activity time is unavailable") {
		t.Fatalf("invented old archive activity: %+v", got)
	}
}

func TestRecoveryEvidenceCodexHomeDiscoveryAndValidation(t *testing.T) {
	for _, mode := range []string{"custom", "default", "invalid", "relative"} {
		t.Run(mode, func(t *testing.T) {
			reader, record := recoveryEvidenceFixture(t, true)
			record.WorkingPath, _ = filepath.EvalSymlinks(record.WorkingPath)
			t.Setenv("HOME", t.TempDir()) // Reader's explicit HomeDir must win for default roots.
			nativeHome := filepath.Join(reader.HomeDir, ".codex")
			if mode == "custom" {
				nativeHome = filepath.Join(t.TempDir(), "custom")
			}
			if err := os.MkdirAll(filepath.Join(nativeHome, "sessions", "2026", "10", "15"), 0700); err != nil {
				t.Fatal(err)
			}
			value := ""
			switch mode {
			case "custom":
				value = nativeHome
			case "invalid":
				value = filepath.Join(t.TempDir(), "missing")
			case "relative":
				value = ".codex"
			}
			t.Setenv("CODEX_HOME", value)
			const id = "019d1111-1111-7111-8111-111111111111"
			stamp := time.Now().UTC().Add(-45 * time.Second).Format(time.RFC3339Nano)
			raw := fmt.Sprintf(`{"timestamp":%q,"type":"session_meta","payload":{"id":%q,"source":"cli","cwd":%q,"originator":"codex-tui"}}
`, stamp, id, record.WorkingPath)
			if err := os.WriteFile(filepath.Join(nativeHome, "sessions", "2026", "10", "15", "rollout-2026-10-15T00-00-00-"+id+".jsonl"), []byte(raw), 0600); err != nil {
				t.Fatal(err)
			}
			got, err := reader.Read(context.Background(), record)
			if err != nil {
				t.Fatal(err)
			}
			if mode == "invalid" || mode == "relative" {
				if !got.Incomplete || len(got.Ranked) != 0 || len(got.Reasons) == 0 || got.Capture.Raw == "" {
					t.Fatalf("unknown home lost screen or inferred native authority: %+v", got)
				}
				return
			}
			if got.Incomplete || len(got.Ranked) != 1 {
				t.Fatalf("native home discovery=%+v", got)
			}
			candidate := got.Ranked[0].Candidate
			preserved, err := reader.Preserve(context.Background(), record, got)
			if err != nil {
				t.Fatal(err)
			}
			// Independently constructed validation runtime must resolve the same source.
			runtime, err := sessioninventory.NewNativeTranscriptRuntime(reader.HomeDir, reader.DataDir, sessioninventory.AgentCodex, "", os.Getenv)
			if err != nil {
				t.Fatal(err)
			}
			if err := sessioninventory.RevalidateRecoveryCandidate(context.Background(), runtime, candidate.Entry, sessioninventory.AgentCodex, id, record.WorkingPath, preserved.Capture.LastActivity); err != nil {
				t.Fatalf("discovered candidate could not validate: %v", err)
			}
		})
	}
}
