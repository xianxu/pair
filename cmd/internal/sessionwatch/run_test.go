package sessionwatch

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/adapt"
)

func TestRunUsesFreshPidfileAndWritesConfig(t *testing.T) {
	home := "/tmp/home"
	data := "/tmp/data"
	sid := "019eff64-6ceb-7e72-9d41-a735a97029ac"
	sessionFile := home + "/.codex/sessions/2026/06/25/rollout-2026-06-25T08-27-12-" + sid + ".jsonl"
	rt := newFakeRuntime(time.Unix(100, 0))
	rt.files[filepath.Join(data, "agent-pid-test")] = fakeFile{content: []byte("999999\n"), mod: time.Unix(1, 0)}
	rt.onSleep = func(time.Duration) {
		rt.files[filepath.Join(data, "agent-pid-test")] = fakeFile{content: []byte("1234\n"), mod: time.Unix(100, 0)}
	}
	rt.alive["1234"] = true
	rt.descendants["1234"] = []string{"1234", "5678"}
	rt.lsof["5678"] = []string{sessionFile}

	err := Run(Options{
		Agent:   "codex",
		Tag:     "test",
		Cwd:     "/repo",
		Args:    []string{"resume", "old", `say "hi"`},
		Home:    home,
		DataDir: data,
		PIDWait: 3 * time.Second,
		Timeout: 5 * time.Second,
		Poll:    100 * time.Millisecond,
	}, rt)
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}

	out := filepath.Join(data, "config-test-codex.json")
	got := string(rt.writes[out])
	if !strings.Contains(got, `"session_id":"`+sid+`"`) || strings.Contains(got, "old") || !strings.Contains(got, `say \"hi\"`) {
		t.Fatalf("config write = %s", got)
	}
	if !rt.hasLog(adapt.Fired, "session_id="+sid) {
		t.Fatalf("logs = %+v, want fired session id", rt.logs)
	}
	ledger := string(rt.writes[filepath.Join(data, "ledger-test.jsonl")])
	if !strings.Contains(ledger, `"agent":"codex"`) || !strings.Contains(ledger, `"session_id":"`+sid+`"`) || !strings.Contains(ledger, `"repo_root":"/repo"`) {
		t.Fatalf("ledger write = %s", ledger)
	}
	if strings.Contains(ledger, "old") || !strings.Contains(ledger, `say \"hi\"`) {
		t.Fatalf("ledger args = %s", ledger)
	}
}

func TestRunUsesRepoIdentityForLedgerWhenCwdIsSubdir(t *testing.T) {
	home := "/tmp/home"
	data := "/tmp/data"
	sid := "019eff64-6ceb-7e72-9d41-a735a97029ac"
	sessionFile := home + "/.codex/sessions/2026/06/25/rollout-2026-06-25T08-27-12-" + sid + ".jsonl"
	rt := newFakeRuntime(time.Unix(100, 0))
	rt.files[filepath.Join(data, "agent-pid-test")] = fakeFile{content: []byte("1234\n"), mod: time.Unix(100, 0)}
	rt.alive["1234"] = true
	rt.descendants["1234"] = []string{"1234"}
	rt.lsof["1234"] = []string{sessionFile}

	err := Run(Options{
		Agent:    "codex",
		Tag:      "test",
		Cwd:      "/repo/cmd/pair",
		RepoRoot: "/repo",
		RepoName: "pair",
		Home:     home,
		DataDir:  data,
		PIDWait:  time.Second,
		Timeout:  time.Second,
		Poll:     100 * time.Millisecond,
	}, rt)
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}

	ledger := string(rt.writes[filepath.Join(data, "ledger-test.jsonl")])
	if !strings.Contains(ledger, `"repo_root":"/repo"`) || !strings.Contains(ledger, `"repo_name":"pair"`) {
		t.Fatalf("ledger write = %s, want repo identity rather than cwd-derived identity", ledger)
	}
	if strings.Contains(ledger, `/repo/cmd/pair`) || strings.Contains(ledger, `"repo_name":"cmd"`) {
		t.Fatalf("ledger write = %s, should not persist pane cwd as repo identity", ledger)
	}
}

func TestRunTreatsSameSecondPidfileAsFresh(t *testing.T) {
	home := "/tmp/home"
	data := "/tmp/data"
	sid := "019eff64-6ceb-7e72-9d41-a735a97029ac"
	sessionFile := home + "/.codex/sessions/2026/06/25/rollout-2026-06-25T08-27-12-" + sid + ".jsonl"
	rt := newFakeRuntime(time.Unix(100, 900_000_000))
	rt.files[filepath.Join(data, "agent-pid-test")] = fakeFile{content: []byte("1234\n"), mod: time.Unix(100, 0)}
	rt.alive["1234"] = true
	rt.descendants["1234"] = []string{"1234"}
	rt.lsof["1234"] = []string{sessionFile}

	err := Run(Options{
		Agent:   "codex",
		Tag:     "test",
		Cwd:     "/repo",
		Home:    home,
		DataDir: data,
		PIDWait: time.Second,
		Timeout: time.Second,
		Poll:    100 * time.Millisecond,
	}, rt)
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if got := string(rt.writes[filepath.Join(data, "config-test-codex.json")]); !strings.Contains(got, sid) {
		t.Fatalf("config write = %s, want same-second pidfile accepted", got)
	}
}

