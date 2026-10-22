package testcouch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/xianxu/pair/cmd/internal/procutil"
	"github.com/xianxu/pair/cmd/internal/testcouch/agentfixture"
)

// RetainedDescendants is a fixed native cohort parented beneath a genuine
// actor. ReleaseParent joins its intermediate parent, deliberately leaving its
// six exact children to the fleet's ownership and cleanup lifetime.
type RetainedDescendants struct {
	Processes              []agentfixture.RetainedProcess
	Parent                 agentfixture.RetainedProcess
	fleet                  *Fleet
	control, id, directory string
}

func (f *Fleet) RetainDescendants(ctx context.Context, n int) (*RetainedDescendants, error) {
	ctx, done, err := f.begin(ctx)
	if err != nil {
		return nil, err
	}
	defer done()
	control, err := f.actorDirectory(n)
	if err != nil {
		return nil, err
	}
	id := fmt.Sprintf("retain-%d", nextCommand.Add(1))
	if err = agentfixture.WriteCommand(control, agentfixture.Command{ID: id, Kind: agentfixture.Retain}); err != nil {
		return nil, err
	}
	cohort := &RetainedDescendants{fleet: f, control: control, id: id, directory: agentfixture.RetainedDirectory(control, id)}
	err = waitUntil(ctx, func() (bool, error) {
		raw, e := os.ReadFile(filepath.Join(cohort.directory, "parent-ready.json"))
		if errors.Is(e, os.ErrNotExist) {
			if result, re := agentfixture.ReadResult(control, id); re == nil {
				return false, fmt.Errorf("retention failed: %+v", result)
			}
			return false, nil
		}
		if e != nil {
			return false, e
		}
		if e = json.Unmarshal(raw, &cohort.Parent); e != nil {
			return false, e
		}
		return true, nil
	})
	if err != nil {
		return nil, err
	}
	cohort.Processes = make([]agentfixture.RetainedProcess, agentfixture.RetainedCount)
	for i := range cohort.Processes {
		raw, e := os.ReadFile(filepath.Join(cohort.directory, fmt.Sprintf("child-%d.json", i)))
		if e != nil {
			return nil, e
		}
		if e = json.Unmarshal(raw, &cohort.Processes[i]); e != nil {
			return nil, e
		}
	}
	for _, p := range append(append([]agentfixture.RetainedProcess(nil), cohort.Processes...), cohort.Parent) {
		if p.PID <= 0 || p.Generation == "" || procutil.Identity(strconv.Itoa(p.PID)) != p.Generation {
			return nil, fmt.Errorf("retained process incarnation unavailable: %+v", p)
		}
		f.record(RunEvent{Kind: "own", Resource: Resource{ID: strconv.Itoa(p.PID), Generation: p.Generation, Kind: "retained"}})
	}
	return cohort, f.ownershipError()
}

func (r *RetainedDescendants) ReleaseParent(ctx context.Context) error {
	ctx, done, err := r.fleet.begin(ctx)
	if err != nil {
		return err
	}
	defer done()
	if err = os.WriteFile(filepath.Join(r.directory, "release-parent"), nil, 0600); err != nil {
		return err
	}
	var result agentfixture.Result
	if err = waitUntil(ctx, func() (bool, error) {
		var e error
		result, e = agentfixture.ReadResult(r.control, r.id)
		if errors.Is(e, os.ErrNotExist) {
			return false, nil
		}
		return e == nil, e
	}); err != nil {
		return err
	}
	if result.Error != "" || result.ExitCode != 0 {
		return fmt.Errorf("retained parent: %+v", result)
	}
	if current := procutil.Identity(strconv.Itoa(r.Parent.PID)); current == r.Parent.Generation {
		return fmt.Errorf("retained parent did not exit")
	}
	return agentfixture.RetireResult(r.control, r.id)
}

// selectCrashCohort reuses the fixture cleanup's positive ownership closure,
// then removes exact protected generations. It never grants authority from a
// PID, process name, or absent environment marker alone.
func selectCrashCohort(rows []procutil.Process, marked map[int]bool, protected []Resource) ([]Resource, error) {
	owned, err := selectOwnedCohort(rows, OwnedRun{}, marked)
	if err != nil {
		return nil, err
	}
	denied := map[string]bool{}
	for _, r := range protected {
		denied[r.ID+"\x00"+r.Generation] = true
	}
	out := make([]Resource, 0, len(owned))
	for _, r := range owned {
		if !denied[r.ID+"\x00"+r.Generation] {
			out = append(out, r)
		}
	}
	return out, nil
}

