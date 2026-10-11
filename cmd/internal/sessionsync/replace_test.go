package sessionsync

import (
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

// A reader outside the flock (the operator's editor reloading a clean
// buffer) polls the file while writes land in a loop: every read is a whole
// file, never a truncated or half-written one (#445).
func TestAReaderOutsideTheFlockNeverSeesATornWrite(t *testing.T) {
	r := newRig(t)
	big := strings.Repeat("a train that runs long enough to make a write take a while\n", 4096)
	r.put(r.p.File, statusDoc+"END\n")
	r.put(r.p.Shadow, statusDoc+"END\n")
	var done atomic.Bool
	torn := make(chan string, 1)
	reads := make(chan int, 1)
	go func() {
		n := 0
		for !done.Load() {
			raw, err := os.ReadFile(r.p.File)
			n++
			if s := string(raw); err != nil || !IsSessionSync(s) || !strings.HasSuffix(s, "END\n") {
				select {
				case torn <- s:
				default:
				}
			}
		}
		reads <- n
	}()
	h := r.hash("Trains")
	for i := 0; i < 40; i++ {
		res, err := r.sync.Write(WriteRequest{File: r.p.File, Section: "Trains", Body: big + "END", IfHash: h})
		if err != nil {
			t.Fatal(err)
		}
		h = res.Hash
	}
	done.Store(true)
	if n := <-reads; n < 100 {
		t.Fatalf("the reader polled only %d times; the test proves nothing", n)
	}
	select {
	case s := <-torn:
		t.Fatalf("a reader saw a partial file (%d bytes)", len(s))
	default:
	}
}

// The replacement keeps the operator's file mode, and writes through a
// symlink to its target instead of replacing the link.
func TestAWriteKeepsTheModeAndTheSymlink(t *testing.T) {
	r := newRig(t)
	real := filepath.Join(t.TempDir(), "real.md")
	r.put(real, statusDoc)
	if err := os.Chmod(real, 0o640); err != nil {
		t.Fatal(err)
	}
	os.Remove(r.p.File)
	if err := os.Symlink(real, r.p.File); err != nil {
		t.Fatal(err)
	}
	if _, err := r.sync.Write(WriteRequest{File: r.p.File, Section: "Trains", Body: "new trains", IfHash: r.hash("Trains")}); err != nil {
		t.Fatal(err)
	}
	if info, err := os.Lstat(r.p.File); err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("the link was replaced: %v %v", info, err)
	}
	info, err := os.Stat(real)
	if err != nil || info.Mode().Perm() != 0o640 {
		t.Fatalf("mode %v, want 0640: %v", info, err)
	}
	if !strings.Contains(r.read(real), "new trains") {
		t.Fatalf("the target did not get the write: %q", r.read(real))
	}
	if left, _ := filepath.Glob(filepath.Join(filepath.Dir(real), ".real.md.tmp-*")); len(left) != 0 {
		t.Fatalf("temp files left behind: %v", left)
	}
}
