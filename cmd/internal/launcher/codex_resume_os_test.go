package launcher

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"

	si "github.com/xianxu/pair/cmd/internal/sessioninventory"
)

func TestCodexResumeOSUsesPrivateAPIOnlyScope(t *testing.T) {
	if _, err := exec.LookPath("sqlite3"); err != nil {
		t.Skip("sqlite3 unavailable")
	}
	python, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("python3 unavailable")
	}
	for _, prefix := range [][]string{nil, {"exec"}} {
		t.Run(fmt.Sprint(prefix), func(t *testing.T) {
			root, err := filepath.EvalSymlinks(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			home, codex, state, cwd := filepath.Join(root, "home"), filepath.Join(root, "native"), filepath.Join(root, "state"), filepath.Join(root, "work")
			sessions := filepath.Join(codex, "sessions", "2026", "10", "15")
			for _, p := range []string{home, codex, state, cwd, sessions, filepath.Join(root, "system"), filepath.Join(root, "writer")} {
				if err := os.MkdirAll(p, 0700); err != nil {
					t.Fatal(err)
				}
			}
			t.Setenv("HOME", home)
			t.Setenv("CODEX_HOME", codex)
			t.Setenv("CODEX_SQLITE_HOME", filepath.Join(root, "wrong-env-state"))
			t.Setenv("CODEX_EXEC_SERVER_URL", "")
			if err := os.WriteFile(filepath.Join(codex, "config.toml"), []byte("forced_login_method='api'\nsqlite_home='../state'\n"), 0600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(codex, "runtime.config.toml"), []byte("[features]\nworktrees=false\n"), 0600); err != nil {
				t.Fatal(err)
			}
			id := "019d1111-1111-7111-8111-111111111111"
			path := filepath.Join(sessions, "rollout-test-"+id+".jsonl")
			raw := nativeValidCodexSessionMeta(id, cwd) + nativeValidCodexTurnContext(cwd)
			if err := os.WriteFile(path, []byte(raw), 0600); err != nil {
				t.Fatal(err)
			}
			db := filepath.Join(state, "state_5.sqlite")
			// Latest admission requires a stable, present WAL. Copy while the
			// private writer is idle and omit SHM, as in the native-mutation test.
			copyCodexWALFixture(t, python, filepath.Join(root, "writer", "state_5.sqlite"), db, id, path, cwd, "")
			reader := si.CodexConfigReader{SystemDir: filepath.Join(root, "system"), ManagedPolicy: func() (si.CodexManagedPolicyStatus, error) { return si.CodexManagedPolicyAbsent, nil }}
			rt := OSRuntime{DataDir: filepath.Join(root, "pair"), codexConfigReader: &reader}
			args := append([]string{"-p", "runtime"}, prefix...)
			args = append(args, "resume", "--last", "continue work")
			selector, err := ParseNativeResumeSelector("codex", args)
			if err != nil {
				t.Fatal(err)
			}
			result, err := rt.ResolveNativeResume(t.Context(), NativeResumeRequest{Agent: "codex", CWD: cwd, Selector: selector})
			if err != nil || result.Status != si.NativeSelectorResolved || result.NativeID != id || result.CodexScope == nil {
				t.Fatalf("resolution=%+v err=%v", result, err)
			}
			if result.CodexScope.CodexHome != codex || result.CodexScope.SQLiteHome != state {
				t.Fatalf("wrong physical namespace: %+v", result.CodexScope)
			}
			if !reflect.DeepEqual(args, selector.Args) {
				t.Fatal("selection changed original args")
			}
			pinned, err := ParseNativeResumeSelector("codex", result.Args)
			if err != nil || pinned.NativeID != id || pinned.Kind != NativeResumeExplicit {
				t.Fatalf("not pinned: %+v %v", pinned, err)
			}
			for _, variant := range []string{"malformed-session-meta", "malformed-turn-context", "mixed-scoped-flags", "absent-wal", "non-wal"} {
				if variant == "malformed-turn-context" && len(prefix) == 0 {
					continue
				}
				t.Run(variant, func(t *testing.T) {
					attemptArgs := append([]string(nil), args...)
					body := fmt.Sprintf("{\"timestamp\":\"2026-10-15T10:00:00Z\",\"type\":\"session_meta\",\"payload\":{\"id\":%q,\"source\":\"cli\",\"cwd\":%q}}\n", id, cwd) + nativeValidCodexTurnContext(cwd)
					if variant == "malformed-turn-context" {
						body = nativeValidCodexSessionMeta(id, cwd) + fmt.Sprintf("{\"timestamp\":\"2026-10-15T10:00:01Z\",\"type\":\"turn_context\",\"payload\":{\"cwd\":%q}}\n", cwd)
					}
					if variant == "mixed-scoped-flags" {
						body = raw
						attemptArgs = append([]string{"-p", "runtime", "--disable", "worktrees"}, prefix...)
						attemptArgs = append(attemptArgs, "resume", "--last", "-c", "features={worktrees=true}", "continue work")
					}
					if variant == "absent-wal" || variant == "non-wal" {
						body = raw
						originalDB, err := os.ReadFile(db)
						if err != nil {
							t.Fatal(err)
						}
						originalWAL, err := os.ReadFile(db + "-wal")
						if err != nil {
							t.Fatal(err)
						}
						t.Cleanup(func() {
							if err := os.WriteFile(db, originalDB, 0600); err != nil {
								t.Error(err)
							}
							if err := os.WriteFile(db+"-wal", originalWAL, 0600); err != nil {
								t.Error(err)
							}
						})
						// The fixture writer has closed and checkpointed its own
						// source. Copy a valid populated main DB, then omit WAL.
						checkpointed, err := os.ReadFile(filepath.Join(root, "writer", "state_5.sqlite"))
						if err != nil {
							t.Fatal(err)
						}
						if err := os.WriteFile(db, checkpointed, 0600); err != nil {
							t.Fatal(err)
						}
						if err := os.Remove(db + "-wal"); err != nil {
							t.Fatal(err)
						}
						if variant == "non-wal" {
							if output, err := exec.Command("sqlite3", db, "PRAGMA journal_mode=DELETE").CombinedOutput(); err != nil {
								t.Fatalf("private rollback fixture: %v %s", err, output)
							}
						}
					}
					if err := os.WriteFile(path, []byte(body), 0600); err != nil {
						t.Fatal(err)
					}
					t.Cleanup(func() {
						if err := os.WriteFile(path, []byte(raw), 0600); err != nil {
							t.Error(err)
						}
					})
					fake := newFakeRuntime()
					resolves, reserves := 0, 0
					fake.nativeResolve = func(ctx context.Context, q NativeResumeRequest) (NativeResumeResolution, error) {
						resolves++
						return rt.ResolveNativeResume(ctx, q)
					}
					fake.nativeReserve = func(context.Context, NativeAdmissionRequest) (string, error) {
						reserves++
						return "unexpected-grant", nil
					}
					opts := baseOpts(LaunchArgs{Agent: "codex", AgentExplicit: true, ForcedTag: "work", AgentArgs: append([]string(nil), attemptArgs...), AgentArgsExplicit: true})
					opts.Env.Cwd = cwd
					opts.Env.Home = home
					opts.SkipConfigPicker = true
					opts.InlineOptOut = map[string]bool{"codex": true}
					code, err := run(t, opts, fake)
					if err != nil || code != 0 || resolves != 1 || reserves != 0 || fake.launchCount != 1 {
						t.Fatalf("unsupported native evidence gained pinned admission: code=%d err=%v resolves=%d reserves=%d launches=%d", code, err, resolves, reserves, fake.launchCount)
					}
					command, err := DecodeAgentCommand(fake.env[AgentCommandEnv])
					if err != nil {
						t.Fatal(err)
					}
					if !reflect.DeepEqual(command.Argv, attemptArgs) || fake.env[NativeAdmissionEnv] != "" || fake.env["PAIR_SESSION_ID"] != "" {
						t.Fatalf("unsupported evidence replaced native invocation: %+v grant=%t session=%q", command, fake.env[NativeAdmissionEnv] != "", fake.env["PAIR_SESSION_ID"])
					}
				})
			}
			// A scoped config must still reproject after Couch adds its
			// selector-independent snapshot switch at the wrapper boundary.
			runtimeArgs := append(append([]string(nil), args...), "-c", "features={worktrees=false}", "--disable", "shell_snapshot")
			runtimeSelector, err := ParseNativeResumeSelector("codex", runtimeArgs)
			if err != nil {
				t.Fatal(err)
			}
			runtimeResult, err := rt.ResolveNativeResume(t.Context(), NativeResumeRequest{Agent: "codex", CWD: cwd, Selector: runtimeSelector})
			if err != nil || runtimeResult.Status != si.NativeSelectorResolved || runtimeResult.NativeID != id {
				t.Fatalf("snapshot injection changed scoped selection: %+v %v", runtimeResult, err)
			}
			// A later arrival cannot retarget explicit selection; the same custom home
			// remains usable even when latest eligibility becomes cloud-unknown.
			if err := os.WriteFile(filepath.Join(codex, "config.toml"), []byte("sqlite_home='../state'\n"), 0600); err != nil {
				t.Fatal(err)
			}
			checked, err := rt.ResolveNativeResume(t.Context(), NativeResumeRequest{Agent: "codex", CWD: cwd, Selector: pinned})
			if err != nil || checked.NativeID != id {
				t.Fatalf("explicit custom-home revalidation=%+v %v", checked, err)
			}
			unknown, err := rt.ResolveNativeResume(t.Context(), NativeResumeRequest{Agent: "codex", CWD: cwd, Selector: selector})
			if err != nil || unknown.Status != si.NativeSelectorUnsupported || unknown.Reason != string(si.CodexConfigCloudUnknown) || unknown.CodexScope != nil {
				t.Fatalf("cloud-unknown gained authority: %+v %v", unknown, err)
			}
		})
	}
}

// Native minimal protocol records: rust-v0.162.1 protocol/src/protocol.rs
// SessionMeta (3152ff), TurnContextItem (3324ff), rollout/src/lib.rs RolloutLine.
// Optional typed fields are omitted; session_id is natively backfilled from id.
func nativeValidCodexSessionMeta(id, cwd string) string {
	return fmt.Sprintf(`{"timestamp":"2026-10-15T10:00:00Z","type":"session_meta","payload":{"id":%q,"timestamp":"2026-10-15T10:00:00Z","cwd":%q,"originator":"codex_cli_rs","cli_version":"0.162.1","source":"cli"}}`+"\n", id, cwd)
}
func nativeValidCodexTurnContext(cwd string) string {
	return fmt.Sprintf(`{"timestamp":"2026-10-15T10:00:01Z","type":"turn_context","payload":{"cwd":%q,"approval_policy":"never","sandbox_policy":{"type":"read-only"},"model":"gpt-5"}}`+"\n", cwd)
}