// CrashActor ends only an already-observed old worker cohort, modeling a
// process crash before public reboot. Its retained descendants, steward and
// supervisor are excluded; no newly discovered process enters the signal set.
func (r *RetainedDescendants) CrashActor(ctx context.Context, n int, expected []Resource) error {
	f := r.fleet
	ctx, done, err := f.begin(ctx)
	if err != nil {
		return err
	}
	defer done()
	if n <= 0 {
		return fmt.Errorf("crash fixture requires a numbered worker")
	}
	actor, err := f.Agent(n)
	if err != nil {
		return err
	}
	hub, err := f.Agent(0)
	if err != nil {
		return err
	}
	worker := f.Workspaces.Workers[n]
	if worker == "" {
		return fmt.Errorf("unknown crash worker")
	}
	reader := procutil.NativeResourceReader{}
	rows, _, err := reader.Table(ctx)
	if err != nil {
		return err
	}
	marked := map[int]bool{}
	wanted := map[string]Resource{}
	for _, p := range expected {
		if p.ID == "" || p.Generation == "" {
			return fmt.Errorf("crash expected identity missing")
		}
		wanted[p.ID+"\x00"+p.Generation] = p
	}
	actorKey := strconv.Itoa(actor.PID) + "\x00" + actor.Generation
	if _, ok := wanted[actorKey]; !ok {
		return fmt.Errorf("crash cohort omits exact old actor")
	}
	for _, p := range rows {
		if _, ok := wanted[strconv.Itoa(p.PID)+"\x00"+p.Identity]; !ok {
			continue
		}
		env, _, e := reader.Environment(ctx, p, "PAIR_TESTCOUCH_RUN_ID", "COUCH_TREE")
		if e == nil && env["PAIR_TESTCOUCH_RUN_ID"] == f.RunID && env["COUCH_TREE"] == worker {
			marked[p.PID] = true
		}
	}
	protected := []Resource{{ID: strconv.Itoa(hub.PID), Generation: hub.Generation, Kind: "steward"}}
	for _, p := range r.Processes {
		protected = append(protected, Resource{ID: strconv.Itoa(p.PID), Generation: p.Generation, Kind: "retained"})
	}
	f.mu.Lock()
	console := f.console
	f.mu.Unlock()
	if console != nil {
		protected = append(protected, Resource{ID: strconv.Itoa(console.cmd.Process.Pid), Generation: console.generation, Kind: "supervisor"})
	}
	selected, err := selectCrashCohort(rows, marked, protected)
	if err != nil {
		return err
	}
	allowed := map[string]bool{}
	for _, p := range selected {
		allowed[p.ID+"\x00"+p.Generation] = true
	}
	state := OwnedRun{}
	for key, p := range wanted {
		live, e := currentCrashGeneration(rows, p)
		if e != nil {
			return e
		}
		if !live {
			if key == actorKey {
				return fmt.Errorf("old actor changed before fixture crash")
			}
			continue
		}
		if !allowed[key] {
			return fmt.Errorf("old crash cohort ownership unavailable for %s/%s", p.ID, p.Generation)
		}
		state = ReduceRun(state, RunEvent{Kind: "own", Resource: p})
	}
	// The existing native effect rechecks identity immediately before every
	// single-PID signal. No discovery or descendant expansion occurs here.
	state, err = stopOwned(ctx, nativeProcesses{}, state, syscall.SIGKILL)
	if err != nil {
		return err
	}
	return waitUntil(ctx, func() (bool, error) {
		var e error
		state, e = stopOwned(ctx, nativeProcesses{}, state, 0)
		return len(state.Resources) == 0, e
	})
}

// A complete native table distinguishes a generation already gone/replaced
// from an unreadable or duplicate row. Only positive absence can be skipped.
func currentCrashGeneration(rows []procutil.Process, target Resource) (bool, error) {
	pid, err := strconv.Atoi(target.ID)
	if err != nil || pid <= 0 || target.Generation == "" {
		return false, fmt.Errorf("invalid crash identity")
	}
	count := 0
	var row procutil.Process
	for _, p := range rows {
		if p.PID == pid {
			count++
			row = p
		}
	}
	if count == 0 {
		return false, nil
	}
	if count != 1 || row.Identity == "" {
		return false, fmt.Errorf("crash identity unknown or ambiguous for pid%d", pid)
	}
	return !row.Zombie && row.Identity == target.Generation, nil
}
