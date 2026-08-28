package sessionledger

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
)

func TestLedgerStoreAppendConcurrentProcesses(t *testing.T) {
	if os.Getenv("SESSIONLEDGER_HELPER") != "" {
		offset, _ := strconv.ParseUint(os.Getenv("SESSIONLEDGER_OFFSET"), 10, 64)
		if _, err := (LedgerStore{Runtime: OSRuntime{}}).Append(os.Getenv("SESSIONLEDGER_PATH"), launchRecord("scope", "work", offset)); err != nil {
			t.Fatal(err)
		}
		return
	}
	path := filepath.Join(t.TempDir(), "ledger.jsonl")
	const writers = 16
	cmds := make([]*exec.Cmd, writers)
	outputs := make([]bytes.Buffer, writers)
	for i := range cmds {
		cmds[i] = exec.Command(os.Args[0], "-test.run=^TestLedgerStoreAppendConcurrentProcesses$")
		cmds[i].Env = append(os.Environ(), "SESSIONLEDGER_HELPER=1", "SESSIONLEDGER_PATH="+path, "SESSIONLEDGER_OFFSET="+strconv.Itoa(i))
		cmds[i].Stdout, cmds[i].Stderr = &outputs[i], &outputs[i]
		if err := cmds[i].Start(); err != nil {
			t.Fatal(err)
		}
	}
	for i, cmd := range cmds {
		if err := cmd.Wait(); err != nil {
			t.Fatalf("helper %d: %v: %s", i, err, outputs[i].Bytes())
		}
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	parsed := ParseLedger(raw)
	if len(parsed.MalformedOrdinals) != 0 || len(parsed.Records) != writers {
		t.Fatalf("records=%d malformed=%v raw=%q", len(parsed.Records), parsed.MalformedOrdinals, raw)
	}
	seen := map[uint64]bool{}
	for _, record := range parsed.Records {
		seen[record.Ordinal] = true
	}
	for ordinal := uint64(1); ordinal <= writers; ordinal++ {
		if !seen[ordinal] {
			t.Errorf("missing ordinal %d", ordinal)
		}
	}
}
