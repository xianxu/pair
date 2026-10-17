package clipcmd

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestClipboardCopyResultRetainsStatusWithoutPublishingStderr(t *testing.T) {
	for _, code := range []string{"0", "7"} {
		t.Run(code, func(t *testing.T) {
			dir := privateClipboardTool(t, `printf '\033[31mprivate notice\033[0m\n' >&2; exit `+code)
			result, err := NewOSRuntime().ClipboardCopyLocalResult(context.Background(), "private notice")
			m := result.Metadata
			if m.Tool != "pbcopy" || m.Executable != filepath.Join(filepath.Dir(dir), "pbcopy") || !m.Started || m.ExitCode == nil || m.StderrBytes != int64(len("\x1b[31mprivate notice\x1b[0m\n")) || m.StderrHash == "" || result.Stderr != "private notice" {
				t.Fatalf("unexpected result: %+v", result)
			}
			if code == "0" {
				if err != nil || m.Status != "copied" || *m.ExitCode != 0 {
					t.Fatalf("successful result: %+v / %v", m, err)
				}
			} else {
				var typed *ClipboardCopyError
				var exit *exec.ExitError
				if !errors.As(err, &typed) || !errors.As(err, &exit) || exit.ExitCode() != 7 || m.Status != "failed" {
					t.Fatalf("typed failure lost: %v", err)
				}
			}
			raw, e := json.Marshal(m)
			if e != nil || strings.Contains(string(raw), "private") || strings.Contains(string(raw), "notice") {
				t.Fatalf("metadata contains content: %s / %v", raw, e)
			}
		})
	}
}

func TestClipboardStderrSummaryBoundsAndHashesFullStream(t *testing.T) {
	var s clipboardStderrSummary
	text := strings.Repeat("x", clipboardStderrLimit+100) + "\x1b[2J"
	for _, part := range []string{text[:32], text[32:]} {
		n, err := s.Write([]byte(part))
		if n != len(part) || err != nil {
			t.Fatal(n, err)
		}
	}
	var result ClipboardCopyResult
	s.apply(&result)
	sum := sha256.Sum256([]byte(text))
	if len(result.Stderr) > clipboardStderrLimit || !result.Metadata.StderrTruncated || result.Metadata.StderrBytes != int64(len(text)) || result.Metadata.StderrHash != hex.EncodeToString(sum[:8]) {
		t.Fatalf("unbounded or inaccurate summary: %+v", result.Metadata)
	}
}

func TestClipboardCopyResultCanceledAndUnavailableAreNotExits(t *testing.T) {
	for _, mode := range []string{"missing", "remote", "canceled", "timeout", "start-failed"} {
		t.Run(mode, func(t *testing.T) {
			privateClipboardTool(t, "exec /bin/sleep 30")
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			want := mode
			switch mode {
			case "missing":
				t.Setenv("PATH", t.TempDir())
				want = "unavailable"
			case "remote":
				t.Setenv("SSH_TTY", "remote")
			case "canceled":
				cancel()
			case "timeout":
				var stop context.CancelFunc
				ctx, stop = context.WithTimeout(ctx, time.Millisecond*20)
				defer stop()
			case "start-failed":
				if err := os.WriteFile(filepath.Join(os.Getenv("PATH"), "pbcopy"), []byte("#!/definitely/missing/interpreter\n"), 0700); err != nil {
					t.Fatal(err)
				}
				want = "failed"
			}
			result, err := NewOSRuntime().ClipboardCopyLocalResult(ctx, "private")
			if err == nil || result.Metadata.Status != want || result.Metadata.ExitCode != nil {
				t.Fatalf("%s: %+v / %v", mode, result, err)
			}
			if mode == "canceled" && (!errors.Is(err, context.Canceled) || !result.Metadata.Canceled) {
				t.Fatal(err)
			}
			if mode == "timeout" && (!errors.Is(err, context.DeadlineExceeded) || !result.Metadata.TimedOut) {
				t.Fatal(err)
			}
		})
	}
}
