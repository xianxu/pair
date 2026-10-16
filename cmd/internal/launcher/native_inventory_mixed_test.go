package launcher

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/xianxu/pair/cmd/internal/sessionledger"
)

type nativeLedgerRestartRuntime struct {
	*fakeRuntime
	ledger OSRuntime
}

func (r nativeLedgerRestartRuntime) ReadLedger(tag string) ([]LedgerEntry, error) {
	return r.ledger.ReadLedger(tag)
}

func TestNativeInventoryMixedLedgerIsolatesUnavailableCodexHome(t *testing.T) {
	for _, kind := range []string{"invalid", "relative"} {
		t.Run(kind, func(t *testing.T) {
			runtime, home, cwd := nativeInventoryFixture(t)
			// A valid default tree must never rescue an invalid explicit namespace.
			defaultDir := filepath.Join(home, ".codex", "sessions", "2026", "08", "28")
			if err := os.MkdirAll(defaultDir, 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(defaultDir, "rollout-test-"+customHomeNativeID+".jsonl"), []byte(nativeValidCodexSessionMeta(customHomeNativeID, cwd)), 0600); err != nil {
				t.Fatal(err)
			}
			t.Chdir(home)
			if kind == "invalid" {
				t.Setenv("CODEX_HOME", filepath.Join(home, "missing"))
			} else {
				t.Setenv("CODEX_HOME", "custom")
			}
			agents := []string{"codex", "claude", "qoder"}
			ids := []string{customHomeNativeID, "019d1111-1111-7111-8111-000000000002", "019d1111-1111-7111-8111-000000000003"}
			var ledger bytes.Buffer
			for index, agent := range agents {
				for _, record := range []sessionledger.Record{
					{Version: 3, Kind: sessionledger.RecordLaunch, ScopeKey: "scope", Tag: "work", Agent: agent, RequestedNativeID: ids[index], RequestOrigin: sessionledger.RequestOriginChosen, BaselineComplete: true},
					{Version: 1, Kind: sessionledger.RecordBinding, ScopeKey: "scope", Tag: "work", Agent: agent, LaunchOrdinal: uint64(2*index + 1), RootNativeID: ids[index]},
				} {
					raw, err := sessionledger.EncodeRecord(record)
					if err != nil {
						t.Fatal(err)
					}
					ledger.Write(raw)
					ledger.WriteByte('\n')
				}
			}
			compat, err := BuildLedgerLine(LedgerEntry{Agent: "codex", Args: []string{}, SessionID: "legacy-codex", RepoRoot: cwd, RepoName: "work"})
			if err != nil {
				t.Fatal(err)
			}
			ledger.WriteString(compat + "\n")
			if err := os.WriteFile(filepath.Join(runtime.DataDir, "ledger-work.jsonl"), ledger.Bytes(), 0600); err != nil {
				t.Fatal(err)
			}
			for index, agent := range agents {
				t.Run(agent, func(t *testing.T) {
					configPath := filepath.Join(runtime.DataDir, "config-work-"+agent+".json")
					config, err := json.Marshal(savedConfig{Agent: agent, Args: []string{}, SessionID: ids[index]})
					if err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(configPath, config, 0600); err != nil {
						t.Fatal(err)
					}
					saved, _, err := readSavedConfigForTag(runtime, configPath, "scope", "work", agent)
					if agent == "codex" {
						if err == nil || saved.SessionID != "" {
							t.Errorf("unavailable Codex gained saved authority: %+v %v", saved, err)
						}
					} else if err != nil || saved.Agent != agent || saved.SessionID != ids[index] {
						t.Errorf("Codex home blocked unrelated saved configuration: agent=%s saved=%+v err=%v", agent, saved, err)
					}
					if actual, err := os.ReadFile(configPath); err != nil || !bytes.Equal(actual, config) {
						t.Errorf("unavailable namespace removed or changed saved config: %v", err)
					}
					fake := newFakeRuntime()
					fake.inferAgent["work"] = agent
					restart := nativeLedgerRestartRuntime{fakeRuntime: fake, ledger: runtime}
					var stderr bytes.Buffer
					code := runRestart(restart, LaunchArgs{}, "📁work", "work", false, &stderr)
					if agent == "codex" {
						if code != 1 || len(fake.writtenMarkers) != 0 || len(fake.killed) != 0 {
							t.Errorf("unavailable Codex restarted: code=%d markers=%v killed=%v", code, fake.writtenMarkers, fake.killed)
						}
					} else if code != 0 || len(fake.killed) != 1 || fake.writtenMarkers["📁work"].SessionID != ids[index] {
						t.Errorf("Codex home blocked unrelated restart: agent=%s code=%d markers=%v killed=%v stderr=%s", agent, code, fake.writtenMarkers, fake.killed, &stderr)
					}
				})
			}
			entries, err := runtime.ReadLedger("work")
			if err != nil {
				t.Errorf("one unavailable agent discarded mixed ledger: %v", err)
				return
			}
			if len(entries) != 4 {
				t.Fatalf("mixed ledger entries lost: %+v", entries)
			}
			for _, entry := range entries {
				if entry.Agent == "codex" && (entry.SessionID != "" || !entry.ResumeBlocked) {
					t.Errorf("unavailable bound/compat Codex entry gained authority: %+v", entry)
				}
			}
		})
	}
}
func TestNativeInventoryReadLedgerPreservesStorageErrors(t *testing.T) {
	runtime := OSRuntime{DataDir: t.TempDir()}
	if _, err := runtime.ReadLedger("work"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("missing ledger error changed: %v", err)
	}
	if err := os.Mkdir(filepath.Join(runtime.DataDir, "ledger-work.jsonl"), 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := runtime.ReadLedger("work"); err == nil {
		t.Fatal("unreadable ledger reported as available")
	}
}
