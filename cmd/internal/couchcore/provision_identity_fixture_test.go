package couchcore

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestProvisionFixtureIdentityTracksRealGit(t *testing.T) {
	t.Parallel()
	f := newProvisionFixture(t)
	f.git(f.Primary, "worktree", "add", "-b", "main-slot1", f.host(1), "main")
	check := func(dir string, slot int) {
		t.Helper()
		top := f.git(dir, "rev-parse", "--show-toplevel")
		common := f.git(dir, "rev-parse", "--path-format=absolute", "--git-common-dir")
		head := f.git(dir, "rev-parse", "HEAD")
		branch := f.git(dir, "branch", "--show-current")
		kind, env, rest := "primary", filepath.Dir(f.Primary), "main"
		if slot > 0 {
			kind, env, rest = "slot", filepath.Dir(top), "main-slot1"
		}
		address := filepath.Base(f.Primary) + ":0"
		if slot > 0 {
			address = filepath.Base(f.Primary) + ":1"
		}
		want := WorkspaceIdentity{SchemaVersion: 2, Repo: filepath.Base(f.Primary), RepoIdentity: common, PrimaryRoot: f.Primary, FleetRoot: filepath.Dir(f.Primary), EnvironmentRoot: env, WorktreeRoot: top, Kind: kind, Address: &address, Slot: &slot, Branch: &branch, Head: &head, RestingBranch: &rest}
		if slot > 0 {
			want.EnvironmentHost = &EnvironmentHost{Repo: want.Repo, Slot: slot, RepoIdentity: common, PrimaryRoot: f.Primary, WorktreeRoot: top}
		}
		raw, err := f.Run(context.Background(), ProvisionCommand{Program: "sdlc", Dir: dir})
		var got WorkspaceIdentity
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(raw, &got); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("identity = %+v, want %+v", got, want)
		}
	}
	check(f.Primary, 0)
	check(f.host(1), 1)
	f.git(f.host(1), "switch", "-c", "issue-work")
	f.git(f.host(1), "commit", "--allow-empty", "-m", "fresh head")
	check(f.host(1), 1)
	f.git(f.host(1), "checkout", "--detach", "HEAD")
	check(f.host(1), 1)
	unknown := filepath.Join(t.TempDir(), "unrecognized-worktree")
	f.git(f.Primary, "worktree", "add", "--detach", unknown, "HEAD")
	if _, err := f.Run(context.Background(), ProvisionCommand{Program: "sdlc", Dir: unknown}); err == nil {
		t.Fatal("unrecognized worktree accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := f.Run(ctx, ProvisionCommand{Program: "sdlc", Dir: f.host(1)}); err == nil {
		t.Fatal("cancelled identity succeeded")
	}
	if err := os.WriteFile(filepath.Join(f.host(1), ".git"), []byte("gitdir: /nonexistent/fixture-admin\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := f.Run(context.Background(), ProvisionCommand{Program: "sdlc", Dir: f.host(1)}); err == nil {
		t.Fatal("corrupt membership succeeded")
	}
}

func TestProvisionFixtureIdentityRejectsForeignRepositoryAtSlotPath(t *testing.T) {
	t.Parallel()
	f := newProvisionFixture(t)
	if err := os.MkdirAll(f.host(1), 0700); err != nil {
		t.Fatal(err)
	}
	f.git(f.host(1), "init", "-b", "main")
	f.git(f.host(1), "commit", "--allow-empty", "-m", "foreign")
	if _, err := f.Run(context.Background(), ProvisionCommand{Program: "sdlc", Dir: f.host(1)}); err == nil {
		t.Fatal("foreign repository accepted as fixture slot")
	}
}

func TestProvisionFixtureIdentityUsesOneGitProcess(t *testing.T) {
	f := newProvisionFixture(t)
	// A Git-file primary must remain supported without adding identity spawns.
	f.git(f.Primary, "init", "--separate-git-dir", filepath.Join(filepath.Dir(f.Primary), "external-git"))
	realGit, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	bin, log := t.TempDir(), filepath.Join(t.TempDir(), "git-calls")
	script := "#!/bin/sh\nIFS= read -r token <&3 && test \"$token\" = \"$PAIR_FIXTURE_LEASE_TOKEN\" || exit 42\nprintf '%s\\n' call >> \"$PAIR_FIXTURE_GIT_TRACE\"\nexec \"$PAIR_FIXTURE_REAL_GIT\" \"$@\"\n"
	if err := os.WriteFile(filepath.Join(bin, "git"), []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	f.IO.Env = append(f.IO.Env, "PAIR_FIXTURE_GIT_TRACE="+log, "PAIR_FIXTURE_REAL_GIT="+realGit)
	lease, err := os.CreateTemp(t.TempDir(), "lease")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = lease.Close() })
	// An inherited descriptor remains readable after its pathname loses read
	// permission. Permission inspection is not proof of descriptor usability.
	token := lease.Name()
	if _, err := lease.WriteString(token + "\n"); err != nil {
		t.Fatal(err)
	}
	f.IO.Env = append(f.IO.Env, "PAIR_FIXTURE_LEASE_TOKEN="+token)
	wrong, err := os.CreateTemp(t.TempDir(), "wrong-lease")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = wrong.Close() })
	if _, err := wrong.WriteString("wrong-token\n"); err != nil {
		t.Fatal(err)
	}
	calls := 0
	for _, tc := range []struct {
		name   string
		file   *os.File
		mode   os.FileMode
		accept bool
	}{
		{"readable", lease, 0600, true},
		{"no-path-permission", lease, 0, true},
		{"absent", nil, 0, false},
		{"wrong-token", wrong, 0600, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.file != nil {
				if err := tc.file.Chmod(tc.mode); err != nil {
					t.Fatal(err)
				}
				if _, err := tc.file.Seek(0, 0); err != nil {
					t.Fatal(err)
				}
			}
			_, err := f.Run(context.Background(), ProvisionCommand{Program: "sdlc", Dir: f.Primary, Lease: tc.file})
			if tc.accept {
				if err != nil {
					t.Fatal(err)
				}
				calls++
			} else if err == nil || !strings.Contains(err.Error(), "exit status 42") {
				t.Fatalf("invalid inherited lease: got %v, want exit42", err)
			}
			raw, err := os.ReadFile(log)
			if err != nil {
				t.Fatal(err)
			}
			if got := strings.Count(string(raw), "call\n"); got != calls {
				t.Fatalf("identity git processes = %d, want %d", got, calls)
			}
		})
	}
}

