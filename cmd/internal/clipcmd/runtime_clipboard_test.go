package clipcmd

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
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
