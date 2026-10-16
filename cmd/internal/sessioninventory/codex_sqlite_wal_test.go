package sessioninventory

import (
	"os/exec"
	"testing"
)

// codexTestFreezeWAL makes a private DB/WAL cohort without leaving a writer or
// copying SHM. No source SQLite process participates in selector observation.
func codexTestFreezeWAL(t *testing.T, path string) {
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