func TestRunDiscoversAgySessionFromLsof(t *testing.T) {
	home := "/tmp/home"
	data := "/tmp/data"
	sid := "123e4567-e89b-12d3-a456-426614174000"
	sessionFile := home + "/.gemini/antigravity-cli/conversations/" + sid + ".db"
	rt := newFakeRuntime(time.Unix(200, 0))
	rt.files[filepath.Join(data, "agent-pid-tag")] = fakeFile{content: []byte("2000\n"), mod: time.Unix(200, 0)}
	rt.alive["2000"] = true
	rt.descendants["2000"] = []string{"2000"}
	rt.lsof["2000"] = []string{sessionFile}

	err := Run(Options{
		Agent:   "agy",
		Tag:     "tag",
		Cwd:     "/repo",
		Args:    []string{"--conversation", "keep"},
		Home:    home,
		DataDir: data,
		PIDWait: time.Second,
		Timeout: time.Second,
		Poll:    100 * time.Millisecond,
	}, rt)
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}

	got := string(rt.writes[filepath.Join(data, "config-tag-agy.json")])
	if !strings.Contains(got, `"session_id":"`+sid+`"`) || !strings.Contains(got, "--conversation") {
		t.Fatalf("agy config write = %s", got)
	}
}

func TestRunLogsNearMissOnce(t *testing.T) {
	home := "/tmp/home"
	data := "/tmp/data"
	bad := home + "/.codex/sessions/2026/06/25/rollout-2026-06-25T08-27-12-not-a-uuid.jsonl"
	rt := newFakeRuntime(time.Unix(300, 0))
	rt.files[filepath.Join(data, "agent-pid-tag")] = fakeFile{content: []byte("3000\n"), mod: time.Unix(300, 0)}
	rt.alive["3000"] = true
	rt.descendants["3000"] = []string{"3000"}
	rt.lsof["3000"] = []string{bad}

	err := Run(Options{
		Agent:   "codex",
		Tag:     "tag",
		Cwd:     "/repo",
		Home:    home,
		DataDir: data,
		PIDWait: time.Second,
		Timeout: 350 * time.Millisecond,
		Poll:    100 * time.Millisecond,
	}, rt)
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}

	if got := rt.countLogs(adapt.NearMiss); got != 1 {
		t.Fatalf("near-miss logs = %d, want 1; logs=%+v", got, rt.logs)
	}
	if !rt.hasLog(adapt.Fail, "no session id") {
		t.Fatalf("logs = %+v, want fail after timeout", rt.logs)
	}
}

func TestRunContinuesPastLsofNearMissToValidCandidate(t *testing.T) {
	home := "/tmp/home"
	data := "/tmp/data"
	sid := "019eff64-6ceb-7e72-9d41-a735a97029ac"
	bad := home + "/.codex/sessions/2026/06/25/rollout-2026-06-25T08-27-12-not-a-uuid.jsonl"
	good := home + "/.codex/sessions/2026/06/25/rollout-2026-06-25T08-27-12-" + sid + ".jsonl"
	rt := newFakeRuntime(time.Unix(350, 0))
	rt.files[filepath.Join(data, "agent-pid-tag")] = fakeFile{content: []byte("3500\n"), mod: time.Unix(350, 0)}
	rt.alive["3500"] = true
	rt.descendants["3500"] = []string{"3500", "3501"}
	rt.lsof["3500"] = []string{bad}
	rt.lsof["3501"] = []string{good}

	err := Run(Options{
		Agent:   "codex",
		Tag:     "tag",
		Cwd:     "/repo",
		Home:    home,
		DataDir: data,
		PIDWait: time.Second,
		Timeout: time.Second,
		Poll:    100 * time.Millisecond,
	}, rt)
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	got := string(rt.writes[filepath.Join(data, "config-tag-codex.json")])
	if !strings.Contains(got, sid) {
		t.Fatalf("config write = %s, want valid sid after near miss", got)
	}
	if rt.countLogs(adapt.NearMiss) != 0 {
		t.Fatalf("near miss should not be logged when a valid candidate is found later: %+v", rt.logs)
	}
}

