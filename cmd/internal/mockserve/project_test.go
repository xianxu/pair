package mockserve

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func writeProjectFile(t *testing.T, dir, name, data string) {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
}
func testProject(t *testing.T) (*Project, string) {
	t.Helper()
	d := t.TempDir()
	writeProjectFile(t, d, "fixtures.json", "{}")
	writeProjectFile(t, d, "mock-1/index.html", "one")
	p, e := OpenProject(d)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { p.Close() })
	return p, d
}
func TestVersionSelection(t *testing.T) {
	for _, n := range []string{"mock-1", "mock-2", "mock-7a", "mock-9999999999999999999999"} {
		if _, e := ParseVersion(n); e != nil {
			t.Errorf("%s: %v", n, e)
		}
	}
	for _, n := range []string{"mock-0", "mock-01", "mock-1A", "mock-1ab", "../mock-1", "mock-1\n", "mock--1"} {
		if _, e := ParseVersion(n); e == nil {
			t.Errorf("accepted %q", n)
		}
	}
	names := []string{"mock-2", "mock-10", "mock-10a", "mock-9z", "unrelated"}
	if got, e := SelectVersion(names, ""); e != nil || got != "mock-10a" {
		t.Fatalf("%q %v", got, e)
	}
	for _, n := range []string{"mock-3", "../mock-2"} {
		if _, e := SelectVersion(names, n); e == nil {
			t.Fatalf("accepted missing/invalid %s", n)
		}
	}
	if got, e := SelectVersion(names, "mock-2"); e != nil || got != "mock-2" {
		t.Fatal(got, e)
	}
}
func TestUseAtomicSelection(t *testing.T) {
	p, d := testProject(t)
	writeProjectFile(t, d, "mock-2/index.html", "two")
	if e := p.Use("mock-1"); e != nil {
		t.Fatal(e)
	}
	for _, v := range []string{"mock-9", "../mock-1"} {
		if e := p.Use(v); e == nil {
			t.Fatal("accepted", v)
		}
	}
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			v := "mock-1"
			if i%2 == 0 {
				v = "mock-2"
			}
			if e := p.Use(v); e != nil {
				t.Error(e)
			}
		}(i)
	}
	wg.Wait()
	q, e := OpenProject(d)
	if e != nil {
		t.Fatal(e)
	}
	defer q.Close()
	s, e := q.Scan(context.Background())
	if e != nil {
		t.Fatal(e)
	}
	if s.Version != "mock-1" && s.Version != "mock-2" {
		t.Fatal(s)
	}
	entries, e := os.ReadDir(d)
	if e != nil {
		t.Fatal(e)
	}
	if len(entries) != 5 {
		t.Fatalf("temp residue: %v", entries)
	}
	// Rename failure must remove the temporary file.
	if e := os.Remove(filepath.Join(d, ".mock-current")); e != nil {
		t.Fatal(e)
	}
	if e := os.Mkdir(filepath.Join(d, ".mock-current"), 0700); e != nil {
		t.Fatal(e)
	}
	if e := p.Use("mock-1"); e == nil {
		t.Fatal("expected rename failure")
	}
	entries, _ = os.ReadDir(d)
	if len(entries) != 5 {
		t.Fatal("temp residue", entries)
	}
}
func TestProjectRootConfinement(t *testing.T) {
	p, d := testProject(t)
	outside := t.TempDir()
	writeProjectFile(t, outside, "secret", "secret")
	if e := os.Symlink(filepath.Join(outside, "secret"), filepath.Join(d, "escape")); e != nil {
		t.Fatal(e)
	}
	if _, e := p.readBounded("escape", 128); e == nil {
		t.Fatal("outside symlink read")
	}
	if _, e := p.readBounded("../secret", 128); e == nil {
		t.Fatal("traversal read")
	}
	if p.existsScreen("mock-1", "../fixtures.json") {
		t.Fatal("invalid screen")
	}
	if !p.existsScreen("mock-1", "index.html") {
		t.Fatal("missing screen")
	}
}

