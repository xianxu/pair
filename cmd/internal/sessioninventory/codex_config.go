package sessioninventory

import (
	"path/filepath"
	"strings"
)

// CodexInvocationMode preserves the distinct native TUI and exec selectors.
type CodexInvocationMode string

const (
	CodexModeTUI  CodexInvocationMode = "tui"
	CodexModeExec CodexInvocationMode = "exec"
)

type CodexConfigInvocation struct {
	StartupCWD                 string
	CWD, Profile               string
	Overrides, Enable, Disable []string
	Mode                       CodexInvocationMode
	Unsupported, OSS           bool
	LocalProvider              string
}

// CodexSelectionScope contains selection inputs only, never raw config or auth.
// Callers own CWDs and must copy it when retaining the scope across attempts.
type CodexSelectionScope struct {
	CodexHome, SQLiteHome, Provider, CWD string
	Worktrees                            bool
	CWDs                                 []string
	Mode                                 CodexInvocationMode
}
type CodexConfigStatus string

const (
	CodexConfigComplete      CodexConfigStatus = "complete"
	CodexConfigCompatibility CodexConfigStatus = "compatibility"
)

type CodexConfigReason string

const (
	CodexConfigCloudUnknown          CodexConfigReason = "cloud-unknown"
	CodexConfigManagedUnsupported    CodexConfigReason = "managed-input-unsupported"
	CodexConfigScopeUnsupported      CodexConfigReason = "invocation-scope-unsupported"
	CodexConfigInvalid               CodexConfigReason = "invalid-configuration"
	CodexConfigObservationIncomplete CodexConfigReason = "observation-incomplete"
)

type CodexConfigProjection struct {
	Scope  CodexSelectionScope
	Status CodexConfigStatus
	Reason CodexConfigReason
}
type codexConfigLayer struct {
	base   string
	values map[string]any
}
type codexConfigEvidence struct {
	scope     CodexSelectionScope
	layers    []codexConfigLayer
	envSQLite string
}

// projectCodexConfig follows rust-v0.162.1: paths bind before merging and an
// API-only local bootstrap excludes cloud auth. It never infers auth from keys.
func projectCodexConfig(e codexConfigEvidence) CodexConfigProjection {
	s := e.scope
	s.Worktrees = true
	var forced string
	invalid := false
	for _, l := range e.layers {
		for _, k := range []string{"sqlite_home", "model_provider", "forced_login_method", "profile"} {
			v, ok := l.values[k]
			if !ok {
				continue
			}
			str, ok := v.(string)
			if !ok {
				invalid = true
				continue
			}
			switch k {
			case "sqlite_home":
				s.SQLiteHome = codexConfigPath(str, l.base, e.scope.CodexHome)
			case "model_provider":
				if str == "" {
					invalid = true
				}
				s.Provider = str
			case "forced_login_method":
				forced = str
			case "profile":
				invalid = true
			}
		}
		if v, ok := l.values["features"]; ok {
			m, ok := v.(map[string]any)
			if !ok {
				invalid = true
				continue
			}
			if v, ok := m["worktrees"]; ok {
				b, ok := v.(bool)
				if !ok {
					invalid = true
				} else {
					s.Worktrees = b
				}
			}
		}
	}
	if s.SQLiteHome == "" {
		if v := strings.TrimSpace(e.envSQLite); v != "" {
			s.SQLiteHome = codexConfigPath(v, s.CWD, e.scope.CodexHome)
		} else {
			s.SQLiteHome = s.CodexHome
		}
	}
	if s.Provider == "" {
		s.Provider = "openai"
	}
	p := CodexConfigProjection{Scope: s, Status: CodexConfigCompatibility, Reason: CodexConfigCloudUnknown}
	if !codexProviderKnown(s.Provider, e.layers) {
		invalid = true
	}
	if invalid || (forced != "" && forced != "api" && forced != "chatgpt") {
		p.Reason = CodexConfigInvalid
		return p
	}
	if forced == "api" {
		p.Status = CodexConfigComplete
		p.Reason = ""
	}
	return p
}
func codexConfigPath(path, base, codexHome string) string {
	// codexHome's parent is not necessarily the user's home; callers resolve tilde
	// in parsed layers with their actual HOME before entering this pure projector.
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	return filepath.Clean(filepath.Join(base, path))
}

// Only native built-ins or explicitly configured named definitions can select
// history. This is a membership projection, not a second provider schema.
func codexProviderKnown(id string, layers []codexConfigLayer) bool {
	defs := map[string]any{}
	for _, l := range layers {
		if raw, exists := l.values["model_providers"]; exists {
			m, ok := raw.(map[string]any)
			if !ok {
				return false
			}
			codexMergeConfig(defs, m)
		}
	}
	if _, overridden := defs[id]; !overridden {
		switch id {
		case "openai", "amazon-bedrock", "amazon-bedrock-runtime", "ollama", "lmstudio":
			return true
		}
	}
	if id == "openai" || id == "ollama" || id == "lmstudio" {
		return false
	}
	d, ok := defs[id].(map[string]any)
	if !ok {
		return false
	}
	name, ok := d["name"].(string)
	return ok && strings.TrimSpace(name) != ""
}
