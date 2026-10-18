package sessioninventory_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/xianxu/pair/cmd/internal/sessioninventorytest"
	"github.com/xianxu/pair/cmd/internal/sessionledger"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

func TestTitleProjectionAgyDatabaseGrowthRecovers(t *testing.T) {
	rt, _, _, _ := titleProviderFixture(t, sessioninventory.AgentAgy)
	db := sessioninventory.Artifact{StorageRoot: "agy-conversations", RelativePath: "019d1111-1111-7111-8111-111111111111.db", Kind: sessioninventory.ArtifactDatabase}
	p := sessioninventory.NewTitleProjection(rt, "scope", "work", sessioninventory.AgentAgy, 1)
	first, err := p.Poll()
	if err != nil {
		t.Fatal(err)
	}
	rt.AppendFile(db, []byte("database growth"), "grow")
	if got, err := p.Poll(); err != nil || got != first {
		t.Fatalf("growth must retain last facts without stopping: got=%+v err=%v first=%+v", got, err, first)
	}
	if got, err := p.Poll(); err != nil || got.Awaiting {
		t.Fatalf("stable readmission=%+v %v", got, err)
	}
}

func agyTitleDatabase() sessioninventory.Artifact {
	return sessioninventory.Artifact{StorageRoot: "agy-conversations", RelativePath: "019d1111-1111-7111-8111-111111111111.db", Kind: sessioninventory.ArtifactDatabase}
}

func mutateAgyTitleFile(t *testing.T, rt *sessioninventorytest.FakeRuntime, a sessioninventory.Artifact, mode string) {
	t.Helper()
	e, err := rt.StatFile(a)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := rt.ReadFile(a, -1)
	if err != nil {
		t.Fatal(err)
	}
	e.MutationToken = "mutation"
	switch mode {
	case "shrink":
		rt.TruncateFile(a, 1, "shrink")
		return
	case "replacement":
		e.StableFileID = "replacement"
	case "same-size":
		raw[0] ^= 1
	case "birth":
		at := time.Now()
		e.BirthTime = &at
	case "generation":
		e.GenerationToken = "new-generation"
	case "missing":
		rt.DeleteFile(a)
		return
	default:
		t.Fatalf("unknown mutation %q", mode)
	}
	rt.PutFile(e, raw)
}
func requireAgyTitleTerminal(t *testing.T, p *sessioninventory.TitleProjection) {
	t.Helper()
	if _, err := p.Poll(); err == nil {
		t.Fatal("mutation did not stop projection")
	}
	if _, err := p.Poll(); err == nil {
		t.Fatal("terminal projection retried")
	}
}

type agyTitleRuntime struct {
	*sessioninventorytest.FakeRuntime
	afterRead   func(sessioninventory.Artifact, int64, int64)
	afterSQLite func(sessioninventory.Artifact, string)
	beforeStat  func(sessioninventory.Artifact)
}

func (r *agyTitleRuntime) ReadAt(a sessioninventory.Artifact, offset, limit int64) ([]byte, bool, error) {
	raw, eof, err := r.FakeRuntime.ReadAt(a, offset, limit)
	if err == nil && r.afterRead != nil {
		r.afterRead(a, offset, limit)
	}
	return raw, eof, err
}
func (r *agyTitleRuntime) QuerySQLite(a sessioninventory.Artifact, query string, limit int64) (sessioninventory.SQLiteResult, error) {
	out, err := r.FakeRuntime.QuerySQLite(a, query, limit)
	if err == nil && r.afterSQLite != nil {
		r.afterSQLite(a, query)
	}
	return out, err
}
func (r *agyTitleRuntime) StatFile(a sessioninventory.Artifact) (sessioninventory.FileEntry, error) {
	if r.beforeStat != nil {
		r.beforeStat(a)
	}
	return r.FakeRuntime.StatFile(a)
}

