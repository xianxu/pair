package clipcmd

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func privateClipboardTool(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("PATH", dir)
	for _, key := range []string{"SSH_CONNECTION", "SSH_CLIENT", "SSH_TTY"} {
		t.Setenv(key, "")
	}
	record := filepath.Join(dir, "clipboard")
	t.Setenv("TEST_CLIPBOARD", record)
	if err := os.WriteFile(filepath.Join(dir, "pbcopy"), []byte("#!/bin/sh\n"+body+"\n"), 0700); err != nil {
		t.Fatal(err)
	}
	return record
}

func TestClipboardCopyLocalWritesExactText(t *testing.T) {
	record := privateClipboardTool(t, `/bin/cat > "$TEST_CLIPBOARD"`)
	text := "full diagnostic 界\nwith a tail\x1b[31m"
	if err := NewOSRuntime().ClipboardCopyLocal(context.Background(), text); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(record)
	if err != nil || string(got) != text {
		t.Fatalf("clipboard = %q, err %v", got, err)
	}
}

func TestClipboardCopyLocalReportsToolFailure(t *testing.T) {
	privateClipboardTool(t, "exit 7")
	err := NewOSRuntime().ClipboardCopyLocal(context.Background(), "notice")
	var exit *exec.ExitError
	if !errors.As(err, &exit) || exit.ExitCode() != 7 {
		t.Fatalf("copy error = %v; want exit 7", err)
	}
}

func TestClipboardCopyLocalDeadline(t *testing.T) {
	privateClipboardTool(t, "exec /bin/sleep 30")
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	if err := NewOSRuntime().ClipboardCopyLocal(ctx, "notice"); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("copy error = %v; want deadline exceeded", err)
	}
}

func TestClipboardCopyLocalUnavailableAndLegacyOptional(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	for _, key := range []string{"SSH_CONNECTION", "SSH_CLIENT", "SSH_TTY"} {
		t.Setenv(key, "")
	}
	if err := NewOSRuntime().ClipboardCopyLocal(context.Background(), "notice"); !errors.Is(err, errClipboardUnavailable) {
		t.Fatalf("copy error = %v; want unavailable", err)
	}
	if err := NewOSRuntime().ClipboardCopy("notice"); err != nil {
		t.Fatalf("legacy optional copy changed: %v", err)
	}
}

func TestClipboardCopyLocalDoesNotWriteForRemoteOrCanceledCalls(t *testing.T) {
	for _, remote := range []string{"SSH_CONNECTION", "SSH_CLIENT", "SSH_TTY", "canceled"} {
		t.Run(remote, func(t *testing.T) {
			record := privateClipboardTool(t, `/bin/cat > "$TEST_CLIPBOARD"`)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			want := errClipboardUnavailable
			if remote == "canceled" {
				cancel()
				want = context.Canceled
			} else {
				t.Setenv(remote, "remote")
			}
			if err := NewOSRuntime().ClipboardCopyLocal(ctx, "notice"); !errors.Is(err, want) {
				t.Fatalf("copy error = %v; want %v", err, want)
			}
			if _, err := os.Stat(record); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("unexpected clipboard write: %v", err)
			}
		})
	}
}

func TestClipboardCopyLocalRetainsToolEvidence(t *testing.T) {
	privateClipboardTool(t, `printf 'pasteboard permission denied\n' >&2; exit 7`)
	err := NewOSRuntime().ClipboardCopyLocal(context.Background(), "notice")
	if err == nil || !strings.Contains(err.Error(), "pbcopy") || !strings.Contains(err.Error(), "pasteboard permission denied") {
		t.Fatalf("local tool evidence lost: %v", err)
	}
}

func TestClipboardInheritedStderrKeepsSuccessfulExit(t *testing.T) {
	for _, mode := range []string{"legacy", "local"} {
		t.Run(mode, func(t *testing.T) {
			record := privateClipboardTool(t, `/bin/cat > "$TEST_CLIPBOARD"
"$TEST_CLIPBOARD_HELPER" -test.run '^TestClipboardInheritedStderrHelper$' &
exit 0`)
			release := filepath.Join(filepath.Dir(record), "release")
			done := filepath.Join(filepath.Dir(record), "done")
			t.Setenv("TEST_CLIPBOARD_HELPER", os.Args[0])
			t.Setenv("TEST_CLIPBOARD_DAEMON", "1")
			t.Setenv("TEST_CLIPBOARD_RELEASE", release)
			t.Setenv("TEST_CLIPBOARD_DONE", done)
			t.Cleanup(func() {
				if err := os.WriteFile(release, []byte("release"), 0600); err != nil {
					t.Error(err)
					return
				}
				deadline := time.After(4 * time.Second)
				for {
					if _, err := os.Stat(done); err == nil {
						return
					}
					select {
					case <-deadline:
						t.Error("bounded stderr holder did not exit")
						return
					case <-time.After(10 * time.Millisecond):
					}
				}
			})
			if mode == "legacy" {
				if err := NewOSRuntime().ClipboardCopy("notice"); err != nil {
					t.Fatalf("stderr drain changed successful exit: %v", err)
				}
				return
			}
			result, err := NewOSRuntime().ClipboardCopyLocalResult(context.Background(), "notice")
			if err != nil || result.Metadata.Status != "copied" || result.Metadata.ExitCode == nil || *result.Metadata.ExitCode != 0 || !result.Metadata.StderrTruncated {
				t.Fatalf("successful process became failure: %+v / %v", result.Metadata, err)
			}
			got, readErr := os.ReadFile(record)
			if readErr != nil || string(got) != "notice" {
				t.Fatal("tool did not consume original stdin")
			}
		})
	}
}

// The tool's child holds stderr until its test owner releases it. This proves
// drain ordering without racing a short sleep; a hard lifetime prevents leaks.
func TestClipboardInheritedStderrHelper(t *testing.T) {
	if os.Getenv("TEST_CLIPBOARD_DAEMON") != "1" {
		return
	}
	defer os.Exit(0)
	deadline := time.After(3 * time.Second)
	for {
		if _, err := os.Stat(os.Getenv("TEST_CLIPBOARD_RELEASE")); err == nil {
			break
		}
		select {
		case <-deadline:
			return
		case <-time.After(10 * time.Millisecond):
		}
	}
	_ = os.WriteFile(os.Getenv("TEST_CLIPBOARD_DONE"), []byte("done"), 0600)
}
