package pairlog

import (
	"bytes"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/commitoutcome"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

func TestPersistSessionLog(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "nested", "log-work.md")
	now := time.Date(2026, 8, 28, 12, 34, 56, 0, time.UTC)
	if err := PersistSessionLog(path, []byte("first\nline"), now, "attempt-first"); err != nil {
		t.Fatal(err)
	}
	if err := PersistSessionLog(path, []byte("second"), now.Add(time.Second), "attempt-second"); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := "## 2026-08-28 12:34:56\n<!-- pair-log-v1 bytes=10 append_id=attempt-first state=submitted -->\n\nfirst\nline\n\n---\n\n" +
		"## 2026-08-28 12:34:57\n<!-- pair-log-v1 bytes=6 append_id=attempt-second state=submitted -->\n\nsecond\n\n---\n\n"
	if string(raw) != want {
		t.Fatalf("log = %q, want %q", raw, want)
	}
	if info, err := os.Stat(path + ".lock"); err != nil || info.IsDir() {
		t.Fatalf("durable lock file missing: info=%v err=%v", info, err)
	}
}

func TestPersistSessionLogRetryAfterPublicationIsIdempotent(t *testing.T) {
	t.Parallel()
	for _, failure := range []string{"directory-sync", "unlock"} {
		failure := failure
		t.Run(failure, func(t *testing.T) {
			t.Parallel()
			path := filepath.Join(t.TempDir(), "log.md")
			runtime := &publicationFailureRuntime{Runtime: OSRuntime{}, failure: failure, remaining: 1}
			store := SessionLogStore{Runtime: runtime}
			now := time.Date(2026, 8, 28, 12, 34, 56, 0, time.UTC)
			body := "please preserve this exact durable user turn now"
			err := store.PersistWithID(path, []byte(body), now, "attempt-a")
			want := commitoutcome.Indeterminate
			if failure == "unlock" {
				want = commitoutcome.Committed
			}
			if got := commitoutcome.Of(err); got != want {
				t.Fatalf("first outcome=%v want=%v err=%v", got, want, err)
			}
			if err := store.PersistWithID(path, []byte(body), now.Add(time.Minute), "attempt-a"); err != nil {
				t.Fatalf("retry: %v", err)
			}
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Count(string(raw), body) != 1 || strings.Count(string(raw), "append_id=attempt-a") != 1 {
				t.Fatalf("retry duplicated published entry: %q", raw)
			}
			parsed := sessioninventory.ParsePairLog(raw, 0)
			parsed.Facts[0].ScopeKey, parsed.Facts[0].Tag, parsed.Facts[0].Agent = "scope", "work", sessioninventory.AgentCodex
			native := []sessioninventory.NativeEventFact{
				{Agent: sessioninventory.AgentCodex, RootNodeID: "root-a", Position: 1, Event: sessioninventory.NativeEvent{Kind: sessioninventory.EventOperator, Text: body}},
				{Agent: sessioninventory.AgentCodex, RootNodeID: "root-a", Position: 2, Event: sessioninventory.NativeEvent{Kind: sessioninventory.EventAssistant}},
			}
			if observations := sessioninventory.QualifyTurnSequence(parsed.Facts, native); len(observations) != 1 {
				t.Fatalf("retry poisoned causal matching: facts=%#v observations=%#v", parsed.Facts, observations)
			}
		})
	}
}

func TestPreparedChangedOrAbandonedEntriesNeverAuthorizeCorrelation(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "log.md")
	runtime := &publicationFailureRuntime{Runtime: OSRuntime{}, failure: "directory-sync", remaining: 1}
	store := SessionLogStore{Runtime: runtime}
	if err := store.PrepareWithID(path, []byte("old unsent body"), time.Time{}, "attempt-old"); commitoutcome.Of(err) != commitoutcome.Indeterminate {
		t.Fatalf("first prepare outcome=%v err=%v", commitoutcome.Of(err), err)
	}
	if err := store.PrepareWithID(path, []byte("edited sent body"), time.Time{}, "attempt-edited"); err != nil {
		t.Fatal(err)
	}
	if err := store.CommitID(path, "attempt-edited"); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	parsed := sessioninventory.ParsePairLog(raw, 0)
	if len(parsed.Entries) != 2 || len(parsed.Facts) != 1 || parsed.Facts[0].AuthoredText != "edited sent body" {
		t.Fatalf("unsent prepared entry authorized correlation: parsed=%#v raw=%q", parsed, raw)
	}
}

