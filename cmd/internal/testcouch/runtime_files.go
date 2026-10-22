package testcouch

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

const maxRuntimeFileBytes int64 = 512 << 20

// Only explicitly selected utilities enter the private PATH. Agent names are
// physical fixture copies and never resolved through an ambient search path.
var runtimeUtilities = []string{
	"sh", "bash", "zsh", "env", "ps", "sed", "awk", "grep", "sort", "tr", "cut", "head", "tail", "cat", "mkdir", "dirname", "basename", "readlink", "uname", "stty", "true", "sleep", "tty", "which", "printf", "date", "chmod", "kill", "ln", "rm", "rmdir", "mktemp", "find", "du", "stat", "touch", "cp", "mv", "install",
}

const nativeTerminalConfig = "host_notification_protocol \"osc9\"\npane_frames false\nshow_startup_tips false\nshow_release_notes false\non_force_close \"quit\"\n"

func installRuntime(b Binaries, p RootsPlan) error {
	return installRuntimeWithTools(b, p, resolveTool)
}

type runtimeFile struct {
	name, source string
	copy         bool
	info         os.FileInfo
}

func installRuntimeWithTools(b Binaries, p RootsPlan, lookup func(string) (string, error)) error {
	physical, err := physicalRoots(p)
	if err != nil {
		return err
	}
	// Runtime files belong only to already-created factory directories.
	for _, key := range []string{"bin", "config"} {
		dir := physical.Mutable[key]
		if dir == "" {
			return fmt.Errorf("runtime requires owned %s directory", key)
		}
		info, err := os.Stat(dir)
		if err != nil {
			return err
		}
		if !info.IsDir() {
			return fmt.Errorf("runtime %s is not a directory", key)
		}
	}
	bin, config := physical.Mutable["bin"], filepath.Join(physical.Mutable["config"], "config.kdl")
	entries, err := os.ReadDir(bin)
	if err != nil {
		return err
	}
	if len(entries) != 0 {
		return fmt.Errorf("runtime bin must be fresh and empty: %s", bin)
	}
	if _, err := os.Lstat(config); !os.IsNotExist(err) {
		return fmt.Errorf("runtime terminal config already exists: %s (%v)", config, err)
	}
	files := []runtimeFile{
		{name: "pair", source: b.Pair, copy: true}, {name: "couch", source: b.Couch, copy: true}, {name: "pair-launch-helper", source: b.Helper, copy: true},
		{name: "claude", source: b.Fixture, copy: true}, {name: "codex", source: b.Fixture, copy: true},
		{name: "git", source: b.Git}, {name: "sdlc", source: b.SDLC}, {name: "weave", source: b.Weave}, {name: "zellij", source: b.Zellij}, {name: "nvim", source: b.Nvim},
	}
	for _, name := range runtimeUtilities {
		path, err := lookup(name)
		if err != nil {
			return fmt.Errorf("runtime utility %s: %w", name, err)
		}
		files = append(files, runtimeFile{name: name, source: path})
	}
	// Inspect every source and destination before any installation effect. All
	// candidates must be real executables; a source symlink is not a candidate.
	for i := range files {
		file := &files[i]
		info, err := os.Lstat(file.source)
		if err != nil {
			return fmt.Errorf("runtime source %s: %w", file.name, err)
		}
		if err := validateToolFacts(file.source, info.Mode()); err != nil {
			return err
		}
		if info.Size() > maxRuntimeFileBytes {
			return fmt.Errorf("runtime source %s exceeds %d bytes", file.name, maxRuntimeFileBytes)
		}
		if file.copy {
			readable, err := os.Open(file.source)
			if err != nil {
				return fmt.Errorf("runtime source %s is unreadable: %w", file.name, err)
			}
			current, statErr := readable.Stat()
			closeErr := readable.Close()
			if statErr != nil || closeErr != nil {
				return errors.Join(statErr, closeErr)
			}
			if !os.SameFile(info, current) {
				return fmt.Errorf("runtime source %s changed during validation", file.name)
			}
		}
		file.info = info
		if _, err := os.Lstat(filepath.Join(bin, file.name)); !os.IsNotExist(err) {
			return fmt.Errorf("runtime destination already exists: %s (%v)", file.name, err)
		}
	}
	for _, file := range files {
		destination := filepath.Join(bin, file.name)
		if file.copy {
			if err := copyRuntimeExecutable(file.source, destination, file.info); err != nil {
				return fmt.Errorf("install %s: %w", file.name, err)
			}
		} else if err := os.Symlink(file.source, destination); err != nil {
			return err
		}
	}
	// This governs direct native-terminal control. Pair's launcher explicitly
	// selects opts.PairHome/zellij in createflow.go and session_handoff_os.go,
	// retaining its product configuration regardless of ZELLIJ_CONFIG_DIR.
	f, err := os.OpenFile(config, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	_, writeErr := io.WriteString(f, nativeTerminalConfig)
	return errors.Join(writeErr, f.Close())
}

func copyRuntimeExecutable(source, destination string, expected os.FileInfo) error {
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	info, err := in.Stat()
	if err != nil {
		return errors.Join(err, in.Close())
	}
	if !os.SameFile(info, expected) || !info.Mode().IsRegular() || info.Size() > maxRuntimeFileBytes {
		return errors.Join(fmt.Errorf("runtime source changed since validation: %s", source), in.Close())
	}
	out, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0700)
	if err != nil {
		return errors.Join(err, in.Close())
	}
	n, copyErr := io.CopyBuffer(out, io.LimitReader(in, maxRuntimeFileBytes+1), make([]byte, 32<<10))
	if n > maxRuntimeFileBytes {
		copyErr = errors.Join(copyErr, fmt.Errorf("runtime source grew beyond %d bytes", maxRuntimeFileBytes))
	}
	if n != expected.Size() {
		copyErr = errors.Join(copyErr, fmt.Errorf("runtime source size changed during copy: %s", source))
	}
	return errors.Join(copyErr, out.Chmod(0700), in.Close(), out.Close())
}
