package remotesession

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
)

func TestRemotePublicationRecoveryReclaimsOnlyOwnedStages(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
	paths, _ := artifactpath.ResolveCouchRemote(root)
	latch, err := OpenStore(paths.Latch())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := latch.Arm(); err != nil {
		t.Fatal(err)
	}
	origins, err := OpenOriginStore(paths.Origins())
	if err != nil {
		t.Fatal(err)
	}
	if err := origins.MarkRemote(originDocA); err != nil {
		t.Fatal(err)
	}
	audit, err := OpenAudit(paths.Audit(), paths.PreviousAudit(), func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	if err := audit.AppendIntent(auditFixture(now)); err != nil {
		t.Fatal(err)
	}
	records := []string{paths.Latch(), paths.Origins(), paths.Audit(), paths.PreviousAudit()}
	// Crash-left staging bytes are not committed authority, even if a complete
	// candidate claims off. Repeated interruptions reuse these exact four slots.
	for cycle := 0; cycle < 16; cycle++ {
		for _, path := range records {
			stage, err := artifactpath.RemotePublicationStage(path)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(stage, []byte(`{"version":1,"phase":"off","generation":9}`), 0600); err != nil {
				t.Fatal(err)
			}
		}
	}
	foreign := filepath.Join(filepath.Dir(paths.Latch()), "foreign-stage")
	if err := os.WriteFile(foreign, []byte("retain"), 0600); err != nil {
		t.Fatal(err)
	}
	reopened, err := OpenStore(paths.Latch())
	if err != nil {
		t.Fatal(err)
	}
	if phase, _ := reopened.Snapshot(); phase != latchKilled {
		t.Fatal("uncommitted stage became authority")
	}
	if _, err := OpenOriginStore(paths.Origins()); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenAudit(paths.Audit(), paths.PreviousAudit(), func() time.Time { return now }); err != nil {
		t.Fatal(err)
	}
	for _, path := range records {
		stage, _ := artifactpath.RemotePublicationStage(path)
		if _, err := os.Lstat(stage); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("owned stage retained: %s %v", stage, err)
		}
	}
	if raw, err := os.ReadFile(foreign); err != nil || string(raw) != "retain" {
		t.Fatal("foreign path touched")
	}
}
func TestRemoteStageRecoveryRefusesUnsafeOrOversizedSlot(t *testing.T) {
	for _, kind := range []string{"symlink", "directory", "oversized"} {
		t.Run(kind, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "latch")
			stage, err := artifactpath.RemotePublicationStage(path)
			if err != nil {
				t.Fatal(err)
			}
			foreign := filepath.Join(filepath.Dir(path), "foreign")
			if err := os.WriteFile(foreign, []byte("retain"), 0600); err != nil {
				t.Fatal(err)
			}
			switch kind {
			case "symlink":
				err = os.Symlink(foreign, stage)
			case "directory":
				err = os.Mkdir(stage, 0700)
			case "oversized":
				err = os.WriteFile(stage, []byte(strings.Repeat("x", maxLatchBytes+1)), 0600)
			}
			if err != nil {
				t.Fatal(err)
			}
			if _, err := OpenStore(path); err == nil {
				t.Fatal("unsafe stage repaired silently")
			}
			if _, err := os.Lstat(stage); err != nil {
				t.Fatal("unsafe stage erased", err)
			}
			if raw, err := os.ReadFile(foreign); err != nil || !bytes.Equal(raw, []byte("retain")) {
				t.Fatal("foreign file changed")
			}
		})
	}
}

func TestRemoteAllStageOwnersRefuseUnsafeSlots(t *testing.T) {
	for _, family := range []string{"latch", "origins", "audit", "previous"} {
		for _, kind := range []string{"symlink", "directory", "oversized"} {
			t.Run(family+"/"+kind, func(t *testing.T) {
				paths, err := artifactpath.ResolveCouchRemote(t.TempDir())
				if err != nil {
					t.Fatal(err)
				}
				path, limit := paths.Latch(), maxLatchBytes
				switch family {
				case "origins":
					path, limit = paths.Origins(), maxOriginsBytes
				case "audit":
					path, limit = paths.Audit(), maxAuditSegmentBytes
				case "previous":
					path, limit = paths.PreviousAudit(), maxAuditSegmentBytes
				}
				if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
					t.Fatal(err)
				}
				stage, err := artifactpath.RemotePublicationStage(path)
				if err != nil {
					t.Fatal(err)
				}
				switch kind {
				case "symlink":
					err = os.Symlink(filepath.Join(filepath.Dir(path), "foreign"), stage)
				case "directory":
					err = os.Mkdir(stage, 0700)
				case "oversized":
					var f *os.File
					f, err = os.Create(stage)
					if err == nil {
						err = f.Truncate(int64(limit) + 1)
						f.Close()
					}
				}
				if err != nil {
					t.Fatal(err)
				}
				switch family {
				case "latch":
					_, err = OpenStore(path)
				case "origins":
					_, err = OpenOriginStore(path)
				default:
					_, err = OpenAudit(paths.Audit(), paths.PreviousAudit(), time.Now)
				}
				if err == nil {
					t.Fatal("unsafe stage accepted")
				}
				if _, err := os.Lstat(stage); err != nil {
					t.Fatal("unsafe slot removed", err)
				}
			})
		}
	}
}