func TestUseAtomicSelectionReaders(t *testing.T) {
	p, d := testProject(t)
	writeProjectFile(t, d, "mock-2/index.html", "two")
	if e := p.Use("mock-1"); e != nil {
		t.Fatal(e)
	}
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			v := "mock-1"
			if i%2 == 0 {
				v = "mock-2"
			}
			if e := p.Use(v); e != nil {
				t.Error(e)
				return
			}
		}
	}()
	defer wg.Wait() // join even when an assertion exits this test goroutine
	for i := 0; i < 200; i++ {
		b, e := p.readBounded(".mock-current", selectionLimit)
		if errors.Is(e, ErrPathChanged) {
			continue
		}
		if e != nil {
			t.Fatal(e)
		}
		if string(b) != "mock-1\n" && string(b) != "mock-2\n" {
			t.Fatalf("partial selection %q", b)
		}
	}
	wg.Wait()
	final, err := p.readBounded(".mock-current", selectionLimit)
	if err != nil || (string(final) != "mock-1\n" && string(final) != "mock-2\n") {
		t.Fatalf("settled selection %q: %v", final, err)
	}
}
func TestProjectBoundedReads(t *testing.T) {
	p, d := testProject(t)
	if _, e := p.readBounded("mock-1/index.html", 2); e == nil {
		t.Fatal("exceeded read bound")
	}
	if _, e := p.readBounded("mock-1", 100); e == nil {
		t.Fatal("accepted directory")
	}
	outside := t.TempDir()
	writeProjectFile(t, outside, "index.html", "outside secret")
	// Replacing a validated version with an outside link cannot escape the root.
	if e := os.Rename(filepath.Join(d, "mock-1"), filepath.Join(d, "saved")); e != nil {
		t.Fatal(e)
	}
	if e := os.Symlink(outside, filepath.Join(d, "mock-1")); e != nil {
		t.Fatal(e)
	}
	if _, e := p.readBounded("mock-1/index.html", 100); e == nil {
		t.Fatal("outside ancestor symlink read")
	}
	if e := p.Use("mock-1"); e == nil {
		t.Fatal("outside version accepted")
	}
}

func TestProjectRejectsReplacementDuringOpen(t *testing.T) {
	for _, kind := range []string{"directory", "file", "file-in-parent", "scanner-directory", "scanner-file", "scanner-file-in-parent"} {
		t.Run(kind, func(t *testing.T) {
			p, dir := testProject(t)
			writeProjectFile(t, dir, "secret.html", "PRIVATE_ROOT_SECRET")
			writeProjectFile(t, dir, "index.html", "PRIVATE_ROOT_SECRET")
			writeProjectFile(t, dir, "mock-1/alternate.html", "different public file")
			matched := false
			p.beforeOpen = func(name string) {
				target := "mock-1"
				if strings.Contains(kind, "file") {
					target = "mock-1/index.html"
				}
				if name != target || matched {
					return
				}
				matched = true
				saved := "saved"
				destination := "."
				if strings.Contains(kind, "file") {
					saved = "mock-1/saved.html"
					destination = "../secret.html"
					if strings.HasSuffix(kind, "in-parent") {
						destination = "alternate.html"
					}
				}
				if err := os.Rename(filepath.Join(dir, target), filepath.Join(dir, saved)); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(destination, filepath.Join(dir, target)); err != nil {
					t.Fatal(err)
				}
			}
			var err error
			if strings.HasPrefix(kind, "scanner-") {
				_, err = p.Scan(t.Context())
			} else {
				_, err = p.readBounded("mock-1/index.html", MaxHTMLBytes)
			}
			if !matched || err == nil {
				t.Fatalf("replacement boundary not refused: matched=%v err=%v", matched, err)
			}
		})
	}
}

