package couchtty

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/ptychild"
	"github.com/xianxu/pair/cmd/internal/runtimebundle"
	"github.com/xianxu/pair/cmd/internal/terminal"
)

// This is a real transport join: Console -> PtyRunner -> Zellij -> pair wrap
// -> a raw, agent-named process. It uses no installed Pair session or data.
// Run with PAIR_LIVE_COUCH_NATIVE=1 PAIR_NATIVE_BINARY=/absolute/candidate.
func TestNativeConsoleWrapperZellij(t *testing.T) {
	if os.Getenv("PAIR_LIVE_COUCH_NATIVE") != "1" {
		t.Skip("set PAIR_LIVE_COUCH_NATIVE=1 and PAIR_NATIVE_BINARY to a freshly built pair")
	}
	for _, wrapped := range []bool{false, true} {
		name := "direct-zellij-baseline"
		if wrapped {
			name = "wrapped-zellij"
		}
		t.Run(name, func(t *testing.T) { runNativeConsoleJoin(t, wrapped) })
	}
}

func runNativeConsoleJoin(t *testing.T, wrapped bool) {
	binary := os.Getenv("PAIR_NATIVE_BINARY")
	if !filepath.IsAbs(binary) {
		t.Fatal("PAIR_NATIVE_BINARY must name an absolute, freshly built candidate")
	}
	zellij, err := exec.LookPath("zellij")
	if err != nil {
		t.Fatal(err)
	}
	python, err := exec.LookPath("python3")
	if err != nil {
		t.Fatal(err)
	}
	// Keep socket paths below the Unix-domain limit on macOS.
	dir, err := os.MkdirTemp("/tmp", "pcn-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	write := func(name, body string, mode os.FileMode) string {
		t.Helper()
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(body), mode); err != nil {
			t.Fatal(err)
		}
		return path
	}
	if err := os.Mkdir(filepath.Join(dir, "pair-data"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "bin"), 0700); err != nil {
		t.Fatal(err)
	}
	// Slug generation is outside terminal transport; keep this unrelated helper local.
	write("bin/pair", "#!/bin/sh\nexit 0\n", 0700)
	paths, err := artifactpath.ResolveScoped(filepath.Join(dir, "pair-data"), "native-fixture")
	if err != nil {
		t.Fatal(err)
	}
	receipts := filepath.Join(dir, "receipts")
	fixture := write("codex", "#!"+python+"\n"+nativeAgentFixture, 0700)
	config := write("config.kdl", "pane_frames false\nshow_startup_tips false\nshow_release_notes false\non_force_close \"quit\"\n", 0600)
	layoutBody := fmt.Sprintf("layout { pane borderless=true command=%q { args \"wrap\" \"--scrollback-log\" %q %q %q; }; }\n", binary, filepath.Join(dir, "scrollback"), fixture, receipts)
	if !wrapped {
		layoutBody = fmt.Sprintf("layout { pane borderless=true command=%q { args %q; }; }\n", fixture, receipts)
	}
	layout := write("layout.kdl", layoutBody, 0600)
	profileEnv, err := runtimebundle.TerminalEnvironment(dir)
	if err != nil {
		t.Fatal(err)
	}
	env := []string{"ZELLIJ=", "ZELLIJ_SESSION_NAME=", "ZELLIJ_PANE_ID=", "XDG_CACHE_HOME=" + filepath.Join(dir, "cache"), "XDG_CONFIG_HOME=" + filepath.Join(dir, "config"), "ZELLIJ_SOCKET_DIR=" + filepath.Join(dir, "socket"), "PAIR_TAG=native-fixture", "PAIR_SCOPE_KEY=", "PAIR_HOME=" + dir, "PAIR_DATA_DIR=" + filepath.Join(dir, "pair-data")}
	session := filepath.Base(dir)
	host := newVTHost(24, 100)
	inputR, inputW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	con := New(host, inputR)
	runner := &couchcore.PtyRunner{Size: con.ChildSize, Sink: con.Deliver, Environment: func() ([]string, error) {
		return profileEnv, nil
	}}
	handle, err := runner.Start(dir, []string{"/bin/sh", "-c", `tty > "$1"; shift; exec "$@"`, "native-launch", paths.OuterTTY(), zellij, "--session", session, "--config", config, "--new-session-with-layout", layout, "--data-dir", filepath.Join(dir, "data")}, env)
	if err != nil {
		t.Fatal(err)
	}
	child := handle.(couchcore.TerminalHandle).Terminal()
	con.Attach(handle.ID(), "native-agent", child)
	done := make(chan int, 1)
	go func() { done <- con.Run() }()
	t.Cleanup(func() {
		if t.Failed() {
			t.Logf("native diagnostic child=%q screen=%q", child.Snapshot(), host.childArea())
		}
		killCtx, cancelKill := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelKill()
		kill := exec.CommandContext(killCtx, zellij, "kill-session", session)
		kill.Env = append(os.Environ(), env...)
		_ = kill.Run()
		con.Stop()
		_ = child.Close()
		_ = inputW.Close()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Error("native Console did not join")
		}
	})
	readReceipts := func() string { b, _ := os.ReadFile(receipts); return string(b) }
	waitUpTo(t, 15*time.Second, "wrapped native process and CPR", func() bool {
		return strings.Contains(readReceipts(), "CPR=") && strings.Contains(host.childArea(), "native 界 e ready")
	})
	if !strings.Contains(readReceipts(), "CPR=1b5b") || !strings.Contains(readReceipts(), "52\n") {
		t.Fatalf("fixture did not receive a cursor response: %s", readReceipts())
	}
	if !strings.Contains(host.row(24), "native-agent") {
		t.Fatalf("lost chrome: %q", host.row(24))
	}
	// Policy consumes the shortcut while the actual enhanced event is encoded
	// separately by each terminal boundary. Paste carries embedded ESC intact.
	_, _ = inputW.Write([]byte("\x1b[97;1u"))
	waitUpTo(t, 3*time.Second, "enhanced key reaches wrapped agent", func() bool { return strings.Contains(readReceipts(), "61") })
	_, _ = inputW.Write([]byte("\x1b[200~paste-界\x1b[31m\x1b[201~"))
	wantPaste := hex.EncodeToString([]byte("\x1b[200~paste-界\x1b[31m\x1b[201~"))
	waitUpTo(t, 3*time.Second, "paste reaches wrapped agent", func() bool { return strings.Contains(strings.ReplaceAll(readReceipts(), "\n", ""), wantPaste) })
	// Focus and a complete drag cross the native transport, not a fake decoder.
	_, _ = inputW.Write([]byte("\x1b[O\x1b[I\x1b[<0;4;4M\x1b[<32;6;4M\x1b[<0;6;4m"))
	waitUpTo(t, 3*time.Second, "focus and drag receipts", func() bool {
		var raw []byte
		for _, line := range strings.Split(readReceipts(), "\n") {
			b, err := hex.DecodeString(line)
			if err == nil {
				raw = append(raw, b...)
			}
		}
		return bytes.Contains(raw, []byte("\x1b[O")) && bytes.Contains(raw, []byte("\x1b[I")) && bytes.Contains(raw, []byte("\x1b[<0;4;4M")) && bytes.Contains(raw, []byte("\x1b[<32;6;4M")) && bytes.Contains(raw, []byte("\x1b[<0;6;4m"))
	})
	// The fixture emits a split combining mark too. Native Zellij 0.45.1 drops
	// it before our endpoint; the direct subtest establishes that boundary.
	if bytes.Contains(child.Snapshot(), []byte("e\xcc\x81")) {
		t.Log("native Zellij retained combining mark")
	} else {
		t.Log("native Zellij omitted combining mark before Endpoint (compare direct baseline); CJK asserted independently")
	}
	if wrapped {
		_, _ = inputW.Write([]byte("\r"))
		waitUpTo(t, 3*time.Second, "one native notification through outer tty", func() bool {
			return strings.Count(host.Written(), "\x1b]777;notify;pair;native-fixture-complete\x1b\\") == 1
		})
	}
	// A real second PTY exercises actor switching, hidden output and panel entry.
	second, err := runner.Start(dir, []string{"/bin/sh", "-c", "printf 'shell-counterpart-ready\\r\\n'; cat"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	secondChild := second.(couchcore.TerminalHandle).Terminal()
	defer secondChild.Close()
	con.Attach(second.ID(), "native-shell", secondChild)
	con.Switch(second.ID())
	waitUpTo(t, 3*time.Second, "real shell actor", func() bool {
		return con.presenter.View().Admitted == secondChild.Endpoint().ID() && strings.Contains(host.childArea(), "shell-counterpart-ready")
	})
	_, _ = inputW.Write([]byte{0})
	waitUpTo(t, 3*time.Second, "panel", func() bool { con.mu.Lock(); defer con.mu.Unlock(); return con.focus.IsPanel() })
	host.SetSize(ptychild.Size{Rows: 28, Cols: 110})
	waitUpTo(t, 3*time.Second, "both real endpoints resized", func() bool {
		a, ea := child.Endpoint().Snapshot(time.Now())
		b, eb := secondChild.Endpoint().Snapshot(time.Now())
		return ea == nil && eb == nil && a.Geometry == (terminal.Geometry{Rows: 27, Cols: 110}) && b.Geometry == (terminal.Geometry{Rows: 27, Cols: 110})
	})
	con.Switch(handle.ID())
	waitUpTo(t, 3*time.Second, "native actor restored", func() bool {
		return con.presenter.View().Admitted == child.Endpoint().ID() && strings.Contains(host.childArea(), "native 界 e ready")
	})
	if err := con.presenter.Flush(context.Background()); err != nil {
		t.Fatal(err)
	}
	if wrapped && strings.Count(host.Written(), "native-fixture-complete") != 1 {
		t.Fatalf("notification duplicated on switch: %q", host.Written())
	}
	assertNativeOracle(t, host.Written(), 110, 28, "native 界 e ready", "native-agent")
	// A no-mouse child makes Zellij own native text selection. Inspect its
	// published highlight while the button is still held, then inspect OSC52.
	if err := os.WriteFile(receipts+".select", []byte("select"), 0600); err != nil {
		t.Fatal(err)
	}
	waitUpTo(t, 3*time.Second, "native no-mouse selection surface", func() bool { return strings.Contains(host.childArea(), "selectable-native-text") })
	before, err := child.Endpoint().Snapshot(time.Now())
	if err != nil {
		t.Fatal(err)
	}
	beforeWire := host.Written()
	_, _ = inputW.Write([]byte("\x1b[<0;1;1M\x1b[<32;10;1M"))
	waitUpTo(t, 3*time.Second, "native held-drag highlight before release", func() bool {
		now, err := child.Endpoint().Snapshot(time.Now())
		if err != nil {
			return false
		}
		for i := 0; i < 9; i++ {
			if !reflect.DeepEqual(before.Cells[i].Style, now.Cells[i].Style) {
				return true
			}
		}
		return false
	})
	if err := con.presenter.Flush(context.Background()); err != nil {
		t.Fatal(err)
	}
	assertNativeHighlight(t, beforeWire, host.Written(), 110, 28)
	_, _ = inputW.Write([]byte("\x1b[<0;10;1m"))
	waitUpTo(t, 3*time.Second, "native selection clipboard", func() bool { return strings.Contains(host.Written(), "\x1b]52;") })
	wire := host.Written()
	start := strings.LastIndex(wire, "\x1b]52;")
	payload := strings.SplitN(wire[start+5:], ";", 2)
	if len(payload) != 2 {
		t.Fatalf("bad clipboard envelope %q", wire[start:])
	}
	encoded := strings.SplitN(payload[1], "\x1b\\", 2)[0]
	copied, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil || string(copied) != "selectabl" {
		t.Fatalf("native selection copied %q (%v), want selectabl", copied, err)
	}
	t.Log("native selection highlighted in endpoint and independent xterm before release; clipboard selectabl emitted on release")
	t.Logf("native Console join (wrapped=%t): CPR, fragmented CJK, synchronized update, enhanced key, paste, focus, drag, shell/panel switches, resize; receipts=%s", wrapped, readReceipts())
}

func assertNativeOracle(t *testing.T, wire string, cols, rows int, body, chrome string) {
	t.Helper()
	request, _ := json.Marshal(map[string]any{"Cols": cols, "Rows": rows, "Chunks": []string{wire}})
	cmd := exec.Command("node", "../../../tests/terminal-oracle/driver.cjs")
	cmd.Stdin = bytes.NewReader(request)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("independent xterm oracle: %v %s", err, out)
	}
	var frames []struct{ Lines []string }
	if err := json.Unmarshal(out, &frames); err != nil {
		t.Fatal(err)
	}
	if len(frames) != 1 || len(frames[0].Lines) != rows {
		t.Fatalf("invalid oracle frame: %s", out)
	}
	if !strings.Contains(strings.Join(frames[0].Lines[:rows-1], "\n"), body) || !strings.Contains(frames[0].Lines[rows-1], chrome) {
		t.Fatalf("independent native frame body/chrome mismatch: %q", frames[0].Lines)
	}
}

