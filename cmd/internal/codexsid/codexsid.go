// Package codexsid resolves a live codex session id by walking the agent's
// process tree for its open rollout file. It's the canonical home for the
// ps-descendants + lsof + rollout-regex walk that slug and sessionwatch each
// grew their own copy of (#93 M3 extracts it for review-target; those two hot
// -path packages can adopt it later).
package codexsid

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// rolloutRE matches ~/.codex/sessions/.../rollout-<...>-<uuid>.jsonl and captures
// the session UUID.
var rolloutRE = regexp.MustCompile(`/\.codex/sessions/.*/rollout-.*([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12})\.jsonl$`)

// ResolveSessionID reads the codex agent's root pid from
// $dataDir/agent-pid-<tag>, BFS-walks its process descendants, and greps each
// process's open files for the live rollout jsonl — returning the session UUID,
// or "" when the pidfile is absent/empty or no rollout is open.
func ResolveSessionID(dataDir, tag string) string {
	raw, err := os.ReadFile(filepath.Join(dataDir, "agent-pid-"+tag))
	if err != nil {
		return ""
	}
	root := strings.TrimSpace(string(raw))
	if root == "" {
		return ""
	}
	for _, pid := range descendants(root) {
		for _, name := range lsofNames(pid) {
			if m := rolloutRE.FindStringSubmatch(name); m != nil {
				return m[1]
			}
		}
	}
	return ""
}

// descendants returns root plus its transitive child pids (BFS over
// `ps -axo pid=,ppid=`). On ps failure it degrades to just [root].
func descendants(root string) []string {
	out, err := exec.Command("ps", "-axo", "pid=,ppid=").Output()
	if err != nil {
		return []string{root}
	}
	children := map[string][]string{}
	for _, line := range strings.Split(string(out), "\n") {
		f := strings.Fields(line)
		if len(f) != 2 {
			continue
		}
		children[f[1]] = append(children[f[1]], f[0])
	}
	queue := []string{root}
	seen := map[string]bool{root: true}
	for i := 0; i < len(queue); i++ {
		for _, c := range children[queue[i]] {
			if c == "" || seen[c] {
				continue
			}
			seen[c] = true
			queue = append(queue, c)
		}
	}
	return queue
}

// lsofNames returns the file paths a pid has open (`lsof -p <pid> -Fn`, the `n`
// lines). Empty on any error.
func lsofNames(pid string) []string {
	out, err := exec.Command("lsof", "-p", pid, "-Fn").Output()
	if err != nil {
		return nil
	}
	var names []string
	for _, line := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(line, "n") {
			names = append(names, strings.TrimPrefix(line, "n"))
		}
	}
	return names
}
