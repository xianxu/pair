package titlepoller

import (
	"fmt"
	"os"
	"testing"

	"github.com/xianxu/pair/cmd/internal/contextcmd"
	"github.com/xianxu/pair/cmd/internal/workbenchperftest"
)

func TestTitleProjectionRepeatedPollWorkIndependentOfLedgerHistory(t *testing.T) {
	const ledgerBytes = 10 << 20
	f := workbenchperftest.NewFixture(t, 1, ledgerBytes)
	o := f.Owners[0]
	file, err := os.OpenFile(o.Transcript, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	_, writeErr := file.WriteString("{\"type\":\"event_msg\",\"payload\":{\"type\":\"token_count\",\"info\":{\"last_token_usage\":{\"input_tokens\":176000}}}}\n")
	closeErr := file.Close()
	if writeErr != nil || closeErr != nil {
		t.Fatalf("usage fixture: %v; close: %v", writeErr, closeErr)
	}
	p := newTitleProjection(contextcmd.Env{Home: f.Home, PairDataDir: o.ScopeDir, PairScopeKey: o.Scope}, 1)
	p.runtime = p.runtime.WithWorkObserver(f.Counts.Observer(o.Tag))
	poll := func() {
		t.Helper()
		if at, ok := p.activity(o.Tag, o.Agent); !ok || at.IsZero() {
			t.Fatalf("activity unresolved: %v %v", at, ok)
		}
		if got := p.count(o.Tag, o.Agent); got != "176k" {
			t.Fatalf("context count=%q, want 176k", got)
		}
	}
	poll()
	cold := f.Counts.Snapshot()
	if cold.ReadFamily("ledger") < ledgerBytes || cold.DecodeFamily("ledger") < ledgerBytes || cold.ReadFamily("native") == 0 {
		t.Fatalf("cold controls disconnected: %+v", cold)
	}
	for i := 0; i < 3; i++ {
		before := f.Counts.Snapshot()
		poll()
		delta := f.Counts.Snapshot().Delta(before)
		if delta.ReadFamily("ledger") != 0 || delta.DecodeFamily("ledger") != 0 {
			t.Fatalf("poll %d reread/redecoded unchanged large ledger: read=%d decoded=%d", i, delta.ReadFamily("ledger"), delta.DecodeFamily("ledger"))
		}
		delta.RequireZero(t)
	}
	before := f.Counts.Snapshot()
	if _, ok := p.activity(o.Tag, o.Agent); !ok {
		t.Fatal("lost warm activity")
	}
	f.Counts.Snapshot().Delta(before).RequireZero(t)
	// A newly chosen launch must invalidate both projections; neither may
	// inherit the established generation that preceded it.
	file, err = os.OpenFile(o.Ledger, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	_, writeErr = fmt.Fprintf(file, "{\"v\":3,\"kind\":\"launch\",\"scope_key\":%q,\"tag\":%q,\"agent\":\"codex\",\"pair_log_offset\":0,\"artifact_boundaries\":[],\"requested_native_id\":\"019d2222-2222-7222-8222-222222222222\",\"request_origin\":\"chosen-id\",\"baseline_complete\":true}\n", o.Scope, o.Tag)
	closeErr = file.Close()
	if writeErr != nil || closeErr != nil {
		t.Fatalf("launch fixture: %v; close: %v", writeErr, closeErr)
	}
	if at, ok := p.activity(o.Tag, o.Agent); ok {
		t.Fatalf("new unbound launch inherited activity %v", at)
	}
	if count := p.count(o.Tag, o.Agent); count != "" {
		t.Fatalf("new unbound launch inherited count %s", count)
	}
}

func TestTitleProjectionCountReusesSamePoll(t *testing.T) {
	f := workbenchperftest.NewFixture(t, 1, 4096)
	o := f.Owners[0]
	p := newTitleProjection(contextcmd.Env{Home: f.Home, PairDataDir: o.ScopeDir, PairScopeKey: o.Scope}, 1)
	p.runtime = p.runtime.WithWorkObserver(f.Counts.Observer(o.Tag))
	if got := p.count(o.Tag, o.Agent); got != "" {
		t.Fatalf("count before poll=%s", got)
	}
	f.Counts.Snapshot().RequireZero(t)
	if _, ok := p.activity(o.Tag, o.Agent); !ok {
		t.Fatal("activity missing")
	}
	before := f.Counts.Snapshot()
	for i := 0; i < 3; i++ {
		p.count(o.Tag, o.Agent)
	}
	f.Counts.Snapshot().Delta(before).RequireZero(t)
	if _, ok := p.activity("another", o.Agent); ok || p.Err() == nil {
		t.Fatal("owner change not terminal")
	}
}
