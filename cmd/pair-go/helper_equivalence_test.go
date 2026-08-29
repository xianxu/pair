package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/sessionledger"
)

// Binary-level route smokes (#104 M3: the standalone helpers are gone, so these
// assert the built `pair` binary's subcommand routes reach their runners rather
// than comparing against a deleted standalone — the in-process routing is
// covered by dispatcher_test.go).

func TestPairGoContextRouteReachesRunner(t *testing.T) {
	bin := t.TempDir()
	pairGo := filepath.Join(bin, "pair-go")
	buildCommand(t, pairGo, ".")

	home, data := writeContextFixture(t)
	env := append(os.Environ(), "HOME="+home, "PAIR_DATA_DIR="+data, "PAIR_SCOPE_KEY=scope")

	r := runCommand(t, env, pairGo, "context", "T", "claude")
	if r.code != 0 {
		t.Fatalf("pair context route exit = %d, want 0\nstderr=%q", r.code, r.stderr)
	}
	if strings.TrimSpace(r.stdout) != "398k" {
		t.Fatalf("pair context route stdout = %q, want 398k", r.stdout)
	}
}

func TestPairGoSlugRouteReachesRunner(t *testing.T) {
	bin := t.TempDir()
	pairGo := filepath.Join(bin, "pair-go")
	buildCommand(t, pairGo, ".")

	// Empty data dir → no config → slugcmd.Run resolves no session_id and no-ops
	// (exit 0, no output, no slug-proposed file).
	data := t.TempDir()
	env := append(os.Environ(),
		"PAIR_TAG=T", "PAIR_DATA_DIR="+data, "PAIR_AGENT=claude", "PAIR_SLUG_NESTED=")

	r := runCommand(t, env, pairGo, "slug")
	if r.code != 0 || r.stdout != "" || r.stderr != "" {
		t.Fatalf("pair slug route: code=%d stdout=%q stderr=%q, want 0/empty/empty", r.code, r.stdout, r.stderr)
	}
	if _, err := os.Stat(filepath.Join(data, "slug-proposed-T")); !os.IsNotExist(err) {
		t.Fatalf("no-session slug must not write a proposal")
	}
}

func buildCommand(t *testing.T, out, pkg string) {
	t.Helper()
	cmd := exec.Command("go", "build", "-o", out, pkg)
	if body, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build %s: %v\n%s", pkg, err, string(body))
	}
}

type commandResult struct {
	code   int
	stdout string
	stderr string
}

func runCommand(t *testing.T, env []string, name string, args ...string) commandResult {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Env = env
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	code := 0
	if err != nil {
		exit, ok := err.(*exec.ExitError)
		if !ok {
			t.Fatalf("run %s: %v", name, err)
		}
		code = exit.ExitCode()
	}
	return commandResult{code: code, stdout: stdout.String(), stderr: stderr.String()}
}

func writeContextFixture(t *testing.T) (home, data string) {
	t.Helper()
	const sessionID = "11111111-1111-4111-8111-111111111111"
	home = t.TempDir()
	data = filepath.Join(home, "data")
	cwd := filepath.Join(home, "repo")
	enc := strings.NewReplacer(".", "-", "/", "-").Replace(cwd)
	proj := filepath.Join(home, ".claude", "projects", enc)
	mustMkdir(t, data)
	mustMkdir(t, cwd)
	mustMkdir(t, proj)
	mustWrite(t, filepath.Join(data, "pane-T-claude.json"), `{"pane_id":"7","cwd":"`+cwd+`","cwd_display":"~/repo"}`)
	transcript := `{"type":"assistant","message":{"model":"claude-opus-4-8","usage":{"input_tokens":397556,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}}` + "\n"
	mustWrite(t, filepath.Join(proj, sessionID+".jsonl"), transcript)
	mustWriteProofLedger(t, home, data, sessionID, int64(len(transcript)))
	return home, data
}

func mustWriteProofLedger(t *testing.T, home, data, sessionID string, size int64) {
	t.Helper()
	runtime := sessioninventory.NewOSRuntime(home, data)
	observations, _ := sessioninventory.ObserveAgentMetadata(runtime, sessioninventory.AgentClaude)
	if len(observations) != 1 {
		t.Fatalf("Claude metadata observations=%d", len(observations))
	}
	entry := observations[0].Entry
	state, _ := json.Marshal(sessioninventory.ScannerState{Version: 1, Agent: sessioninventory.AgentClaude, NativeID: sessionID, IdentityAnchor: sessionID, Role: sessioninventory.RoleRoot, ScannerSchema: "claude-v1", FirstRecordValidated: true})
	proof := sessionledger.AuthorizationProof{Version: 1, RootNativeID: sessionID, ScannerSchema: "claude-v1", ScannerState: state, Artifacts: []sessionledger.ArtifactProof{{StorageRoot: entry.Artifact.StorageRoot, RelativePath: entry.Artifact.RelativePath, StableFileID: string(entry.StableFileID), GenerationToken: string(entry.GenerationToken), MutationToken: string(entry.MutationToken), Size: size, ParserCompleteOffset: size}}}
	launch, err := sessionledger.EncodeRecord(sessionledger.Record{Version: 2, Kind: sessionledger.RecordLaunch, ScopeKey: "scope", Tag: "T", Agent: "claude", LaunchArtifactBoundaries: []sessionledger.LaunchArtifactBoundary{}})
	if err != nil {
		t.Fatal(err)
	}
	binding, err := sessionledger.EncodeRecord(sessionledger.Record{Version: 2, Kind: sessionledger.RecordBinding, ScopeKey: "scope", Tag: "T", Agent: "claude", LaunchOrdinal: 1, RootNativeID: sessionID, AuthorizationProof: &proof})
	if err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(data, "ledger-T.jsonl"), string(launch)+"\n"+string(binding)+"\n")
}

func mustMkdir(t *testing.T, d string) {
	t.Helper()
	if err := os.MkdirAll(d, 0o755); err != nil {
		t.Fatal(err)
	}
}

func mustWrite(t *testing.T, p, s string) {
	t.Helper()
	if err := os.WriteFile(p, []byte(s), 0o644); err != nil {
		t.Fatal(err)
	}
}
