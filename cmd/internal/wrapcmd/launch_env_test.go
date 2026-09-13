package wrapcmd

import (
	"bytes"
	"github.com/creack/pty"
	"github.com/xianxu/pair/cmd/internal/launcher"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestLaunchEnvCommandExactArguments(t *testing.T) {
	t.Setenv("PAIR_TAG", "")
	t.Setenv("PAIR_DATA_DIR", "")
	target := filepath.Join(t.TempDir(), "args")
	want := []string{"", "a b", "雪", "$HOME", "$(echo nope)", "*", "a\nline"}
	args := append([]string{"-c", `printf '%s\000' "$@" > "$TARGET"`, "fake"}, want...)
	raw, err := launcher.EncodeAgentCommand(launcher.AgentCommand{Executable: "sh", Argv: args})
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv(launcher.AgentCommandEnv, raw)
	t.Setenv("TARGET", target)
	master, tty, err := pty.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer master.Close()
	defer tty.Close()
	var stderr bytes.Buffer
	if code := Run([]string{"--from-launch-env"}, tty, io.Discard, &stderr); code != 0 {
		t.Fatalf("code %d: %s", code, &stderr)
	}
	actual, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	got := strings.Split(strings.TrimSuffix(string(actual), "\x00"), "\x00")
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("argv %#v want %#v", got, want)
	}
	// Inherited command authority cannot override an ordinary explicit command.
	if code := Run([]string{"sh", "-c", "exit 7"}, tty, io.Discard, &stderr); code != 7 {
		t.Fatalf("ordinary command exit %d", code)
	}
}
func TestLaunchEnvRequiresSoleCommandAuthority(t *testing.T) {
	t.Setenv(launcher.AgentCommandEnv, `{"executable":"sh","argv":[]}`)
	for _, args := range [][]string{{"--from-launch-env", "sh"}, {"--from-launch-env", "--from-launch-env"}} {
		var stderr bytes.Buffer
		if code := Run(args, strings.NewReader(""), io.Discard, &stderr); code != 1 {
			t.Fatalf("accepted %#v", args)
		}
	}
	t.Setenv(launcher.AgentCommandEnv, "")
	var stderr bytes.Buffer
	if code := Run([]string{"--from-launch-env"}, strings.NewReader(""), io.Discard, &stderr); code != 1 {
		t.Fatal("accepted missing env")
	}
}
