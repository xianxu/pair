package couchcmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/couchtty"
	"github.com/xianxu/pair/cmd/internal/ptychild"
	"github.com/xianxu/pair/cmd/internal/testwait"
)

func TestRenicerConsoleWarningIsOncePerExactBinding(t *testing.T) {
	console, host, input := warningConsole(t)
	target := ptychild.NewFakeChild(nil)
	t.Cleanup(func() { target.Exit(0) })
	console.AttachActor("priority-target", couchcore.ActorID("3"), "/priority-target", "priority target", target)
	binding := slot3
	binding.Scope = "legacy" // Public fixture attachment's exact thread namespace.
	machine := slotMachine()
	machine.set(pr(300, 200, 10, "agent@1"))
	r := newConsoleRenicer(console, machine, func() []couchmessage.Binding { return []couchmessage.Binding{binding} }, func(couchmessage.Binding) (int, error) { return 300, nil }, 10, 100)
	r.pass()
	r.pass()
	openWarningAttention(t, input, host, "relaunch the slot")
	if !strings.Contains(host.Written(), couchtty.AttentionGlyph+"1") || strings.Contains(host.Written(), couchtty.AttentionGlyph+"2") {
		t.Fatal("priority warning was not once per binding")
	}
	if _, err := input.Write([]byte("\r")); err != nil {
		t.Fatal(err)
	}
	if _, err := input.Write([]byte("EXACT-PRIORITY-TARGET")); err != nil {
		t.Fatal(err)
	}
	testwait.Until(t, "warning selects its exact binding rather than current focus", func() bool {
		return bytes.Contains(bytes.Join(target.Writes(), nil), []byte("EXACT-PRIORITY-TARGET"))
	})
}

func TestRenicerConsoleNormalPriorityDoesNotWarn(t *testing.T) {
	console, host, input := warningConsole(t)
	r := newConsoleRenicer(console, slotMachine(), func() []couchmessage.Binding { return []couchmessage.Binding{slot3} }, func(couchmessage.Binding) (int, error) { return 300, nil }, 10, 100)
	r.pass()
	// A known independent warning is an output barrier for the completed pass.
	console.Warn(couchtty.Notice{Body: "independent output barrier"})
	openWarningAttention(t, input, host, "independent output barrier")
	if strings.Contains(host.Written(), "relaunch the slot") || strings.Contains(host.Written(), couchtty.AttentionGlyph+"2") {
		t.Fatal("normal agent priority raised attention")
	}
}
