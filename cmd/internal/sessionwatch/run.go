package sessionwatch

import (
	"path/filepath"
	"strings"
	"time"

	"github.com/xianxu/pair/cmd/internal/adapt"
)

// Options are the watcher inputs after CLI/env resolution.
type Options struct {
	Agent   string
	Tag     string
	Cwd     string
	Args    []string
	Home    string
	DataDir string
	PIDWait time.Duration
	Timeout time.Duration
	Poll    time.Duration
}

// Runtime is the IO boundary for the session watcher.
type Runtime interface {
	Now() time.Time
	Sleep(time.Duration)
	ReadFile(path string) ([]byte, error)
	ModTime(path string) (time.Time, error)
	BirthTime(path string) (time.Time, error)
	ListFiles(root string) ([]string, error)
	Descendants(root string) ([]string, error)
	LsofPaths(pid string) ([]string, error)
	ProcessAlive(pid string) bool
	AtomicWrite(path string, data []byte) error
	Log(outcome adapt.Outcome, detail string)
}

// Run discovers the async agent session id and writes config-<tag>-<agent>.json.
func Run(opts Options, rt Runtime) error {
	spec, ok := SpecForAgent(opts.Agent, opts.Home)
	if !ok || opts.Tag == "" || opts.DataDir == "" {
		return nil
	}
	if opts.PIDWait <= 0 {
		opts.PIDWait = 2 * time.Second
	}
	if opts.Timeout <= 0 {
		opts.Timeout = 60 * time.Second
	}
	if opts.Poll <= 0 {
		opts.Poll = 100 * time.Millisecond
	}

	watchStart := rt.Now()
	pidFile := filepath.Join(opts.DataDir, "agent-pid-"+opts.Tag)
	out := filepath.Join(opts.DataDir, "config-"+opts.Tag+"-"+opts.Agent+".json")

	pidDeadline := watchStart.Add(opts.PIDWait)
	for {
		if fresh, _ := freshPID(pidFile, watchStart, rt); fresh {
			break
		}
		if !rt.Now().Before(pidDeadline) {
			break
		}
		rt.Sleep(opts.Poll)
	}

	rootPID := ""
	agentStart := time.Time{}
	if fresh, mod := freshPID(pidFile, watchStart, rt); fresh {
		if data, err := rt.ReadFile(pidFile); err == nil {
			rootPID = strings.TrimSpace(string(data))
			agentStart = mod
		}
	}

	legacyExisting := map[string]bool{}
	if rootPID == "" {
		files, _ := rt.ListFiles(spec.WatchDir)
		for _, file := range files {
			legacyExisting[file] = true
		}
	}

	nmLogged := false
	deadline := watchStart.Add(opts.Timeout)
	for rt.Now().Before(deadline) {
		if rootPID != "" && !rt.ProcessAlive(rootPID) {
			return nil
		}

		result := discover(spec, rootPID, agentStart, legacyExisting, rt)
		if result.ID != "" {
			payload, err := ConfigJSON(ConfigPayload{
				Agent:     opts.Agent,
				Args:      StripResumeArgs(opts.Agent, opts.Args),
				SessionID: result.ID,
			})
			if err != nil {
				return err
			}
			if err := rt.AtomicWrite(out, payload); err != nil {
				return err
			}
			rt.Log(adapt.Fired, "session_id="+result.ID)
			return nil
		}
		if result.NearMiss && !nmLogged {
			rt.Log(adapt.NearMiss, "matched session file but no id extracted: "+filepath.Base(result.Path))
			nmLogged = true
		}

		rt.Sleep(opts.Poll)
	}

	rt.Log(adapt.Fail, "no session id within 60s deadline (agent="+opts.Agent+")")
	return nil
}

func freshPID(pidFile string, since time.Time, rt Runtime) (bool, time.Time) {
	mod, err := rt.ModTime(pidFile)
	if err != nil {
		return false, time.Time{}
	}
	return !mod.Before(since), mod
}

func discover(spec AgentSpec, rootPID string, agentStart time.Time, legacyExisting map[string]bool, rt Runtime) SessionID {
	if rootPID != "" {
		pids, _ := rt.Descendants(rootPID)
		for _, pid := range pids {
			paths, _ := rt.LsofPaths(pid)
			for _, path := range paths {
				if result := spec.Match(path); result.ID != "" || result.NearMiss {
					return result
				}
			}
		}
		if !agentStart.IsZero() {
			return discoverByBirth(spec, agentStart, rt)
		}
		return SessionID{}
	}
	files, _ := rt.ListFiles(spec.WatchDir)
	for _, file := range files {
		if legacyExisting[file] {
			continue
		}
		if result := spec.Match(file); result.ID != "" || result.NearMiss {
			return result
		}
	}
	return SessionID{}
}

func discoverByBirth(spec AgentSpec, agentStart time.Time, rt Runtime) SessionID {
	files, _ := rt.ListFiles(spec.WatchDir)
	candidates := make([]SessionID, 0, 1)
	for _, file := range files {
		birth, err := rt.BirthTime(file)
		if err != nil || birth.Before(agentStart) {
			continue
		}
		result := spec.Match(file)
		if result.Matched {
			candidates = append(candidates, result)
		}
	}
	if len(candidates) == 1 {
		return candidates[0]
	}
	return SessionID{}
}
