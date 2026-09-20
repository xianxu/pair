package sessioninventory_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/sessioninventorytest"
)

const v1Launch = `{"v":1,"kind":"launch","scope_key":"scope","tag":"work","agent":"codex","pair_log_offset":0,"native_watermarks":[]}` + "\n"

// A ledger grows with every launch by construction, so a whole-file cap is a
// cliff every long-lived thread reaches. The owner ledger was read with an
// 8 MiB one; the operator's thread crossed it on its 14th relaunch and became
// unresumable (#237).
func TestQuerySessionReadsALedgerLargerThanTheOldWholeFileCap(t *testing.T) {
	t.Parallel()
	runtime := sessioninventorytest.NewFakeRuntime()
	pairRoot := sessioninventory.StorageRoot{Name: "pair-data", Path: "/pair/scope"}
	runtime.SetPairDataRoot(pairRoot)
	ledger := sessioninventory.Artifact{StorageRoot: pairRoot.Name, RelativePath: "ledger-work.jsonl"}
	rows := (8<<20)/len(v1Launch) + 2
	runtime.PutFile(sessioninventory.FileEntry{Artifact: ledger}, []byte(strings.Repeat(v1Launch, rows)))

	query, err := sessioninventory.QuerySession(runtime, "scope", "work", sessioninventory.AgentCodex)
	if err != nil {
		t.Fatalf("a %d-row ledger over 8 MiB must still resolve: %v", rows, err)
	}
	if query.Status != sessioninventory.BindingProvisional {
		t.Fatalf("query = %#v, want the latest launch found (provisional)", query)
	}
}

// Launch snapshots grow too: a valid record must not acquire a reader-only cap.
func TestInventoryReadsASingleLargeLedgerRecord(t *testing.T) {
	t.Parallel()
	runtime := sessioninventorytest.NewFakeRuntime()
	pairRoot := sessioninventory.StorageRoot{Name: "pair-data", Path: "/pair/scope"}
	runtime.SetPairDataRoot(pairRoot)
	ledger := sessioninventory.Artifact{StorageRoot: pairRoot.Name, RelativePath: "ledger-work.jsonl"}
	large := strings.TrimSuffix(v1Launch, "}\n") + strings.Repeat(" ", 8<<20) + "}\n"
	runtime.PutFile(sessioninventory.FileEntry{Artifact: ledger}, []byte(large))

	query, err := sessioninventory.QuerySession(runtime, "scope", "work", sessioninventory.AgentCodex)
	if err != nil || query.Status != sessioninventory.BindingProvisional || hasDiagnostic(query.Diagnostics, sessioninventory.DiagnosticPairRecordMalformed) {
		t.Fatalf("large valid ledger query: status=%s err=%v diagnostics=%v", query.Status, err, query.Diagnostics)
	}
	inventory, err := sessioninventory.RecoverPairBindings(runtime, sessioninventory.Inventory{}, "current", "scope", []sessioninventory.Agent{sessioninventory.AgentCodex})
	if err != nil || hasDiagnostic(inventory.Diagnostics, sessioninventory.DiagnosticStorageUnreadable) || hasDiagnostic(inventory.Diagnostics, sessioninventory.DiagnosticPairRecordMalformed) {
		t.Fatalf("large valid ledger inventory: %v, %v", err, inventory.Diagnostics)
	}
}

func TestRecoverPairBindingsReadsLargeLogAndConfig(t *testing.T) {
	for _, name := range []string{"log-work.md", "config-work-codex.json"} {
		t.Run(name, func(t *testing.T) {
			runtime := sessioninventorytest.NewFakeRuntime()
			root := sessioninventory.StorageRoot{Name: "pair-data", Path: "/pair/scope"}
			runtime.SetPairDataRoot(root)
			content := append(bytes.Repeat([]byte(" "), 64<<20), []byte(`{"agent":"codex","session_id":"native-root"}`)...)
			runtime.PutFile(sessioninventory.FileEntry{Artifact: sessioninventory.Artifact{StorageRoot: root.Name, RelativePath: name}}, content)
			inventory, err := sessioninventory.RecoverPairBindings(runtime, sessioninventory.Inventory{}, "current", "scope", []sessioninventory.Agent{sessioninventory.AgentCodex})
			if err != nil || hasDiagnostic(inventory.Diagnostics, sessioninventory.DiagnosticStorageUnreadable) || hasDiagnostic(inventory.Diagnostics, sessioninventory.DiagnosticPairRecordMalformed) {
				t.Fatalf("large artifact: %v, %v", err, inventory.Diagnostics)
			}
			if strings.HasPrefix(name, "config") && !hasDiagnostic(inventory.Diagnostics, sessioninventory.DiagnosticBindingStale) {
				t.Fatal("config was not parsed")
			}
		})
	}
}

// A ledger mid-append has an unterminated last line. ParseLedger already
// treats it as a malformed ordinal; the reader must hand it over rather than
// fail, or a concurrent append would make the thread momentarily unresumable.
func TestQuerySessionToleratesALedgerMidAppend(t *testing.T) {
	t.Parallel()
	runtime := sessioninventorytest.NewFakeRuntime()
	pairRoot := sessioninventory.StorageRoot{Name: "pair-data", Path: "/pair/scope"}
	runtime.SetPairDataRoot(pairRoot)
	ledger := sessioninventory.Artifact{StorageRoot: pairRoot.Name, RelativePath: "ledger-work.jsonl"}
	partial := []byte(v1Launch + `{"v":1,"kind":"launch","scope_key":"scope","tag":"work","agent":"cod`)
	runtime.PutFile(sessioninventory.FileEntry{Artifact: ledger}, partial)

	query, err := sessioninventory.QuerySession(runtime, "scope", "work", sessioninventory.AgentCodex)
	if err != nil {
		t.Fatalf("mid-append ledger must still resolve: %v", err)
	}
	if query.Status != sessioninventory.BindingProvisional || !hasDiagnostic(query.Diagnostics, sessioninventory.DiagnosticPairRecordMalformed) {
		t.Fatalf("query = %#v, want the complete launch found and the partial row diagnosed as malformed", query)
	}
}

// The same ledger family is read by the full-inventory path with a 64 MiB
// whole-file cap — the same class, a later cliff. It reads per record too.
func TestRecoverPairBindingsReadsALedgerLargerThanTheOldWholeFileCap(t *testing.T) {
	t.Parallel()
	runtime := sessioninventorytest.NewFakeRuntime()
	pairRoot := sessioninventory.StorageRoot{Name: "pair-data", Path: "/pair/scope"}
	runtime.SetPairDataRoot(pairRoot)
	ledger := sessioninventory.Artifact{StorageRoot: pairRoot.Name, RelativePath: "ledger-work.jsonl"}
	rows := (64<<20)/len(v1Launch) + 2
	runtime.PutFile(sessioninventory.FileEntry{Artifact: ledger}, bytes.Repeat([]byte(v1Launch), rows))

	inventory, err := sessioninventory.RecoverPairBindings(runtime, sessioninventory.Inventory{}, "current", "scope", []sessioninventory.Agent{sessioninventory.AgentCodex})
	if err != nil {
		t.Fatal(err)
	}
	if hasDiagnostic(inventory.Diagnostics, sessioninventory.DiagnosticStorageUnreadable) {
		t.Fatalf("a %d-row ledger over 64 MiB was diagnosed unreadable: %#v", rows, inventory.Diagnostics)
	}
}