func TestCommitIDExposesFactsOnlyAfterSubmittedMarkerPublication(t *testing.T) {
	t.Parallel()
	for _, failure := range []string{"lock", "read", "open", "chmod", "write", "sync", "close", "rename", "directory-sync", "unlock"} {
		failure := failure
		t.Run(failure, func(t *testing.T) {
			t.Parallel()
			path := filepath.Join(t.TempDir(), "log.md")
			if err := (SessionLogStore{Runtime: OSRuntime{}}).PrepareWithID(path, []byte("dispatched body"), time.Time{}, "attempt-a"); err != nil {
				t.Fatal(err)
			}
			var runtime Runtime = failingRuntime{Runtime: OSRuntime{}, fail: failure}
			if failure == "directory-sync" || failure == "unlock" {
				runtime = &publicationFailureRuntime{Runtime: OSRuntime{}, failure: failure, remaining: 1}
			}
			err := (SessionLogStore{Runtime: runtime}).CommitID(path, "attempt-a")
			if err == nil {
				t.Fatal("CommitID returned nil error")
			}
			raw, readErr := os.ReadFile(path)
			if readErr != nil {
				t.Fatal(readErr)
			}
			facts := sessioninventory.ParsePairLog(raw, 0).Facts
			published := failure == "directory-sync" || failure == "unlock"
			if (len(facts) == 1) != published {
				t.Fatalf("failure=%s facts=%#v err=%v", failure, facts, err)
			}
			if published && facts[0].AuthoredText != "dispatched body" {
				t.Fatalf("failure=%s facts=%#v", failure, facts)
			}
		})
	}
}

func TestPersistSessionLogRejectsAppendIDReuseForDifferentBody(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "log.md")
	store := SessionLogStore{Runtime: OSRuntime{}}
	if err := store.PersistWithID(path, []byte("first"), time.Time{}, "attempt-a"); err != nil {
		t.Fatal(err)
	}
	err := store.PersistWithID(path, []byte("different"), time.Time{}, "attempt-a")
	if err == nil || commitoutcome.Of(err) != commitoutcome.NotAuthoritative {
		t.Fatalf("reuse err=%v outcome=%v", err, commitoutcome.Of(err))
	}
	raw, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if strings.Contains(string(raw), "different") {
		t.Fatalf("ID collision changed log: %q", raw)
	}
}

func TestPersistSessionLogRejectsNonAppendableLegacyTail(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "log.md")
	existing := []byte("## 2026-08-28 01:02:03\n\nlegacy without separator")
	if err := os.WriteFile(path, existing, 0o600); err != nil {
		t.Fatal(err)
	}
	err := (SessionLogStore{Runtime: OSRuntime{}}).PersistWithID(path, []byte("new"), time.Time{}, "attempt-new")
	if err == nil || commitoutcome.Of(err) != commitoutcome.NotAuthoritative {
		t.Fatalf("err=%v outcome=%v", err, commitoutcome.Of(err))
	}
	raw, readErr := os.ReadFile(path)
	if readErr != nil || !bytes.Equal(raw, existing) {
		t.Fatalf("raw=%q readErr=%v", raw, readErr)
	}
}

func TestPersistSessionLogRoundTripsArbitraryMarkdown(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "log-work.md")
	body := []byte("before\n\n---\n\n## 2026-08-28 01:02:03\n\nafter")
	if err := PersistSessionLog(path, body, time.Date(2026, 8, 28, 12, 34, 56, 0, time.UTC), "attempt-markdown"); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	parsed := sessioninventory.ParsePairLog(raw, 0)
	if len(parsed.MalformedOffsets) != 0 || len(parsed.Facts) != 1 || parsed.Facts[0].Text != string(body) {
		t.Fatalf("parsed=%#v raw=%q", parsed, raw)
	}
}

