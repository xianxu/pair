package launcher

import (
	"time"
)

// Env is the explicit runtime environment for launch decisions.
type Env struct {
	Home             string
	XDGData          string
	Cwd              string
	RepoRoot         string
	Now              time.Time
	HistoryD         int
	DataDir          string
	CouchThreadScope string
	CouchThreadTag   string
}

// CouchHosted reports whether Couch launched this Pair process's session:
// Couch sets COUCH_THREAD_SCOPE and COUCH_THREAD_TAG on every thread it
// launches (couchcore/launch_existing.go), and either marks it hosted. The
// launcher's hosted refusals and Pair's hosted help wording share this rule
// (#282).
func (e Env) CouchHosted() bool { return e.CouchThreadScope != "" || e.CouchThreadTag != "" }

// CouchHostedEnv is CouchHosted read from a process environment.
func CouchHostedEnv(getenv func(string) string) bool {
	return Env{CouchThreadScope: getenv("COUCH_THREAD_SCOPE"), CouchThreadTag: getenv("COUCH_THREAD_TAG")}.CouchHosted()
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
	if env.RepoRoot == "" {
		env.RepoRoot = env.Cwd
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
	base := DefaultTag(env.RepoRoot)
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
