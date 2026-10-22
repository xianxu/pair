package workbenchshortcut_test

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/pairlifecycletest"
	"github.com/xianxu/pair/cmd/internal/procutil"
	"github.com/xianxu/pair/cmd/internal/testpair"
	wb "github.com/xianxu/pair/cmd/internal/workbenchshortcut"
	"github.com/xianxu/pair/cmd/internal/zellijexec"
	"github.com/xianxu/pair/cmd/internal/zellijpane"
)

type nativeTerminalOwner struct {
	PID            int
	Pane, Identity string
}

// The helper keeps exactly the production argv. In particular testing's flag
// parser must never interpret "term" before native process observation.
func TestMain(m *testing.M) {
	if os.Getenv("PAIR_NATIVE_TERMINAL_HELPER") == "1" {
		if err := nativeTerminalHelper(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func nativeTerminalHelper() error {
	if len(os.Args) != 2 || os.Args[1] != "term" {
		return fmt.Errorf("helper argv: %q", os.Args)
	}
	owner := nativeTerminalOwner{PID: os.Getpid(), Pane: os.Getenv("ZELLIJ_PANE_ID"), Identity: (procutil.Source{}).Identity(os.Getpid())}
	reg := wb.TerminalPaneRegistry{DataDir: os.Getenv("PAIR_DATA_DIR"), Tag: os.Getenv("PAIR_TAG")}
	if owner.Identity == "" || owner.Pane == "" {
		return fmt.Errorf("missing helper identity: %+v", owner)
	}
	if err := reg.Register(owner.Pane, owner.PID); err != nil {
		return err
	}
	data, err := json.Marshal(owner)
	if err != nil {
		return err
	}
	path := filepath.Join(os.Getenv("PAIR_NATIVE_TERMINAL_READY"), owner.Pane)
	if err := os.WriteFile(path+".tmp", data, 0600); err != nil {
		return err
	}
	if err := os.Rename(path+".tmp", path); err != nil {
		return err
	}
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		return err
	}
	if strings.TrimSpace(line) != "release" {
		return fmt.Errorf("unexpected release: %q", line)
	}
	return syscall.Exec("/bin/sh", []string{"sh", "-i"}, os.Environ())
}

func TestTerminalOwnerNativeExecConformance(t *testing.T) {
	if os.Getenv("PAIR_LIVE_ZELLIJ") != "1" {
		t.Skip("set PAIR_LIVE_ZELLIJ=1 for disposable native terminal owner conformance")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	defer cancel()
	base := t.TempDir()
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	// testpair's shared-image placement also supports the test executable: no
	// executable copy means no second AMFI assessment under the private HOME.
	t.Setenv("PAIR_TEST_BINARY", self)
	pair := testpair.AtContext(t, ctx, filepath.Join(base, "pair"))
	original, err := os.Stat(self)
	if err != nil {
		t.Fatal(err)
	}
	alias, err := os.Stat(pair)
	if err != nil || !os.SameFile(original, alias) {
		t.Fatalf("helper alias changed executable inode: %v", err)
	}
	zellij, err := exec.LookPath("zellij")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(zellij, filepath.Join(base, "zellij")); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"HOME", "XDG_CONFIG_HOME", "XDG_DATA_HOME", "XDG_CACHE_HOME", "ZELLIJ_CONFIG_DIR"} {
		t.Setenv(key, t.TempDir())
	}
	// Short owned root reserves enough room for Zellij's version and session
	// suffix even when the test's inherited TMPDIR is long.
	runtimeDir, err := os.MkdirTemp("", "to-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(runtimeDir); err != nil {
			t.Error(err)
		}
	})
	t.Setenv("XDG_RUNTIME_DIR", runtimeDir)
	t.Setenv("ZELLIJ_SOCKET_DIR", filepath.Join(runtimeDir, "s"))
	t.Setenv("SHELL", "/bin/sh")
	t.Setenv("ENV", "")
	t.Setenv("BASH_ENV", "")
	for _, key := range []string{"ZELLIJ", "ZELLIJ_SESSION_NAME", "ZELLIJ_PANE_ID", "PAIR_SESSION"} {
		t.Setenv(key, "")
		if err := os.Unsetenv(key); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", base+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("PAIR_NATIVE_TERMINAL_HELPER", "1")
	ready := t.TempDir()
	t.Setenv("PAIR_NATIVE_TERMINAL_READY", ready)
	t.Setenv("PAIR_DATA_DIR", t.TempDir())
	t.Setenv("PAIR_TAG", "native-owner")
	write := func(name, value string) string {
		t.Helper()
		p := filepath.Join(base, name)
		if err := os.WriteFile(p, []byte(value), 0600); err != nil {
			t.Fatal(err)
		}
		return p
	}
	config := write("config.kdl", "pane_frames false\nshow_startup_tips false\nshow_release_notes false\nsession_serialization false\n")
	layout := write("layout.kdl", "layout {\n pane name=\"terminal\" command=\"pair\" { args \"term\"; }\n pane name=\"terminal-control\" command=\"pair\" { args \"term\"; }\n}\n")
	var entropy [6]byte
	if _, err := rand.Read(entropy[:]); err != nil {
		t.Fatal(err)
	}
	session := "to-" + hex.EncodeToString(entropy[:])
	if len(filepath.Join(runtimeDir, "s", "0.45.1", session)) >= 104 {
		t.Fatal("private native socket exceeds Unix address budget")
	}
	fixture, err := pairlifecycletest.StartControlledZellijWithOptions(ctx, session, pairlifecycletest.ZellijOptions{ConfigFile: config, LayoutFile: layout})
	if err != nil {
		t.Fatal(err)
	}
	var owners []nativeTerminalOwner
	alive := func(pid int) bool { return procutil.Alive(strconv.Itoa(pid)) }
	wait := func(description string, check func() bool) {
		t.Helper()
		ticker := time.NewTicker(25 * time.Millisecond)
		defer ticker.Stop()
		for {
			if check() {
				return
			}
			select {
			case <-ctx.Done():
				t.Fatalf("waiting for %s: %v; startup=%q", description, ctx.Err(), fixture.StartupOutput())
			case <-ticker.C:
			}
		}
	}
	t.Cleanup(func() {
		if err := fixture.Close(); err != nil {
			t.Errorf("owned fixture cleanup: %v", err)
		}
		cleanupCtx, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		ticker := time.NewTicker(25 * time.Millisecond)
		defer ticker.Stop()
		for {
			remaining := false
			for _, o := range owners {
				if alive(o.PID) && (procutil.Source{}).Identity(o.PID) == o.Identity {
					remaining = true
				}
			}
			if !remaining {
				t.Logf("cleanup verified: session=%s helper generations=%+v", session, owners)
				return
			}
			select {
			case <-cleanupCtx.Done():
				t.Errorf("owned helper survived cleanup: %+v", owners)
				return
			case <-ticker.C:
			}
		}
	})
	action := func(args ...string) []byte {
		t.Helper()
		argv := append([]string{"--session", session, "action"}, args...)
		command := exec.CommandContext(ctx, "zellij", argv...)
		var stdout, diagnostic bytes.Buffer
		command.Stdout, command.Stderr = &stdout, &diagnostic
		var err error
		if len(args) > 0 && args[0] == "list-panes" {
			err = zellijexec.Run(command)
		} else {
			err = command.Run()
		}
		out := stdout.Bytes()
		if err != nil {
			t.Fatalf("native action %q: %v: %s; startup=%q", args, err, diagnostic.Bytes(), fixture.StartupOutput())
		}
		return out
	}
	observe := func() []zellijpane.Pane {
		return zellijpane.Parse(action("list-panes", "--json", "--command", "--state", "--all"))
	}
	wait("both registered native helpers", func() bool {
		entries, err := os.ReadDir(ready)
		if err != nil {
			t.Fatal(err)
		}
		owners = nil
		for _, entry := range entries {
			if strings.HasSuffix(entry.Name(), ".tmp") {
				continue
			}
			data, err := os.ReadFile(filepath.Join(ready, entry.Name()))
			if err != nil {
				t.Fatal(err)
			}
			var o nativeTerminalOwner
			if err := json.Unmarshal(data, &o); err != nil {
				t.Fatal(err)
			}
			owners = append(owners, o)
		}
		return len(owners) == 2
	})
	reg := wb.TerminalPaneRegistry{DataDir: os.Getenv("PAIR_DATA_DIR"), Tag: os.Getenv("PAIR_TAG")}
	f, err := os.OpenFile(reg.Path(), os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString("future record with unknown fields\n"); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	registryBefore, err := os.ReadFile(reg.Path())
	if err != nil {
		t.Fatal(err)
	}
	ids := func() []string {
		t.Helper()
		result, err := reg.LiveOwnerIDs(alive, procutil.Source{})
		if err != nil {
			t.Fatal(err)
		}
		return result
	}
	var before zellijpane.Pane
	wait("native Pair metadata", func() bool {
		for _, p := range observe() {
			if p.Title == "terminal" {
				before = p
				return strings.Contains(p.TerminalCommand, "pair term") && strings.Contains(p.PaneCommand, "pair term")
			}
		}
		return false
	})
	var handed, control nativeTerminalOwner
	for _, o := range owners {
		if o.Pane == before.ID {
			handed = o
		} else {
			control = o
		}
	}
	if handed.PID == 0 || control.PID == 0 {
		t.Fatalf("helper/pane binding: %+v, pane=%+v", owners, before)
	}
	beforeIDs := ids()
	if !wb.Registered(beforeIDs, handed.Pane) || !wb.Registered(beforeIDs, control.Pane) || wb.RoleForPaneWith(before, beforeIDs) != wb.PaneRoleRightTerminal {
		t.Fatalf("before ownership: ids=%v pane=%+v", beforeIDs, before)
	}
	args, ok := (procutil.Source{}).Command(handed.PID)
	if !ok || len(args) != 2 || filepath.Base(args[0]) != "pair" || args[1] != "term" {
		t.Fatalf("native helper argv: %q readable=%v", args, ok)
	}
	t.Logf("before: owner=%+v argv=%q pane=%+v ids=%v", handed, args, before, beforeIDs)
	action("focus-pane-id", handed.Pane)
	if _, err := fixture.WriteInput([]byte("release\n")); err != nil {
		t.Fatal(err)
	}
	wait("native shell exec", func() bool {
		args, ok := (procutil.Source{}).Command(handed.PID)
		return ok && reflect.DeepEqual(args, []string{"sh", "-i"})
	})
	ack := filepath.Join(base, "shell-ack")
	// Native argv confirms the helper has consumed the release before sending
	// shell input, so its buffered reader cannot prefetch the acknowledgement.
	if _, err := fixture.WriteInput([]byte("printf '%s\\n' \"$$\" > " + strconv.Quote(ack) + "\n")); err != nil {
		t.Fatal(err)
	}
	wait("shell PID acknowledgement", func() bool {
		data, err := os.ReadFile(ack)
		if os.IsNotExist(err) {
			return false
		}
		if err != nil {
			t.Fatal(err)
		}
		return strings.TrimSpace(string(data)) == strconv.Itoa(handed.PID)
	})
	var after zellijpane.Pane
	wait("current shell pane command", func() bool {
		for _, p := range observe() {
			if !p.IsPlugin && p.ID == handed.Pane {
				after = p
				return p.PaneCommand == "sh -i"
			}
		}
		return false
	})
	args, ok = (procutil.Source{}).Command(handed.PID)
	if !ok || !reflect.DeepEqual(args, []string{"sh", "-i"}) || !alive(handed.PID) || (procutil.Source{}).Identity(handed.PID) != handed.Identity {
		t.Fatalf("exec changed PID/start identity or wrong argv: owner=%+v args=%q readable=%v", handed, args, ok)
	}
	if after.TerminalCommand != before.TerminalCommand || after.Title != before.Title {
		t.Fatalf("native stale metadata premise changed: before=%+v after=%+v", before, after)
	}
	afterIDs := ids()
	if wb.Registered(afterIDs, handed.Pane) || !wb.Registered(afterIDs, control.Pane) || wb.RoleForPaneWith(after, afterIDs) == wb.PaneRoleRightTerminal {
		t.Fatalf("shell retained terminal role: ids=%v pane=%+v", afterIDs, after)
	}
	controlFound := false
	for _, p := range observe() {
		if !p.IsPlugin && p.ID == control.Pane {
			controlFound = true
			if wb.RoleForPaneWith(p, afterIDs) != wb.PaneRoleRightTerminal {
				t.Fatalf("live control lost role: %+v", p)
			}
		}
	}
	if !controlFound || (procutil.Source{}).Identity(control.PID) != control.Identity {
		t.Fatalf("control generation or pane disappeared: %+v", control)
	}
	registryAfter, err := os.ReadFile(reg.Path())
	if err != nil || !bytes.Equal(registryBefore, registryAfter) {
		t.Fatalf("owner reads mutated registry/unknown record: %v", err)
	}
	t.Logf("after: same owner=%+v argv=%q pane=%+v ids=%v; registry unchanged", handed, args, after, afterIDs)
}
