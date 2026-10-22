package testcouch

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/xianxu/pair/cmd/internal/testcouch/agentfixture"
)

// Reporter is the diagnostic surface shared by test wrappers and standalone
// owners. It carries no fatal-exit or implicit cleanup authority.
type Reporter interface {
	Logf(string, ...any)
	Failed() bool
}

type OwnerConfig struct {
	// AdmissionContext bounds construction only; nil uses the fleet lifetime.
	AdmissionContext     context.Context
	ScratchParent        string
	Reporter             Reporter
	Denied               []string
	AllowScratchFallback bool
}

type quietReporter struct{}

func (quietReporter) Logf(string, ...any) {}
func (quietReporter) Failed() bool        { return false }

func legacyScratchParent() string {
	base := os.TempDir()
	if info, err := os.Stat("/Volumes/TestScratch"); err == nil && info.IsDir() {
		base = "/Volumes/TestScratch"
	}
	// Existing test wrappers permit the platform's conventional /tmp alias;
	// explicit standalone callers must instead name the physical directory.
	if physical, err := filepath.EvalSymlinks(base); err == nil {
		return physical
	}
	return base
}

func ownerDenied(extra []string) []string {
	denied := append([]string(nil), extra...)
	for _, key := range []string{"HOME", "COUCH_ISOLATED_ROOT", "COUCH_STORE_DIR", "COUCH_IDENTITY_DIR", "COUCH_PAIR_DATA_DIR", "PAIR_DATA_DIR", "COUCH_TREE", "ZELLIJ_SOCKET_DIR", "CODEX_HOME", "CLAUDE_CONFIG_DIR"} {
		if value := os.Getenv(key); value != "" {
			denied = append(denied, value)
		}
	}
	return denied
}

func validateScratchParent(parent string) error {
	if !filepath.IsAbs(parent) || filepath.Clean(parent) != parent {
		return fmt.Errorf("canonical physical absolute scratch parent required")
	}
	physical, err := filepath.EvalSymlinks(parent)
	if err != nil {
		return fmt.Errorf("scratch parent: %w", err)
	}
	if physical != parent {
		return fmt.Errorf("physical scratch parent required, not alias %q", parent)
	}
	info, err := os.Stat(parent)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("scratch parent must be an existing directory")
	}
	return nil
}

func socketPlacementFits(p RootsPlan) bool {
	return len(filepath.Join(p.Root, "m", "wrapper-2147483647-0000000000000000000000000000000000000000.sock")) <= 103
}

// NewOwned constructs the same isolated owner used by New without retaining a
// testing.TB. The caller owns Close and AssertCleaned even when an error returns
// a nonnil partial Fleet. ctx owns the fleet lifetime, not its cleanup budget.
func NewOwned(ctx context.Context, b Binaries, o Options, cfg OwnerConfig) (*Fleet, error) {
	if ctx == nil {
		return nil, fmt.Errorf("fleet lifetime context required")
	}
	admission := cfg.AdmissionContext
	if admission == nil {
		admission = ctx
	}
	check := func() error {
		if err := ctx.Err(); err != nil {
			return err
		}
		return admission.Err()
	}
	if err := check(); err != nil {
		return nil, err
	}
	// Both cancellation sources bound construction. Completing admission revokes
	// this temporary context, not the independently owned fleet lifetime.
	construction, cancelConstruction := context.WithCancel(admission)
	stopLifetime := context.AfterFunc(ctx, cancelConstruction)
	defer stopLifetime()
	defer cancelConstruction()
	if err := validateOptions(o); err != nil {
		return nil, err
	}
	if err := check(); err != nil {
		return nil, err
	}
	if err := validateScratchParent(cfg.ScratchParent); err != nil {
		return nil, err
	}
	if o.AgentKind == "" {
		o.AgentKind = "claude"
	}
	if o.RuntimeBudget == 0 {
		o.RuntimeBudget = 120 * time.Second
	}
	var nonce [16]byte
	if err := check(); err != nil {
		return nil, err
	}
	if _, err := rand.Read(nonce[:]); err != nil {
		return nil, err
	}
	denied := ownerDenied(cfg.Denied)
	if err := check(); err != nil {
		return nil, err
	}
	p, err := allocateRoots(cfg.ScratchParent, denied)
	if err != nil {
		return nil, err
	}
	lifetime, cancel := context.WithCancel(ctx)
	reporter := cfg.Reporter
	if reporter == nil {
		reporter = quietReporter{}
	}
	f := &Fleet{Roots: p, Options: o, Binaries: b, RunID: hex.EncodeToString(nonce[:]), t: reporter, ctx: lifetime, cancel: cancel, idle: make(chan struct{})}
	close(f.idle)
	if err := check(); err != nil {
		return f, err
	}
	if !socketPlacementFits(p) {
		if !cfg.AllowScratchFallback {
			return f, fmt.Errorf("scratch parent exceeds native wrapper socket path limit")
		}
		short, err := filepath.EvalSymlinks("/tmp")
		if err != nil {
			return f, err
		}
		// No process has been started. Remove only this unpublished fresh lease
		// before selecting the legacy short socket directory.
		if err := check(); err != nil {
			return f, err
		}
		if err := os.RemoveAll(p.Root); err != nil {
			return f, fmt.Errorf("remove unstarted long socket root: %w", err)
		}
		if err := check(); err != nil {
			cancel()
			return nil, err
		}
		p, err = allocateRoots(short, denied)
		if err != nil {
			// The original lease was removed, so there is no hidden owner to return.
			cancel()
			return nil, err
		}
		f.Roots = p
		if err := check(); err != nil {
			return f, err
		}
		if !socketPlacementFits(p) {
			return f, fmt.Errorf("fallback exceeds native wrapper socket path limit")
		}
	}
	if err := check(); err != nil {
		return f, err
	}
	if err := installRuntimeContext(construction, b, p); err != nil {
		return f, err
	}
	if err := check(); err != nil {
		return f, err
	}
	if err := os.WriteFile(filepath.Join(p.Root, agentfixture.OwnershipFile), []byte(f.RunID), 0600); err != nil {
		return f, err
	}
	if err := check(); err != nil {
		return f, err
	}
	f.Environment, err = runtimeEnvironment(p, p.Mutable["bin"], o.AgentKind, f.RunID, b.Source, filepath.Join(p.Mutable["bin"], "couch"))
	if err != nil {
		return f, err
	}
	if err := check(); err != nil {
		return f, err
	}
	f.record(RunEvent{Kind: "ready"})
	if err := errors.Join(check(), f.ownershipError()); err != nil {
		return f, err
	}
	reporter.Logf("native fleet isolation: root=%s run=%s agent=%s workers=%d", p.Root, f.RunID, o.AgentKind, o.Workers)
	return f, check()
}

// Done exposes the current owned supervisor's completion without polling or
// transferring its cleanup authority. An absent supervisor disables selection.
func (f *Fleet) Done() <-chan struct{} {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.console == nil {
		return nil
	}
	return f.console.done
}
