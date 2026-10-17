package wrapcmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/notifytransport"
	"github.com/xianxu/pair/cmd/internal/testwait"
	"github.com/xianxu/pair/cmd/internal/workbenchshortcut"
)

func TestDraftRouteRefusalDefaultNotifiesExistingBroker(t *testing.T) {
	isolateNotificationSockets(t)
	binding := filepath.Join(t.TempDir(), "wrapper.pid")
	t.Setenv("PAIR_PAIR_WRAP_PID_PATH", binding)
	broker, err := notifytransport.Start(binding, os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	defer broker.Close()
	var stderr bytes.Buffer
	p := &proxy{stderr: &stderr, draftRouteRuntime: &fakeDraftRouteRuntime{panes: []byte(`[]`)}}
	if !p.executeWorkbenchDecision(workbenchshortcut.ShortcutDecision{DraftLuaFunction: "PRIVATE_FUNCTION", FocusDraft: true}) {
		t.Fatal("draft chord not handled")
	}
	select {
	case message := <-broker.Messages():
		if message != "Pair action refused: draft context could not be confirmed." {
			t.Fatalf("notice=%q", message)
		}
	case <-testwait.After(t):
		t.Fatal("draft refusal never reached broker")
	}
	if !strings.Contains(stderr.String(), "global shortcut") {
		t.Fatalf("missing retained diagnostic: %q", &stderr)
	}
}

func TestDraftRouteRefusalRetainsBrokerFailure(t *testing.T) {
	isolateNotificationSockets(t)
	t.Setenv("PAIR_PAIR_WRAP_PID_PATH", filepath.Join(t.TempDir(), "missing-wrapper.pid"))
	var stderr bytes.Buffer
	p := &proxy{stderr: &stderr, draftRouteRuntime: &fakeDraftRouteRuntime{panes: []byte(`[]`)}}
	p.executeWorkbenchDecision(workbenchshortcut.ShortcutDecision{DraftLuaFunction: "PairConfirmQuit", FocusDraft: true})
	if !strings.Contains(stderr.String(), "global shortcut") || !strings.Contains(stderr.String(), "refusal notification failed") {
		t.Fatalf("lost broker delivery failure: %q", &stderr)
	}
}

func TestDraftRouteRefusalPreservesInjectedReporter(t *testing.T) {
	isolateNotificationSockets(t)
	t.Setenv("PAIR_PAIR_WRAP_PID_PATH", filepath.Join(t.TempDir(), "missing-wrapper.pid"))
	var stderr bytes.Buffer
	var reported error
	p := &proxy{stderr: &stderr, draftRouteRuntime: &fakeDraftRouteRuntime{panes: []byte(`[]`)}, shortcutErrorReporter: func(err error) { reported = err }}
	p.executeWorkbenchDecision(workbenchshortcut.ShortcutDecision{DraftLuaFunction: "PairConfirmQuit", FocusDraft: true})
	if reported == nil || stderr.Len() != 0 {
		t.Fatalf("reporter=%v stderr=%q", reported, &stderr)
	}
}
