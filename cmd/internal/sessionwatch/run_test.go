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
	rt.files[sessionFile] = fakeFile{content: rootSessionMeta(sid), birth: rt.now}

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
	rt.files[sessionFile] = fakeFile{content: rootSessionMeta(sid), birth: rt.now}

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

func TestRunDoesNotWriteConfigWhenLedgerAppendFails(t *testing.T) {
	home := "/tmp/home"
	data := "/tmp/data"
	sid := "019eff64-6ceb-7e72-9d41-a735a97029ac"
	sessionFile := home + "/.codex/sessions/2026/06/25/rollout-2026-06-25T08-27-12-" + sid + ".jsonl"
	rt := newFakeRuntime(time.Unix(100, 0))
	rt.files[filepath.Join(data, "agent-pid-test")] = fakeFile{content: []byte("1234\n"), mod: time.Unix(100, 0)}
	rt.alive["1234"] = true
	rt.descendants["1234"] = []string{"1234"}
	rt.lsof["1234"] = []string{sessionFile}
	rt.files[sessionFile] = fakeFile{content: rootSessionMeta(sid), birth: rt.now}
	rt.writeErr[filepath.Join(data, "ledger-test.jsonl")] = errors.New("ledger write failed")

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
	if err == nil {
		t.Fatalf("Run error = nil, want ledger write error")
	}
	if _, ok := rt.writes[filepath.Join(data, "config-test-codex.json")]; ok {
		t.Fatalf("config should not be written when ledger append fails")
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
	rt.files[sessionFile] = fakeFile{content: rootSessionMeta(sid), birth: rt.now}

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

func TestPIDFileFreshUsesExactNativeBoundAndLegacySecondTolerance(t *testing.T) {
	bound := time.Unix(100, 500)
	for _, tc := range []struct {
		name   string
		mod    time.Time
		bound  time.Time
		legacy time.Time
		want   bool
	}{
		{name: "native newer", mod: time.Unix(100, 501), bound: bound, want: true},
		{name: "native exact", mod: bound, bound: bound, want: true},
		{name: "native older same second", mod: time.Unix(100, 499), bound: bound, want: false},
		{name: "legacy older same second", mod: time.Unix(100, 0), legacy: time.Unix(100, 900), want: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := pidFileFresh(tc.mod, tc.bound, tc.legacy)
			if got != tc.want {
				t.Fatalf("pidFileFresh = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestRunAcceptsPIDWrittenAfterGenerationBoundBeforeWatcherStart(t *testing.T) {
	home := "/tmp/home"
	data := "/tmp/data"
	sid := "019eff64-6ceb-7e72-9d41-a735a97029ac"
	sessionFile := home + "/.codex/sessions/2026/06/25/rollout-2026-06-25T08-27-12-" + sid + ".jsonl"
	bound := time.Unix(100, 100)
	rt := newFakeRuntime(time.Unix(100, 900))
	rt.files[filepath.Join(data, "agent-pid-test")] = fakeFile{content: []byte("1234\n"), mod: time.Unix(100, 200)}
	rt.alive["1234"] = true
	rt.descendants["1234"] = []string{"1234"}
	rt.lsof["1234"] = []string{sessionFile}
	rt.files[sessionFile] = fakeFile{content: rootSessionMeta(sid), birth: time.Unix(100, 300)}

	err := Run(Options{Agent: "codex", Tag: "test", Cwd: "/repo", Home: home, DataDir: data,
		PIDNotBefore: bound, PIDWait: time.Second, Timeout: time.Second, Poll: 100 * time.Millisecond}, rt)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(rt.writes[filepath.Join(data, "config-test-codex.json")]); !strings.Contains(got, sid) {
		t.Fatalf("config write = %s, want generation-bound pid accepted", got)
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
	rt.onSleep = func(d time.Duration) {
		if d == time.Second {
			rt.alive["3000"] = false
		}
	}

	err := Run(Options{
		Agent:    "codex",
		Tag:      "tag",
		Cwd:      "/repo",
		Home:     home,
		DataDir:  data,
		PIDWait:  time.Second,
		Timeout:  350 * time.Millisecond,
		Poll:     100 * time.Millisecond,
		SlowPoll: time.Second,
	}, rt)
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}

	if got := rt.countLogs(adapt.NearMiss); got != 1 {
		t.Fatalf("near-miss logs = %d, want 1; logs=%+v", got, rt.logs)
	}
	if rt.hasLog(adapt.Fail, "no session id") {
		t.Fatalf("logs = %+v, process-bound watch should exit without timeout failure", rt.logs)
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
	rt.files[good] = fakeFile{content: rootSessionMeta(sid), birth: rt.now}

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
			rt.files[good] = fakeFile{content: rootSessionMeta(sid), mod: time.Unix(360, 0)}
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

func TestRunDiscoversSessionAfterStartupTimeoutForEveryAsyncAgent(t *testing.T) {
	tests := []struct {
		agent string
		sid   string
		path  func(home, sid string) string
	}{
		{
			agent: "codex",
			sid:   "019eff64-6ceb-7e72-9d41-a735a97029ac",
			path: func(home, sid string) string {
				return home + "/.codex/sessions/2026/08/18/rollout-2026-08-18T14-47-32-" + sid + ".jsonl"
			},
		},
		{
			agent: "agy",
			sid:   "123e4567-e89b-12d3-a456-426614174000",
			path: func(home, sid string) string {
				return home + "/.gemini/antigravity-cli/conversations/" + sid + ".db"
			},
		},
		{
			agent: "muse",
			sid:   "223e4567-e89b-12d3-a456-426614174000",
			path: func(home, sid string) string {
				return home + "/.local/share/muse/sessions/" + sid + "/session.jsonl"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.agent, func(t *testing.T) {
			home := "/tmp/home"
			data := "/tmp/data"
			rt := newFakeRuntime(time.Unix(500, 0))
			rt.files[filepath.Join(data, "agent-pid-tag")] = fakeFile{content: []byte("5000\n"), mod: rt.now}
			rt.alive["5000"] = true
			rt.descendants["5000"] = []string{"5000"}
			rt.onSleep = func(d time.Duration) {
				if d == time.Minute {
					path := tt.path(home, tt.sid)
					rt.lsof["5000"] = []string{path}
					if tt.agent == "codex" {
						rt.files[path] = fakeFile{content: rootSessionMeta(tt.sid), birth: rt.now}
					}
				}
			}

			err := Run(Options{
				Agent:   tt.agent,
				Tag:     "tag",
				Cwd:     "/repo",
				Home:    home,
				DataDir: data,
				PIDWait: 100 * time.Millisecond,
				Timeout: 300 * time.Millisecond,
				Poll:    100 * time.Millisecond,
			}, rt)
			if err != nil {
				t.Fatalf("Run error: %v", err)
			}

			got := string(rt.writes[filepath.Join(data, "config-tag-"+tt.agent+".json")])
			if !strings.Contains(got, `"session_id":"`+tt.sid+`"`) {
				t.Fatalf("config write = %s, want delayed session %s", got, tt.sid)
			}
			if got := countDuration(rt.sleeps, time.Minute); got != 1 {
				t.Fatalf("slow sleeps = %d, want 1; all sleeps=%v", got, rt.sleeps)
			}
		})
	}
}

func TestRunStopsAtSlowPollWhenBoundProcessExits(t *testing.T) {
	data := "/tmp/data"
	rt := newFakeRuntime(time.Unix(600, 0))
	rt.files[filepath.Join(data, "agent-pid-tag")] = fakeFile{content: []byte("6000\n"), mod: rt.now}
	rt.alive["6000"] = true
	rt.onSleep = func(d time.Duration) {
		if d == time.Minute {
			rt.alive["6000"] = false
		}
	}

	err := Run(Options{
		Agent:   "codex",
		Tag:     "tag",
		Cwd:     "/repo",
		Home:    "/tmp/home",
		DataDir: data,
		PIDWait: 100 * time.Millisecond,
		Timeout: 300 * time.Millisecond,
		Poll:    100 * time.Millisecond,
	}, rt)
	if err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if got := countDuration(rt.sleeps, time.Minute); got != 1 {
		t.Fatalf("slow sleeps = %d, want 1; all sleeps=%v", got, rt.sleeps)
	}
	if _, ok := rt.writes[filepath.Join(data, "config-tag-codex.json")]; ok {
		t.Fatal("config should not be written after the bound process exits")
	}
}

func TestRunCodexLsofSkipsSubagentForRoot(t *testing.T) {
	home := "/tmp/home"
	data := "/tmp/data"
	rootSID := "019e8178-79c2-7862-91db-e8fa1be3b162"
	subSID := "01a017b6-af00-7c91-a656-0611a3750669"
	rootPath := home + "/.codex/sessions/2026/08/18/rollout-root-" + rootSID + ".jsonl"
	subPath := home + "/.codex/sessions/2026/08/18/rollout-sub-" + subSID + ".jsonl"
	rt := newFakeRuntime(time.Unix(370, 0))
	rt.files[filepath.Join(data, "agent-pid-tag")] = fakeFile{content: []byte("3700\n"), mod: rt.now}
	rt.files[rootPath] = fakeFile{content: []byte(`{"type":"session_meta","payload":{"id":"` + rootSID + `","parent_thread_id":null,"source":"cli"}}` + "\n"), birth: rt.now}
	rt.files[subPath] = fakeFile{content: []byte(`{"type":"session_meta","payload":{"id":"` + subSID + `","parent_thread_id":"` + rootSID + `","source":{"subagent":{}}}}` + "\n"), birth: rt.now}
	rt.alive["3700"] = true
	rt.descendants["3700"] = []string{"3700"}
	rt.lsof["3700"] = []string{subPath, rootPath}

	if err := Run(Options{Agent: "codex", Tag: "tag", Cwd: "/repo", Home: home, DataDir: data, PIDWait: time.Second, Timeout: time.Second, Poll: 100 * time.Millisecond}, rt); err != nil {
		t.Fatal(err)
	}
	got := string(rt.writes[filepath.Join(data, "config-tag-codex.json")])
	if !strings.Contains(got, rootSID) || strings.Contains(got, subSID) {
		t.Fatalf("config = %s, want root %s", got, rootSID)
	}
}

func TestRunCodexLsofContinuesPastMalformedMetadata(t *testing.T) {
	home := "/tmp/home"
	data := "/tmp/data"
	rootSID := "019e8178-79c2-7862-91db-e8fa1be3b162"
	badSID := "01a017b6-af00-7c91-a656-0611a3750669"
	rootPath := home + "/.codex/sessions/2026/08/18/rollout-root-" + rootSID + ".jsonl"
	badPath := home + "/.codex/sessions/2026/08/18/rollout-bad-" + badSID + ".jsonl"
	rt := newFakeRuntime(time.Unix(375, 0))
	rt.files[filepath.Join(data, "agent-pid-tag")] = fakeFile{content: []byte("3750\n"), mod: rt.now}
	rt.files[badPath] = fakeFile{content: []byte("{not-json}\n"), birth: rt.now}
	rt.files[rootPath] = fakeFile{content: []byte(`{"type":"session_meta","payload":{"id":"` + rootSID + `","parent_thread_id":null,"source":"cli"}}` + "\n"), birth: rt.now}
	rt.alive["3750"] = true
	rt.descendants["3750"] = []string{"3750"}
	rt.lsof["3750"] = []string{badPath, rootPath}

	if err := Run(Options{Agent: "codex", Tag: "tag", Cwd: "/repo", Home: home, DataDir: data, PIDWait: time.Second, Timeout: time.Second, Poll: 100 * time.Millisecond}, rt); err != nil {
		t.Fatal(err)
	}
	got := string(rt.writes[filepath.Join(data, "config-tag-codex.json")])
	if !strings.Contains(got, rootSID) || strings.Contains(got, badSID) {
		t.Fatalf("config = %s, want root after malformed candidate", got)
	}
}

func TestRunCodexBirthFallbackSkipsNewerSubagent(t *testing.T) {
	home := "/tmp/home"
	data := "/tmp/data"
	rootSID := "019e8178-79c2-7862-91db-e8fa1be3b162"
	subSID := "01a017b6-af00-7c91-a656-0611a3750669"
	rootPath := home + "/.codex/sessions/2026/08/18/rollout-root-" + rootSID + ".jsonl"
	subPath := home + "/.codex/sessions/2026/08/18/rollout-sub-" + subSID + ".jsonl"
	rt := newFakeRuntime(time.Unix(380, 0))
	rt.files[filepath.Join(data, "agent-pid-tag")] = fakeFile{content: []byte("3800\n"), mod: rt.now}
	rt.files[rootPath] = fakeFile{content: []byte(`{"type":"session_meta","payload":{"id":"` + rootSID + `","parent_thread_id":null,"source":"exec"}}` + "\n"), birth: rt.now.Add(time.Second)}
	rt.files[subPath] = fakeFile{content: []byte(`{"type":"session_meta","payload":{"id":"` + subSID + `","parent_thread_id":"` + rootSID + `","source":{"subagent":{}}}}` + "\n"), birth: rt.now.Add(2 * time.Second)}
	rt.alive["3800"] = true

	if err := Run(Options{Agent: "codex", Tag: "tag", Cwd: "/repo", Home: home, DataDir: data, PIDWait: time.Second, Timeout: time.Second, Poll: 100 * time.Millisecond}, rt); err != nil {
		t.Fatal(err)
	}
	got := string(rt.writes[filepath.Join(data, "config-tag-codex.json")])
	if !strings.Contains(got, rootSID) || strings.Contains(got, subSID) {
		t.Fatalf("config = %s, want root %s", got, rootSID)
	}
}

func TestRunCodexSubagentOnlyWritesNoConfig(t *testing.T) {
	home := "/tmp/home"
	data := "/tmp/data"
	subSID := "01a017b6-af00-7c91-a656-0611a3750669"
	parent := "019e8178-79c2-7862-91db-e8fa1be3b162"
	subPath := home + "/.codex/sessions/2026/08/18/rollout-sub-" + subSID + ".jsonl"
	rt := newFakeRuntime(time.Unix(390, 0))
	rt.files[filepath.Join(data, "agent-pid-tag")] = fakeFile{content: []byte("3900\n"), mod: rt.now}
	rt.files[subPath] = fakeFile{content: []byte(`{"type":"session_meta","payload":{"id":"` + subSID + `","parent_thread_id":"` + parent + `","source":{"subagent":{}}}}` + "\n"), birth: rt.now}
	rt.alive["3900"] = true
	rt.lsof["3900"] = []string{subPath}
	rt.onSleep = func(time.Duration) { rt.alive["3900"] = false }

	if err := Run(Options{Agent: "codex", Tag: "tag", Cwd: "/repo", Home: home, DataDir: data, PIDWait: time.Second, Timeout: time.Second, Poll: 100 * time.Millisecond}, rt); err != nil {
		t.Fatal(err)
	}
	if got := rt.writes[filepath.Join(data, "config-tag-codex.json")]; got != nil {
		t.Fatalf("subagent-only config = %s, want none", got)
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

func TestRunStopsWhenPIDIsReusedByAnotherProcess(t *testing.T) {
	home := "/tmp/home"
	data := "/tmp/data"
	pidFile := filepath.Join(data, "agent-pid-tag")
	sid := "019eff64-6ceb-7e72-9d41-a735a97029ac"
	sessionFile := home + "/.codex/sessions/2026/06/25/rollout-2026-06-25T08-27-12-" + sid + ".jsonl"
	rt := newFakeRuntime(time.Unix(500, 0))
	rt.files[pidFile] = fakeFile{content: []byte("1234\n"), mod: rt.now}
	rt.alive["1234"] = true
	rt.identities["1234"] = "incarnation-a"
	rt.descendants["1234"] = []string{"1234"}
	rt.onSleep = func(time.Duration) {
		rt.identities["1234"] = "incarnation-b"
		rt.lsof["1234"] = []string{sessionFile}
		rt.files[sessionFile] = fakeFile{content: rootSessionMeta(sid), birth: rt.now}
	}

	if err := Run(Options{Agent: "codex", Tag: "tag", Cwd: "/repo", Home: home, DataDir: data,
		PIDWait: time.Second, Timeout: 100 * time.Millisecond, Poll: 100 * time.Millisecond, SlowPoll: time.Second}, rt); err != nil {
		t.Fatal(err)
	}
	if got := rt.writes[filepath.Join(data, "config-tag-codex.json")]; got != nil {
		t.Fatalf("reused-pid config = %s, want none", got)
	}
}

func TestRunRevalidatesProcessIdentityAfterDiscovery(t *testing.T) {
	home := "/tmp/home"
	data := "/tmp/data"
	sid := "019eff64-6ceb-7e72-9d41-a735a97029ac"
	sessionFile := home + "/.codex/sessions/2026/06/25/rollout-2026-06-25T08-27-12-" + sid + ".jsonl"
	rt := newFakeRuntime(time.Unix(600, 0))
	rt.files[filepath.Join(data, "agent-pid-tag")] = fakeFile{content: []byte("1234\n"), mod: rt.now}
	rt.files[sessionFile] = fakeFile{content: rootSessionMeta(sid), birth: rt.now}
	rt.alive["1234"] = true
	rt.identities["1234"] = "incarnation-a"
	rt.descendants["1234"] = []string{"1234"}
	rt.lsof["1234"] = []string{sessionFile}
	rt.onLsof = func(string) { rt.identities["1234"] = "incarnation-b" }

	if err := Run(Options{Agent: "codex", Tag: "tag", Cwd: "/repo", Home: home, DataDir: data,
		PIDWait: time.Second, Timeout: time.Second, Poll: 100 * time.Millisecond}, rt); err != nil {
		t.Fatal(err)
	}
	if got := rt.writes[filepath.Join(data, "config-tag-codex.json")]; got != nil {
		t.Fatalf("mid-discovery reused-pid config = %s, want none", got)
	}
}

func countDuration(ds []time.Duration, want time.Duration) int {
	var n int
	for _, d := range ds {
		if d == want {
			n++
		}
	}
	return n
}

type fakeFile struct {
	content []byte
	mod     time.Time
	birth   time.Time
}

func rootSessionMeta(sid string) []byte {
	return []byte(`{"type":"session_meta","payload":{"id":"` + sid + `","parent_thread_id":null,"source":"cli"}}` + "\n")
}

type fakeLog struct {
	outcome adapt.Outcome
	detail  string
}

type fakeRuntime struct {
	now         time.Time
	files       map[string]fakeFile
	alive       map[string]bool
	identities  map[string]string
	descendants map[string][]string
	lsof        map[string][]string
	writes      map[string][]byte
	writeErr    map[string]error
	logs        []fakeLog
	sleeps      []time.Duration
	onSleep     func(time.Duration)
	onLsof      func(string)
}

func newFakeRuntime(now time.Time) *fakeRuntime {
	return &fakeRuntime{
		now:         now,
		files:       map[string]fakeFile{},
		alive:       map[string]bool{},
		identities:  map[string]string{},
		descendants: map[string][]string{},
		lsof:        map[string][]string{},
		writes:      map[string][]byte{},
		writeErr:    map[string]error{},
	}
}

func (f *fakeRuntime) Now() time.Time { return f.now }

func (f *fakeRuntime) Sleep(d time.Duration) {
	f.sleeps = append(f.sleeps, d)
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

func (f *fakeRuntime) ReadFirstLine(path string) ([]byte, error) {
	data, err := f.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if i := strings.IndexByte(string(data), '\n'); i >= 0 {
		return data[:i+1], nil
	}
	return nil, errors.New("unterminated first line")
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

func (f *fakeRuntime) LsofPaths(pid string) ([]string, error) {
	if f.onLsof != nil {
		f.onLsof(pid)
	}
	return f.lsof[pid], nil
}
func (f *fakeRuntime) ProcessIdentity(pid string) string {
	if !f.alive[pid] {
		return ""
	}
	if identity := f.identities[pid]; identity != "" {
		return identity
	}
	return "process:" + pid
}
func (f *fakeRuntime) AtomicWrite(path string, data []byte) error {
	if err := f.writeErr[path]; err != nil {
		return err
	}
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
