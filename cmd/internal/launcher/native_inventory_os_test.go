package launcher

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/sessionledger"
)

const customHomeNativeID = "019d1111-1111-7111-8111-000000000001"

func nativeInventoryFixture(t *testing.T) (OSRuntime, string, string) {
	t.Helper()
	home, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("CODEX_HOME", filepath.Join(home, "custom"))
	cwd := filepath.Join(home, "work")
	data := filepath.Join(home, "pair")
	for _, dir := range []string{cwd, data, filepath.Join(home, "custom", "sessions", "2026", "08", "28")} {
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
	}
	native := filepath.Join(home, "custom", "sessions", "2026", "08", "28", "rollout-test-"+customHomeNativeID+".jsonl")
	raw := fmt.Sprintf("{\"type\":\"session_meta\",\"payload\":{\"id\":%q,\"source\":\"cli\",\"cwd\":%q}}\n", customHomeNativeID, cwd)
	if err := os.WriteFile(native, []byte(raw), 0600); err != nil {
		t.Fatal(err)
	}
	return OSRuntime{DataDir: data}, home, cwd
}
func nativeInventoryLedger(t *testing.T, r OSRuntime, agent string, bound bool) {
	t.Helper()
	raw, err := sessionledger.EncodeRecord(sessionledger.Record{Version: 3, Kind: sessionledger.RecordLaunch, ScopeKey: "scope", Tag: "work", Agent: agent, RequestedNativeID: customHomeNativeID, RequestOrigin: sessionledger.RequestOriginChosen, BaselineComplete: true})
	if err != nil {
		t.Fatal(err)
	}
	raw = append(raw, '\n')
	if bound {
		binding, err := sessionledger.EncodeRecord(sessionledger.Record{Version: 1, Kind: sessionledger.RecordBinding, ScopeKey: "scope", Tag: "work", Agent: agent, LaunchOrdinal: 1, RootNativeID: customHomeNativeID})
		if err != nil {
			t.Fatal(err)
		}
		raw = append(raw, binding...)
		raw = append(raw, '\n')
	}
	if err := os.WriteFile(filepath.Join(r.DataDir, "ledger-work.jsonl"), raw, 0600); err != nil {
		t.Fatal(err)
	}
}
func TestNativeInventoryCustomHomeFlowsThroughLedgerAndIdentity(t *testing.T) {
	r, _, cwd := nativeInventoryFixture(t)
	nativeInventoryLedger(t, r, "codex", false)
	entries, err := r.ReadLedger("work")
	if err != nil || len(entries) != 1 || entries[0].SessionID != customHomeNativeID || entries[0].ResumeBlocked {
		t.Fatalf("custom-home ledger=%+v err=%v", entries, err)
	}
	if !r.AgentSessionExists("codex", customHomeNativeID, cwd) {
		t.Fatal("custom home native root not found")
	}
	id, status := r.EstablishedSessionID("scope", "work", "codex")
	if id != customHomeNativeID || status != sessioninventory.BindingProvisional {
		t.Fatalf("custom-home identity=%q %s", id, status)
	}
	nativeInventoryLedger(t, r, "codex", true)
	id, status = r.EstablishedSessionID("scope", "work", "codex")
	if id != customHomeNativeID || status != sessioninventory.BindingEstablished {
		t.Fatalf("custom-home binding=%q %s", id, status)
	}
}
func TestNativeInventoryInvalidCustomHomeCannotReuseDefault(t *testing.T) {
	r, home, cwd := nativeInventoryFixture(t)
	nativeInventoryLedger(t, r, "codex", true)
	defaultDir := filepath.Join(home, ".codex", "sessions", "2026", "08", "28")
	if err := os.MkdirAll(defaultDir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(defaultDir, "rollout-test-"+customHomeNativeID+".jsonl"), []byte("default root"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CODEX_HOME", filepath.Join(home, "missing"))
	if entries, err := r.ReadLedger("work"); err == nil {
		t.Fatalf("invalid custom home projected ledger: %+v", entries)
	}
	if id, status := r.EstablishedSessionID("scope", "work", "codex"); id != "" || status == sessioninventory.BindingEstablished {
		t.Fatalf("invalid custom home projected identity=%q %s", id, status)
	}
	if !r.AgentSessionExists("codex", customHomeNativeID, cwd) {
		t.Fatal("unknown custom home incorrectly proved candidate absence")
	}
	nativeInventoryLedger(t, r, "claude", true)
	if entries, err := r.ReadLedger("work"); err != nil || len(entries) != 1 || entries[0].SessionID != customHomeNativeID {
		t.Fatalf("Codex home affected Claude ledger: %+v %v", entries, err)
	}
	if id, status := r.EstablishedSessionID("scope", "work", "claude"); id != customHomeNativeID || status != sessioninventory.BindingEstablished {
		t.Fatalf("Codex home affected Claude binding: %q %s", id, status)
	}
}

func TestNativeInventoryRelativeHomeUsesCallersDirectory(t *testing.T) {
	r, home, _ := nativeInventoryFixture(t)
	t.Setenv("CODEX_HOME", "custom")
	t.Chdir(home)
	for _, cwd := range []string{"", home} {
		native, err := r.nativeInventory("codex", cwd)
		if err != nil {
			t.Fatal(err)
		}
		roots := native.NativeRoots(sessioninventory.AgentCodex)
		if len(roots) != 1 || roots[0].Path != filepath.Join(home, "custom", "sessions") {
			t.Fatalf("cwd=%q roots=%+v", cwd, roots)
		}
	}
	nativeInventoryLedger(t, r, "codex", false)
	if entries, err := r.ReadLedger("work"); err != nil || len(entries) != 1 || entries[0].SessionID != customHomeNativeID {
		t.Fatalf("relative home ledger=%+v %v", entries, err)
	}
	if id, _ := r.EstablishedSessionID("scope", "work", "codex"); id != customHomeNativeID {
		t.Fatalf("relative home ID=%q", id)
	}
	if !r.AgentSessionExists("codex", customHomeNativeID, home) {
		t.Fatal("relative home candidate missed")
	}
	t.Setenv("CODEX_HOME", filepath.Join(home, "missing"))
	if _, err := r.nativeInventory("codex", home); err == nil || err.Error() != "Codex native home unavailable" {
		t.Fatalf("invalid home error=%v", err)
	}
	if _, err := r.nativeInventory("claude", ""); err != nil {
		t.Fatalf("Codex namespace affected Claude: %v", err)
	}
	if r.AgentSessionExists("claude", customHomeNativeID, home) {
		t.Fatal("invalid Codex namespace implied a Claude candidate")
	}
}