const nativeAgentFixture = `import os,sys,time,tty,select
fd=os.open('/dev/tty',os.O_RDWR)
tty.setraw(fd)
log=open(sys.argv[1],'a',buffering=1)
os.write(fd,b'\x1b[6n')
reply=b''
end=time.monotonic()+5
while time.monotonic()<end and not reply.endswith(b'R'):
 if select.select([fd],[],[],.1)[0]: reply+=os.read(fd,4096)
log.write('CPR='+reply.hex()+'\n')
os.write(fd,b'\x1b[?2004h\x1b[?1004h\x1b[?1002h\x1b[?1006h\x1b[?2026h\x1b[2J\x1b[H')
for part in ['native '.encode(),b'\xe7',b'\x95',b'\x8c',b' e',b'\xcc',b'\x81',b' ready']:
 os.write(fd,part)
 time.sleep(.005)
os.write(fd,b'\x1b[?2026l')
while True:
 if os.path.exists(sys.argv[1]+'.select'):
  os.unlink(sys.argv[1]+'.select')
  os.write(fd,b'\x1b[?1000l\x1b[?1002l\x1b[?1003l\x1b[?1006l\x1b[H\x1b[2Kselectable-native-text')
 if not select.select([fd],[],[],.02)[0]: continue
 data=os.read(fd,65536)
 if not data: break
 log.write(data.hex()+'\n')
 if b'\r' in data: os.write(fd,b'\x1b]9;native-fixture-complete\x07\x1b]9;native-fixture-complete\x07')
`

func assertNativeHighlight(t *testing.T, before, after string, cols, rows int) {
	t.Helper()
	if !strings.HasPrefix(after, before) {
		t.Fatal("host wire capture lost prefix")
	}
	request, _ := json.Marshal(map[string]any{"Cols": cols, "Rows": rows, "Chunks": []string{before, after[len(before):]}})
	cmd := exec.Command("node", "../../../tests/terminal-oracle/driver.cjs")
	cmd.Stdin = bytes.NewReader(request)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("native highlight oracle: %v %s", err, out)
	}
	var frames []struct {
		Cells [][]struct {
			FG, BG  int
			Inverse bool
		}
	}
	if err := json.Unmarshal(out, &frames); err != nil {
		t.Fatal(err)
	}
	if len(frames) != 2 {
		t.Fatal("missing selection oracle frames")
	}
	for i := 0; i < 9; i++ {
		if frames[0].Cells[0][i] != frames[1].Cells[0][i] {
			return
		}
	}
	t.Fatal("native held selection did not reach independent xterm styles")
}
