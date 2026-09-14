package launcher

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type fakeRetentionUse struct {
	id        string
	events    *[]string
	finishErr error
	spawnErr  error
}

func (f *fakeRetentionUse) ReservationID() string { return f.id }
func (f *fakeRetentionUse) BeforeSpawn() error {
	*f.events = append(*f.events, "spawn")
	return f.spawnErr
}
func (f *fakeRetentionUse) Finish(success bool) error {
	*f.events = append(*f.events, fmt.Sprintf("finish:%v", success))
	return f.finishErr
}

type retainedRuntime struct {
	traceTail bool
	*fakeRuntime
	events                        []string
	beginErr, finishErr, spawnErr error
}

func (r *retainedRuntime) BeginRetention(dataDir, tag string, create bool) (RetentionUse, error) {
	r.events = append(r.events, fmt.Sprintf("begin:%v", create))
	if r.beginErr != nil {
		return nil, r.beginErr
	}
	return &fakeRetentionUse{id: "start-id", events: &r.events, finishErr: r.finishErr, spawnErr: r.spawnErr}, nil
}
func (r *retainedRuntime) EnsureThreadAddress(scope RepoScope, tag string, couch bool) error {
	r.events = append(r.events, "claim")
	return r.fakeRuntime.EnsureThreadAddress(scope, tag, couch)
}
func (r *retainedRuntime) LaunchSession(session, configDir, layout string) (int, error) {
	r.events = append(r.events, "launch")
	return r.fakeRuntime.LaunchSession(session, configDir, layout)
}
func (r *retainedRuntime) AttachSession(session, configDir string) (int, error) {
	r.events = append(r.events, "attach")
	return r.fakeRuntime.AttachSession(session, configDir)
}

func TestRetentionCreateReservesBeforeEffectsAndFinishesSuccess(t *testing.T) {
	r := &retainedRuntime{fakeRuntime: newFakeRuntime()}
	opts := baseOpts(LaunchArgs{Agent: "claude", ForcedTag: "work"})
	var output bytes.Buffer
	code, err := RunLaunch(opts, r, &output)
	if err != nil || code != 0 {
		t.Fatalf("launch %d %v %s", code, err, output.String())
	}
	if fmt.Sprint(r.events) != "[begin:true claim spawn launch finish:true]" {
		t.Fatalf("order %v", r.events)
	}
	if r.env["PAIR_RETENTION_START_ID"] != "start-id" || r.env["PAIR_RETENTION_PROTOCOL"] != "1" {
		t.Fatalf("missing child handshake env %+v", r.env)
	}
}

func TestRetentionCreateFailureStopsEffectsOrRetainsFailedStart(t *testing.T) {
	for _, failure := range []string{"begin", "spawn", "launch", "finish"} {
		t.Run(failure, func(t *testing.T) {
			r := &retainedRuntime{fakeRuntime: newFakeRuntime()}
			switch failure {
			case "begin":
				r.beginErr = errors.New("retention unavailable")
			case "spawn":
				r.spawnErr = errors.New("spawn boundary commit failed")
			case "launch":
				r.launchErr = errors.New("launch failed")
			case "finish":
				r.finishErr = errors.New("use commit failed")
			}
			var output bytes.Buffer
			code, _ := RunLaunch(baseOpts(LaunchArgs{Agent: "claude", ForcedTag: "work"}), r, &output)
			if code != 1 || output.Len() == 0 {
				t.Fatalf("failure lost %d %s", code, output.String())
			}
			if failure == "spawn" && r.launchCount != 0 {
				t.Fatal("launched after reservation persistence failed")
			}
			if failure == "begin" && len(r.threadClaims) != 0 {
				t.Fatal("claim before retention")
			}
			if failure == "launch" && r.events[len(r.events)-1] != "finish:false" {
				t.Fatal("failed launch marked use")
			}
		})
	}
}

func TestRetentionAttachHoldsLifetimeAndMarksOnlySuccess(t *testing.T) {
	for _, code := range []int{0, 1} {
		r := &retainedRuntime{fakeRuntime: newFakeRuntime()}
		r.attachCode = code
		opts := baseOpts(LaunchArgs{})
		got, err := AttachExistingSession(opts, opts.Env, r, "work", "pair-work", "claude")
		if err != nil || got != code {
			t.Fatalf("attach %d %v", got, err)
		}
		want := fmt.Sprintf("[begin:false attach finish:%v]", code == 0)
		if fmt.Sprint(r.events) != want {
			t.Fatalf("events %v", r.events)
		}
	}
}

