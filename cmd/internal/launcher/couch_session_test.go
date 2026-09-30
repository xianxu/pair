package launcher

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCouchSessionLeadingProtocolFlag(t *testing.T) {
	for _, argv := range [][]string{{"--couch-session-v1", "resume", "1-repo-2"}, {"--couch-session-v1", "resume", "1-repo-2", "--layout3"}} {
		args, err := ParseArgs(argv)
		if err != nil {
			t.Fatalf("managed argv %v: %v", argv, err)
		}
		if args.ForcedTag != "1-repo-2" {
			t.Fatalf("lost exact tag: %+v", args)
		}
	}
	for _, argv := range [][]string{{"--layout3", "--couch-session-v1", "resume", "work"}, {"--couch-session-v1", "--help"}, {"--couch-session-v1", "codex"}, {"resume", "work", "--couch-session-v1"}} {
		if _, err := ParseArgs(argv); err == nil {
			t.Fatalf("accepted unsupported managed argv %v", argv)
		}
	}
}

func TestCouchSessionProtocolSkewRejectsBeforeEffects(t *testing.T) {
	const old = "/tmp/pair-355-prechange"
	if _, err := os.Stat(old); err != nil {
		t.Skip("immutable pre-change binary unavailable")
	}
	for _, disposition := range []string{"create", "attach"} {
		t.Run(disposition, func(t *testing.T) {
			root := t.TempDir()
			raw, _ := json.Marshal(map[string]string{"scope": "0123456789abcdef", "tag": "1-repo-2", "name": "📁1-3", "nonce": "nonce", "disposition": disposition})
			argv := []string{"--couch-session-v1", "resume", "1-repo-2"}
			if disposition == "create" {
				argv = append(argv, "--layout3")
			}
			cmd := exec.Command(old, append([]string{"launch"}, argv...)...)
			cmd.Dir = root
			cmd.Env = []string{"HOME=" + root, "PATH=/usr/bin:/bin", "PAIR_HOME=" + mustPairHome(t), "XDG_DATA_HOME=" + filepath.Join(root, "data"), "PAIR_COUCH_SESSION_INTENT=" + string(raw)}
			out, err := cmd.CombinedOutput()
			if err == nil || !bytes.Contains(out, []byte("is a flag, not an agent")) {
				t.Fatalf("old binary did not reject protocol: %v %s", err, out)
			}
			entries, err := os.ReadDir(root)
			if err != nil || len(entries) != 0 {
				t.Fatalf("old binary created resources: %v %v", entries, err)
			}
		})
	}
}

func TestCouchSessionUnpairedIntentConsumedBeforeDispatch(t *testing.T) {
	t.Setenv("PAIR_COUCH_SESSION_INTENT", `{"scope":"0123456789abcdef","tag":"work","name":"📁1-1","nonce":"nonce","disposition":"attach"}`)
	var out, stderr bytes.Buffer
	code, err := LaunchNative([]string{"--help"}, t.TempDir(), &out, &stderr)
	if err != nil || code != 2 || !strings.Contains(stderr.String(), "couch session") {
		t.Fatalf("unpaired intent accepted: code=%d err=%v out=%s stderr=%s", code, err, out.String(), stderr.String())
	}
	if os.Getenv("PAIR_COUCH_SESSION_INTENT") != "" {
		t.Fatal("intent leaked to children")
	}
}