func TestTitleProjectionAgyDatabaseGrowthAdmissionSites(t *testing.T) {
	for _, binding := range []string{"proofless", "stale-proof"} {
		for _, site := range []string{"database-header", "after-validation", "transcript-boundary"} {
			t.Run(binding+"/"+site, func(t *testing.T) {
				fake, a, ledger, _ := titleProviderFixture(t, sessioninventory.AgentAgy)
				if binding == "stale-proof" {
					installAgyTitleProof(t, fake, a, ledger)
				}
				fake.AppendFile(a, []byte(`{"type":"PLANNER_RESPONSE","content":"`+strings.Repeat("x", 5000)+`"}`+"\n"), "padding")
				rt := &agyTitleRuntime{FakeRuntime: fake}
				grew := false
				grow := func() {
					if !grew {
						grew = true
						fake.AppendFile(agyTitleDatabase(), []byte("growth"), "growth")
					}
				}
				rt.afterRead = func(artifact sessioninventory.Artifact, offset, limit int64) {
					if site == "database-header" && artifact.StorageRoot == "agy-conversations" {
						grow()
					}
					if site == "transcript-boundary" && artifact.StorageRoot == a.StorageRoot && offset > 0 && limit == 4096 {
						grow()
					}
				}
				rt.afterSQLite = func(_ sessioninventory.Artifact, query string) {
					if site == "after-validation" && query == agyFactsQuery {
						grow()
					}
				}
				p := sessioninventory.NewTitleProjection(rt, "scope", "work", sessioninventory.AgentAgy, 1)
				if _, err := p.Poll(); err != nil {
					t.Fatalf("growth during %s: %v", site, err)
				}
				if !grew {
					t.Fatal("growth interleaving never ran")
				}
				rt.afterRead, rt.afterSQLite = nil, nil
				if got, err := p.Poll(); err != nil || got.Activity.IsZero() || got.Awaiting {
					t.Fatalf("stable readmission=%+v %v", got, err)
				}
			})
		}
	}
}

func TestTitleProjectionAgyDatabaseGrowthTailSites(t *testing.T) {
	for _, site := range []string{"between-polls", "boundary", "suffix"} {
		t.Run(site, func(t *testing.T) {
			fake, a, _, tail := titleProviderFixture(t, sessioninventory.AgentAgy)
			fake.AppendFile(a, []byte(`{"content":"`+strings.Repeat("x", 5000)+`"}`+"\n"), "padding")
			rt := &agyTitleRuntime{FakeRuntime: fake}
			p := sessioninventory.NewTitleProjection(rt, "scope", "work", sessioninventory.AgentAgy, 1)
			first, err := p.Poll()
			if err != nil {
				t.Fatal(err)
			}
			old, err := fake.StatFile(a)
			if err != nil {
				t.Fatal(err)
			}
			fake.AppendFile(a, tail, "tail")
			grew := false
			grow := func() {
				if !grew {
					grew = true
					fake.AppendFile(agyTitleDatabase(), []byte("growth"), "growth")
				}
			}
			if site == "between-polls" {
				grow()
			}
			rt.afterRead = func(artifact sessioninventory.Artifact, offset, limit int64) {
				if artifact.StorageRoot != a.StorageRoot {
					return
				}
				if site == "boundary" && offset == old.Size-4096 && limit == 4096 {
					grow()
				}
				if site == "suffix" && offset == old.Size {
					grow()
				}
			}
			if got, err := p.Poll(); err != nil || got != first {
				t.Fatalf("growth must retain prior display: %+v %v; prior=%+v", got, err, first)
			}
			if !grew {
				t.Fatal("growth interleaving never ran")
			}
			rt.afterRead = nil
			if got, err := p.Poll(); err != nil || got.Awaiting || got.Activity.IsZero() {
				t.Fatalf("tail readmission=%+v %v", got, err)
			}
		})
	}
}

func TestTitleProjectionAgyGrowthCannotHideTranscriptMutation(t *testing.T) {
	for _, mode := range []string{"same-size", "rewrite-and-append", "shrink", "replacement", "birth", "generation", "missing"} {
		t.Run(mode, func(t *testing.T) {
			fake, a, _, tail := titleProviderFixture(t, sessioninventory.AgentAgy)
			p := sessioninventory.NewTitleProjection(fake, "scope", "work", sessioninventory.AgentAgy, 1)
			if _, err := p.Poll(); err != nil {
				t.Fatal(err)
			}
			fake.AppendFile(agyTitleDatabase(), []byte("growth"), "growth")
			if mode == "rewrite-and-append" {
				e, err := fake.StatFile(a)
				if err != nil {
					t.Fatal(err)
				}
				raw, err := fake.ReadFile(a, -1)
				if err != nil {
					t.Fatal(err)
				}
				raw = bytes.Replace(raw, []byte("prompt"), []byte("PROMPT"), 1)
				e.MutationToken = "rewritten-and-grown"
				fake.PutFile(e, append(raw, tail...))
			} else {
				mutateAgyTitleFile(t, fake, a, mode)
			}
			requireAgyTitleTerminal(t, p)
		})
	}
}

