package sessioninventory

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestCodexConfigProjectionPrecedenceAndCloudProof(t *testing.T) {
	for _, tc := range []struct {
		forced string
		status CodexConfigStatus
	}{{"api", CodexConfigComplete}, {"chatgpt", CodexConfigCompatibility}, {"", CodexConfigCompatibility}} {
		in := codexConfigEvidence{scope: CodexSelectionScope{CodexHome: "/home/codex", CWD: "/work", Mode: CodexModeExec}, envSQLite: "relative", layers: []codexConfigLayer{{base: "/base", values: map[string]any{"forced_login_method": tc.forced, "sqlite_home": "db", "model_provider": "custom", "model_providers": map[string]any{"custom": map[string]any{"name": "Custom"}}}}, {base: "/profile", values: map[string]any{"sqlite_home": "profile-db"}}}}
		p := projectCodexConfig(in)
		if p.Status != tc.status {
			t.Fatalf("forced %q: %+v", tc.forced, p)
		}
		if p.Scope.SQLiteHome != "/profile/profile-db" || p.Scope.Provider != "custom" {
			t.Fatalf("wrong origins: %+v", p)
		}
	}
}

func TestCodexConfigOSPrivateHomeAndScopedRuntime(t *testing.T) {
	h := t.TempDir()
	cwd := t.TempDir()
	ch := filepath.Join(h, "custom")
	if err := os.Mkdir(ch, 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", h)
	t.Setenv("CODEX_HOME", ch)
	t.Setenv("CODEX_SQLITE_HOME", "env-db")
	if err := os.WriteFile(filepath.Join(ch, "config.toml"), []byte("forced_login_method='api'\nmodel_provider='local'\nsqlite_home='base-db'\n[model_providers.local]\nname='Local'\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ch, "work.config.toml"), []byte("sqlite_home='profile-db'\n[features]\nworktrees=false\n"), 0600); err != nil {
		t.Fatal(err)
	}
	io := newCodexConfigOSReader()
	io.mdm = func() (bool, bool) { return false, true }
	io.systemRoot = filepath.Join(h, "missing-system")
	p, err := readCodexSelectionConfig(context.Background(), CodexConfigInvocation{CWD: cwd, Profile: "work", Mode: CodexModeExec}, io)
	if err != nil || p.Status != CodexConfigComplete {
		t.Fatalf("%+v %v", p, err)
	}
	physicalCH, _ := filepath.EvalSymlinks(ch)
	if p.Scope.SQLiteHome != filepath.Join(physicalCH, "profile-db") || p.Scope.Worktrees {
		t.Fatalf("%+v", p)
	}
	original := NewOSRuntime(h, filepath.Join(h, "pair"))
	scoped, err := original.WithCodexScope(p.Scope)
	if err != nil {
		t.Fatal(err)
	}
	if scoped.NativeRoots(AgentCodex)[0].Path != filepath.Join(physicalCH, "sessions") {
		t.Fatal(scoped.NativeRoots(AgentCodex))
	}
	if original.NativeRoots(AgentCodex)[0].Path == scoped.NativeRoots(AgentCodex)[0].Path {
		t.Fatal("mutated original roots")
	}
}

func codexTestConfig(t *testing.T, body string) (codexConfigOSReader, CodexConfigInvocation, string, string) {
	t.Helper()
	h := t.TempDir()
	ch := filepath.Join(h, "codex")
	cwd := filepath.Join(h, "work")
	for _, p := range []string{ch, cwd} {
		if err := os.Mkdir(p, 0700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(ch, "config.toml"), []byte(body+"\n[model_providers.base]\nname='Base'\n[model_providers.profile]\nname='Profile'\n[model_providers.last]\nname='Last'\n"), 0600); err != nil {
		t.Fatal(err)
	}
	r := newCodexConfigOSReader()
	r.systemRoot = filepath.Join(h, "system")
	r.mdm = func() (bool, bool) { return false, true }
	r.getenv = func(k string) string {
		switch k {
		case "HOME":
			return h
		case "CODEX_HOME":
			return ch
		case "CODEX_SQLITE_HOME", "CODEX_EXEC_SERVER_URL":
			return ""
		default:
			t.Fatalf("forbidden environment read: %s", k)
			return ""
		}
	}
	physicalCH, _ := filepath.EvalSymlinks(ch)
	return r, CodexConfigInvocation{CWD: cwd, Mode: CodexModeTUI}, physicalCH, cwd
}
func TestCodexConfigReaderCompatibilityBoundaries(t *testing.T) {
	for _, which := range []string{"cloud", "malformed", "legacy", "policy", "unknown-policy", "requirements", "managed", "environments", "exec-server", "bad-profile", "budget", "symlink", "empty-provider"} {
		t.Run(which, func(t *testing.T) {
			r, q, ch, cwd := codexTestConfig(t, "forced_login_method='api'\n")
			write := func(path, body string) {
				t.Helper()
				if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte(body), 0600); err != nil {
					t.Fatal(err)
				}
			}
			switch which {
			case "cloud":
				write(filepath.Join(ch, "config.toml"), "")
			case "malformed":
				write(filepath.Join(ch, "config.toml"), "forced_login_method=[")
			case "legacy":
				write(filepath.Join(ch, "config.toml"), "forced_login_method='api'\nprofile='old'")
			case "policy":
				r.mdm = func() (bool, bool) { return true, true }
			case "unknown-policy":
				r.mdm = func() (bool, bool) { return false, false }
			case "requirements":
				write(filepath.Join(r.systemRoot, "requirements.toml"), "")
			case "managed":
				write(filepath.Join(r.systemRoot, "managed_config.toml"), "")
			case "environments":
				write(filepath.Join(ch, "environments.toml"), "")
			case "exec-server":
				old := r.getenv
				r.getenv = func(k string) string {
					if k == "CODEX_EXEC_SERVER_URL" {
						return "nonsecret endpoint"
					}
					return old(k)
				}
			case "bad-profile":
				q.Profile = "../secret"
			case "budget":
				write(filepath.Join(ch, "config.toml"), strings.Repeat(" ", codexConfigFileLimit+1))
			case "symlink":
				if err := os.Symlink(filepath.Join(ch, "config.toml"), filepath.Join(cwd, "config-link")); err != nil {
					t.Fatal(err)
				}
				r.read = func(ctx context.Context, path string) ([]byte, error) {
					if path == filepath.Join(ch, "config.toml") {
						return codexReadConfigFile(filepath.Join(cwd, "config-link"))
					}
					return codexReadConfigFile(path)
				}
			case "empty-provider":
				write(filepath.Join(ch, "config.toml"), "forced_login_method='api'\nmodel_provider=''\n")
			}
			p, err := readCodexSelectionConfig(context.Background(), q, r)
			if err != nil {
				t.Fatal(err)
			}
			if p.Status != CodexConfigCompatibility {
				t.Fatalf("must retain native behavior: %+v", p)
			}
		})
	}
}
func TestCodexConfigReaderCLIPathAndProfileOrigins(t *testing.T) {
	r, q, ch, cwd := codexTestConfig(t, "forced_login_method='api'\nsqlite_home='base'\nmodel_provider='base'\n")
	if err := os.WriteFile(filepath.Join(ch, "work.config.toml"), []byte("sqlite_home='profile'\nmodel_provider='profile'\n"), 0600); err != nil {
		t.Fatal(err)
	}
	q.Profile = "work"
	q.Overrides = []string{"sqlite_home='~/db'", "model_provider=first", "model_provider='last'"}
	q.Disable = []string{"worktrees"}
	p, err := readCodexSelectionConfig(context.Background(), q, r)
	if err != nil || p.Status != CodexConfigComplete {
		t.Fatalf("%+v %v", p, err)
	}
	home := r.getenv("HOME")
	if p.Scope.SQLiteHome != filepath.Join(home, "db") || p.Scope.Provider != "last" || p.Scope.Worktrees {
		t.Fatalf("%+v cwd=%s", p, cwd)
	}
	q.Overrides = []string{"forced_login_method=chatgpt"}
	p, err = readCodexSelectionConfig(context.Background(), q, r)
	if err != nil || p.Reason != CodexConfigCloudUnknown {
		t.Fatalf("%+v %v", p, err)
	}
}
func TestCodexConfigReaderProjectsCannotSupplyProviderButMayRedirectSQLite(t *testing.T) {
	r, q, ch, cwd := codexTestConfig(t, "forced_login_method='api'\nmodel_provider='base'\n")
	if err := os.MkdirAll(filepath.Join(cwd, ".git"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cwd, ".git", "HEAD"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(cwd, ".codex"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cwd, ".codex", "config.toml"), []byte("sqlite_home='project-db'\nmodel_provider='hostile'\n"), 0600); err != nil {
		t.Fatal(err)
	}
	physical, _ := filepath.EvalSymlinks(cwd)
	for _, trust := range []string{"trusted", "untrusted"} {
		q.Overrides = []string{`projects={"` + physical + `"={trust_level='` + trust + `'}}`}
		p, err := readCodexSelectionConfig(context.Background(), q, r)
		if err != nil || p.Status != CodexConfigComplete {
			t.Fatalf("%+v %v", p, err)
		}
		want := ch
		if trust == "trusted" {
			want = filepath.Join(physical, ".codex", "project-db")
		}
		chPhysical, _ := filepath.EvalSymlinks(want)
		if chPhysical != "" {
			want = chPhysical
		}
		if p.Scope.SQLiteHome != want || p.Scope.Provider != "base" {
			t.Fatalf("trust %s: %+v want %s", trust, p, want)
		}
	}
}
func TestCodexConfigReaderEnvironmentAndNamedReadFirewall(t *testing.T) {
	r, q, ch, _ := codexTestConfig(t, "forced_login_method='api'\n")
	seen := map[string]bool{}
	baseRead := r.read
	r.read = func(ctx context.Context, path string) ([]byte, error) {
		seen[path] = true
		name := filepath.Base(path)
		if strings.Contains(name, "auth") || strings.Contains(name, "credential") {
			t.Fatalf("forbidden credential read %s", path)
		}
		return baseRead(ctx, path)
	}
	p, err := readCodexSelectionConfig(context.Background(), q, r)
	if err != nil || p.Status != CodexConfigComplete {
		t.Fatalf("%+v %v", p, err)
	}
	if !seen[filepath.Join(ch, "config.toml")] {
		t.Fatal("config reader disconnected")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := readCodexSelectionConfig(ctx, q, r); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}

func TestCodexConfigRelativeHomeUsesStartupNotSelectedDirectory(t *testing.T) {
	r, q, ch, cwd := codexTestConfig(t, "forced_login_method='api'\n")
	q.StartupCWD = filepath.Dir(ch)
	q.CWD = cwd
	old := r.getenv
	r.getenv = func(k string) string {
		if k == "CODEX_HOME" {
			return "codex"
		}
		return old(k)
	}
	p, err := readCodexSelectionConfig(context.Background(), q, r)
	if err != nil || p.Status != CodexConfigComplete || p.Scope.CodexHome != ch {
		t.Fatalf("%+v %v", p, err)
	}
	q.Overrides = []string{"sqlite_home='relative-db'"}
	p, err = readCodexSelectionConfig(context.Background(), q, r)
	physicalCWD, _ := filepath.EvalSymlinks(cwd)
	if err != nil || p.Scope.SQLiteHome != filepath.Join(physicalCWD, "relative-db") {
		t.Fatalf("%+v %v", p, err)
	}
}

func TestCodexConfigNestedCWDLogicalAliasRemainsCompatibility(t *testing.T) {
	r, q, _, cwd := codexTestConfig(t, "forced_login_method='api'\n")
	alias := filepath.Join(filepath.Dir(cwd), "alias")
	if err := os.Symlink(cwd, alias); err != nil {
		t.Fatal(err)
	}
	q.CWD = alias
	p, err := readCodexSelectionConfig(context.Background(), q, r)
	if err != nil || p.Status != CodexConfigCompatibility || p.Reason != CodexConfigScopeUnsupported {
		t.Fatalf("%+v %v", p, err)
	}
}

func TestCodexConfigCLIInlineTableAndNoSiblingInjection(t *testing.T) {
	r, q, _, _ := codexTestConfig(t, "forced_login_method='api'\n")
	q.Overrides = []string{"features={worktrees=false}"}
	p, err := readCodexSelectionConfig(context.Background(), q, r)
	if err != nil || p.Status != CodexConfigComplete || p.Scope.Worktrees {
		t.Fatalf("inline table %+v %v", p, err)
	}
	r, q, _, _ = codexTestConfig(t, "")
	q.Overrides = []string{"unrelated='text'\nforced_login_method='api'"}
	p, err = readCodexSelectionConfig(context.Background(), q, r)
	if err != nil || p.Status != CodexConfigCompatibility || p.Reason != CodexConfigCloudUnknown {
		t.Fatalf("sibling injection %+v %v", p, err)
	}
}

func TestCodexConfigReaderPlatformObservationFailsClosed(t *testing.T) {
	r, q, _, _ := codexTestConfig(t, "forced_login_method='api'\n")
	reader := CodexConfigReader{SystemDir: r.systemRoot}
	p, err := reader.Read(context.Background(), q, r.getenv)
	if err != nil {
		t.Fatal(err)
	}
	present, known := codexForcedPreferences()
	want := CodexConfigCompatibility
	if known && !present {
		want = CodexConfigComplete
	}
	if p.Status != want {
		t.Fatalf("platform known=%v present=%v projection=%+v", known, present, p)
	}
	if (!known || present) && p.Reason != CodexConfigManagedUnsupported {
		t.Fatalf("unknown policy gained authority: %+v", p)
	}
	reader.ManagedPolicy = func() (CodexManagedPolicyStatus, error) { return CodexManagedPolicyUnknown, nil }
	p, err = reader.Read(context.Background(), q, r.getenv)
	if err != nil || p.Status != CodexConfigCompatibility {
		t.Fatalf("explicit unknown %+v %v", p, err)
	}
}

func TestCodexConfigCLIStatementKeyCannotForgeBootstrapProof(t *testing.T) {
	r, q, _, _ := codexTestConfig(t, "")
	q.Overrides = []string{"# ignored\nforced_login_method=api"}
	p, err := readCodexSelectionConfig(context.Background(), q, r)
	if err != nil {
		t.Fatal(err)
	}
	if p.Status == CodexConfigComplete {
		t.Fatalf("literal CLI key forged API-only proof: %+v", p)
	}
}
func TestCodexConfigUnknownProviderCannotAuthorize(t *testing.T) {
	r, q, _, _ := codexTestConfig(t, "forced_login_method='api'\nmodel_provider='unconfigured-provider'\n")
	p, err := readCodexSelectionConfig(context.Background(), q, r)
	if err != nil {
		t.Fatal(err)
	}
	if p.Status == CodexConfigComplete {
		t.Fatalf("missing native provider accepted: %+v", p)
	}
}

// osfs checks cancellation after Lstat and immediately before its secure open;
// this controlled context replaces the entry precisely at that boundary.
type codexSwapOpenContext struct {
	context.Context
	checks int
	swap   func()
}

func (c *codexSwapOpenContext) Err() error {
	c.checks++
	if c.checks == 2 {
		c.swap()
	}
	return c.Context.Err()
}

func TestCodexConfigMetadataOpenRefusesSwapToSymlinkOrFIFO(t *testing.T) {
	for _, kind := range []string{"symlink", "fifo"} {
		t.Run(kind, func(t *testing.T) {
			d := t.TempDir()
			path := filepath.Join(d, "config.toml")
			target := filepath.Join(d, "private-target")
			if err := os.WriteFile(path, []byte("forced_login_method='api'"), 0600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(target, []byte("fixture-not-authorized-for-reading"), 0600); err != nil {
				t.Fatal(err)
			}
			var swapErr error
			ctx := &codexSwapOpenContext{Context: context.Background(), swap: func() {
				if swapErr = os.Rename(path, path+".original"); swapErr != nil {
					return
				}
				if kind == "symlink" {
					swapErr = os.Symlink(target, path)
				} else {
					swapErr = syscall.Mkfifo(path, 0600)
				}
			}}
			result := make(chan error, 1)
			go func() {
				b, err := newCodexConfigOSReader().read(ctx, path)
				if len(b) != 0 {
					err = fmt.Errorf("unsafe read returned bytes")
				}
				result <- err
			}()
			select {
			case err := <-result:
				if swapErr != nil {
					t.Fatal(swapErr)
				}
				if err == nil {
					t.Fatal("swapped file accepted")
				}
				if kind == "symlink" && !errors.Is(err, syscall.ELOOP) {
					t.Fatalf("symlink was opened before refusal: %v", err)
				}
			case <-time.After(time.Second):
				// Release a counterfactual blocking FIFO open so failure cannot leak a goroutine.
				f, err := os.OpenFile(path, os.O_RDWR|syscall.O_NONBLOCK, 0600)
				if err == nil {
					f.Close()
				}
				t.Fatal("metadata open blocked on replacement FIFO")
			}
		})
	}
}
func TestCodexConfigMissingProfileIsEmptyNativeLayer(t *testing.T) {
	r, q, _, _ := codexTestConfig(t, "forced_login_method='api'\n")
	q.Profile = "missing"
	p, err := readCodexSelectionConfig(context.Background(), q, r)
	if err != nil || p.Status != CodexConfigComplete {
		t.Fatalf("%+v %v", p, err)
	}
}

func TestCodexConfigKnownFeatureFlagsKeepSelectionScope(t *testing.T) {
	for _, tc := range []struct {
		name            string
		enable, disable []string
		status          CodexConfigStatus
		worktrees       bool
	}{
		{name: "shell-snapshot-disable", disable: []string{"shell_snapshot"}, status: CodexConfigComplete},
		{name: "shell-snapshot-enable", enable: []string{"shell_snapshot"}, status: CodexConfigComplete},
		{name: "worktrees-enable", enable: []string{"worktrees", "shell_snapshot"}, status: CodexConfigComplete, worktrees: true},
		{name: "worktrees-disable", disable: []string{"worktrees", "shell_snapshot"}, status: CodexConfigComplete},
		{name: "unknown-disable", disable: []string{"future_selection_feature"}, status: CodexConfigCompatibility},
		{name: "unknown-enable", enable: []string{"future_selection_feature"}, status: CodexConfigCompatibility},
	} {
		t.Run(tc.name, func(t *testing.T) {
			reader, inv, _, _ := codexTestConfig(t, "forced_login_method='api'\n[features]\nworktrees=false\n")
			inv.Enable = tc.enable
			inv.Disable = tc.disable
			projection, err := readCodexSelectionConfig(context.Background(), inv, reader)
			if err != nil || projection.Status != tc.status {
				t.Fatalf("feature projection: %+v %v", projection, err)
			}
			if projection.Status == CodexConfigComplete && projection.Scope.Worktrees != tc.worktrees {
				t.Fatalf("worktrees changed: %+v", projection.Scope)
			}
		})
	}
}

func TestCodexConfigCLITableAssignmentRestoresNativeWorktreeDefault(t *testing.T) {
	q := CodexConfigInvocation{CWD: "/work", Overrides: []string{"features={worktrees=false}", "features={shell_snapshot=false}"}}
	cli, err := codexCLIConfigLayer(q, "/home")
	if err != nil {
		t.Fatal(err)
	}
	p := projectCodexConfig(codexConfigEvidence{scope: CodexSelectionScope{CodexHome: "/home/codex", CWD: "/work", Mode: CodexModeTUI}, layers: []codexConfigLayer{{base: "/home/codex", values: map[string]any{"forced_login_method": "api"}}, cli}})
	if p.Status != CodexConfigComplete || !p.Scope.Worktrees {
		t.Fatalf("native final features assignment restores default worktrees=true: %+v CLI=%+v", p, cli.values)
	}
}
func TestCodexConfigCLITableAssignmentReplacesSelectorTables(t *testing.T) {
	for _, tc := range []struct {
		name      string
		overrides []string
		want      map[string]any
	}{
		{"features root", []string{"features={worktrees=false}", "features={shell_snapshot=false}"}, map[string]any{"features": map[string]any{"shell_snapshot": false}}},
		{"projects root", []string{"projects={first={trust_level='trusted'}}", "projects={second={trust_level='untrusted'}}"}, map[string]any{"projects": map[string]any{"second": map[string]any{"trust_level": "untrusted"}}}},
		{"providers root", []string{"model_providers={first={name='First'}}", "model_providers={second={name='Second'}}"}, map[string]any{"model_providers": map[string]any{"second": map[string]any{"name": "Second"}}}},
		{"provider definition", []string{"model_providers.custom={name='Custom',base_url='https://first.invalid'}", "model_providers.custom={base_url='https://second.invalid'}"}, map[string]any{"model_providers": map[string]any{"custom": map[string]any{"base_url": "https://second.invalid"}}}},
		{"scalar ancestor", []string{"features=false", "features.worktrees=false"}, map[string]any{"features": map[string]any{"worktrees": false}}},
		{"ordinary feature leaf", []string{"features.worktrees=false", "features.worktrees={enabled=true}"}, map[string]any{"features": map[string]any{"worktrees": map[string]any{"enabled": true}}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			layer, err := codexCLIConfigLayer(CodexConfigInvocation{CWD: "/work", Overrides: tc.overrides}, "/home")
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(layer.values, tc.want) {
				t.Fatalf("got %#v want %#v", layer.values, tc.want)
			}
		})
	}
}

func TestCodexConfigCLIStructuredFeatureAssignment(t *testing.T) {
	for _, feature := range []string{"code_mode", "multi_agent_v2", "network_proxy", "sleep_tool"} {
		for _, prefix := range []string{"features.", "profiles.work.features."} {
			for _, tc := range []struct {
				name   string
				values []string
				want   map[string]any
			}{
				{"boolean then table", []string{"false", "{extra=1}"}, map[string]any{"enabled": false, "extra": int64(1)}},
				{"table then boolean", []string{"{extra=1}", "true"}, map[string]any{"enabled": true, "extra": int64(1)}},
				{"table then table", []string{"{first=1}", "{second=2}"}, map[string]any{"first": int64(1), "second": int64(2)}},
			} {
				t.Run(prefix+feature+"/"+tc.name, func(t *testing.T) {
					var overrides []string
					for _, v := range tc.values {
						overrides = append(overrides, prefix+feature+"="+v)
					}
					layer, err := codexCLIConfigLayer(CodexConfigInvocation{Overrides: overrides}, "/home")
					if err != nil {
						t.Fatal(err)
					}
					var got any = layer.values
					for _, segment := range strings.Split(prefix+feature, ".") {
						got = got.(map[string]any)[segment]
					}
					if !reflect.DeepEqual(got, tc.want) {
						t.Fatalf("got %#v want %#v", got, tc.want)
					}
				})
			}
			t.Run(prefix+feature+"/intermediate boolean", func(t *testing.T) {
				layer, err := codexCLIConfigLayer(CodexConfigInvocation{Overrides: []string{prefix + feature + "=false", prefix + feature + ".extra=1"}}, "/home")
				if err != nil {
					t.Fatal(err)
				}
				var got any = layer.values
				for _, segment := range strings.Split(prefix+feature, ".") {
					got = got.(map[string]any)[segment]
				}
				want := map[string]any{"enabled": false, "extra": int64(1)}
				if !reflect.DeepEqual(got, want) {
					t.Fatalf("got %#v want %#v", got, want)
				}
			})
		}
	}
}
func TestCodexConfigCLITableReplacementDropsBootstrapTrustAndProvider(t *testing.T) {
	t.Run("project API proof", func(t *testing.T) {
		r, q, _, cwd := codexTestConfig(t, "")
		physical, _ := filepath.EvalSymlinks(cwd)
		folder := filepath.Join(cwd, ".codex")
		if err := os.Mkdir(folder, 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(folder, "config.toml"), []byte("forced_login_method='api'\n"), 0600); err != nil {
			t.Fatal(err)
		}
		q.Overrides = []string{`projects={"` + physical + `"={trust_level='trusted'}}`}
		p, err := readCodexSelectionConfig(context.Background(), q, r)
		if err != nil || p.Status != CodexConfigComplete {
			t.Fatalf("positive proof disconnected: %+v %v", p, err)
		}
		q.Overrides = append(q.Overrides, "projects={}")
		p, err = readCodexSelectionConfig(context.Background(), q, r)
		if err != nil || p.Status == CodexConfigComplete {
			t.Fatalf("deleted CLI trust survived table replacement: %+v %v", p, err)
		}
	})
	t.Run("provider definition", func(t *testing.T) {
		r, q, _, _ := codexTestConfig(t, "forced_login_method='api'\nmodel_provider='custom'\n")
		q.Overrides = []string{"model_providers={custom={name='Custom'}}"}
		p, err := readCodexSelectionConfig(context.Background(), q, r)
		if err != nil || p.Status != CodexConfigComplete {
			t.Fatalf("positive provider disconnected: %+v %v", p, err)
		}
		q.Overrides = append(q.Overrides, "model_providers={}")
		p, err = readCodexSelectionConfig(context.Background(), q, r)
		if err != nil || p.Status == CodexConfigComplete {
			t.Fatalf("deleted CLI provider survived: %+v %v", p, err)
		}
	})
}
func TestCodexConfigIndependentLayersStillMergeRecursively(t *testing.T) {
	r, q, _, _ := codexTestConfig(t, "forced_login_method='api'\n[features]\nworktrees=false\n")
	q.Overrides = []string{"features={shell_snapshot=false}"}
	p, err := readCodexSelectionConfig(context.Background(), q, r)
	if err != nil || p.Status != CodexConfigComplete || p.Scope.Worktrees {
		t.Fatalf("independent CLI layer erased base worktrees=false: %+v %v", p, err)
	}
}
