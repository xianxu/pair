// pair-go is the development dispatcher for the future primary Go CLI. Its
// launch route is a compatibility handoff to the current shell launcher.
package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/xianxu/pair/cmd/internal/changelogcmd"
	"github.com/xianxu/pair/cmd/internal/continuationcmd"
	"github.com/xianxu/pair/cmd/internal/dispatcher"
	"github.com/xianxu/pair/cmd/internal/entrypoint"
	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/runtimebundle"
	"github.com/xianxu/pair/cmd/internal/scribecmd"
	"github.com/xianxu/pair/cmd/internal/sessionwatch"
	"github.com/xianxu/pair/cmd/internal/wrapcmd"
)

var defaultPairHome string

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	return runWithLegacyRuntime(args, stdout, stderr, osLegacyRuntime{})
}

type legacyRuntime interface {
	Executable() (string, error)
	PairHome() string
	DefaultPairHome() string
	Stat(path string) error
	Environ() []string
	EmbeddedAssetRoot() (string, error)
	Exec(label string, path string, argv []string, env []string) int
}

func runWithLegacyRuntime(args []string, stdout, stderr io.Writer, rt legacyRuntime) int {
	exe, err := rt.Executable()
	if err != nil {
		if len(args) > 0 && args[0] == "launch" {
			_, _ = fmt.Fprintf(stderr, "pair-go launch: cannot resolve current executable: %v\n", err)
			return 1
		}
		res := dispatcher.Dispatch(args)
		return writeResult(res, stdout, stderr)
	}

	switch entrypoint.ClassifyInvocation(exe, args, dispatcher.DispatchNames()) {
	case entrypoint.ModePublicPair:
		return runLegacyLaunch("pair", exe, args, stderr, rt)
	case entrypoint.ModePairGoLaunch:
		return runLegacyLaunch("pair-go launch", exe, args[1:], stderr, rt)
	default:
		if len(args) > 0 && dispatcher.IsImplemented(args[0]) && dispatcher.IsStreaming(args[0]) {
			return runStreamingSubcommand(args, os.Stdin, stdout, stderr)
		}
		res := dispatcher.Dispatch(args)
		return writeResult(res, stdout, stderr)
	}
}

// runStreamingSubcommand routes subcommands that need real stdio — a live
// stderr consumer (changelog), stdin (continuation), or a long lifetime
// (session-watch) — straight to their runner with pass-through streams,
// bypassing the buffered dispatcher.Dispatch. stdin is a parameter so the seam
// is unit-testable with a fake. Only implemented streaming subcommands reach
// here (gated by the caller), so an unknown arg is a programming error.
func runStreamingSubcommand(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	switch args[0] {
	case "wrap":
		return wrapcmd.Run(args[1:], stdin, stdout, stderr)
	case "scribe":
		return scribecmd.Run(args[1:], stdin, stdout, stderr)
	case "changelog":
		return changelogcmd.Run(args[1:], stderr)
	case "continuation":
		return continuationcmd.Run(args[1:], stdin, stdout, stderr, time.Now)
	case "session-watch":
		return sessionwatch.RunCLI(args[1:], os.Getenv, stderr)
	default:
		_, _ = fmt.Fprintf(stderr, "pair-go: %s: streaming subcommand has no runner wired\n", args[0])
		return 2
	}
}