func TestConcurrentSelectionReadRefusesChangedIdentity(t *testing.T) {
	p, dir := testProject(t)
	writeProjectFile(t, dir, "mock-2/index.html", "two")
	if err := p.Use("mock-1"); err != nil {
		t.Fatal(err)
	}
	changed := false
	p.beforeOpen = func(name string) {
		if name != ".mock-current" || changed {
			return
		}
		changed = true
		if err := p.Use("mock-2"); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := p.readBounded(".mock-current", selectionLimit); !errors.Is(err, ErrPathChanged) {
		t.Fatalf("changed selection identity must refuse: %v", err)
	}
	p.beforeOpen = nil
	b, err := p.readBounded(".mock-current", selectionLimit)
	if err != nil || string(b) != "mock-2\n" {
		t.Fatalf("confirmed selection %q %v", b, err)
	}
}

func TestSelectionInterruptedWriterHelper(t *testing.T) {
	dir := os.Getenv("PAIR_MOCK_INTERRUPTED_WRITER")
	if dir == "" {
		return
	}
	p, err := OpenProject(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	p.beforeSelectionCommit = func() { fmt.Println("staged"); select {} }
	if err := p.Use("mock-2"); err != nil {
		t.Fatal(err)
	}
}

func TestUseRecoversKilledWriter(t *testing.T) {
	p, dir := testProject(t)
	writeProjectFile(t, dir, "mock-2/index.html", "two")
	if err := p.Use("mock-1"); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestSelectionInterruptedWriterHelper$")
	cmd.Env = append(os.Environ(), "PAIR_MOCK_INTERRUPTED_WRITER="+dir)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	var stopOnce sync.Once
	stopHelper := func() { stopOnce.Do(func() { cmd.Process.Kill(); cmd.Wait() }) }
	defer stopHelper()
	ready := make(chan bool, 1)
	go func() { s := bufio.NewScanner(stdout); ready <- s.Scan() && s.Text() == "staged" }()
	select {
	case ok := <-ready:
		if !ok {
			t.Fatal("helper never staged")
		}
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	stage, err := os.ReadFile(filepath.Join(dir, ".mock-current.tmp"))
	if err != nil || string(stage) != "mock-2\n" {
		t.Fatalf("stage %q: %v", stage, err)
	}
	pointer, err := os.ReadFile(filepath.Join(dir, ".mock-current"))
	if err != nil || string(pointer) != "mock-1\n" {
		t.Fatalf("pointer %q: %v", pointer, err)
	}
	waiting := make(chan struct{})
	p.selectionContended = func() { close(waiting) }
	writerDone := make(chan struct{})
	var writerErr error
	go func() { defer close(writerDone); writerErr = p.Use("mock-1") }()
	// Release the helper's OS lock and join the waiting writer on every exit,
	// before testProject's cleanup closes the project handle.
	defer func() { stopHelper(); <-writerDone }()
	select {
	case <-waiting:
	case <-writerDone:
		t.Fatalf("writer bypassed live lock: %v", writerErr)
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	same, err := os.ReadFile(filepath.Join(dir, ".mock-current.tmp"))
	if err != nil || string(same) != string(stage) {
		t.Fatalf("live staging changed %q: %v", same, err)
	}
	pointer, err = os.ReadFile(filepath.Join(dir, ".mock-current"))
	if err != nil || string(pointer) != "mock-1\n" {
		t.Fatalf("live pointer changed %q: %v", pointer, err)
	}
	stopHelper()
	select {
	case <-writerDone:
		if writerErr != nil {
			t.Fatal(writerErr)
		}
	case <-ctx.Done():
		t.Fatal("lock was not released after kill")
	}
	if _, err := os.Lstat(filepath.Join(dir, ".mock-current.tmp")); !os.IsNotExist(err) {
		t.Fatalf("stage survived recovery: %v", err)
	}
	pointer, err = os.ReadFile(filepath.Join(dir, ".mock-current"))
	if err != nil || string(pointer) != "mock-1\n" {
		t.Fatalf("recovered pointer %q: %v", pointer, err)
	}
}

func TestSelectionReservedNamesExcludedFromScan(t *testing.T) {
	p, dir := testProject(t)
	// Exactly fill the public entry allowance before adding the two reserved names.
	for i := 3; i < MaxEntries; i++ {
		writeProjectFile(t, dir, fmt.Sprintf("entry-%d", i), "")
	}
	baseline, err := p.Scan(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{".mock-current.lock", ".mock-current.tmp"} {
		f, err := os.Create(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		if err := f.Truncate(MaxBytes + 1); err != nil {
			t.Fatal(err)
		}
		f.Close()
	}
	entries, bytes := 0, int64(0)
	p.scanWork = func(e int, b int64) { entries += e; bytes += b }
	after, err := p.Scan(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if after != baseline || entries != MaxEntries+2 || bytes != baseline.Bytes {
		t.Fatalf("reserved names affected snapshot/work: before=%+v after=%+v entries=%d bytes=%d", baseline, after, entries, bytes)
	}
	writeProjectFile(t, dir, ".mock-current.tmp", "changed interrupted data")
	after, err = p.Scan(t.Context())
	if err != nil || after != baseline {
		t.Fatalf("staging edit caused reload: %+v %v", after, err)
	}
}

func TestUseRejectsUnsafeLock(t *testing.T) {
	for _, kind := range []string{"symlink", "directory", "replacement"} {
		t.Run(kind, func(t *testing.T) {
			p, dir := testProject(t)
			target := filepath.Join(dir, ".mock-current.lock")
			switch kind {
			case "symlink":
				writeProjectFile(t, dir, "secret", "untouched")
				if err := os.Symlink("secret", target); err != nil {
					t.Fatal(err)
				}
			case "directory":
				if err := os.Mkdir(target, 0700); err != nil {
					t.Fatal(err)
				}
			case "replacement":
				writeProjectFile(t, dir, ".mock-current.lock", "")
				p.beforeOpen = func(name string) {
					if name != ".mock-current.lock" {
						return
					}
					if err := os.Rename(target, target+".old"); err != nil {
						t.Fatal(err)
					}
					writeProjectFile(t, dir, ".mock-current.lock", "")
				}
			}
			if err := p.Use("mock-1"); err == nil {
				t.Fatal("unsafe lock accepted")
			}
			if _, err := os.Lstat(filepath.Join(dir, ".mock-current")); !os.IsNotExist(err) {
				t.Fatalf("selection changed: %v", err)
			}
		})
	}
}

func TestSelectionReservedScopeAndKinds(t *testing.T) {
	for _, kind := range []string{"directory", "symlink"} {
		t.Run(kind, func(t *testing.T) {
			p, dir := testProject(t)
			before, err := p.Scan(t.Context())
			if err != nil {
				t.Fatal(err)
			}
			for _, name := range []string{selectionLockName, selectionStageName} {
				if kind == "directory" {
					writeProjectFile(t, dir, name+"/nested", "private")
				} else if err := os.Symlink("/missing-private-path", filepath.Join(dir, name)); err != nil {
					t.Fatal(err)
				}
			}
			after, err := p.Scan(t.Context())
			if err != nil || after != before {
				t.Fatalf("private kind changed scan: %+v %v", after, err)
			}
			if err := p.Use("mock-1"); err == nil {
				t.Fatal("nonregular lock accepted")
			}
		})
	}
	p, dir := testProject(t)
	before, err := p.Scan(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	writeProjectFile(t, dir, "mock-1/"+selectionLockName, "content")
	writeProjectFile(t, dir, "mock-1/"+selectionStageName, "more")
	after, err := p.Scan(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if before.Digest == after.Digest || after.Entries != before.Entries+2 || after.Bytes != before.Bytes+11 {
		t.Fatalf("nested names excluded: before=%+v after=%+v", before, after)
	}
}

func TestUseRejectsUnsafeStage(t *testing.T) {
	for _, kind := range []string{"directory", "symlink", "replacement"} {
		t.Run(kind, func(t *testing.T) {
			p, dir := testProject(t)
			target := filepath.Join(dir, selectionStageName)
			switch kind {
			case "directory":
				if err := os.Mkdir(target, 0700); err != nil {
					t.Fatal(err)
				}
			case "symlink":
				writeProjectFile(t, dir, "secret", "untouched")
				if err := os.Symlink("secret", target); err != nil {
					t.Fatal(err)
				}
			case "replacement":
				p.beforeSelectionCommit = func() {
					if err := os.Rename(target, target+".old"); err != nil {
						t.Fatal(err)
					}
					writeProjectFile(t, dir, selectionStageName, "mock-99\n")
				}
			}
			if err := p.Use("mock-1"); err == nil {
				t.Fatal("unsafe stage accepted")
			}
			if _, err := os.Lstat(filepath.Join(dir, ".mock-current")); !os.IsNotExist(err) {
				t.Fatalf("selection changed: %v", err)
			}
			if kind == "symlink" {
				data, err := os.ReadFile(filepath.Join(dir, "secret"))
				if err != nil || string(data) != "untouched" {
					t.Fatalf("symlink target affected: %q %v", data, err)
				}
			}
		})
	}
}