func TestProvisionFixtureIdentityRejectsMalformedGitFacts(t *testing.T) {
	f := newProvisionFixture(t)
	bin, response := t.TempDir(), filepath.Join(t.TempDir(), "git-response")
	if err := os.WriteFile(filepath.Join(bin, "git"), []byte("#!/bin/sh\ncat \"$PAIR_FIXTURE_GIT_RESPONSE\"\n"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	f.IO.Env = append(f.IO.Env, "PAIR_FIXTURE_GIT_RESPONSE="+response)
	facts := []string{f.Primary, filepath.Join(f.Primary, ".git"), f.Base, "main"}
	cases := map[string][]string{
		"missing":        facts[:3],
		"extra":          append(append([]string(nil), facts...), "surplus"),
		"relative top":   {"relative", facts[1], facts[2], facts[3]},
		"foreign common": {facts[0], filepath.Join(t.TempDir(), ".git"), facts[2], facts[3]},
		"nonhex head":    {facts[0], facts[1], strings.Repeat("z", 40), facts[3]},
		"short head":     {facts[0], facts[1], "0123", facts[3]},
		"empty branch":   {facts[0], facts[1], facts[2], ""},
	}
	for name, output := range cases {
		t.Run(name, func(t *testing.T) {
			if err := os.WriteFile(response, []byte(strings.Join(output, "\n")+"\n"), 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := f.Run(context.Background(), ProvisionCommand{Program: "sdlc", Dir: f.Primary}); err == nil {
				t.Fatal("malformed facts accepted")
			}
		})
	}
}