func TestTitleProjectionAgyGrowthRevalidatesSQLiteAuthority(t *testing.T) {
	for _, mode := range []string{"header", "schema", "identity"} {
		t.Run(mode, func(t *testing.T) {
			rt, _, _, _ := titleProviderFixture(t, sessioninventory.AgentAgy)
			db := agyTitleDatabase()
			p := sessioninventory.NewTitleProjection(rt, "scope", "work", sessioninventory.AgentAgy, 1)
			if _, err := p.Poll(); err != nil {
				t.Fatal(err)
			}
			rt.AppendFile(db, []byte("growth"), "growth")
			if _, err := p.Poll(); err != nil {
				t.Fatalf("growth permanently stopped before revalidation: %v", err)
			}
			switch mode {
			case "header":
				e, _ := rt.StatFile(db)
				raw, _ := rt.ReadFile(db, -1)
				raw[0] = 'X'
				e.MutationToken = "bad-header"
				rt.PutFile(e, raw)
			case "schema":
				rt.PutSQLite(db, agySchemaQuery, sessioninventory.SQLiteResult{Columns: []string{"name", "type"}, Rows: [][]string{{"cascade_id", "BLOB"}}})
			case "identity":
				rt.PutSQLite(db, agyFactsQuery, sessioninventory.SQLiteResult{Rows: [][]string{{"wrong-id", "text", "integer", "integer"}}})
			}
			requireAgyTitleTerminal(t, p)
		})
	}
}

func installAgyTitleProof(t *testing.T, rt *sessioninventorytest.FakeRuntime, transcript, ledger sessioninventory.Artifact) {
	t.Helper()
	const id = "019d1111-1111-7111-8111-111111111111"
	validation, _, err := sessioninventory.NewIncrementalInventory(rt, sessioninventory.Catalog{Version: sessioninventory.CatalogVersion}).ValidateNamedContent(sessioninventory.AgentAgy, id)
	if err != nil {
		t.Fatal(err)
	}
	state, err := json.Marshal(validation.State)
	if err != nil {
		t.Fatal(err)
	}
	proof := sessionledger.AuthorizationProof{Version: 1, RootNativeID: id, ScannerSchema: validation.State.ScannerSchema, ScannerState: state}
	for _, artifact := range []sessioninventory.Artifact{agyTitleDatabase(), transcript} {
		e, err := rt.StatFile(artifact)
		if err != nil {
			t.Fatal(err)
		}
		proof.Artifacts = append(proof.Artifacts, sessionledger.ArtifactProof{StorageRoot: artifact.StorageRoot, RelativePath: artifact.RelativePath, StableFileID: string(e.StableFileID), GenerationToken: string(e.GenerationToken), MutationToken: string(e.MutationToken), Size: e.Size, ParserCompleteOffset: e.Size})
	}
	raw, err := rt.ReadFile(ledger, -1)
	if err != nil {
		t.Fatal(err)
	}
	records := sessionledger.ParseLedger(raw).Records
	var updated []byte
	for _, record := range records {
		if record.Kind == sessionledger.RecordBinding {
			record.AuthorizationProof = &proof
		}
		row, err := sessionledger.EncodeRecord(record)
		if err != nil {
			t.Fatal(err)
		}
		updated = append(updated, append(row, '\n')...)
	}
	e, err := rt.StatFile(ledger)
	if err != nil {
		t.Fatal(err)
	}
	e.MutationToken = "proof"
	rt.PutFile(e, updated)
}

func agyTitleWork(rt *sessioninventorytest.FakeRuntime, operation sessioninventorytest.Operation) int {
	return rt.OperationCountForRoot(operation, "agy-conversations") + rt.OperationCountForRoot(operation, "agy-brain")
}

