package testcouch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

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
