package sessionwatch

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"time"

	"github.com/xianxu/pair/cmd/internal/adapt"
)

// Options are the watcher inputs after CLI/env resolution.
type Options struct {
	Agent    string
	Tag      string
	Cwd      string
	RepoRoot string
	RepoName string
	Args     []string
	Home     string
	DataDir  string
	PIDWait  time.Duration
	Timeout  time.Duration
	Poll     time.Duration
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

type sessionLedgerEntry struct {
	Agent      string    `json:"agent"`
	Args       []string  `json:"args"`
	SessionID  string    `json:"session_id"`
	Started    time.Time `json:"started"`
	LastActive time.Time `json:"last_active"`
	RepoRoot   string    `json:"repo_root"`
	RepoName   string    `json:"repo_name"`
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
	repoRoot := opts.RepoRoot
	if repoRoot == "" {
		repoRoot = opts.Cwd
	}
	repoName := opts.RepoName
	if repoName == "" {
		repoName = filepath.Base(filepath.Clean(repoRoot))
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
			if err := appendSessionLedger(rt, filepath.Join(opts.DataDir, "ledger-"+opts.Tag+".jsonl"), sessionLedgerEntry{
				Agent:      opts.Agent,
				Args:       StripResumeArgs(opts.Agent, opts.Args),
				SessionID:  result.ID,
				Started:    watchStart,
				LastActive: rt.Now(),
				RepoRoot:   repoRoot,
				RepoName:   repoName,
			}); err != nil {
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

func appendSessionLedger(rt Runtime, path string, entry sessionLedgerEntry) error {
	raw := ""
	if existing, err := rt.ReadFile(path); err == nil {
		raw = string(existing)
	}
	line, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	if raw != "" && !strings.HasSuffix(raw, "\n") {
		raw += "\n"
	}
	raw += string(line) + "\n"
	return rt.AtomicWrite(path, []byte(raw))
}

func freshPID(pidFile string, since time.Time, rt Runtime) (bool, time.Time) {
	mod, err := rt.ModTime(pidFile)
	if err != nil {
		return false, time.Time{}
	}
	return mod.Unix() >= since.Unix(), mod
}

func discover(spec AgentSpec, rootPID string, agentStart time.Time, legacyExisting map[string]bool, rt Runtime) SessionID {
	if rootPID != "" {
		nearMiss := SessionID{}
		pids, _ := rt.Descendants(rootPID)
		for _, pid := range pids {
			paths, _ := rt.LsofPaths(pid)
			for _, path := range paths {
				result := spec.Match(path)
				if result.ID != "" {
					return result
				}
				if result.NearMiss && !nearMiss.NearMiss {
					nearMiss = result
				}
			}
		}
		if !agentStart.IsZero() {
			if result := discoverByBirth(spec, agentStart, rt); result.ID != "" {
				return result
			} else if result.NearMiss && !nearMiss.NearMiss {
				nearMiss = result
			}
		}
		return nearMiss
	}
	nearMiss := SessionID{}
	files, _ := rt.ListFiles(spec.WatchDir)
	for _, file := range files {
		if legacyExisting[file] {
			continue
		}
		result := spec.Match(file)
		if result.ID != "" {
			return result
		}
		if result.NearMiss && !nearMiss.NearMiss {
			nearMiss = result
		}
	}
	return nearMiss
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
	for _, candidate := range candidates {
		if candidate.NearMiss {
			return candidate
		}
	}
	return SessionID{}
}