func TestTitleProjectionAgyGrowthWorkScalesWithInvalidations(t *testing.T) {
	type work struct{ read, sql, stat, list int }
	run := func(cycles int) work {
		rt, _, _, _ := titleProviderFixture(t, sessioninventory.AgentAgy)
		p := sessioninventory.NewTitleProjection(rt, "scope", "work", sessioninventory.AgentAgy, 1)
		if _, err := p.Poll(); err != nil {
			t.Fatal(err)
		}
		snapshot := func() work {
			return work{agyTitleWork(rt, sessioninventorytest.OperationReadAt), agyTitleWork(rt, sessioninventorytest.OperationSQLite), agyTitleWork(rt, sessioninventorytest.OperationStatFile), agyTitleWork(rt, sessioninventorytest.OperationListFiles)}
		}
		before := snapshot()
		for cycle := 0; cycle < cycles; cycle++ {
			rt.AppendFile(agyTitleDatabase(), []byte("growth"), sessioninventory.MutationToken(fmt.Sprintf("growth-%d", cycle)))
			sqlBefore := agyTitleWork(rt, sessioninventorytest.OperationSQLite)
			if _, err := p.Poll(); err != nil {
				t.Fatalf("cycle %d growth: %v", cycle, err)
			}
			if got := agyTitleWork(rt, sessioninventorytest.OperationSQLite); got != sqlBefore {
				t.Fatalf("growth tick ran SQLite: before=%d after=%d", sqlBefore, got)
			}
			if got, err := p.Poll(); err != nil || got.Activity.IsZero() {
				t.Fatalf("cycle %d readmission=%+v %v", cycle, got, err)
			}
			sqlAfter := agyTitleWork(rt, sessioninventorytest.OperationSQLite)
			if delta := sqlAfter - sqlBefore; delta != 2 {
				t.Fatalf("proofless cycle %d SQLite=%d want 2", cycle, delta)
			}
			warm := snapshot()
			readFileBefore := agyTitleWork(rt, sessioninventorytest.OperationReadFile)
			for tick := 0; tick < 3; tick++ {
				if _, err := p.Poll(); err != nil {
					t.Fatal(err)
				}
			}
			after := snapshot()
			if after.read != warm.read || after.sql != warm.sql || after.list != warm.list || agyTitleWork(rt, sessioninventorytest.OperationReadFile) != readFileBefore {
				t.Fatalf("unchanged ticks did native history/SQLite/listing work: before=%+v after=%+v", warm, after)
			}
		}
		after := snapshot()
		return work{after.read - before.read, after.sql - before.sql, after.stat - before.stat, after.list - before.list}
	}
	small, big := run(8), run(16)
	t.Logf("8 cycles=%+v; 16 cycles=%+v", small, big)
	if big.read != 2*small.read || big.sql != 2*small.sql || big.stat > 2*small.stat || big.list != 2*small.list {
		t.Fatalf("growth recovery work must scale linearly: 8=%+v 16=%+v", small, big)
	}
}

func TestTitleProjectionAgyGrowthCancellationRemainsReusable(t *testing.T) {
	for _, phase := range []string{"detection", "readmission"} {
		t.Run(phase, func(t *testing.T) {
			fake, _, _, _ := titleProviderFixture(t, sessioninventory.AgentAgy)
			rt := &agyTitleRuntime{FakeRuntime: fake}
			p := sessioninventory.NewTitleProjection(rt, "scope", "work", sessioninventory.AgentAgy, 1)
			if _, err := p.Poll(); err != nil {
				t.Fatal(err)
			}
			fake.AppendFile(agyTitleDatabase(), []byte("growth"), "growth")
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if phase == "detection" {
				rt.beforeStat = func(a sessioninventory.Artifact) {
					if a.StorageRoot == "agy-conversations" {
						cancel()
					}
				}
			} else {
				if _, err := p.Poll(); err != nil {
					t.Fatalf("growth detection=%v", err)
				}
				rt.afterSQLite = func(sessioninventory.Artifact, string) { cancel() }
			}
			if _, err := p.PollContext(ctx); !errors.Is(err, context.Canceled) {
				t.Fatalf("cancel during %s=%v", phase, err)
			}
			rt.beforeStat, rt.afterSQLite = nil, nil
			for tick := 0; tick < 2; tick++ {
				if _, err := p.Poll(); err != nil {
					t.Fatalf("resume after %s tick=%d: %v", phase, tick, err)
				}
			}
		})
	}
}

func TestTitleProjectionAgyGrowthCannotHideOwnerSupersession(t *testing.T) {
	fake, a, ledger, tail := titleProviderFixture(t, sessioninventory.AgentAgy)
	rt := &agyTitleRuntime{FakeRuntime: fake}
	p := sessioninventory.NewTitleProjection(rt, "scope", "work", sessioninventory.AgentAgy, 1)
	if _, err := p.Poll(); err != nil {
		t.Fatal(err)
	}
	old, err := fake.StatFile(a)
	if err != nil {
		t.Fatal(err)
	}
	fake.AppendFile(a, tail, "tail")
	changed := false
	rt.afterRead = func(artifact sessioninventory.Artifact, offset, _ int64) {
		if changed || artifact.StorageRoot != a.StorageRoot || offset != old.Size {
			return
		}
		changed = true
		fake.AppendFile(agyTitleDatabase(), []byte("growth"), "growth")
		launch, err := sessionledger.EncodeRecord(sessionledger.Record{Version: 3, Kind: sessionledger.RecordLaunch, ScopeKey: "scope", Tag: "work", Agent: "agy", RequestedNativeID: "019d1111-1111-7111-8111-111111111111", RequestOrigin: sessionledger.RequestOriginChosen, BaselineComplete: true, LaunchArtifactBoundaries: []sessionledger.LaunchArtifactBoundary{}})
		if err != nil {
			t.Fatal(err)
		}
		fake.AppendFile(ledger, append(launch, '\n'), "new-launch")
	}
	requireAgyTitleTerminal(t, p)
	if !changed {
		t.Fatal("owner supersession interleaving never ran")
	}
}
