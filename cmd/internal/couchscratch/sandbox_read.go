package couchscratch

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

// SettingsLimit caps each optional settings read at one MiB.
const SettingsLimit = 1 << 20

// SandboxSettings supplies non-credential paths and a replaceable file boundary.
type SandboxSettings struct {
	Agent, Home, Project, ClaudeHome, CodexHome, ManagedRoot string
	Argv                                                     []string
	ReadFile                                                 func(string) ([]byte, error)
}

// ReadSandboxPolicy reads only the supported local settings projection.
func ReadSandboxPolicy(in SandboxSettings) SandboxPolicy {
	if !filepath.IsAbs(in.Home) || !filepath.IsAbs(in.Project) {
		return SandboxPolicy{}
	}
	known, readOnly := sandboxInvocation(in.Agent, in.Argv)
	if !known {
		return SandboxPolicy{}
	}
	read := in.ReadFile
	if read == nil {
		read = readSandboxFile
	}
	load := func(path string) ([]byte, bool) {
		data, err := read(path)
		if errors.Is(err, os.ErrNotExist) {
			return nil, true
		}
		return data, err == nil && len(data) <= SettingsLimit
	}
	switch in.Agent {
	case "claude":
		managed := in.ManagedRoot
		if managed == "" {
			managed = "/Library/Application Support/ClaudeCode"
		}
		if _, err := os.Stat(managed); !errors.Is(err, os.ErrNotExist) {
			return SandboxPolicy{}
		}
		base := in.ClaudeHome
		if base == "" {
			base = filepath.Join(in.Home, ".claude")
		}
		if !filepath.IsAbs(base) {
			return SandboxPolicy{}
		}
		var layers []SettingsLayer
		for _, source := range []struct{ path, base string }{
			{filepath.Join(base, "settings.json"), base},
			{filepath.Join(in.Project, ".claude", "settings.json"), in.Project},
			{filepath.Join(in.Project, ".claude", "settings.local.json"), in.Project},
		} {
			data, ok := load(source.path)
			if !ok {
				return SandboxPolicy{}
			}
			if data != nil {
				layers = append(layers, SettingsLayer{Data: data, Base: source.base})
			}
		}
		return ProjectClaudeSettings(in.Home, layers)
	case "codex":
		base := in.CodexHome
		if base == "" {
			base = filepath.Join(in.Home, ".codex")
		}
		if !filepath.IsAbs(base) {
			return SandboxPolicy{}
		}
		var layers [][]byte
		for _, path := range []string{filepath.Join(base, "config.toml"), filepath.Join(in.Project, ".codex", "config.toml")} {
			data, ok := load(path)
			if !ok {
				return SandboxPolicy{}
			}
			if data != nil {
				layers = append(layers, data)
			}
		}
		if readOnly {
			return SandboxPolicy{Known: true, Enabled: true}
		}
		return ProjectCodexSettings(layers)
	}
	return SandboxPolicy{}
}

func sandboxInvocation(agent string, argv []string) (bool, bool) {
	readOnly := false
	for i := 0; i < len(argv); i++ {
		arg := argv[i]
		key, _, _ := strings.Cut(arg, "=")
		if agent == "claude" && (key == "--settings" || key == "--setting-sources") {
			return false, false
		}
		if agent != "codex" {
			continue
		}
		if key == "--config" || key == "--profile" || key == "--permission-profile" || strings.HasPrefix(arg, "-c") && !strings.HasPrefix(arg, "--") || strings.HasPrefix(arg, "-p") && !strings.HasPrefix(arg, "--") {
			return false, false
		}
		if strings.HasPrefix(arg, "-s") && !strings.HasPrefix(arg, "--") && key != "-s" {
			return false, false
		}
		if key == "--sandbox" || key == "-s" {
			_, value, inline := strings.Cut(arg, "=")
			if !inline {
				i++
				if i >= len(argv) {
					return false, false
				}
				value = argv[i]
			}
			if value != "workspace-write" && value != "danger-full-access" && value != "read-only" {
				return false, false
			}
			readOnly = readOnly || value == "read-only"
		}
	}
	return true, readOnly
}

// readSandboxFile cannot hang opening a FIFO and never writes native settings.
func readSandboxFile(path string) ([]byte, error) {
	f, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !st.Mode().IsRegular() || st.Size() > SettingsLimit {
		return nil, errors.New("unsupported settings file")
	}
	return io.ReadAll(io.LimitReader(f, SettingsLimit+1))
}
