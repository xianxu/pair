package couchcmd

import (
	"context"
	"path/filepath"
	"sync"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/testpair"
)

// Probes a public read-only request during startup, including a surviving CLI
// reconnecting before the constructor returns. All sockets are private.
func TestDrainStatusDuringServicePublication(t *testing.T) {
	for range 50 {
		root := testpair.ShortRoot(t)
		broker, registry := filepath.Join(root, "broker"), filepath.Join(root, "registry")
		ctx, cancel := context.WithCancel(context.Background())
		var clients sync.WaitGroup
		clients.Add(1)
		go func() {
			defer clients.Done()
			for ctx.Err() == nil {
				var response couchmessage.Response
				_ = couchmessage.Call(ctx, broker, couchmessage.Request{Op: "drain", Drain: &couchmessage.DrainRequest{Action: "status"}}, &response)
			}
		}()
		world := newMessageWorld()
		service, err := newMessageService(context.Background(), broker, registry, world.authority(), couchmessage.NewPaneMailbox(), func(string) (couchcore.SlotGitStatus, bool) { return couchcore.SlotGitStatus{}, false }, nil)
		cancel()
		clients.Wait()
		if err != nil {
			t.Fatal(err)
		}
		service.Close()
	}
}
