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

	"github.com/xianxu/pair/cmd/internal/dispatcher"
	"github.com/xianxu/pair/cmd/internal/entrypoint"
	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/runtimebundle"
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

	switch entrypoint.ClassifyInvocation(exe, args) {
	case entrypoint.ModePublicPair:
		return runLegacyLaunch("pair", exe, args, stderr, rt)
	case entrypoint.ModePairGoLaunch:
		return runLegacyLaunch("pair-go launch", exe, args[1:], stderr, rt)
	default:
		res := dispatcher.Dispatch(args)
		return writeResult(res, stdout, stderr)
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
