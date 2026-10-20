package couchscratch

import (
	"encoding/json"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"
	"github.com/xianxu/pair/cmd/internal/strictjson"
)

// SettingsLayer binds relative paths to the native settings scope.
type SettingsLayer struct {
	Data []byte
	Base string
}

// ProjectClaudeSettings projects only standard local sandbox settings.
func ProjectClaudeSettings(home string, layers []SettingsLayer) SandboxPolicy {
	p := SandboxPolicy{Known: true}
	for _, layer := range layers {
		if strictjson.RejectDuplicateKeys(layer.Data) != nil {
			return SandboxPolicy{}
		}
		var doc map[string]any
		if json.Unmarshal(layer.Data, &doc) != nil || doc == nil {
			return SandboxPolicy{}
		}
		raw, exists := doc["sandbox"]
		if !exists {
			continue
		}
		s, ok := raw.(map[string]any)
		if !ok {
			return SandboxPolicy{}
		}
		if raw, exists = s["enabled"]; exists {
			p.Enabled, ok = raw.(bool)
			if !ok {
				return SandboxPolicy{}
			}
		}
		raw, exists = s["filesystem"]
		if !exists {
			continue
		}
		fs, ok := raw.(map[string]any)
		if !ok {
			return SandboxPolicy{}
		}
		// This narrow projection does not interpret filesystem-layer overrides.
		if _, exists := fs["disabled"]; exists {
			return SandboxPolicy{}
		}
		for key, dest := range map[string]*[]string{"allowWrite": &p.Allowed, "denyWrite": &p.Denied} {
			raw, exists := fs[key]
			if !exists {
				continue
			}
			paths, ok := raw.([]any)
			if !ok {
				return SandboxPolicy{}
			}
			for _, item := range paths {
				path, ok := item.(string)
				if !ok || path == "" || strings.ContainsAny(path, "\x00*?[]\\") {
					return SandboxPolicy{}
				}
				if path == "~" || strings.HasPrefix(path, "~/") {
					if !filepath.IsAbs(home) {
						return SandboxPolicy{}
					}
					path = filepath.Join(home, strings.TrimPrefix(strings.TrimPrefix(path, "~"), "/"))
				} else if !filepath.IsAbs(path) {
					if !filepath.IsAbs(layer.Base) {
						return SandboxPolicy{}
					}
					path = filepath.Join(layer.Base, path)
				}
				*dest = append(*dest, filepath.Clean(path))
			}
		}
	}
	return p
}

// ProjectCodexSettings projects only the small TMPDIR-related policy surface.
func ProjectCodexSettings(layers [][]byte) SandboxPolicy {
	p := SandboxPolicy{Known: true}
	for _, raw := range layers {
		var doc map[string]any
		if toml.Unmarshal(raw, &doc) != nil {
			return SandboxPolicy{}
		}
		if _, exists := doc["profile"]; exists {
			return SandboxPolicy{}
		}
		if value, exists := doc["sandbox_mode"]; exists {
			switch value {
			case "workspace-write", "danger-full-access":
			case "read-only":
				return SandboxPolicy{Known: true, Enabled: true}
			default:
				return SandboxPolicy{}
			}
		}
		if value, exists := doc["sandbox_workspace_write"]; exists {
			workspace, ok := value.(map[string]any)
			if !ok {
				return SandboxPolicy{}
			}
			if value, exists := workspace["exclude_tmpdir_env_var"]; exists {
				excluded, ok := value.(bool)
				if !ok {
					return SandboxPolicy{}
				}
				if excluded {
					return SandboxPolicy{Known: true, Enabled: true}
				}
			}
		}
	}
	// Native workspace-write includes the TMPDIR supplied by Couch. For this
	// optional root it imposes no additional directory-list restriction.
	return p
}
