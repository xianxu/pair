package wrapcmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func startupRecords(t *testing.T, path, label string) []map[string]any {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var records []map[string]any
	for _, line := range strings.Split(string(raw), "\n") {
		_, payload, ok := strings.Cut(line, "] "+label+": ")
		if !ok {
			continue
		}
		text, err := strconv.Unquote(payload)
		if err != nil {
			t.Fatal(err)
		}
		var record map[string]any
		if err := json.Unmarshal([]byte(text), &record); err != nil {
			t.Fatal(err)
		}
		records = append(records, record)
	}
	return records
}

func TestWrapperStartupFailureRetainsAllowlistedIdentity(t *testing.T) {
	for _, mode := range []string{"arguments", "storage-lease"} {
		t.Run(mode, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "wrapper.log")
			t.Setenv("PAIR_WRAP_LOG", path)
			for key, value := range map[string]string{"PAIR_SCOPE_KEY": "scope", "PAIR_TAG": "", "PAIR_DATA_DIR": "", "PAIR_SESSION_NAME": "actual-session", "PAIR_LAUNCH_NONCE": "actual-nonce", "SECRET_TOKEN": "never-retain-this", "PAIR_ORIENTATION_PROMPT": "never-retain-prompt"} {
				t.Setenv(key, value)
			}
			if mode == "storage-lease" {
				t.Setenv("PAIR_TAG", "actual-tag")
			}
			var stdout, stderr bytes.Buffer
			if code := Run([]string{"--never-retain-argv"}, strings.NewReader(""), &stdout, &stderr); code != 1 {
				t.Fatalf("code=%d", code)
			}
			identity := startupRecords(t, path, "STARTUP-identity")
			if len(identity) != 1 || len(identity[0]) != 5 || identity[0]["actualPID"] != float64(os.Getpid()) || identity[0]["PAIR_SCOPE_KEY"] != "scope" || identity[0]["PAIR_SESSION_NAME"] != "actual-session" || identity[0]["PAIR_LAUNCH_NONCE"] != "actual-nonce" {
				t.Fatalf("identity=%v", identity)
			}
			failures := startupRecords(t, path, "STARTUP-error")
			if len(failures) != 1 || failures[0]["stage"] != mode {
				t.Fatalf("failures=%v", failures)
			}
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(raw), "never-retain") || strings.Contains(string(raw), "SECRET_TOKEN") {
				t.Fatalf("unallowlisted values retained: %q", raw)
			}
			if stdout.Len() != 0 || !strings.Contains(stderr.String(), "pair-wrap:") {
				t.Fatalf("original output changed: stdout=%q stderr=%q", stdout.String(), stderr.String())
			}
		})
	}
}

func TestWrapperStartupIdentityValuesAreBoundedAndEscaped(t *testing.T) {
	path := filepath.Join(t.TempDir(), "wrapper.log")
	t.Setenv("PAIR_WRAP_LOG", path)
	t.Setenv("PAIR_DATA_DIR", "")
	t.Setenv("PAIR_TAG", "")
	for _, key := range []string{"PAIR_SCOPE_KEY", "PAIR_SESSION_NAME", "PAIR_LAUNCH_NONCE"} {
		t.Setenv(key, strings.Repeat("\"\n界", 300))
	}
	Run(nil, strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	records := startupRecords(t, path, "STARTUP-identity")
	if len(records) != 1 {
		t.Fatalf("records=%v", records)
	}
	for _, key := range []string{"PAIR_SCOPE_KEY", "PAIR_SESSION_NAME", "PAIR_LAUNCH_NONCE"} {
		v := records[0][key].(string)
		if len(v) > 128 || !strings.HasSuffix(v, "<truncated>") {
			t.Fatalf("unbounded/misleading %s length=%d", key, len(v))
		}
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) > 4096 || bytes.Count(raw, []byte("\n")) != 2 {
		t.Fatalf("unbounded/injected log: bytes=%d", len(raw))
	}
}

func TestWrapperStartupDiagnosticUnavailablePreservesFailure(t *testing.T) {
	t.Setenv("PAIR_DATA_DIR", "")
	t.Setenv("PAIR_TAG", "")
	t.Setenv("PAIR_WRAP_LOG", t.TempDir())
	var out, stderr bytes.Buffer
	if code := Run(nil, strings.NewReader(""), &out, &stderr); code != 1 {
		t.Fatalf("code=%d", code)
	}
	if out.Len() != 0 || !strings.Contains(stderr.String(), "usage: pair-wrap") || !strings.Contains(stderr.String(), "startup diagnostics unavailable") {
		t.Fatalf("output=%q error=%q", out.String(), stderr.String())
	}
}

func TestWrapperStartupUnsetLogHasNoEffects(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	t.Setenv("PAIR_WRAP_LOG", "")
	t.Setenv("PAIR_DATA_DIR", "")
	t.Setenv("PAIR_TAG", "")
	var out, stderr bytes.Buffer
	if code := Run(nil, strings.NewReader(""), &out, &stderr); code != 1 {
		t.Fatalf("code=%d", code)
	}
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != 0 {
		t.Fatalf("unexpected effects: %v %v", entries, err)
	}
	if strings.Contains(stderr.String(), "diagnostics") {
		t.Fatalf("unset diagnostic output: %q", stderr.String())
	}
}

func TestWrapperChildExitIsNotStartupFailure(t *testing.T) {
	isolateNotificationSockets(t)
	path := filepath.Join(t.TempDir(), "wrapper.log")
	t.Setenv("PAIR_WRAP_LOG", path)
	t.Setenv("PAIR_DATA_DIR", "")
	t.Setenv("PAIR_TAG", "")
	var out, stderr bytes.Buffer
	code := Run([]string{"/bin/sh", "-c", "printf ordinary-child-output; exit 7"}, strings.NewReader(""), &out, &stderr)
	if code != 7 || !strings.Contains(out.String(), "ordinary-child-output") {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, out.String(), stderr.String())
	}
	if records := startupRecords(t, path, "STARTUP-error"); len(records) != 0 {
		t.Fatalf("child exit mislabeled startup: %v", records)
	}
	if records := startupRecords(t, path, "STARTUP-identity"); len(records) != 1 {
		t.Fatalf("identity=%v", records)
	}
}

func TestWrapperExecFailureReportsStageWithoutExecutableOrArguments(t *testing.T) {
	isolateNotificationSockets(t)
	path := filepath.Join(t.TempDir(), "wrapper.log")
	t.Setenv("PAIR_WRAP_LOG", path)
	t.Setenv("PAIR_DATA_DIR", "")
	t.Setenv("PAIR_TAG", "")
	var out, stderr bytes.Buffer
	if code := Run([]string{filepath.Join(t.TempDir(), "secret-executable"), "secret-argument"}, strings.NewReader(""), &out, &stderr); code != 1 {
		t.Fatalf("code=%d", code)
	}
	records := startupRecords(t, path, "STARTUP-error")
	if len(records) != 1 || records[0]["stage"] != "native-pty" || records[0]["errno"] == float64(0) {
		t.Fatalf("records=%v", records)
	}
	// Existing ordinary debug rows may mention the executable basename. Only
	// allowlisted startup rows enter the new qualification collector.
	raw, err := json.Marshal(append(records, startupRecords(t, path, "STARTUP-identity")...))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(raw, []byte("secret-")) {
		t.Fatalf("executable/argv leaked into startup records: %q", raw)
	}
	if !strings.Contains(stderr.String(), "secret-executable") || out.Len() != 0 {
		t.Fatalf("original output changed: stdout=%q stderr=%q", out.String(), stderr.String())
	}
}
