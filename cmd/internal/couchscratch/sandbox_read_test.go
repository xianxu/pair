package couchscratch

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/sys/unix"
)

func TestSandboxFileReadBoundary(t *testing.T) {
	root := t.TempDir()
	p := filepath.Join(root, "settings.json")
	data := []byte(`{"sandbox":{"enabled":false},"future":9007199254740993}`)
	if err := os.WriteFile(p, data, 0600); err != nil {
		t.Fatal(err)
	}
	before, err := os.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	got, err := readSandboxFile(p)
	if err != nil || string(got) != string(data) {
		t.Fatalf("read=%q error=%v", got, err)
	}
	after, err := os.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(before, after) || !before.ModTime().Equal(after.ModTime()) {
		t.Fatal("settings changed")
	}
	fifo := filepath.Join(root, "fifo")
	if err := unix.Mkfifo(fifo, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := readSandboxFile(fifo); err == nil {
		t.Fatal("special settings source admitted")
	}
	if err := os.WriteFile(p, make([]byte, SettingsLimit+1), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := readSandboxFile(p); err == nil {
		t.Fatal("oversized source admitted")
	}
}

func TestSandboxPolicyReadCountIndependentOfAllowances(t *testing.T) {
	for _, n := range []int{8, 16} {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			paths := []string{DefaultRoot}
			for i := 0; i < n; i++ {
				paths = append(paths, fmt.Sprintf("/other/%d", i))
			}
			data, err := json.Marshal(map[string]any{"sandbox": map[string]any{"enabled": true, "filesystem": map[string]any{"allowWrite": paths}}})
			if err != nil {
				t.Fatal(err)
			}
			reads, bytesRead := 0, 0
			in := SandboxSettings{Agent: "claude", Home: "/fixture/home", Project: "/fixture/repo", ManagedRoot: filepath.Join(t.TempDir(), "absent-managed")}
			in.ReadFile = func(string) ([]byte, error) { reads++; bytesRead += len(data); return data, nil }
			if !ScratchAllowed(DefaultRoot, ReadSandboxPolicy(in)) {
				t.Fatal("valid allowances refused")
			}
			if reads != 3 || bytesRead != 3*len(data) {
				t.Fatalf("n=%d reads=%d bytes=%d", n, reads, bytesRead)
			}
		})
	}
}

func TestSandboxInvocationProjection(t *testing.T) {
	for _, tc := range []struct {
		agent           string
		argv            []string
		known, readOnly bool
	}{
		{"claude", nil, true, false},
		{"claude", []string{"--settings={}"}, false, false},
		{"claude", []string{"--setting-sources", "user"}, false, false},
		{"codex", []string{"--sandbox", "workspace-write"}, true, false},
		{"codex", []string{"--sandbox=read-only"}, true, true},
		{"codex", []string{"-s=read-only"}, true, true},
		{"codex", []string{"-sread-only"}, false, false},
		{"codex", []string{"--sandbox"}, false, false},
		{"codex", []string{"--sandbox=other"}, false, false},
		{"codex", []string{"-cfoo=1"}, false, false},
		{"codex", []string{"--profile=p"}, false, false},
	} {
		known, readOnly := sandboxInvocation(tc.agent, tc.argv)
		if known != tc.known || readOnly != tc.readOnly {
			t.Fatalf("%s %q -> %v/%v want %v/%v", tc.agent, tc.argv, known, readOnly, tc.known, tc.readOnly)
		}
	}
}

