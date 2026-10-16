package launcher

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
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
		for _, shape := range []string{"typed-only", "compatibility-only", "mixed"} {
			for _, otherAgents := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/%s/others=%v", kind, shape, otherAgents), func(t *testing.T) {
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
					agents := []string{"codex"}
					if otherAgents {
						agents = append(agents, "claude", "qoder")
					}
					ids := []string{customHomeNativeID, "019d1111-1111-7111-8111-000000000002", "019d1111-1111-7111-8111-000000000003"}
					var ledger bytes.Buffer
					ordinal := uint64(1)
					for index, agent := range agents {
						if agent == "codex" && shape == "compatibility-only" {
							continue
						}
						for _, record := range []sessionledger.Record{
							{Version: 3, Kind: sessionledger.RecordLaunch, ScopeKey: "scope", Tag: "work", Agent: agent, RequestedNativeID: ids[index], RequestOrigin: sessionledger.RequestOriginChosen, BaselineComplete: true},
							{Version: 1, Kind: sessionledger.RecordBinding, ScopeKey: "scope", Tag: "work", Agent: agent, LaunchOrdinal: ordinal, RootNativeID: ids[index]},
						} {
							raw, err := sessionledger.EncodeRecord(record)
							if err != nil {
								t.Fatal(err)
							}
							ledger.Write(raw)
							ledger.WriteByte('\n')
						}
						ordinal += 2
					}
					if shape != "typed-only" {
						compat, err := BuildLedgerLine(LedgerEntry{Agent: "codex", Args: []string{}, SessionID: "legacy-codex", RepoRoot: cwd, RepoName: "work"})
						if err != nil {
							t.Fatal(err)
						}
						ledger.WriteString(compat + "\n")
					}
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
					wantEntries := len(agents)
					if shape == "mixed" {
						wantEntries++
					}
					if len(entries) != wantEntries {
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

// Available namespaces retain the compatibility projection; legacy IDs do not
// become typed saved-config authority merely because their home is readable.
func TestNativeInventoryAvailableCompatibilityDoesNotGainAuthority(t *testing.T) {
	for _, kind := range []string{"custom", "default"} {
		t.Run(kind, func(t *testing.T) {
			runtime, home, cwd := nativeInventoryFixture(t)
			if kind == "default" {
				t.Setenv("CODEX_HOME", "")
				dir := filepath.Join(home, ".codex", "sessions", "2026", "08", "28")
				if err := os.MkdirAll(dir, 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(dir, "rollout-test-"+customHomeNativeID+".jsonl"), []byte(nativeValidCodexSessionMeta(customHomeNativeID, cwd)), 0600); err != nil {
					t.Fatal(err)
				}
			}
			legacy := LedgerEntry{Agent: "codex", Args: []string{"--no-alt-screen"}, SessionID: customHomeNativeID, RepoRoot: cwd, RepoName: "work"}
			line, err := BuildLedgerLine(legacy)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(runtime.DataDir, "ledger-work.jsonl"), []byte(line+"\n"), 0600); err != nil {
				t.Fatal(err)
			}
			entries, err := runtime.ReadLedger("work")
			if err != nil || len(entries) != 1 {
				t.Fatalf("available compatibility ledger: %+v %v", entries, err)
			}
			if entries[0].Typed || entries[0].ResumeBlocked || entries[0].SessionID != customHomeNativeID || len(entries[0].Args) != 1 || entries[0].Args[0] != "--no-alt-screen" {
				t.Fatalf("available compatibility semantics changed: %+v", entries[0])
			}
			configPath := filepath.Join(runtime.DataDir, "config-work-codex.json")
			config, err := json.Marshal(savedConfig{Agent: "codex", Args: legacy.Args, SessionID: customHomeNativeID})
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(configPath, config, 0600); err != nil {
				t.Fatal(err)
			}
			saved, warnings, err := readSavedConfigForTag(runtime, configPath, "scope", "work", "codex")
			if err != nil || saved.Agent != "codex" || saved.SessionID != "" || len(warnings) != 1 {
				t.Fatalf("legacy ID gained saved authority: %+v warnings=%v err=%v", saved, warnings, err)
			}
			if _, err := os.Stat(configPath); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("baseline fresh cleanup changed: %v", err)
			}
		})
	}
}

func TestNativeInventoryLedgerWithoutCodexRemainsUnchanged(t *testing.T) {
	for _, kind := range []string{"empty", "compatibility", "typed"} {
		t.Run(kind, func(t *testing.T) {
			runtime, home, cwd := nativeInventoryFixture(t)
			t.Setenv("CODEX_HOME", filepath.Join(home, "missing"))
			var raw string
			switch kind {
			case "compatibility":
				line, err := BuildLedgerLine(LedgerEntry{Agent: "claude", Args: []string{}, SessionID: customHomeNativeID, RepoRoot: cwd, RepoName: "work"})
				if err != nil {
					t.Fatal(err)
				}
				raw = line + "\n"
			case "typed":
				nativeInventoryLedger(t, runtime, "claude", true)
			}
			if kind != "typed" {
				if err := os.WriteFile(filepath.Join(runtime.DataDir, "ledger-work.jsonl"), []byte(raw), 0600); err != nil {
					t.Fatal(err)
				}
			}
			entries, err := runtime.ReadLedger("work")
			if err != nil {
				t.Fatal(err)
			}
			if kind == "empty" {
				if len(entries) != 0 {
					t.Fatalf("empty ledger invented state: %+v", entries)
				}
				return
			}
			if len(entries) != 1 || entries[0].Agent != "claude" || entries[0].ResumeBlocked || entries[0].SessionID != customHomeNativeID {
				t.Fatalf("unrepresented Codex affected ledger: %+v", entries)
			}
		})
	}
}
