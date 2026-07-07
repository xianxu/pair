// pair-go is the primary Go CLI. Its launch route drives the native launcher
// in-process (bin/pair-shell retired, #99 M5c).
package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/xianxu/pair/cmd/internal/changelogcmd"
	"github.com/xianxu/pair/cmd/internal/clipcmd"
	"github.com/xianxu/pair/cmd/internal/continuationcmd"
	"github.com/xianxu/pair/cmd/internal/dispatcher"
	"github.com/xianxu/pair/cmd/internal/entrypoint"
	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/runtimebundle"
	"github.com/xianxu/pair/cmd/internal/scribecmd"
	"github.com/xianxu/pair/cmd/internal/sessionwatch"
	"github.com/xianxu/pair/cmd/internal/titlepoller"
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
	EmbeddedAssetRoot() (string, error)
	// LaunchNative runs the in-process native launcher (#99 M5c — the sole
	// launcher, bin/pair-shell retired) and returns its exit code. Behind the seam
	// so the launch route is unit-testable without real zellij.
	LaunchNative(args []string, root string, stdout, stderr io.Writer) int
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

	mode, dispatchArgs := entrypoint.ResolveInvocation(exe, args, dispatcher.DispatchNames())
	switch mode {
	case entrypoint.ModePublicPair:
		return runLegacyLaunch("pair", exe, args, stdout, stderr, rt)
	case entrypoint.ModePairGoLaunch:
		return runLegacyLaunch("pair-go launch", exe, args[1:], stdout, stderr, rt)
	default:
		if fam, rest, ok := dispatcher.Resolve(dispatchArgs); ok && fam.Status == "implemented" && fam.Streaming {
			return runStreamingSubcommand(fam.Name, rest, os.Stdin, stdout, stderr)
		}
		res := dispatcher.Dispatch(dispatchArgs)
		return writeResult(res, stdout, stderr)
	}
}

// runStreamingSubcommand routes subcommands that need real stdio — a live
// stderr consumer (changelog render), stdin (continuation, clip copy-on-select),
// or a long lifetime (session-watch, title) — straight to their runner with
// pass-through streams, bypassing the buffered dispatcher.Dispatch. `name` is
// the resolved family name (dispatcher.Resolve) and `rest` the args after the
// command tokens; stdin is a parameter so the seam is unit-testable with a fake.
// Only implemented streaming families reach here (gated by the caller), so an
// unknown name is a programming error.
func runStreamingSubcommand(name string, rest []string, stdin io.Reader, stdout, stderr io.Writer) int {
	switch name {
	case "wrap":
		return wrapcmd.Run(rest, stdin, stdout, stderr)
	case "scribe":
		return scribecmd.Run(rest, stdin, stdout, stderr)
	case "changelog render":
		return changelogcmd.Run(rest, stderr)
	case "continuation":
		return continuationcmd.Run(rest, stdin, stdout, stderr, time.Now)
	case "session-watch":
		return sessionwatch.RunCLI(rest, os.Getenv, stderr)
	case "title":
		return titlepoller.RunCLI(rest, os.Getenv, stderr)
	case "clip copy-on-select":
		return clipcmd.RunCopyOnSelectCLI(rest, stdin, os.Getenv, stderr)
	default:
		_, _ = fmt.Fprintf(stderr, "pair-go: %s: streaming subcommand has no runner wired\n", name)
		return 2
	}
}

func runLegacyLaunch(label string, executable string, args []string, stdout, stderr io.Writer, rt legacyRuntime) int {
	root, err := entrypoint.ResolveAssetRoot(entrypoint.AssetRootInput{
		PairHome:        rt.PairHome(),
		Executable:      executable,
		DefaultPairHome: rt.DefaultPairHome(),
		ValidRoot: func(root string) bool {
			return rt.Stat(entrypoint.ValidRootMarker(root)) == nil
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
				ValidRoot: func(root string) bool {
					return rt.Stat(entrypoint.ValidRootMarker(root)) == nil
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
	// The native launcher is the sole launcher (#99 M5c — bin/pair-shell retired).
	// It handles every flow in-process and always returns a real exit code.
	return rt.LaunchNative(args, root.Root, stdout, stderr)
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

// LaunchNative drives the in-process native launcher — the sole launcher now
// (#99 M5c); it always returns a real exit code (no shell to fall back to).
func (osLegacyRuntime) LaunchNative(args []string, root string, stdout, stderr io.Writer) int {
	code, _ := launcher.LaunchNative(args, root, stdout, stderr)
	return code
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