func TestRunContinuesPastLegacyNearMissToValidCandidate(t *testing.T) {
	home := "/tmp/home"
	data := "/tmp/data"
	sid := "019eff64-6ceb-7e72-9d41-a735a97029ac"
	bad := home + "/.codex/sessions/2026/06/25/rollout-2026-06-25T08-27-12-not-a-uuid.jsonl"
	good := home + "/.codex/sessions/2026/06/25/rollout-2026-06-25T08-27-12-" + sid + ".jsonl"
	rt := newFakeRuntime(time.Unix(360, 0))
	var sleeps int
	rt.onSleep = func(time.Duration) {
		sleeps++
		if sleeps == 2 {
			rt.files[bad] = fakeFile{mod: time.Unix(360, 0)}
			rt.files[good] = fakeFile{mod: time.Unix(360, 0)}
		}
	}

	err := Run(Options{
		Agent:   "codex",
		Tag:     "tag",
		Cwd:     "/repo",
		Home:    home,
		DataDir: data,
		PIDWait: 100 * time.Millisecond,
		Timeout: time.Second,
		Poll:    100 * time.Millisecond,
	}, rt)
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	got := string(rt.writes[filepath.Join(data, "config-tag-codex.json")])
	if !strings.Contains(got, sid) {
		t.Fatalf("config write = %s, want valid sid after legacy near miss", got)
	}
}

func TestRunLogsFailOnTimeout(t *testing.T) {
	rt := newFakeRuntime(time.Unix(400, 0))
	err := Run(Options{
		Agent:   "codex",
		Tag:     "tag",
		Cwd:     "/repo",
		Home:    "/tmp/home",
		DataDir: "/tmp/data",
		PIDWait: 100 * time.Millisecond,
		Timeout: 300 * time.Millisecond,
		Poll:    100 * time.Millisecond,
	}, rt)
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if !rt.hasLog(adapt.Fail, "no session id") {
		t.Fatalf("logs = %+v, want fail", rt.logs)
	}
}

type fakeFile struct {
	content []byte
	mod     time.Time
	birth   time.Time
}

type fakeLog struct {
	outcome adapt.Outcome
	detail  string
}

type fakeRuntime struct {
	now         time.Time
	files       map[string]fakeFile
	alive       map[string]bool
	descendants map[string][]string
	lsof        map[string][]string
	writes      map[string][]byte
	logs        []fakeLog
	onSleep     func(time.Duration)
}

func newFakeRuntime(now time.Time) *fakeRuntime {
	return &fakeRuntime{
		now:         now,
		files:       map[string]fakeFile{},
		alive:       map[string]bool{},
		descendants: map[string][]string{},
		lsof:        map[string][]string{},
		writes:      map[string][]byte{},
	}
}

func (f *fakeRuntime) Now() time.Time { return f.now }

func (f *fakeRuntime) Sleep(d time.Duration) {
	if f.onSleep != nil {
		f.onSleep(d)
	}
	f.now = f.now.Add(d)
}

func (f *fakeRuntime) ReadFile(path string) ([]byte, error) {
	file, ok := f.files[path]
	if !ok {
		return nil, errors.New("missing")
	}
	return file.content, nil
}

func (f *fakeRuntime) ModTime(path string) (time.Time, error) {
	file, ok := f.files[path]
	if !ok {
		return time.Time{}, errors.New("missing")
	}
	return file.mod, nil
}

func (f *fakeRuntime) BirthTime(path string) (time.Time, error) {
	file, ok := f.files[path]
	if !ok {
		return time.Time{}, errors.New("missing")
	}
	if file.birth.IsZero() {
		return file.mod, nil
	}
	return file.birth, nil
}

func (f *fakeRuntime) ListFiles(root string) ([]string, error) {
	var out []string
	for path := range f.files {
		if strings.HasPrefix(path, root) {
			out = append(out, path)
		}
	}
	return out, nil
}

func (f *fakeRuntime) Descendants(root string) ([]string, error) {
	if out := f.descendants[root]; len(out) > 0 {
		return out, nil
	}
	return []string{root}, nil
}

func (f *fakeRuntime) LsofPaths(pid string) ([]string, error) { return f.lsof[pid], nil }
func (f *fakeRuntime) ProcessAlive(pid string) bool           { return f.alive[pid] }
func (f *fakeRuntime) AtomicWrite(path string, data []byte) error {
	f.writes[path] = append([]byte(nil), data...)
	return nil
}
func (f *fakeRuntime) Log(outcome adapt.Outcome, detail string) {
	f.logs = append(f.logs, fakeLog{outcome: outcome, detail: detail})
}

func (f *fakeRuntime) hasLog(outcome adapt.Outcome, detail string) bool {
	for _, log := range f.logs {
		if log.outcome == outcome && strings.Contains(log.detail, detail) {
			return true
		}
	}
	return false
}

func (f *fakeRuntime) countLogs(outcome adapt.Outcome) int {
	var n int
	for _, log := range f.logs {
		if log.outcome == outcome {
			n++
		}
	}
	return n
}