func mustPairHome(t *testing.T) string {
	t.Helper()
	p, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestCouchSessionIntentValidation(t *testing.T) {
	scope, _ := ResolveRepoScope("/repo")
	valid := CouchSessionIntent{Scope: scope.Key, Tag: "1-repo-2", Name: "📁1-3", Nonce: "claim", Disposition: "create"}
	for _, tc := range []struct {
		name   string
		mutate func(*CouchSessionIntent)
		flag   bool
		tag    string
		layout bool
		good   bool
	}{
		{name: "create", flag: true, tag: valid.Tag, good: true},
		{name: "attach", flag: true, tag: valid.Tag, mutate: func(i *CouchSessionIntent) { i.Disposition = "attach" }, good: true},
		{name: "missing flag", tag: valid.Tag},
		{name: "different tag", flag: true, tag: "other"},
		{name: "bad scope", flag: true, tag: valid.Tag, mutate: func(i *CouchSessionIntent) { i.Scope = "../bad" }},
		{name: "missing nonce", flag: true, tag: valid.Tag, mutate: func(i *CouchSessionIntent) { i.Nonce = "" }},
		{name: "attach layout", flag: true, tag: valid.Tag, layout: true, mutate: func(i *CouchSessionIntent) { i.Disposition = "attach" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			intent := valid
			if tc.mutate != nil {
				tc.mutate(&intent)
			}
			raw, _ := json.Marshal(intent)
			_, err := applyCouchSessionIntent(LaunchArgs{CouchSessionV1: tc.flag, ForcedTag: tc.tag, Layout: LayoutRequest{Mode: Layout3, Explicit: tc.layout}}, string(raw))
			if (err == nil) != tc.good {
				t.Fatalf("validation err=%v good=%v", err, tc.good)
			}
		})
	}
	if _, err := applyCouchSessionIntent(LaunchArgs{CouchSessionV1: true, ForcedTag: valid.Tag}, ""); err == nil {
		t.Fatal("missing intent accepted")
	}
}

func TestCouchSessionAssignmentsUseExactNameAndRefuseFallback(t *testing.T) {
	scope, _ := ResolveRepoScope("/repo")
	intent := CouchSessionIntent{Scope: scope.Key, Tag: "1-repo-2", Name: "📁1-3", Nonce: "claim", Disposition: "create"}
	for _, single := range []bool{false, true} {
		for _, scenario := range []string{"exact", "occupied", "budget", "scope", "tag", "warm absent"} {
			t.Run(fmt.Sprintf("single=%v/%s", single, scenario), func(t *testing.T) {
				rt := newFakeRuntime()
				current := intent
				root, tag := "/repo", intent.Tag
				var live []Session
				switch scenario {
				case "occupied":
					live = []Session{{Name: intent.Name, State: SessionDetached}}
				case "budget":
					rt.maxSessionNameBytes = 3
				case "scope":
					root = "/other"
				case "tag":
					tag = "other"
				case "warm absent":
					current.Disposition = "attach"
				}
				var stderr bytes.Buffer
				var name string
				var ok bool
				if single {
					name, _, ok = assignSingleSessionName(rt, live, root, tag, &stderr, &current)
				} else {
					_, names, _, success := assignLaunchSessionNames(rt, live, root, "/global", LaunchArgs{ForcedTag: tag, CouchSessionV1: true, CouchSession: &current}, "unused", &stderr)
					ok = success
					name = names[tag]
				}
				if scenario == "exact" {
					if !ok || name != intent.Name {
						t.Fatalf("name=%q ok=%v error=%s", name, ok, &stderr)
					}
				} else if ok {
					t.Fatalf("fallback accepted scenario %s name=%q", scenario, name)
				}
			})
		}
	}
}

func TestCouchSessionIndexReplacementRetainsOnlyCurrentExactAddress(t *testing.T) {
	root := t.TempDir()
	rt := NewOSRuntime(root, "/pair")
	own := SessionNameEntry{RepoRoot: "/repo", RepoName: "repo", ScopeKey: "scope", Tag: "same", SessionName: "📁1-1"}
	otherScope := SessionNameEntry{RepoRoot: "/other", RepoName: "other", ScopeKey: "other", Tag: "same", SessionName: "📁2-1"}
	otherTag := SessionNameEntry{RepoRoot: "/repo", RepoName: "repo", ScopeKey: "scope", Tag: "different", SessionName: "📁1-2"}
	for _, entry := range []SessionNameEntry{own, otherScope, otherTag} {
		if err := rt.AppendSessionNameIndex(entry); err != nil {
			t.Fatal(err)
		}
	}
	own.SessionName = "📁1-3"
	if err := rt.ReplaceSessionNameIndex(own); err != nil {
		t.Fatal(err)
	}
	own.SessionName = "📁1-4"
	if err := rt.ReplaceSessionNameIndex(own); err != nil {
		t.Fatal(err)
	}
	index, err := rt.ReadSessionNameIndex()
	if err != nil {
		t.Fatal(err)
	}
	if len(index.Entries) != 3 {
		t.Fatalf("binding history grew: %+v", index.Entries)
	}
	for _, entry := range []SessionNameEntry{own, otherScope, otherTag} {
		found := false
		for _, got := range index.Entries {
			found = found || got == entry
		}
		if !found {
			t.Fatalf("entry lost: %+v", entry)
		}
	}
	path := filepath.Join(root, "session-names.jsonl")
	if err := os.WriteFile(path, []byte("corrupt\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := rt.ReplaceSessionNameIndex(own); err == nil {
		t.Fatal("replaced unreadable authority")
	}
	raw, _ := os.ReadFile(path)
	if string(raw) != "corrupt\n" {
		t.Fatal("modified corrupt index")
	}
}

type couchSessionRuntime struct {
	*fakeRuntime
	ownerState            SessionOwnerState
	observed, revalidated int
	changed               bool
	changeDuringLayout    bool
}

func (f *couchSessionRuntime) ObserveSessionOwner(ctx context.Context, name, root, scope, tag string) (SessionOwnerObservation, error) {
	f.observed++
	owner, _ := artifactpath.NewStorageOwner(root, scope, tag)
	return SessionOwnerObservation{State: f.ownerState, Name: name, Owner: owner, Server: SessionServerIdentity{PID: 99, Identity: "start", Session: name}}, nil
}
func (f *couchSessionRuntime) RevalidateSessionOwner(context.Context, SessionOwnerObservation) error {
	f.revalidated++
	if f.changed {
		return errors.New("server generation changed")
	}
	return nil
}
func (f *couchSessionRuntime) ReplaceSessionNameIndex(entry SessionNameEntry) error {
	if f.appendIndexErr != nil {
		return f.appendIndexErr
	}
	var retained []SessionNameEntry
	for _, existing := range f.sessionIndex.Entries {
		if existing.ScopeKey != entry.ScopeKey || existing.Tag != entry.Tag {
			retained = append(retained, existing)
		}
	}
	f.sessionIndex.Entries = append(retained, entry)
	return nil
}

func managedSessionOptions(disposition string) LaunchOptions {
	opts := baseOpts(LaunchArgs{CouchSessionV1: true, ForcedTag: "1-work-2"})
	scope, _ := ResolveRepoScope(opts.Env.Cwd)
	opts.GlobalDataDir = "/data"
	opts.Args.CouchSession = &CouchSessionIntent{Scope: scope.Key, Tag: opts.Args.ForcedTag, Name: "📁1-3", Nonce: "claim", Disposition: disposition}
	opts.Env.CouchThreadScope = scope.Key
	opts.Env.CouchThreadTag = opts.Args.ForcedTag
	return opts
}

func TestCouchSessionRunLaunchRequiresOwnedStableWarmSession(t *testing.T) {
	for _, scenario := range []string{"owned", "foreign", "unknown", "absent", "changed", "disappeared", "cold profile"} {
		t.Run(scenario, func(t *testing.T) {
			opts := managedSessionOptions("attach")
			rt := &couchSessionRuntime{fakeRuntime: newFakeRuntime(), ownerState: SessionOwnerOwned}
			rt.sessions = []Session{{Name: opts.Args.CouchSession.Name, State: SessionDetached}}
			switch scenario {
			case "foreign":
				rt.ownerState = SessionOwnerForeign
			case "unknown":
				rt.ownerState = SessionOwnerUnknown
			case "absent":
				rt.ownerState = SessionOwnerAbsent
			case "changed":
				rt.changed = true
			case "disappeared":
				rt.sessions = nil
			case "cold profile":
				opts.Args.ResumeRequired = true
			}
			var stderr bytes.Buffer
			code, err := RunLaunch(opts, rt, &stderr)
			if scenario == "owned" {
				if err != nil || code != 0 || len(rt.attached) != 1 || rt.attached[0] != opts.Args.CouchSession.Name || rt.observed != 1 || rt.revalidated != 2 {
					t.Fatalf("warm attach code=%d err=%v attached=%v stderr=%s", code, err, rt.attached, &stderr)
				}
			} else if err != nil || code == 0 || len(rt.attached) != 0 {
				t.Fatalf("unsafe warm attach code=%d err=%v attached=%v stderr=%s", code, err, rt.attached, &stderr)
			}
			if rt.launchCount != 0 {
				t.Fatal("warm attach fell back to create")
			}
			if scenario != "owned" && len(rt.files) != 0 {
				t.Fatalf("refusal mutated sidecars: %v", rt.files)
			}
		})
	}
}

func TestCouchSessionRunLaunchCreatesExactNameAndReplacesAssociation(t *testing.T) {
	opts := managedSessionOptions("create")
	rt := &couchSessionRuntime{fakeRuntime: newFakeRuntime()}
	rt.sessionIndex.Entries = []SessionNameEntry{{ScopeKey: opts.Args.CouchSession.Scope, Tag: opts.Args.ForcedTag, SessionName: "📁1-1"}}
	var stderr bytes.Buffer
	code, err := RunLaunch(opts, rt, &stderr)
	if err != nil || code != 0 || rt.launched != opts.Args.CouchSession.Name {
		t.Fatalf("create code=%d err=%v name=%q stderr=%s", code, err, rt.launched, &stderr)
	}
	if len(rt.sessionIndex.Entries) != 1 || rt.sessionIndex.Entries[0].SessionName != opts.Args.CouchSession.Name {
		t.Fatalf("index=%+v", rt.sessionIndex.Entries)
	}
}

func (f *couchSessionRuntime) ProbeLiveLayout(name string) (LayoutMode, error) {
	mode, err := f.fakeRuntime.ProbeLiveLayout(name)
	if f.changeDuringLayout {
		f.changed = true
	}
	return mode, err
}
func TestCouchSessionWarmGenerationRevalidatedAfterLayoutProbe(t *testing.T) {
	opts := managedSessionOptions("attach")
	rt := &couchSessionRuntime{fakeRuntime: newFakeRuntime(), ownerState: SessionOwnerOwned, changeDuringLayout: true}
	rt.sessions = []Session{{Name: opts.Args.CouchSession.Name, State: SessionDetached}}
	var stderr bytes.Buffer
	code, err := RunLaunch(opts, rt, &stderr)
	if err != nil || code == 0 || len(rt.attached) != 0 || rt.launchCount != 0 {
		t.Fatalf("changed generation reached handoff: code=%d err=%v attached=%v stderr=%s", code, err, rt.attached, &stderr)
	}
	if rt.observed != 1 {
		t.Fatalf("replaced original observation instead of revalidating: %d", rt.observed)
	}
}
