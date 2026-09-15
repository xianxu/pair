package terminal

import (
	"bytes"
	"context"
	vt "github.com/charmbracelet/x/vt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestProfileQueryLiteralReplies(t *testing.T) {
	for _, tc := range []struct{ query, want string }{
		{"\x1b[c", "\x1b[?1;2c"}, {"\x1b[0c", "\x1b[?1;2c"}, {"\x1b[>c", "\x1b[>0;1;0c"},
		{"\x1b[1c", ""}, {"\x1b[>2c", ""},
		{"\x1bP+q544e\x1b\\", "\x1bP1+r544e=706169722d76742d323536636f6c6f72\x1b\\"},
		{"\x1bP+q436f;6b63757531\x1b\\", "\x1bP1+r436f=323536;6b63757531=1b4f41\x1b\\"},
		{"\x1bP+q4d73\x1b\\", "\x1bP0+r\x1b\\"},
		{"\x1bP+q736978656c\x1b\\", "\x1bP0+r\x1b\\"},
		{"\x1bP+q787465726d\x1b\\", "\x1bP0+r\x1b\\"},
		{"\x1bP+qzz\x1b\\", "\x1bP0+r\x1b\\"},
	} {
		t.Run(tc.query, func(t *testing.T) {
			for split := 0; split <= len(tc.query); split++ {
				e := vt.NewEmulator(10, 3)
				var out bytes.Buffer
				e.SetReplyWriter(&out)
				if err := DefaultProfile().InstallQueries(e, &out); err != nil {
					t.Fatal(err)
				}
				e.Write([]byte(tc.query[:split]))
				e.Write([]byte(tc.query[split:]))
				e.Close()
				if out.String() != tc.want {
					t.Fatalf("split%d reply%q want%q", split, out.String(), tc.want)
				}
			}
		})
	}
}
func TestProfileRejectsUnsupportedAdvertising(t *testing.T) {
	for _, p := range []Profile{{TERM: "xterm-256color"}, {TERM: PairTERM, Graphics: true}, {TERM: PairTERM, ClipboardRead: true}} {
		if p.Validate() == nil {
			t.Fatalf("accepted unsupported profile%+v", p)
		}
	}
}
func TestProfileTerminfoCompiledContract(t *testing.T) {
	source, err := os.ReadFile("../../../terminfo/pair-vt-256color.ti")
	if err != nil {
		t.Fatal(err)
	}
	if string(source) != DefaultProfile().TerminfoSource() {
		t.Fatal("committed terminfo differs from profile capability contract")
	}
	dir := t.TempDir()
	cmd := exec.Command("tic", "-x", "-o", dir, "../../../terminfo/pair-vt-256color.ti")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("compile terminfo: %v %s", err, out)
	}
	cmd = exec.Command("infocmp", "-1", "-x", "-A", dir, PairTERM)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("read compiled terminfo: %v %s", err, out)
	}
	for _, forbidden := range []string{"\tbce,", "\tccc,", "\tinitc=", "\tmc0=", "\tmc4=", "\tmc5=", "\tMs=", "\tsixel"} {
		if strings.Contains(string(out), forbidden) {
			t.Errorf("advertised unsupported capability%s", forbidden)
		}
	}
	for cap, want := range map[string]string{"colors": "256\n", "cup": "\x1b[3;4H", "smcup": "\x1b[?1049h", "rmcup": "\x1b[?1049l", "sc": "\x1b7", "rc": "\x1b8", "kcuu1": "\x1bOA", "setaf": "\x1b[38;5;2m", "setab": "\x1b[48;5;2m"} {
		args := []string{"-T", PairTERM, cap}
		if cap == "cup" {
			args = append(args, "2", "3")
		} else if cap == "setaf" || cap == "setab" {
			args = append(args, "2")
		}
		cmd := exec.Command("tput", args...)
		cmd.Env = append(os.Environ(), "TERMINFO="+dir)
		got, err := cmd.Output()
		if err != nil || string(got) != want {
			t.Errorf("compiled%s=%q err%v want%q", cap, got, err, want)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "p", PairTERM)); err != nil {
		if _, hexerr := os.Stat(filepath.Join(dir, "70", PairTERM)); hexerr != nil {
			t.Fatalf("compiled entry missing: %v", err)
		}
	}
}

func TestProfileQueriesUseEndpointReplyTransport(t *testing.T) {
	e, out := newEndpointTest(t, "profile")
	if _, err := e.Feed([]byte("\x1b[c\x1b[>c\x1bP+q544e;436f\x1b\\"), time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := e.Flush(context.Background()); err != nil {
		t.Fatal(err)
	}
	want := "\x1b[?1;2c\x1b[>0;1;0c\x1bP1+r544e=706169722d76742d323536636f6c6f72;436f=323536\x1b\\"
	if string(out.Bytes()) != want {
		t.Fatalf("endpoint reply%q want%q", out.Bytes(), want)
	}
}
