package launcher

import (
	"time"
)

// Env is the explicit runtime environment for launch decisions.
type Env struct {
	Home     string
	XDGData  string
	Cwd      string
	Now      time.Time
	HistoryD int
	DataDir  string
}

// SessionSource supplies zellij session state.
type SessionSource interface {
	Snapshot() ([]Session, error)
}

// HistoricalScanner supplies recently touched Pair tags.
type HistoricalScanner interface {
	Scan(base string, cutoff time.Time) ([]HistoricalTag, error)
}

// LaunchOutcome is the domain outcome from the launcher core.
type LaunchOutcome struct {
	Args     LaunchArgs
	Env      Env
	Decision LaunchDecision
}

// Run builds a pure snapshot from injected sources and returns a domain launch
// outcome. The dispatcher maps this to process stdout/stderr/exit status.
func Run(argv []string, env Env, sessions SessionSource, history HistoricalScanner) (LaunchOutcome, error) {
	args, err := ParseArgs(argv)
	if err != nil {
		return LaunchOutcome{}, err
	}
	if env.DataDir == "" {
		env.DataDir = ResolveDataDir(env.Home, env.XDGData)
	}
	if env.HistoryD == 0 {
		env.HistoryD = 14
	}
	if env.Now.IsZero() {
		env.Now = time.Now()
	}

	sessionRows, err := sessions.Snapshot()
	if err != nil {
		return LaunchOutcome{}, err
	}
	base := DefaultTag(env.Cwd)
	historical, err := history.Scan(base, env.Now.Add(-time.Duration(env.HistoryD)*24*time.Hour))
	if err != nil {
		return LaunchOutcome{}, err
	}

	decision, err := DecideLaunch(args, SessionSnapshot{
		BaseTag:    base,
		Sessions:   sessionRows,
		Historical: historical,
	})
	if err != nil {
		return LaunchOutcome{}, err
	}
	return LaunchOutcome{Args: args, Env: env, Decision: decision}, nil
}
