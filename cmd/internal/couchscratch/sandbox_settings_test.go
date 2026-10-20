package couchscratch

import (
	"fmt"
	"testing"
)

func TestProjectClaudeSettings(t *testing.T) {
	for _, tc := range []struct {
		name string
		data []string
		want bool
	}{
		{"absent", nil, true},
		{"operator user shape", []string{`{"sandbox":{"enabled":true,"filesystem":{"allowWrite":[".","..","/tmp","~/.npm","/Volumes/TestScratch"],"denyWrite":[]},"excludedCommands":["couch *","sdlc *"]},"future":{"n":9007199254740993}}`}, true},
		{"incident", []string{`{"sandbox":{"enabled":true,"filesystem":{"allowWrite":["/tmp"]}}}`}, false},
		{"merged arrays", []string{`{"sandbox":{"enabled":true,"filesystem":{"allowWrite":["/Volumes/TestScratch"]}}}`, `{"sandbox":{"filesystem":{"allowWrite":["/other"]}}}`}, true},
		{"local disable", []string{`{"sandbox":{"enabled":true}}`, `{"sandbox":{"enabled":false}}`}, true},
		{"deny overrides", []string{`{"sandbox":{"enabled":true,"filesystem":{"allowWrite":["/Volumes"]}}}`, `{"sandbox":{"filesystem":{"denyWrite":["/Volumes/TestScratch"]}}}`}, false},
		{"null enabled", []string{`{"sandbox":{"enabled":null}}`}, false},
		{"null filesystem", []string{`{"sandbox":{"filesystem":null}}`}, false},
		{"wrong array", []string{`{"sandbox":{"enabled":true,"filesystem":{"allowWrite":"/Volumes"}}}`}, false},
		{"duplicate", []string{`{"sandbox":{"enabled":true,"enabled":false}}`}, false},
		{"malformed", []string{`{`}, false},
		{"unknown glob", []string{`{"sandbox":{"enabled":true,"filesystem":{"allowWrite":["/Volumes/*"]}}}`}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var layers []SettingsLayer
			for _, raw := range tc.data {
				layers = append(layers, SettingsLayer{Data: []byte(raw), Base: "/project"})
			}
			p := ProjectClaudeSettings("/home/operator", layers)
			if got := ScratchAllowed("/Volumes/TestScratch", p); got != tc.want {
				t.Fatalf("allowed=%v policy=%+v want=%v", got, p, tc.want)
			}
		})
	}
}

func TestProjectCodexSettings(t *testing.T) {
	for _, tc := range []struct {
		raw  string
		want bool
	}{
		{"", true},
		{"sandbox_mode = 'workspace-write'", true},
		{"sandbox_mode = 'danger-full-access'", true},
		{"sandbox_mode = 'read-only'", false},
		{"[sandbox_workspace_write]\nexclude_tmpdir_env_var = true", false},
		{"[sandbox_workspace_write]\nexclude_tmpdir_env_var = false", true},
		{"[sandbox_workspace_write]\nexclude_tmpdir_env_var = 'false'", false},
		{"sandbox_mode = 'new-mode'", false},
		{"profile = 'restricted'", false},
		{"sandbox_mode = [", false},
		{"sandbox_mode='workspace-write'\nfuture_field = 9007199254740993", true},
	} {
		t.Run(fmt.Sprintf("%q", tc.raw), func(t *testing.T) {
			p := ProjectCodexSettings([][]byte{[]byte(tc.raw)})
			if got := ScratchAllowed("/Volumes/TestScratch", p); got != tc.want {
				t.Fatalf("allowed=%v policy=%+v want=%v", got, p, tc.want)
			}
		})
	}
}

func FuzzProjectSandboxSettings(f *testing.F) {
	f.Add([]byte(`{"sandbox":{"enabled":true}}`))
	f.Add([]byte("sandbox_mode='workspace-write'"))
	f.Fuzz(func(t *testing.T, raw []byte) {
		_ = ProjectClaudeSettings("/home", []SettingsLayer{{Data: raw, Base: "/project"}})
		_ = ProjectCodexSettings([][]byte{raw})
	})
}
