package couchcmd

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/sessionledger"
)

type agyTitleNativeWork struct{ read, decode, sqlite int64 }

type growingAgyTitleRuntime struct {
	sessioninventory.OSRuntime
	t                *testing.T
	sqlite, database string
	mode             string
	armed, grew      bool
	suffixOffset     int64
	work             agyTitleNativeWork
}

func (r *growingAgyTitleRuntime) growDatabase() {
	r.t.Helper()
	before, err := os.Stat(r.database)
	if err != nil {
		r.t.Fatal(err)
	}
	out, err := exec.CommandContext(r.t.Context(), r.sqlite, r.database, "insert into growth values (zeroblob(16384));").CombinedOutput()
	if err != nil {
		r.t.Fatalf("grow disposable SQLite: %v: %s", err, out)
	}
	after, err := os.Stat(r.database)
	if err != nil || !os.SameFile(before, after) || after.Size() <= before.Size() {
		r.t.Fatalf("SQLite write did not grow same file: before=%v after=%v err=%v", before, after, err)
	}
	r.armed, r.grew = false, true
}
func (r *growingAgyTitleRuntime) ReadAt(a sessioninventory.Artifact, offset, limit int64) ([]byte, bool, error) {
	raw, eof, err := r.OSRuntime.ReadAt(a, offset, limit)
	if a.StorageRoot == "agy-brain" || a.StorageRoot == "agy-conversations" {
		r.work.read += int64(len(raw))
	}
	if err == nil && r.armed && a.StorageRoot == "agy-brain" {
		if (r.mode == "admission" && offset > 0 && limit == 4096) || (r.mode == "mid-suffix" && offset >= r.suffixOffset) {
			r.growDatabase()
		}
	}
	return raw, eof, err
}
func (r *growingAgyTitleRuntime) DecodeWork(a sessioninventory.Artifact, n int64) {
	if a.StorageRoot == "agy-brain" || a.StorageRoot == "agy-conversations" {
		r.work.decode += n
	}
}
func (r *growingAgyTitleRuntime) QuerySQLite(a sessioninventory.Artifact, query string, limit int64) (sessioninventory.SQLiteResult, error) {
	return r.QuerySQLiteContext(r.t.Context(), a, query, limit)
}
func (r *growingAgyTitleRuntime) QuerySQLiteContext(ctx context.Context, a sessioninventory.Artifact, query string, limit int64) (sessioninventory.SQLiteResult, error) {
	r.work.sqlite++
	return r.OSRuntime.QuerySQLiteContext(ctx, a, query, limit)
}

type growingAgyOSTitles struct {
	*osCouchTitles
	runtime *growingAgyTitleRuntime
	source  *osCouchTitleSource
	opens   int
}

func (o *growingAgyOSTitles) Open(target couchcore.TitleTarget) (couchTitleSource, error) {
	src, err := o.osCouchTitles.Open(target)
	if err != nil || target.Agent != "agy" {
		return src, err
	}
	s := src.(*osCouchTitleSource)
	o.runtime.OSRuntime = s.runtime
	s.projection = sessioninventory.NewTitleProjection(o.runtime, target.Address.RepoScope, string(target.Address.Tag), sessioninventory.AgentAgy, target.LaunchOrdinal)
	o.source = s
	o.opens++
	return s, nil
}