func TestRetentionOSRuntimeKeepsUncertainStartAfterParentLeaseCloses(t *testing.T) {
	t.Setenv("PAIR_SCOPE_KEY", "")
	root := t.TempDir()
	data := filepath.Join(root, "repos", "scope")
	runtime := NewScopedOSRuntime(root, data, "/assets")
	use, err := runtime.BeginRetention(data, "work", true)
	if err != nil {
		t.Fatal(err)
	}
	guard := use.(*launchRetention)
	state, err := guard.coordinator.ReadOwner(guard.owner)
	if err != nil || len(state.Starts) != 1 || len(state.Processes) != 1 {
		t.Fatalf("missing reservation/lifetime %+v %v", state, err)
	}
	before := state.Activity.LastUse
	if err := use.BeforeSpawn(); err != nil {
		t.Fatal(err)
	}
	if err := use.Finish(false); err != nil {
		t.Fatal(err)
	}
	state, err = guard.coordinator.ReadOwner(guard.owner)
	if err != nil || len(state.Starts) != 1 || len(state.Processes) != 0 || !state.Activity.LastUse.Equal(before) {
		t.Fatalf("failed spawned launch lost protection/refreshed use %+v %v", state, err)
	}
}

func TestRetentionOSRuntimeCancelsOnlyPreSpawnAndRecordsSuccessfulAttach(t *testing.T) {
	t.Setenv("PAIR_SCOPE_KEY", "")
	root := t.TempDir()
	runtime := NewScopedOSRuntime(root, root, "/assets")
	use, err := runtime.BeginRetention(root, "work", true)
	if err != nil {
		t.Fatal(err)
	}
	guard := use.(*launchRetention)
	before, err := guard.coordinator.ReadOwner(guard.owner)
	if err != nil {
		t.Fatal(err)
	}
	if err := use.Finish(false); err != nil {
		t.Fatal(err)
	}
	state, err := guard.coordinator.ReadOwner(guard.owner)
	if err != nil || len(state.Starts) != 0 || len(state.Processes) != 0 || !state.Activity.LastUse.Equal(before.Activity.LastUse) {
		t.Fatalf("pre-spawn cancel %+v %v", state, err)
	}
	use, err = runtime.BeginRetention(root, "work", false)
	if err != nil {
		t.Fatal(err)
	}
	guard = use.(*launchRetention)
	at := before.Activity.LastUse.Add(time.Hour)
	guard.coordinator.Now = func() time.Time { return at }
	if err := use.Finish(true); err != nil {
		t.Fatal(err)
	}
	state, err = guard.coordinator.ReadOwner(guard.owner)
	if err != nil || len(state.Intents) != 0 || len(state.Processes) != 0 || !state.Activity.LastUse.Equal(at) {
		t.Fatalf("successful attach %+v %v", state, err)
	}
}

func TestRetentionOSRuntimeRejectsOutsideRootBeforeCreatingIt(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "not-created")
	runtime := NewScopedOSRuntime(root, root, "/assets")
	if _, err := runtime.BeginRetention(outside, "work", true); err == nil {
		t.Fatal("outside namespace accepted")
	}
	if _, err := os.Stat(outside); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("invalid namespace created")
	}
}

func TestRetentionOSRuntimeRejectsScopeSymlinkBeforeEffects(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, "repos")); err != nil {
		t.Fatal(err)
	}
	data := filepath.Join(root, "repos", "scope")
	runtime := NewScopedOSRuntime(root, data, "/assets")
	if _, err := runtime.BeginRetention(data, "work", true); err == nil {
		t.Fatal("symlink namespace accepted")
	}
	if _, err := os.Stat(filepath.Join(outside, "scope")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("retention escaped before validating physical namespace")
	}
}

func (r *retainedRuntime) TakeRestartMarker(session string) (RestartMarker, bool) {
	if r.traceTail {
		r.events = append(r.events, "restart-read")
	}
	return r.fakeRuntime.TakeRestartMarker(session)
}
func (f *fakeRetentionUse) WriteChanged(target string, write func() (bool, error)) error {
	*f.events = append(*f.events, "write-use")
	_, err := write()
	return err
}

