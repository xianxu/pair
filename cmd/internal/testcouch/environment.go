package testcouch

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/xianxu/pair/cmd/internal/pathscope"
	"github.com/xianxu/pair/cmd/internal/runtimebundle"
	"github.com/xianxu/pair/cmd/internal/testpair"
)

func allocateRoots(base string, denied []string) (RootsPlan, error) {
	if !filepath.IsAbs(base) || filepath.Clean(base) != base {
		return RootsPlan{}, fmt.Errorf("canonical absolute scratch parent required")
	}
	physical, err := pathscope.CanonicalFuture(base)
	if err != nil {
		return RootsPlan{}, err
	}
	for _, live := range denied {
		live, err = pathscope.CanonicalFuture(live)
		if err != nil {
			return RootsPlan{}, err
		}
		if pathscope.Within(live, physical) {
			return RootsPlan{}, fmt.Errorf("scratch parent is within live root %q", live)
		}
	}
	root, err := os.MkdirTemp(base, "pc-")
	if err != nil {
		return RootsPlan{}, err
	}
	ok := false
	defer func() {
		if !ok {
			_ = os.Remove(root)
		}
	}()
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return RootsPlan{}, err
	}
	p := RootsPlan{Root: root, Mutable: map[string]string{}, Denied: append([]string(nil), denied...)}
	for _, name := range []string{"home", "data", "store", "identity", "config", "cache", "runtime", "tmp", "queue", "fleet", "sockets", "control", "bin", "artifacts"} {
		p.Mutable[name] = filepath.Join(root, name)
	}
	p.Mutable["codex"] = filepath.Join(root, "home", ".codex")
	p.Mutable["claude"] = filepath.Join(root, "home", ".claude")
	p.Mutable["capacity"] = filepath.Join(root, "capacity")
	p, err = physicalRoots(p)
	if err != nil {
		return RootsPlan{}, err
	}
	// The root is a newly allocated lease; no process can see it until this
	// constructor succeeds. RemoveAll is confined to that fresh lease on failure.
	defer func() {
		if !ok {
			_ = os.RemoveAll(root)
		}
	}()
	for name, path := range p.Mutable {
		if name == "capacity" {
			err = os.WriteFile(path, nil, 0600)
		} else {
			err = os.MkdirAll(path, 0700)
		}
		if err != nil {
			return RootsPlan{}, err
		}
	}
	if _, err := testpair.SeedCouchIdentity(p.Mutable["identity"]); err != nil {
		return RootsPlan{}, err
	}
	ok = true
	return p, nil
}

func runtimeEnvironment(p RootsPlan, bin, agent, runID, source, couch string) ([]string, error) {
	if !filepath.IsAbs(source) || filepath.Clean(source) != source {
		return nil, fmt.Errorf("runtime environment requires the selected absolute checkout")
	}
	if _, err := physicalRoots(p); err != nil {
		return nil, err
	}
	if agent != "claude" && agent != "codex" {
		return nil, fmt.Errorf("unsupported fixture personality %q", agent)
	}
	terminal, err := runtimebundle.TerminalEnvironment(p.Mutable["data"])
	if err != nil {
		return nil, err
	}
	env := []string{
		"PATH=" + bin, "HOME=" + p.Mutable["home"], "TMPDIR=" + p.Mutable["tmp"],
		"XDG_DATA_HOME=" + p.Mutable["data"], "XDG_CONFIG_HOME=" + p.Mutable["config"], "XDG_CACHE_HOME=" + p.Mutable["cache"], "XDG_RUNTIME_DIR=" + p.Mutable["runtime"],
		"COUCH_ISOLATED_ROOT=" + p.Root, "COUCH_STORE_DIR=" + p.Mutable["store"], "COUCH_IDENTITY_DIR=" + p.Mutable["identity"], "COUCH_PAIR_DATA_DIR=" + filepath.Join(p.Mutable["data"], "pair"),
		"PAIR_AGENT=" + agent, "PAIR_HOME=" + source, "CODEX_HOME=" + p.Mutable["codex"], "CLAUDE_CONFIG_DIR=" + p.Mutable["claude"],
		"ZELLIJ_SOCKET_DIR=" + p.Mutable["sockets"], "ZELLIJ_CONFIG_DIR=" + p.Mutable["config"], "ZELLIJ_DATA_DIR=" + p.Mutable["data"],
		"WF_SDLC_QUEUE_ROOT=" + p.Mutable["queue"], "WF_TEST_LOCK_FILE=" + p.Mutable["capacity"], "WF_SDLC_QUEUE_FLEET_ROOT=" + p.Mutable["fleet"],
		"PAIR_TESTCOUCH_ROOT=" + p.Root, "PAIR_TESTCOUCH_CONTROL=" + p.Mutable["control"], "PAIR_TESTCOUCH_RUN_ID=" + runID, "PAIR_TESTCOUCH_COUCH=" + couch,
		"PAIR_WRAP_LOG=" + startupLogPath(p),
		"GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null", "GIT_TERMINAL_PROMPT=0", "GIT_ALLOW_PROTOCOL=file",
		"GIT_AUTHOR_NAME=Couch Fixture", "GIT_AUTHOR_EMAIL=fixture@example.invalid", "GIT_COMMITTER_NAME=Couch Fixture", "GIT_COMMITTER_EMAIL=fixture@example.invalid",
		"LANG=en_US.UTF-8", "LC_ALL=en_US.UTF-8", "SHELL=/bin/sh", "EDITOR=/usr/bin/true", "VISUAL=/usr/bin/true",
	}
	env = append(env, terminal...)
	helper := filepath.Join(source, "tests", "lib", "pair-test-env.sh")
	entries, err := testpair.EnvironmentFrom(helper, p.Mutable["home"], env...)
	if err != nil {
		return nil, err
	}
	// The shared shell helper appends explicit assignments after its defaults.
	// Exec uses the final value for each key; publish that same effective
	// environment once per key so a replacement map has identical semantics.
	seen := make(map[string]bool, len(entries))
	unique := make([]string, 0, len(entries))
	for i := len(entries) - 1; i >= 0; i-- {
		entry := entries[i]
		key, _, ok := strings.Cut(entry, "=")
		if !ok || key == "" || strings.ContainsRune(entry, 0) {
			return nil, fmt.Errorf("invalid selected runtime environment entry")
		}
		if !seen[key] {
			seen[key] = true
			unique = append(unique, entry)
		}
	}
	for i, j := 0, len(unique)-1; i < j; i, j = i+1, j-1 {
		unique[i], unique[j] = unique[j], unique[i]
	}
	return unique, nil
}
