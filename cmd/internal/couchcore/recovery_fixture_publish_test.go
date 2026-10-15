package couchcore

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// recoveryFixturePublisher installs the source-agent fixture's file publisher.
// beforeWrite is a test-only barrier after opening the write target.
func recoveryFixturePublisher(beforeWrite string) string {
	return `publish_fixture_value() {
 destination=$1
 value=$2
 stage="${destination}.tmp.$$"
 {
 ` + beforeWrite + `
 printf '%s' "$value"
 } > "$stage" && mv -f "$stage" "$destination"
}
`
}

func TestRecoveryFixturePublishesCompleteValuesAtomically(t *testing.T) {
	for _, value := range []string{"12345", "terminal_7"} {
		t.Run(value, func(t *testing.T) {
			base := t.TempDir()
			destination := filepath.Join(base, "published")
			opened := filepath.Join(base, "opened")
			release := filepath.Join(base, "release")
			// A FIFO makes the interleaving explicit: the child opens its write target,
			// signals that point, then cannot write its value until this test releases it.
			if out, err := exec.Command("mkfifo", release).CombinedOutput(); err != nil {
				t.Fatalf("mkfifo: %v %s", err, out)
			}
			releasePipe, err := os.OpenFile(release, os.O_RDWR, 0600)
			if err != nil {
				t.Fatal(err)
			}
			defer releasePipe.Close()
			barrier := `: > "$3"; IFS= read -r release < "$4"`
			script := recoveryFixturePublisher(barrier) + `publish_fixture_value "$1" "$2" "$3" "$4"`
			ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, "sh", "-c", script, "fixture", destination, value, opened, release)
			var output strings.Builder
			cmd.Stdout, cmd.Stderr = &output, &output
			if err := cmd.Start(); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = cmd.Process.Kill(); _ = cmd.Wait() })
			waitFile(t, opened)
			_, statErr := os.Stat(destination)
			// Always release the child before reporting the assertion, including on the
			// old direct-write implementation where the destination is already empty.
			if _, err := releasePipe.WriteString("continue\n"); err != nil {
				t.Fatal(err)
			}
			if err := cmd.Wait(); err != nil {
				t.Fatalf("publisher: %v %s", err, output.String())
			}
			if !os.IsNotExist(statErr) {
				t.Errorf("public destination visible before complete write: %v", statErr)
			}
			body, err := os.ReadFile(destination)
			if err != nil {
				t.Fatal(err)
			}
			if string(body) != value {
				t.Fatalf("published %q, want %q", body, value)
			}
		})
	}
}
