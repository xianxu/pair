package couchtty

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestNativeEvidenceLifetimeHelper(t *testing.T) {
	if os.Getenv("PAIR_NATIVE_EVIDENCE_HELPER") != "1" {
		t.Skip("subprocess helper")
	}
	dirs := []string{nativeEvidenceDirectory(t, "reattach")}
	if err := os.WriteFile(filepath.Join(dirs[0], "sample.raw"), []byte("synthetic"), 0600); err != nil {
		t.Fatal(err)
	}
	// The process-owner cleanup is registered after scratch allocation, just
	// like Console cleanup in the native fixture. Scratch must still exist.
	t.Cleanup(func() {
		if _, err := os.Stat(dirs[0]); err != nil {
			t.Errorf("owner cleanup lost scratch: %v", err)
		}
		t.Logf("bounded failure diagnostics: %q", nativeEvidenceDiagnostic([]byte("synthetic")))
		data, _ := json.Marshal(dirs)
		if err := os.WriteFile(os.Getenv("PAIR_NATIVE_EVIDENCE_RECORD"), data, 0600); err != nil {
			t.Error(err)
		}
	})
	if os.Getenv("PAIR_NATIVE_EVIDENCE_FAIL") == "1" {
		t.Error("intentional failure verifies cleanup")
	}
}

func TestNativeEvidenceLifetime(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(map[bool]string{false: "success", true: "failure"}[fail], func(t *testing.T) {
			record := filepath.Join(t.TempDir(), "directories.json")
			cmd := exec.Command(os.Args[0], "-test.run=^TestNativeEvidenceLifetimeHelper$")
			cmd.Env = append(os.Environ(), "PAIR_NATIVE_EVIDENCE_HELPER=1", "PAIR_NATIVE_EVIDENCE_RECORD="+record, "PAIR_NATIVE_EVIDENCE_FAIL="+map[bool]string{false: "0", true: "1"}[fail])
			out, err := cmd.CombinedOutput()
			if (err != nil) != fail {
				t.Fatalf("subprocess err=%v output=%s", err, out)
			}
			if bytes.Contains(out, []byte("owner cleanup lost scratch")) {
				t.Fatalf("cleanup ordering: %s", out)
			}
			data, err := os.ReadFile(record)
			if err != nil {
				t.Fatal(err)
			}
			var dirs []string
			if err := json.Unmarshal(data, &dirs); err != nil {
				t.Fatal(err)
			}
			if len(dirs) != 1 {
				t.Fatalf("invocation evidence=%v", dirs)
			}
			for _, dir := range dirs {
				// Keep even a deliberately broken cleanup mutation invocation-owned.
				t.Cleanup(func() {
					if err := os.RemoveAll(dir); err != nil {
						t.Errorf("remove failed regression scratch: %v", err)
					}
				})
				if _, err := os.Stat(dir); !os.IsNotExist(err) {
					t.Errorf("evidence survived invocation: %s err=%v", dir, err)
				}
			}
		})
	}
}

func TestNativeEvidenceDiagnosticBound(t *testing.T) {
	raw := []byte(strings.Repeat("x", 1<<20))
	got := nativeEvidenceDiagnostic(raw)
	if len(got) > 4096 || !bytes.Equal(got, raw[:4096]) {
		t.Fatalf("diagnostic size=%d", len(got))
	}
	if got := nativeEvidenceDiagnostic([]byte("short")); string(got) != "short" {
		t.Fatalf("short=%q", got)
	}
}

// testing.T owns every evidence directory for this invocation, including failed
// tests. TempDir cleanup reports removal errors rather than silently retaining
// data. Artifacts are diagnostic scratch, never implicit cross-run retention.
func nativeEvidenceDirectory(t testing.TB, family string) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), family)
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	return dir
}

func nativeEvidenceDiagnostic(raw []byte) []byte {
	return raw[:min(len(raw), 4096)]
}
