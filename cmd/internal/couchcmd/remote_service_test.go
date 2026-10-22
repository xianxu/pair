package couchcmd

import (
	"context"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/broadcast"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/remotesession"
)

func TestRemoteServiceLocalActionsNeverCarryLinkOrReset(t *testing.T) {
	rig := &serviceRig{t: t, world: newMessageWorld()}
	rig.init()
	host, err := broadcast.NewProtectedHost(&broadcast.FakeProtectedTunnel{FakeTunnel: &broadcast.FakeTunnel{}})
	if err != nil {
		t.Fatal(err)
	}
	remote, _, err := remotesession.NewSupervisor(remotesession.SupervisorConfig{Namespace: t.TempDir(), Origin: "https://remote.example.com", Host: host})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		remote.Off()
		if err := remote.Join(context.Background()); err != nil {
			t.Error(err)
		}
	})
	rig.s.remote.Store(&remoteControl{supervisor: remote})
	for _, op := range []string{"remote-status", "remote-on", "remote-off"} {
		var result couchmessage.Response
		if err := couchmessage.Call(t.Context(), rig.brokerSock, couchmessage.Request{Op: op}, &result); err != nil {
			t.Fatal(err)
		}
		if result.Code != "ok" || result.Remote == nil {
			t.Fatalf("%s %+v", op, result)
		}
	}
	remote.Kill()
	if err := remote.Join(t.Context()); err != nil {
		t.Fatal(err)
	}
	result := rig.s.handleRemote(couchmessage.Request{Op: "remote-on"})
	if result.Code != "refused" {
		t.Fatal("local on reset kill", result)
	}
	if strings.Contains(result.Error, "https://") {
		t.Fatal("link leaked in refusal")
	}
}
