package zellijtest

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// ReplyEnvelope preserves all three native reply channels, including absence.
type ReplyEnvelope struct {
	Stdout string `json:"stdout"`
	Stderr string `json:"stderr"`
	Code   int    `json:"code"`
}

var generatedReplies sync.Map

func replyScript() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		p := filepath.Join(cwd, "tests", "lib", "zellij_fixture_replies.py")
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
		parent := filepath.Dir(cwd)
		if parent == cwd {
			return "", fmt.Errorf("Zellij fixture generator not found")
		}
		cwd = parent
	}
}

func replyRun(t testing.TB, args ...string) ReplyEnvelope {
	t.Helper()
	script, err := replyScript()
	if err != nil {
		t.Fatal(err)
	}
	keyBytes, err := json.Marshal(append([]string{script}, args...))
	if err != nil {
		t.Fatal(err)
	}
	key := string(keyBytes)
	if cached, ok := generatedReplies.Load(key); ok {
		return cached.(ReplyEnvelope)
	}
	python, err := exec.LookPath("python3")
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(python, append([]string{"-B", script, "--envelope"}, args...)...)
	raw, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("fixture generator: %v: %s", err, raw)
	}
	var r ReplyEnvelope
	if err := json.Unmarshal(raw, &r); err != nil {
		t.Fatal(err)
	}
	generatedReplies.Store(key, r)
	return r
}

// Reply renders at setup time so timed native queries spawn no extra process.
func Reply(t testing.TB, kind string, payload any) ReplyEnvelope {
	t.Helper()
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	return replyRun(t, "--kind", kind, "--data", string(raw))
}

func Case(t testing.TB, name string) ReplyEnvelope {
	t.Helper()
	return replyRun(t, "--case", name)
}

func shellQuoteReply(s string) string { return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'" }

func (r ReplyEnvelope) Shell() string {
	return fmt.Sprintf("printf '%%s' %s; printf '%%s' %s >&2; exit %d", shellQuoteReply(r.Stdout), shellQuoteReply(r.Stderr), r.Code)
}

func ShellReply(t testing.TB, kind string, payload any) string {
	t.Helper()
	return Reply(t, kind, payload).Shell()
}

func ShellCase(t testing.TB, name string) string {
	t.Helper()
	return Case(t, name).Shell()
}

// ReplyCommandFrom uses an already passed absolute test helper path. It adds no
// ambient session or protocol environment to the fixture's controlled children.
func ReplyCommandFrom(envScript string) (string, error) {
	if !filepath.IsAbs(envScript) {
		return "", fmt.Errorf("test helper path must be absolute")
	}
	script := filepath.Join(filepath.Dir(envScript), "zellij_fixture_replies.py")
	if _, err := os.Stat(script); err != nil {
		return "", err
	}
	python, err := exec.LookPath("python3")
	if err != nil {
		return "", err
	}
	python, err = filepath.Abs(python)
	if err != nil {
		return "", err
	}
	return shellQuoteReply(python) + " -B " + shellQuoteReply(script), nil
}

func ReplyCommand(t testing.TB) string {
	t.Helper()
	script, err := replyScript()
	if err != nil {
		t.Fatal(err)
	}
	command, err := ReplyCommandFrom(filepath.Join(filepath.Dir(script), "pair-test-env.sh"))
	if err != nil {
		t.Fatal(err)
	}
	return command
}
