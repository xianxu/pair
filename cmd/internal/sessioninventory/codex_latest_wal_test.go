package sessioninventory_test

import (
	"context"
	"errors"
	"fmt"
	si "github.com/xianxu/pair/cmd/internal/sessioninventory"
	"os/exec"
	"path/filepath"
	"testing"
)

func codexLatestFreezeWAL(t *testing.T, path string) {
	t.Helper()
	python, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("python3 unavailable")
	}
	const script = `import sqlite3,sys
p=sys.argv[1]
c=sqlite3.connect(p)
c.execute('PRAGMA journal_mode=WAL')
c.execute('PRAGMA wal_autocheckpoint=0')
v=c.execute('PRAGMA user_version').fetchone()[0]
c.execute('PRAGMA user_version='+str(v+1))
c.commit()
db=open(p,'rb').read();wal=open(p+'-wal','rb').read()
c.close()
open(p,'wb').write(db);open(p+'-wal','wb').write(wal)
`
	if out, err := exec.Command(python, "-c", script, path).CombinedOutput(); err != nil {
		t.Fatalf("freeze private WAL: %v %s", err, out)
	}
}
func (f *codexLatestFixture) freezeWAL() {
	f.t.Helper()
	codexLatestFreezeWAL(f.t, filepath.Join(f.state, "state_5.sqlite"))
}

type codexCleanupFailureRuntime struct {
	si.OSRuntime
	closes *int
}

func (r codexCleanupFailureRuntime) NewNativeSelectorSQLiteSnapshot(ctx context.Context, a si.Artifact, source si.NativeSelectorSQLiteSnapshotSource, remaining int64) (si.NativeSelectorSQLiteSnapshot, error) {
	snapshot, err := r.OSRuntime.NewNativeSelectorSQLiteSnapshot(ctx, a, source, remaining)
	if err != nil {
		return nil, err
	}
	return codexCleanupFailureSnapshot{NativeSelectorSQLiteSnapshot: snapshot, closes: r.closes}, nil
}

type codexCleanupFailureSnapshot struct {
	si.NativeSelectorSQLiteSnapshot
	closes *int
}

func (s codexCleanupFailureSnapshot) Close() error {
	*s.closes++
	return errors.Join(s.NativeSelectorSQLiteSnapshot.Close(), errors.New("injected private snapshot cleanup failure"))
}
func TestCodexLatestSnapshotCleanupFailureCannotFinalizeAuthority(t *testing.T) {
	for _, scoped := range []bool{true, false} {
		t.Run(fmt.Sprint(scoped), func(t *testing.T) {
			f := newCodexLatestFixture(t)
			f.add(1, 100, "cli", f.cwd, "")
			f.freezeWAL()
			scope := si.CodexSelectionScope{CodexHome: f.state, SQLiteHome: f.state, Provider: "openai", CWD: f.cwd, CWDs: []string{f.cwd}, Mode: si.CodexModeTUI}
			native, err := f.rt.WithCodexScope(scope)
			if err != nil {
				t.Fatal(err)
			}
			closes := 0
			runtime := codexCleanupFailureRuntime{OSRuntime: native, closes: &closes}
			config := si.NativeSelectorConfig{Provider: "openai", ProviderResolved: true, SQLiteHome: f.state, SQLiteHomeResolved: true}
			if scoped {
				config.CodexScope = &scope
			}
			got, err := si.ResolveNativeSelector(t.Context(), runtime, si.NativeSelectorRequest{Agent: si.AgentCodex, CWD: f.cwd, Latest: true, Config: config})
			if got.Status == si.NativeSelectorResolved || got.NativeID != "" || closes != 1 {
				t.Fatalf("cleanup failure gained authority result=%+v err=%v closes=%d", got, err, closes)
			}
			if scoped && err == nil {
				t.Fatal("scoped cleanup failure not reported")
			}
		})
	}
}