func TestCouchTitlesOSAgyDatabaseGrowthRecovers(t *testing.T) {
	sqlite, err := exec.LookPath("sqlite3")
	if err != nil {
		t.Skip("sqlite3 is not installed")
	}
	for _, mode := range []string{"admission", "between-polls", "mid-suffix"} {
		t.Run(mode, func(t *testing.T) {
			f := newCouchTitleOSFixture(t)
			titleFixtureWrite(t, f.paths.Pane("codex"), []byte(`{"pane_id":"8"}`))
			paths, err := artifactpath.Resolve(artifactpath.Address{DataDir: f.data, RepoScope: "scope", Tag: "agy-work"})
			if err != nil {
				t.Fatal(err)
			}
			row := couchTitleRow("scope")
			row.Address.Tag = "agy-work"
			row.Agent = "agy"
			row.TitleTarget.Address = row.Address
			row.TitleTarget.Agent = "agy"
			row.TitleTarget.Session = "agy-session"
			row.TitleTarget.Sessions = map[string]bool{"agy-session": true}
			const id = "55555555-5555-4555-8555-555555555555"
			rt := sessioninventory.NewOSRuntime(f.home, paths.ScopeDir())
			var database, transcript string
			for _, root := range rt.NativeRoots(sessioninventory.AgentAgy) {
				if root.Name == "agy-conversations" {
					database = filepath.Join(root.Path, id+".db")
				}
				if root.Name == "agy-brain" {
					transcript = filepath.Join(root.Path, id, ".system_generated", "logs", "transcript.jsonl")
				}
			}
			if database == "" || transcript == "" {
				t.Fatal("Agy native roots missing")
			}
			if err := os.MkdirAll(filepath.Dir(database), 0700); err != nil {
				t.Fatal(err)
			}
			out, err := exec.CommandContext(t.Context(), sqlite, database, "create table trajectory_meta(cascade_id TEXT, trajectory_type INTEGER, source INTEGER); insert into trajectory_meta values ('"+id+"', 1, 1); create table growth(payload BLOB);").CombinedOutput()
			if err != nil {
				t.Fatalf("create disposable Agy SQLite: %v: %s", err, out)
			}
			initial := `{"type":"USER_INPUT","content":"` + strings.Repeat("x", 5000) + `"}` + "\n"
			titleFixtureWrite(t, transcript, []byte(initial))
			initialActivity := time.Now().Add(-time.Second)
			if err := os.Chtimes(transcript, initialActivity, initialActivity); err != nil {
				t.Fatal(err)
			}
			var ledger []byte
			for _, record := range []sessionledger.Record{
				{Version: 3, Kind: sessionledger.RecordLaunch, ScopeKey: "scope", Tag: "agy-work", Agent: "agy", RequestedNativeID: id, RequestOrigin: sessionledger.RequestOriginChosen, BaselineComplete: true, LaunchArtifactBoundaries: []sessionledger.LaunchArtifactBoundary{}},
				{Version: 3, Kind: sessionledger.RecordBinding, ScopeKey: "scope", Tag: "agy-work", Agent: "agy", LaunchOrdinal: 1, RootNativeID: id, ConfirmationReason: sessionledger.ConfirmationChosen},
			} {
				raw, err := sessionledger.EncodeRecord(record)
				if err != nil {
					t.Fatal(err)
				}
				ledger = append(ledger, append(raw, '\n')...)
			}
			titleFixtureWrite(t, paths.Ledger(), ledger)
			titleFixtureWrite(t, paths.Pane("agy"), []byte(`{"pane_id":"7"}`))
			u := newCouchTitleUpdater(f.data, f.home, "", func(context.Context, couchcore.TitleTarget) bool { return true })
			io := &growingAgyOSTitles{osCouchTitles: u.io.(*osCouchTitles), runtime: &growingAgyTitleRuntime{t: t, sqlite: sqlite, database: database, mode: mode, armed: mode == "admission", suffixOffset: int64(len(initial))}}
			u.io = io
			rows := []couchcore.ActionableThreadSummary{row, f.row}
			var source *osCouchTitleSource
			poll := func() (map[couchcore.ThreadAddress]time.Time, map[couchcore.ThreadAddress]bool) {
				t.Helper()
				activity, failed := u.Update(t.Context(), rows)
				if source == nil {
					source = io.source
				}
				if source == nil || io.source != source || io.opens != 1 || len(u.entries) != 2 {
					t.Fatalf("replaced or lost source: opens=%d entries=%d", io.opens, len(u.entries))
				}
				if failed[f.row.Address] || activity[f.row.Address].IsZero() {
					t.Fatalf("steady neighbor failed: %v %v", activity, failed)
				}
				return activity, failed
			}
			// Admission growth may defer initial display by one existing tick.
			for i := 0; i < 3; i++ {
				poll()
			}
			if io.runtime.work.sqlite == 0 || io.runtime.work.read == 0 || io.runtime.work.decode == 0 {
				t.Fatalf("fixture bypassed real native validation: %+v", io.runtime.work)
			}
			if mode == "admission" && io.runtime.work.sqlite < 4 {
				t.Fatalf("admission growth bypassed SQLite revalidation: %+v", io.runtime.work)
			}
			beforeRecovery := io.runtime.work
			if mode != "admission" {
				io.runtime.armed = mode == "mid-suffix"
				if mode == "between-polls" {
					io.runtime.growDatabase()
				}
			}
			// The fresh activity must come from the transcript after database
			// growth, through the same updater source and compact projection.
			file, err := os.OpenFile(transcript, os.O_APPEND|os.O_WRONLY, 0600)
			if err != nil {
				t.Fatal(err)
			}
			_, writeErr := file.WriteString(`{"type":"PLANNER_RESPONSE","content":"fresh activity"}` + "\n")
			closeErr := file.Close()
			if writeErr != nil || closeErr != nil {
				t.Fatalf("append transcript: %v %v", writeErr, closeErr)
			}
			freshActivity := initialActivity.Add(2 * time.Second)
			if err := os.Chtimes(transcript, freshActivity, freshActivity); err != nil {
				t.Fatal(err)
			}
			var activity map[couchcore.ThreadAddress]time.Time
			var failed map[couchcore.ThreadAddress]bool
			for i := 0; i < 4; i++ {
				activity, failed = poll()
			}
			if !io.runtime.grew || failed[row.Address] || !activity[row.Address].After(initialActivity) {
				t.Fatalf("same source did not recover SQLite growth: injected=%t failed=%v activity=%v", io.runtime.grew, failed, activity)
			}
			if mode != "admission" && io.runtime.work.sqlite <= beforeRecovery.sqlite {
				t.Fatalf("tail growth accepted without SQLite revalidation: before=%+v after=%+v", beforeRecovery, io.runtime.work)
			}
			raw, err := os.ReadFile(f.log)
			if err != nil {
				t.Fatal(err)
			}
			wantAgy := "--session\nagy-session\naction\nrename-pane\n--pane-id\n7\nagy\n"
			if strings.Count(string(raw), wantAgy) != 1 || strings.Count(string(raw), "rename-pane\n") != 2 || strings.Count(string(raw), "\n") != 14 {
				t.Fatalf("generic Agy/steady neighbor command contract: %q", raw)
			}
			warmWork := io.runtime.work
			for i := 0; i < 10; i++ {
				poll()
			}
			if io.runtime.work != warmWork {
				t.Fatalf("unchanged recovered source repeated native work: before=%+v after=%+v", warmWork, io.runtime.work)
			}
			after, err := os.ReadFile(f.log)
			if err != nil || string(after) != string(raw) {
				t.Fatalf("unchanged source repeated commands: %q %v", after, err)
			}
			t.Logf("same-source recovery: opens=%d native_read_bytes=%d native_decode_bytes=%d sqlite_queries=%d; 10 stable polls add zero native work and zero commands", io.opens, warmWork.read, warmWork.decode, warmWork.sqlite)
		})
	}
}
