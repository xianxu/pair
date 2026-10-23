package couchcmd

import (
	"context"
	"errors"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchtty"
)

func TestScratchMetadataConfigurationFailureRaisesAttention(t *testing.T) {
	raw, _, _ := scratchWiringRuntime(t)
	raw.scratchServers = func(context.Context) (bool, error) { return false, errors.New("native metadata authority unavailable") }
	rt, lease, err := raw.prepareSingleton(true)
	if err != nil {
		t.Fatal(err)
	}
	defer lease.Close()
	ns, err := rt.ResolveNamespace()
	if err != nil {
		t.Fatal(err)
	}
	c, err := rt.NewCouchWith(couchcore.NewFakeRunner(), ns)
	if err != nil {
		t.Fatal(err)
	}
	console, host, input := warningConsole(t)
	c.ScratchStartup(withScratchWarnings(t.Context(), console), func(text string) { console.Notify(couchtty.Notice{Body: text}) })
	openWarningAttention(t, input, host, "Zellij metadata unchanged")
}