func TestReadSandboxPolicySettingsScopes(t *testing.T) {
	root := t.TempDir()
	in := SandboxSettings{Home: filepath.Join(root, "home"), Project: filepath.Join(root, "repo"), ManagedRoot: filepath.Join(root, "managed"), Agent: "claude"}
	files := map[string][]byte{}
	reads := 0
	in.ReadFile = func(path string) ([]byte, error) {
		reads++
		if data, ok := files[path]; ok {
			return data, nil
		}
		return nil, os.ErrNotExist
	}
	user := filepath.Join(in.Home, ".claude", "settings.json")
	project := filepath.Join(in.Project, ".claude", "settings.json")
	local := filepath.Join(in.Project, ".claude", "settings.local.json")
	files[user] = []byte(`{"sandbox":{"enabled":true,"filesystem":{"allowWrite":["/Volumes/TestScratch"],"denyWrite":[]}},"unrelated":{"future":true}}`)
	if p := ReadSandboxPolicy(in); !ScratchAllowed(DefaultRoot, p) {
		t.Fatalf("operator settings silently disabled RAM: %+v", p)
	}
	if reads != 3 {
		t.Fatalf("read count=%d", reads)
	}
	files[project] = []byte(`{"sandbox":{"filesystem":{"denyWrite":["/Volumes"]}}}`)
	if ScratchAllowed(DefaultRoot, ReadSandboxPolicy(in)) {
		t.Fatal("project deny ignored")
	}
	files[local] = []byte(`{"sandbox":{"enabled":false}}`)
	if !ScratchAllowed(DefaultRoot, ReadSandboxPolicy(in)) {
		t.Fatal("local disabled ignored")
	}
	files[local] = []byte(`{`)
	if ReadSandboxPolicy(in).Known {
		t.Fatal("malformed local allowed")
	}
	delete(files, local)
	delete(files, project)
	in.Argv = []string{"--settings", "{}"}
	if ReadSandboxPolicy(in).Known {
		t.Fatal("unsupported CLI layer allowed")
	}
	in.Argv = nil
	if err := os.Mkdir(in.ManagedRoot, 0700); err != nil {
		t.Fatal(err)
	}
	if ReadSandboxPolicy(in).Known {
		t.Fatal("managed policy treated as ordinary local settings")
	}
}

func TestReadSandboxPolicyCodexAndUnknownInputs(t *testing.T) {
	root := t.TempDir()
	in := SandboxSettings{Agent: "codex", Home: root, Project: root, CodexHome: filepath.Join(root, "custom-codex"), ManagedRoot: filepath.Join(root, "managed")}
	files := map[string][]byte{}
	reads := []string{}
	in.ReadFile = func(path string) ([]byte, error) {
		reads = append(reads, path)
		if data, ok := files[path]; ok {
			return data, nil
		}
		return nil, os.ErrNotExist
	}
	if !ScratchAllowed(DefaultRoot, ReadSandboxPolicy(in)) {
		t.Fatal("Codex default TMPDIR must be admitted")
	}
	if len(reads) != 2 || reads[0] != filepath.Join(in.CodexHome, "config.toml") {
		t.Fatalf("wrong routing/count: %v", reads)
	}
	files[reads[0]] = []byte("[sandbox_workspace_write]\nexclude_tmpdir_env_var=true")
	if ScratchAllowed(DefaultRoot, ReadSandboxPolicy(in)) {
		t.Fatal("excluded TMPDIR admitted")
	}
	delete(files, reads[0])
	in.Argv = []string{"--sandbox", "read-only"}
	if ScratchAllowed(DefaultRoot, ReadSandboxPolicy(in)) {
		t.Fatal("read-only CLI admitted")
	}
	in.Argv = []string{"-s=read-only"}
	if ScratchAllowed(DefaultRoot, ReadSandboxPolicy(in)) {
		t.Fatal("short inline read-only CLI admitted")
	}
	in.Argv = []string{"--sandbox=workspace-write", "--disable", "shell_snapshot"}
	if !ScratchAllowed(DefaultRoot, ReadSandboxPolicy(in)) {
		t.Fatal("known workspace CLI declined")
	}
	in.Argv = []string{"--profile", "restricted"}
	if ReadSandboxPolicy(in).Known {
		t.Fatal("unresolved profile admitted")
	}
	in.Argv = nil
	in.ReadFile = func(string) ([]byte, error) { return nil, errors.New("unreadable") }
	if ReadSandboxPolicy(in).Known {
		t.Fatal("unreadable config treated as absent")
	}
	in.ReadFile = func(string) ([]byte, error) { return make([]byte, SettingsLimit+1), nil }
	if ReadSandboxPolicy(in).Known {
		t.Fatal("oversized config admitted")
	}
	in.Agent = "unqualified-agent"
	if ReadSandboxPolicy(in).Known {
		t.Fatal("unknown agent admitted")
	}
}
