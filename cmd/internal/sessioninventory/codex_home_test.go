package sessioninventory

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveCodexHomeExplicitDoesNotReadHOME(t *testing.T) {
	want, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	got, err := ResolveCodexHome("", func(key string) string {
		if key != "CODEX_HOME" {
			t.Fatalf("explicit home queried %q", key)
		}
		return want
	})
	if err != nil || got != want {
		t.Fatalf("home=%q err=%v; want %q", got, err, want)
	}
}

func TestResolveCodexHomeRelativeUsesStartupCWD(t *testing.T) {
	startup := t.TempDir()
	want := filepath.Join(startup, "native")
	if err := os.Mkdir(want, 0700); err != nil {
		t.Fatal(err)
	}
	want, err := filepath.EvalSymlinks(want)
	if err != nil {
		t.Fatal(err)
	}
	got, err := ResolveCodexHome(startup, func(key string) string {
		if key == "CODEX_HOME" {
			return "native"
		}
		if key == "HOME" {
			return startup
		}
		t.Fatalf("unexpected environment key %q", key)
		return ""
	})
	if err != nil || got != want {
		t.Fatalf("home=%q err=%v; want %q", got, err, want)
	}
}

func TestResolveCodexHomeDefaultNeedNotExist(t *testing.T) {
	home := t.TempDir()
	got, err := ResolveCodexHome("", func(key string) string {
		switch key {
		case "CODEX_HOME":
			return ""
		case "HOME":
			return home
		default:
			t.Fatalf("unexpected environment key %q", key)
			return ""
		}
	})
	if err != nil || got != filepath.Join(home, ".codex") {
		t.Fatalf("home=%q err=%v", got, err)
	}
	if _, err := os.Stat(got); !os.IsNotExist(err) {
		t.Fatalf("default home unexpectedly exists: %v", err)
	}
}

func TestResolveCodexHomeUnobservableInputsFailClosed(t *testing.T) {
	for _, tc := range []struct{ name, startup, home, explicit string }{
		{name: "missing fallback HOME", startup: t.TempDir()},
		{name: "relative fallback HOME", startup: t.TempDir(), home: "relative"},
		{name: "relative explicit without startup", explicit: "native"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ResolveCodexHome(tc.startup, func(key string) string {
				switch key {
				case "CODEX_HOME":
					return tc.explicit
				case "HOME":
					return tc.home
				default:
					t.Fatalf("unexpected environment key %q", key)
					return ""
				}
			})
			if err == nil || got != "" {
				t.Fatalf("unobservable home=%q err=%v", got, err)
			}
		})
	}
}

func TestResolveCodexHomeExplicitDirectoryValidation(t *testing.T) {
	base := t.TempDir()
	file := filepath.Join(base, "file")
	if err := os.WriteFile(file, nil, 0600); err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{file, filepath.Join(base, "missing"), " native ", "native\x00"} {
		t.Run(value, func(t *testing.T) {
			got, err := ResolveCodexHome(base, func(key string) string {
				if key == "CODEX_HOME" {
					return value
				}
				return base
			})
			if err == nil || got != "" {
				t.Fatalf("invalid home=%q err=%v", got, err)
			}
		})
	}
	alias := filepath.Join(base, "alias")
	if err := os.Symlink(base, alias); err != nil {
		t.Fatal(err)
	}
	want, err := filepath.EvalSymlinks(base)
	if err != nil {
		t.Fatal(err)
	}
	got, err := ResolveCodexHome(base, func(key string) string {
		if key == "CODEX_HOME" {
			return alias
		}
		return base
	})
	if err != nil || got != want {
		t.Fatalf("symlink home=%q err=%v; want %q", got, err, want)
	}
}

func TestNativeTranscriptRuntimeSharesHomeRouting(t *testing.T) {
	fallback := t.TempDir()
	custom := t.TempDir()
	physical, err := filepath.EvalSymlinks(custom)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, value, startup, want string
		fail                       bool
	}{
		{name: "default", want: filepath.Join(fallback, ".codex", "sessions")},
		{name: "custom", value: custom, want: filepath.Join(physical, "sessions")},
		{name: "invalid", value: filepath.Join(custom, "missing"), fail: true},
		{name: "historical relative", value: "native", fail: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			runtime, err := NewNativeTranscriptRuntime(fallback, t.TempDir(), AgentCodex, tc.startup, func(key string) string {
				if key != "CODEX_HOME" {
					t.Fatalf("constructor queried %q instead of explicit fallback home", key)
				}
				return tc.value
			})
			if tc.fail {
				if err == nil {
					t.Fatal("unavailable home accepted")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			roots := runtime.NativeRoots(AgentCodex)
			if len(roots) != 1 || roots[0].Name != "codex-sessions" || roots[0].Path != tc.want {
				t.Fatalf("roots=%+v; want %q", roots, tc.want)
			}
		})
	}
	for _, agent := range []Agent{AgentClaude, AgentQoder} {
		t.Run(string(agent), func(t *testing.T) {
			base := NewOSRuntime(fallback, "pairdata")
			runtime, err := NewNativeTranscriptRuntime(fallback, "pairdata", agent, "", func(key string) string { t.Fatalf("non-Codex queried %q", key); return "invalid" })
			if err != nil {
				t.Fatal(err)
			}
			got, want := runtime.NativeRoots(agent), base.NativeRoots(agent)
			if len(got) != len(want) {
				t.Fatalf("roots=%+v; want %+v", got, want)
			}
			for i := range want {
				if got[i] != want[i] {
					t.Fatalf("roots=%+v; want %+v", got, want)
				}
			}
		})
	}
}
