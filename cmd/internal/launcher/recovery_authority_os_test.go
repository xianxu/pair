package launcher

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/procutil"
	"github.com/xianxu/pair/cmd/internal/storagegc"
	"github.com/xianxu/pair/cmd/internal/threadrecord"
	"golang.org/x/sys/unix"
)

func TestOSRecoveryAuthorityFreshChecksDurableClaimAndCapture(t *testing.T) {
	for _, scenario := range []string{"valid", "capture changed", "record mismatch", "expired", "fifo record", "cancelled"} {
		t.Run(scenario, func(t *testing.T) {
			dir := t.TempDir()
			physical, err := filepath.EvalSymlinks(dir)
			if err != nil {
				t.Fatal(err)
			}
			dir = physical
			a := testRecoveryAuthority()
			scope, _ := ResolveRepoScope(dir)
			a.Address.RepoScope = scope.Key
			a.Intent.Context.Address = a.Address
			a.Intent.NativeID = ""
			a.Intent.SelectedKey = "fresh-key"
			a.RecordPath = filepath.Join(dir, "thread.json")
			c := &a.Intent.Context
			c.CreatedAt = time.Now().UTC().Add(-time.Minute)
			c.Capture.CapturedAt = c.CreatedAt
			c.ExpiresAt = c.Capture.CapturedAt.Add(storagegc.CaptureRetentionPeriod)
			paths, _ := artifactpath.ResolveScoped(dir, "work")
			c.Capture.Token = "20261013T120000"
			archived, err := paths.ParkedScrollbackArtifacts(c.Capture.Token)
			if err != nil {
				t.Fatal(err)
			}
			c.Capture.Raw = archived.Raw
			c.Capture.Events = archived.Events
			for _, p := range []string{c.Capture.Raw, c.Capture.Events} {
				if err := os.WriteFile(p, []byte("capture"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			c.Capture.RawIdentity, err = storagegc.CaptureIdentity(c.Capture.Raw)
			if err != nil {
				t.Fatal(err)
			}
			c.Capture.EventsIdentity, err = storagegc.CaptureIdentity(c.Capture.Events)
			if err != nil {
				t.Fatal(err)
			}
			if scenario == "expired" {
				c.CreatedAt = time.Now().UTC().Add(-2 * storagegc.CaptureRetentionPeriod)
				c.Capture.CapturedAt = c.CreatedAt
				c.ExpiresAt = c.CreatedAt.Add(storagegc.CaptureRetentionPeriod)
			}
			metadata, err := json.Marshal(artifactpath.CaptureMetadata{Version: 1, Raw: filepath.Base(archived.Raw), Events: filepath.Base(archived.Events), CapturedAt: c.Capture.CapturedAt, RawIdentity: c.Capture.RawIdentity, EventsIdentity: &c.Capture.EventsIdentity})
			if err != nil {
				t.Fatal(err)
			}
			if err = os.WriteFile(archived.Metadata, metadata, 0600); err != nil {
				t.Fatal(err)
			}
			record := recoveryAuthorityRecord(a, dir, "codex", os.Getpid(), procutil.Identity(strconv.Itoa(os.Getpid())))
			if scenario == "record mismatch" {
				record.Incarnations[0].Start.RecoveryIntent.SelectedKey = "other-fresh-key"
			}
			raw, err := json.Marshal(record)
			if err != nil {
				t.Fatal(err)
			}
			if scenario == "fifo record" {
				if err = unix.Mkfifo(a.RecordPath, 0600); err != nil {
					t.Fatal(err)
				}
			} else if err = os.WriteFile(a.RecordPath, raw, 0600); err != nil {
				t.Fatal(err)
			}
			if scenario == "capture changed" {
				if err = os.WriteFile(c.Capture.Raw, []byte("different capture"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			if scenario == "cancelled" {
				cancel()
			}
			done := make(chan error, 1)
			go func() {
				done <- (OSRuntime{DataDir: dir}).ValidateRecoveryAuthority(ctx, a, scope, "work", "codex", dir)
			}()
			select {
			case err := <-done:
				if (err == nil) != (scenario == "valid") {
					t.Fatalf("validation: %v", err)
				}
				if scenario == "cancelled" && !errors.Is(err, context.Canceled) {
					t.Fatalf("cancellation: %v", err)
				}
			case <-time.After(2 * time.Second):
				t.Fatal("authority IO blocked")
			}
			if scenario == "valid" {
				after, err := os.ReadFile(a.RecordPath)
				if err != nil || string(after) != string(raw) {
					t.Fatal("authority check mutated record")
				}
			}
		})
	}
}

func TestRecoveryRecordDoesNotTreatUnknownAsReady(t *testing.T) {
	a := testRecoveryAuthority()
	a.Intent.Context.ExpiresAt = a.Intent.Context.Capture.CapturedAt.Add(storagegc.CaptureRetentionPeriod)
	record := recoveryAuthorityRecord(a, "/repo", "codex", 42, "helper")
	record.Incarnations[0].State = "unknown"
	record.RecoveryContext = threadrecord.CloneRecoveryContext(&a.Intent.Context)
	record.Incarnations[0].Start = nil
	if ValidateRecoveryRecord(a, record, RepoScope{Key: a.Address.RepoScope}, "work", "codex", "/repo", 42, "helper", a.Intent.Context.CreatedAt.Add(time.Minute)) == nil {
		t.Fatal("unknown record authorized another launch")
	}
}

func TestOSTaggedFreshAuthorityWithoutScreen(t *testing.T) {
	for _, screen := range []bool{false, true} {
		t.Run(strconv.FormatBool(screen), func(t *testing.T) {
			dir, err := filepath.EvalSymlinks(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			scope, err := ResolveRepoScope(dir)
			if err != nil {
				t.Fatal(err)
			}
			a := taggedAuthority(threadrecord.RecoveryFresh)
			a.Address.RepoScope = scope.Key
			a.Intent.Address = a.Address
			a.Intent.CreatedAt = time.Now().UTC().Add(-time.Minute)
			a.RecordPath = filepath.Join(dir, "thread.json")
			if screen {
				capture := testRecoveryAuthority().Intent.Context.Capture
				a.Intent.Orientation = &threadrecord.RecoveryOrientation{Capture: &capture}
			}
			record := recoveryAuthorityRecord(a, dir, "codex", os.Getpid(), procutil.Identity(strconv.Itoa(os.Getpid())))
			raw, err := json.Marshal(record)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(a.RecordPath, raw, 0600); err != nil {
				t.Fatal(err)
			}
			if err := (OSRuntime{DataDir: dir}).ValidateRecoveryAuthority(context.Background(), a, scope, "work", "codex", dir); err != nil {
				t.Fatalf("screenless Fresh refused: %v", err)
			}
			after, err := os.ReadFile(a.RecordPath)
			if err != nil || string(after) != string(raw) {
				t.Fatal("authority check mutated durable claim")
			}
		})
	}
}

func TestOSTaggedProvenAuthorityRejectsMissingBinding(t *testing.T) {
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", t.TempDir())
	scope, err := ResolveRepoScope(dir)
	if err != nil {
		t.Fatal(err)
	}
	a := taggedAuthority(threadrecord.RecoveryProven)
	a.Address.RepoScope = scope.Key
	a.Intent.Address = a.Address
	a.Intent.CreatedAt = time.Now().UTC().Add(-time.Minute)
	a.RecordPath = filepath.Join(dir, "thread.json")
	record := recoveryAuthorityRecord(a, dir, "codex", os.Getpid(), procutil.Identity(strconv.Itoa(os.Getpid())))
	raw, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(a.RecordPath, raw, 0600); err != nil {
		t.Fatal(err)
	}
	err = (OSRuntime{DataDir: dir}).ValidateRecoveryAuthority(context.Background(), a, scope, "work", "codex", dir)
	if LaunchDiagnosticOf(err) != NativeBindingChanged {
		t.Fatalf("missing native binding not revalidated: %v", err)
	}
}
