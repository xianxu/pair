package couchcore

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/storagegc"
	"github.com/xianxu/pair/cmd/internal/threadrecord"
)

func nativeRecoveryOSFixture(t *testing.T) (OSRecoveryEvidenceReader, ThreadRecord, artifactpath.Paths, int64) {
	t.Helper()
	r, record := recoveryEvidenceFixture(t, true)
	var err error
	record.WorkingPath, err = filepath.EvalSymlinks(record.WorkingPath)
	if err != nil {
		t.Fatal(err)
	}
	paths, err := r.recoveryPaths(record)
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(r.HomeDir, ".codex", "sessions", "2026", "10", "20")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	const id = "019d1111-1111-7111-8111-111111111111"
	stamp := time.Now().UTC().Add(-45 * time.Second).Format(time.RFC3339Nano)
	raw := fmt.Sprintf(`{"timestamp":%q,"type":"session_meta","payload":{"id":%q,"source":"cli","cwd":%q}}
{"timestamp":%q,"type":"response_item","payload":{"type":"message","role":"assistant","content":[{"type":"output_text","text":"retained native working context"}]}}
`, stamp, id, record.WorkingPath, stamp)
	if err := os.WriteFile(filepath.Join(dir, "rollout-2026-10-20T00-00-00-"+id+".jsonl"), []byte(raw), 0600); err != nil {
		t.Fatal(err)
	}
	return r, record, paths, int64(len(raw))
}

func TestRecoveryOversizedReplayDoesNotConsumeNativeBudget(t *testing.T) {
	r, record, paths, nativeSize := nativeRecoveryOSFixture(t)
	files, err := paths.ScrollbackArtifacts("codex")
	if err != nil {
		t.Fatal(err)
	}
	events, err := os.Stat(files.Events)
	if err != nil {
		t.Fatal(err)
	}
	const rawSize = int64(200 << 20) // Replay refuses this before opening raw bytes.
	if err := os.Truncate(files.Raw, rawSize); err != nil {
		t.Fatal(err)
	}
	consumed := sessioninventory.RecoveryTotalReadLimit - rawSize - 2*events.Size() - nativeSize/2
	got, err := r.ReadRequest(context.Background(), record, RecoveryObservationRequest{ObservedAt: time.Now().UTC(), BytesAlreadyRead: consumed})
	if err != nil || got.Incomplete || len(got.Ranked) != 1 || got.BytesRead != consumed+nativeSize {
		t.Fatalf("unread oversized raw charged to native scan: %+v err=%v native bytes=%d", got, err, nativeSize)
	}
	if got.Screen.Availability != ScreenUnavailable || !strings.Contains(got.Screen.Reason, "replay") {
		t.Fatalf("missing replay-bound diagnosis: %+v", got.Screen)
	}
}

func TestRecoveryEarlyScreenFailureRetainsRawFileActivity(t *testing.T) {
	for _, kind := range []string{"missing-events", "retained-device", "archive-metadata", "archive-device"} {
		t.Run(kind, func(t *testing.T) {
			r, record, paths, _ := nativeRecoveryOSFixture(t)
			files, err := paths.ScrollbackArtifacts("codex")
			if err != nil {
				t.Fatal(err)
			}
			rawPath := files.Raw
			switch kind {
			case "missing-events":
				if err := os.Remove(files.Events); err != nil {
					t.Fatal(err)
				}
			case "retained-device":
				rawID, err := storagegc.CaptureIdentity(files.Raw)
				if err != nil {
					t.Fatal(err)
				}
				eventID, err := storagegc.CaptureIdentity(files.Events)
				if err != nil {
					t.Fatal(err)
				}
				rawID.Device++
				record.RecoveryContext = &threadrecord.RecoveryContext{Capture: threadrecord.RecoveryCapture{Raw: files.Raw, Events: files.Events, RawIdentity: rawID, EventsIdentity: eventID, CapturedAt: rawID.ModifiedAt}}
			default:
				before, err := r.Read(context.Background(), record)
				if err != nil {
					t.Fatal(err)
				}
				archived, err := r.Preserve(context.Background(), record, before)
				if err != nil {
					t.Fatal(err)
				}
				record.RecoveryContext = &threadrecord.RecoveryContext{Capture: archived.Capture}
				rawPath = archived.Capture.Raw
				if kind == "archive-device" {
					record.RecoveryContext.Capture.RawIdentity.Device++
				} else {
					archive, err := paths.ParkedScrollbackArtifacts(archived.Capture.Token)
					if err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(archive.Metadata, []byte("{invalid"), 0600); err != nil {
						t.Fatal(err)
					}
				}
			}
			stat, err := os.Stat(rawPath)
			if err != nil {
				t.Fatal(err)
			}
			got, err := r.Read(context.Background(), record)
			if err != nil || got.Incomplete || len(got.Ranked) != 1 || got.Source == nil || got.Screen.Availability != ScreenUnavailable || got.Screen.Reason == "" || !got.Screen.LastActivity.Equal(stat.ModTime()) || got.Screen.ActivitySource != "capture-file activity" {
				t.Fatalf("early failure lost descriptive raw activity: %+v %v", got, err)
			}
		})
	}
}
