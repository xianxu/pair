package sessioninventory

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

// This explicit opt-in checks the real noncredential configuration reader only.
// Ordinary tests never inspect the developer's Codex installation or credentials.
func TestCodexMachineReadOnlyConfigConformance(t *testing.T) {
	if os.Getenv("PAIR_CODEX_READONLY_CONFORMANCE") != "1" {
		t.Skip("explicit real-machine read-only conformance only")
	}
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	r := newCodexConfigOSReader()
	read := r.read
	files, bytes := 0, 0
	r.read = func(ctx context.Context, path string) ([]byte, error) {
		// Refuse an accidental expansion of the production reader's file surface
		// before touching the machine; never print native paths or file contents.
		switch filepath.Base(path) {
		case "config.toml", "requirements.toml", "managed_config.toml", "environments.toml":
		default:
			t.Fatal("unexpected noncredential config input")
		}
		files++
		b, err := read(ctx, path)
		bytes += len(b)
		t.Logf("input=%s absent=%t nofollow=%t failed=%t", filepath.Base(path), os.IsNotExist(err), errors.Is(err, syscall.ELOOP), err != nil)
		return b, err
	}
	for _, mode := range []CodexInvocationMode{CodexModeTUI, CodexModeExec} {
		got, err := readCodexSelectionConfig(t.Context(), CodexConfigInvocation{StartupCWD: cwd, CWD: cwd, Mode: mode}, r)
		if err != nil {
			t.Fatal(err)
		}
		// This machine has no source-proven API-only local bootstrap, and its
		// project metadata may also exceed the secure observation boundary. Do not
		// inspect auth to improve the result or run native startup as an oracle.
		if got.Status != CodexConfigCompatibility || (got.Reason != CodexConfigCloudUnknown && got.Reason != CodexConfigObservationIncomplete) {
			t.Fatalf("machine projection: status=%s reason=%s", got.Status, got.Reason)
		}
		t.Logf("mode=%s status=%s reason=%s; no selection or admission", mode, got.Status, got.Reason)
	}
	t.Logf("bounded noncredential reads: files=%d bytes=%d", files, bytes)
}