func TestPersistSessionLogConcurrentProcessesLoseNoEntries(t *testing.T) {
	if os.Getenv("PAIRLOG_HELPER") != "" {
		path := os.Getenv("PAIRLOG_PATH")
		body := os.Getenv("PAIRLOG_BODY")
		if err := PersistSessionLog(path, []byte(body), time.Unix(0, 0).UTC(), "attempt-"+body); err != nil {
			t.Fatal(err)
		}
		return
	}
	path := filepath.Join(t.TempDir(), "log.md")
	const writers = 12
	cmds := make([]*exec.Cmd, writers)
	outputs := make([]bytes.Buffer, writers)
	for i := range cmds {
		cmds[i] = exec.Command(os.Args[0], "-test.run=^TestPersistSessionLogConcurrentProcessesLoseNoEntries$")
		cmds[i].Env = append(os.Environ(), "PAIRLOG_HELPER=1", "PAIRLOG_PATH="+path, "PAIRLOG_BODY=entry-"+string(rune('a'+i)))
		cmds[i].Stdout = &outputs[i]
		cmds[i].Stderr = &outputs[i]
		if err := cmds[i].Start(); err != nil {
			t.Fatal(err)
		}
	}
	for i, cmd := range cmds {
		if err := cmd.Wait(); err != nil {
			t.Fatalf("helper: %v: %s", err, outputs[i].Bytes())
		}
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < writers; i++ {
		body := "entry-" + string(rune('a'+i))
		if strings.Count(string(raw), "\n\n"+body+"\n\n---\n\n") != 1 {
			t.Errorf("%q count != 1 in %q", body, raw)
		}
	}
}

func TestPersistSessionLogFailuresPreserveExistingLog(t *testing.T) {
	t.Parallel()
	for _, failure := range []string{"mkdir", "lock", "read", "open", "chmod", "write", "sync", "close", "rename"} {
		failure := failure
		t.Run(failure, func(t *testing.T) {
			t.Parallel()
			path := filepath.Join(t.TempDir(), "log.md")
			existing := sessioninventory.EncodePairLogEntry([]byte("existing"), time.Time{})
			if err := os.WriteFile(path, existing, 0o600); err != nil {
				t.Fatal(err)
			}
			store := SessionLogStore{Runtime: failingRuntime{Runtime: OSRuntime{}, fail: failure}}
			if err := store.PersistWithID(path, []byte("new"), time.Time{}, "attempt-new"); err == nil {
				t.Fatal("Persist returned nil error")
			}
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(raw, existing) {
				t.Fatalf("failed append changed log to %q", raw)
			}
		})
	}
}

type failingRuntime struct {
	Runtime
	fail string
}

func (f failingRuntime) MkdirAll(path string, mode os.FileMode) error {
	if f.fail == "mkdir" {
		return os.ErrPermission
	}
	return f.Runtime.MkdirAll(path, mode)
}

func (f failingRuntime) Lock(path string) (io.Closer, error) {
	if f.fail == "lock" {
		return nil, os.ErrPermission
	}
	return f.Runtime.Lock(path)
}

func (f failingRuntime) CreateTemp(dir, pattern string) (File, error) {
	if f.fail == "open" {
		return nil, os.ErrPermission
	}
	file, err := f.Runtime.CreateTemp(dir, pattern)
	if err != nil {
		return nil, err
	}
	return failingFile{File: file, fail: f.fail}, nil
}

func (f failingRuntime) ReadFile(path string) ([]byte, error) {
	if f.fail == "read" {
		return nil, os.ErrPermission
	}
	return f.Runtime.ReadFile(path)
}

func (f failingRuntime) Rename(oldPath, newPath string) error {
	if f.fail == "rename" {
		return os.ErrPermission
	}
	return f.Runtime.Rename(oldPath, newPath)
}

type failingFile struct {
	File
	fail string
}

func (f failingFile) Chmod(mode os.FileMode) error {
	if f.fail == "chmod" {
		return os.ErrPermission
	}
	return f.File.Chmod(mode)
}

type publicationFailureRuntime struct {
	Runtime
	failure   string
	remaining int
}

func (r *publicationFailureRuntime) Lock(path string) (io.Closer, error) {
	lock, err := r.Runtime.Lock(path)
	if err != nil {
		return nil, err
	}
	return publicationFailureLock{Closer: lock, runtime: r}, nil
}

func (r *publicationFailureRuntime) SyncDirectory(path string) error {
	if r.failure == "directory-sync" && r.remaining > 0 {
		r.remaining--
		return errors.New("injected directory sync failure")
	}
	return r.Runtime.SyncDirectory(path)
}

type publicationFailureLock struct {
	io.Closer
	runtime *publicationFailureRuntime
}

func (l publicationFailureLock) Close() error {
	err := l.Closer.Close()
	if l.runtime.failure == "unlock" && l.runtime.remaining > 0 {
		l.runtime.remaining--
		return errors.Join(err, errors.New("injected unlock failure"))
	}
	return err
}

func (f failingFile) Write(content []byte) (int, error) {
	if f.fail == "write" {
		return 0, os.ErrPermission
	}
	return f.File.Write(content)
}

func (f failingFile) Sync() error {
	if f.fail == "sync" {
		return os.ErrPermission
	}
	return f.File.Sync()
}

func (f failingFile) Close() error {
	err := f.File.Close()
	if f.fail == "close" {
		return errors.Join(err, os.ErrPermission)
	}
	return err
}