func runLegacyLaunch(label string, executable string, args []string, stderr io.Writer, rt legacyRuntime) int {
	root, err := entrypoint.ResolveAssetRoot(entrypoint.AssetRootInput{
		PairHome:        rt.PairHome(),
		Executable:      executable,
		DefaultPairHome: rt.DefaultPairHome(),
		PairShellExists: func(root string) bool {
			return rt.Stat(entrypoint.PairShellPath(root)) == nil
		},
	})
	if err != nil {
		embeddedRoot, embeddedErr := rt.EmbeddedAssetRoot()
		if embeddedErr == nil && embeddedRoot != "" {
			root, err = entrypoint.ResolveAssetRoot(entrypoint.AssetRootInput{
				PairHome:        rt.PairHome(),
				Executable:      executable,
				DefaultPairHome: rt.DefaultPairHome(),
				EmbeddedRoot:    embeddedRoot,
				PairShellExists: func(root string) bool {
					return rt.Stat(entrypoint.PairShellPath(root)) == nil
				},
			})
		}
		if err != nil {
			if embeddedErr != nil {
				_, _ = fmt.Fprintf(stderr, "%s: embedded runtime extraction failed: %v\n", label, embeddedErr)
			}
			_, _ = fmt.Fprintf(stderr, "%s: %v; run make build or make install, or source ../ariadne/construct/dev-aliases.sh in a dev shell\n", label, err)
			return 1
		}
	}
	// Native launcher preview (#99 M2): under PAIR_NATIVE_LAUNCH, run the
	// in-process create path. It declines (ErrFallbackToShell) anything it
	// doesn't own yet — attach/pick, in-pane launches, unsupported verbs — so
	// the shell below stays the default until the M4 cutover flips it.
	if os.Getenv("PAIR_NATIVE_LAUNCH") != "" {
		if code, err := launcher.LaunchNative(args, root.Root, stderr); err == nil {
			return code
		}
	}

	req := entrypoint.ResolveLegacyLaunch(root, args)
	return rt.Exec(label, req.Path, req.Argv, withEnv(rt.Environ(), "PAIR_HOME", root.Root))
}

type osLegacyRuntime struct{}

func (osLegacyRuntime) Executable() (string, error) {
	return os.Executable()
}

func (osLegacyRuntime) PairHome() string {
	return os.Getenv("PAIR_HOME")
}

func (osLegacyRuntime) DefaultPairHome() string {
	return defaultPairHome
}

func (osLegacyRuntime) Stat(path string) error {
	path = filepath.Clean(path)
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("is a directory")
	}
	return nil
}

func (osLegacyRuntime) Environ() []string {
	return os.Environ()
}

func (osLegacyRuntime) EmbeddedAssetRoot() (string, error) {
	dataDir := runtimeDataDir(os.Getenv("PAIR_DATA_DIR"), os.Getenv("HOME"), os.Getenv("XDG_DATA_HOME"))
	res, err := runtimebundle.Extract(runtimebundle.StoreInput{
		StoreRoot: filepath.Join(dataDir, "runtime"),
		Manifest:  runtimebundle.EmbeddedManifest(),
		ReadAsset: runtimebundle.EmbeddedAsset,
		Keep:      2,
	})
	if err != nil {
		return "", err
	}
	return res.PairHome, nil
}

func runtimeDataDir(pairDataDir, home, xdgDataHome string) string {
	if pairDataDir != "" {
		return pairDataDir
	}
	return launcher.ResolveDataDir(home, xdgDataHome)
}

func (osLegacyRuntime) Exec(label string, path string, argv []string, env []string) int {
	if err := syscall.Exec(path, argv, env); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%s: exec %s failed: %v\n", label, path, err)
		return 1
	}
	return 0
}

func writeResult(res dispatcher.Result, stdout, stderr io.Writer) int {
	if res.Stdout != "" {
		_, _ = io.WriteString(stdout, res.Stdout)
	}
	if res.Stderr != "" {
		_, _ = io.WriteString(stderr, res.Stderr)
	}
	return res.ExitCode
}

func withEnv(env []string, key, value string) []string {
	prefix := key + "="
	out := make([]string, 0, len(env)+1)
	replaced := false
	for _, entry := range env {
		if strings.HasPrefix(entry, prefix) {
			if !replaced {
				out = append(out, prefix+value)
				replaced = true
			}
			continue
		}
		out = append(out, entry)
	}
	if !replaced {
		out = append(out, prefix+value)
	}
	return out
}