func TestRetentionHeldThroughPostHandoffReads(t *testing.T) {
	r := &retainedRuntime{fakeRuntime: newFakeRuntime(), traceTail: true}
	var out bytes.Buffer
	code, _ := RunLaunch(baseOpts(LaunchArgs{Agent: "claude", ForcedTag: "work"}), r, &out)
	if code != 0 {
		t.Fatal(out.String())
	}
	if fmt.Sprint(r.events) != "[begin:true claim spawn launch restart-read finish:true]" {
		t.Fatalf("released before cleanup/restart reads: %v", r.events)
	}
}
func TestRetentionContinuationWriteIsUseEvenWhenLaunchFails(t *testing.T) {
	r := &retainedRuntime{fakeRuntime: newFakeRuntime()}
	r.launchErr = errors.New("failed after seed")
	opts := baseOpts(LaunchArgs{Agent: "claude", ForcedTag: "work"})
	opts.ContinueText = "preserve this"
	var out bytes.Buffer
	code, _ := RunLaunch(opts, r, &out)
	if code != 1 {
		t.Fatal("expected launch failure")
	}
	if !strings.Contains(fmt.Sprint(r.events), "write-use") {
		t.Fatalf("successful seed write lacked independent use: %v", r.events)
	}
}

func TestRetentionRestartProtectsNewTagBeforeReadingIt(t *testing.T) {
	r := &retainedRuntime{fakeRuntime: newFakeRuntime(), traceTail: true}
	r.restartMarkers["📁work-old"] = RestartMarker{Tag: "new", Agent: "claude"}
	var out bytes.Buffer
	code, _ := RunLaunch(baseOpts(LaunchArgs{Agent: "claude", ForcedTag: "old"}), r, &out)
	if code != 0 {
		t.Fatal(out.String())
	}
	if !strings.Contains(fmt.Sprint(r.events), "restart-read begin:false finish:false finish:true") {
		t.Fatalf("new restart tag lacked non-use lifetime: %v", r.events)
	}
}

func TestRetentionOSRuntimeSupportsExplicitSelectedDirectoryOverride(t *testing.T) {
	t.Setenv("PAIR_SCOPE_KEY", "")
	defaultRoot := t.TempDir()
	selected := filepath.Join(t.TempDir(), "custom-pair-data")
	runtime := NewScopedOSRuntime(defaultRoot, selected, "/assets")
	use, err := runtime.BeginRetention(selected, "work", false)
	if err != nil {
		t.Fatal(err)
	}
	guard := use.(*launchRetention)
	physical, _ := filepath.EvalSymlinks(selected)
	if guard.owner.DataDir != physical || guard.owner.RepoScope != "" {
		t.Fatalf("override misowned %+v", guard.owner)
	}
	if err := use.Finish(false); err != nil {
		t.Fatal(err)
	}
}

func TestRetentionBackgroundAttachDoesNotRefreshUse(t *testing.T) {
	t.Setenv("PAIR_RETENTION_BACKGROUND", "1")
	t.Setenv("PAIR_SCOPE_KEY", "")
	root := t.TempDir()
	runtime := NewScopedOSRuntime(root, root, "/assets")
	use, err := runtime.BeginRetention(root, "work", false)
	if err != nil {
		t.Fatal(err)
	}
	guard := use.(*launchRetention)
	before, err := guard.coordinator.ReadOwner(guard.owner)
	if err != nil {
		t.Fatal(err)
	}
	guard.coordinator.Now = func() time.Time { return before.Activity.LastUse.Add(time.Hour) }
	if err := use.Finish(true); err != nil {
		t.Fatal(err)
	}
	after, err := guard.coordinator.ReadOwner(guard.owner)
	if err != nil || !after.Activity.LastUse.Equal(before.Activity.LastUse) {
		t.Fatalf("background attach refreshed %+v %v", after, err)
	}
}

func TestRetentionSeedCommitFailureLeavesBlockingIntent(t *testing.T) {
	t.Setenv("PAIR_SCOPE_KEY", "")
	root := t.TempDir()
	runtime := NewScopedOSRuntime(root, root, "/assets")
	use, err := runtime.BeginRetention(root, "work", true)
	if err != nil {
		t.Fatal(err)
	}
	guard := use.(*launchRetention)
	writes := 0
	guard.coordinator.BeforePersist = func() error {
		writes++
		if writes == 2 {
			return errors.New("activity commit failed")
		}
		return nil
	}
	target := filepath.Join(root, "draft-work.md")
	if err := guard.WriteChanged(target, func() (bool, error) { return true, os.WriteFile(target, []byte("saved seed"), 0600) }); err == nil {
		t.Fatal("expected metadata failure after saved content")
	}
	if err := guard.Finish(false); err != nil {
		t.Fatal(err)
	}
	state, err := guard.coordinator.ReadOwner(guard.owner)
	if err != nil || len(state.Intents) != 1 {
		t.Fatalf("saved content lost blocking intent %+v %v", state, err)
	}
}
