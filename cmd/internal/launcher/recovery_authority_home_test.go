package launcher

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/procutil"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/storagegc"
	"github.com/xianxu/pair/cmd/internal/threadrecord"
)

func TestRecoveryAuthorityNativeHomeDiscoveryToFinalValidation(t *testing.T) {
	for _, mode := range []string{"custom", "default", "invalid", "relative"} {
		t.Run(mode, func(t *testing.T) {
			dir, err := filepath.EvalSymlinks(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			home := filepath.Join(dir, "home")
			t.Setenv("HOME", home)
			nativeHome := filepath.Join(home, ".codex")
			if mode != "default" {
				nativeHome = filepath.Join(home, "custom")
			}
			t.Setenv("CODEX_HOME", "")
			if mode != "default" {
				t.Setenv("CODEX_HOME", nativeHome)
			}
			if err := os.MkdirAll(filepath.Join(nativeHome, "sessions", "2026", "10", "15"), 0700); err != nil {
				t.Fatal(err)
			}
			const id = "019d1111-1111-7111-8111-111111111111"
			at := time.Now().UTC().Add(-time.Minute)
			raw := fmt.Sprintf(`{"timestamp":%q,"type":"session_meta","payload":{"id":%q,"source":"cli","cwd":%q,"originator":"codex-tui"}}
`, at.Format(time.RFC3339Nano), id, dir)
			if err := os.WriteFile(filepath.Join(nativeHome, "sessions", "2026", "10", "15", "rollout-test-"+id+".jsonl"), []byte(raw), 0600); err != nil {
				t.Fatal(err)
			}
			native, err := sessioninventory.NewNativeTranscriptRuntime(home, dir, sessioninventory.AgentCodex, "", os.Getenv)
			if err != nil {
				t.Fatal(err)
			}
			captured := time.Now().UTC()
			produced := at.Add(-time.Minute)
			evidence := sessioninventory.CollectRecoveryEvidence(context.Background(), native, sessioninventory.RecoveryCollectRequest{Agent: sessioninventory.AgentCodex, TargetCWD: dir, ProducingLaunch: &produced, CapturedAt: &captured})
			if evidence.Incomplete || len(evidence.Ranked) != 1 {
				t.Fatalf("discovery=%+v", evidence)
			}
			candidate := evidence.Ranked[0].Candidate
			a := testRecoveryAuthority()
			scope, err := ResolveRepoScope(dir)
			if err != nil {
				t.Fatal(err)
			}
			a.Address.RepoScope = scope.Key
			a.RecordPath = filepath.Join(dir, "thread.json")
			a.Intent.NativeID = id
			c := &a.Intent.Context
			c.Address = a.Address
			c.CreatedAt = captured
			c.Capture.CapturedAt = captured
			c.Capture.LastActivity = captured
			c.ExpiresAt = captured.Add(storagegc.CaptureRetentionPeriod)
			paths, err := artifactpath.ResolveScoped(dir, "work")
			if err != nil {
				t.Fatal(err)
			}
			c.Capture.Token = "20261015T120000"
			archived, err := paths.ParkedScrollbackArtifacts(c.Capture.Token)
			if err != nil {
				t.Fatal(err)
			}
			c.Capture.Raw, c.Capture.Events = archived.Raw, archived.Events
			for _, path := range []string{archived.Raw, archived.Events} {
				if err := os.WriteFile(path, []byte("capture"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			c.Capture.RawIdentity, err = storagegc.CaptureIdentity(archived.Raw)
			if err != nil {
				t.Fatal(err)
			}
			c.Capture.EventsIdentity, err = storagegc.CaptureIdentity(archived.Events)
			if err != nil {
				t.Fatal(err)
			}
			metadata, err := json.Marshal(artifactpath.CaptureMetadata{Version: 1, Raw: filepath.Base(archived.Raw), Events: filepath.Base(archived.Events), CapturedAt: captured, RawIdentity: c.Capture.RawIdentity, EventsIdentity: &c.Capture.EventsIdentity})
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(archived.Metadata, metadata, 0600); err != nil {
				t.Fatal(err)
			}
			e := candidate.Entry
			c.Candidates = []threadrecord.RecoveryCandidate{{Key: "native-key", Agent: "codex", NativeID: id, ArtifactRoot: e.Artifact.StorageRoot, ArtifactRelativePath: e.Artifact.RelativePath, StableFileID: string(e.StableFileID), GenerationToken: string(e.GenerationToken), MutationToken: string(e.MutationToken), Size: e.Size, ModTime: *e.ModTime, LastActivity: candidate.CreatedAt, Eligible: true}}
			record := recoveryAuthorityRecord(a, dir, "codex", os.Getpid(), procutil.Identity(strconv.Itoa(os.Getpid())))
			encoded, err := json.Marshal(record)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(a.RecordPath, encoded, 0600); err != nil {
				t.Fatal(err)
			}
			if mode == "invalid" {
				t.Setenv("CODEX_HOME", filepath.Join(home, "missing"))
			}
			if mode == "relative" {
				t.Setenv("CODEX_HOME", filepath.Join("home", "custom"))
			}
			err = (OSRuntime{DataDir: dir}).ValidateRecoveryAuthority(context.Background(), a, scope, "work", "codex", dir)
			allowed := mode == "custom" || mode == "default"
			if (err == nil) != allowed {
				t.Fatalf("authority=%v", err)
			}
		})
	}
}
