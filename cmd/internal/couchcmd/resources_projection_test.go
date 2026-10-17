package couchcmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

func TestResourceProjectionTransportCapacity(t *testing.T) {
	v := couchcore.ResourceView{}
	for i := 0; i < couchcore.ResourceMaxSlots; i++ {
		v.Slots = append(v.Slots, couchcore.ResourceSlotView{Slot: fmt.Sprintf("pair:%d", i), ProcessCount: 32, Reason: strings.Repeat("<", 4096)})
	}
	for i := 0; i < couchcore.ResourceMaxProcesses; i++ {
		p := couchcore.ResourceProcessView{Name: strings.Repeat("<", 128)}
		p.Key.PID = i + 1
		v.Processes = append(v.Processes, p)
		v.Orphans = append(v.Orphans, p)
		v.Unattributed = append(v.Unattributed, p)
		rss := uint64(i)
		v.OrphanNvim = append(v.OrphanNvim, couchcore.OrphanNvim{
			Key: p.Key, Age: time.Duration(i) * time.Hour,
			Cwd: "/" + strings.Repeat("<", 128), Root: "/" + strings.Repeat("<", 128),
			RSSBytes: &rss, Reason: "unattributed embedded nvim",
		})
	}
	for i := 0; i < couchcore.ResourceMaxEpisodes; i++ {
		v.Episodes = append(v.Episodes, couchcore.ResourceEpisode{Key: fmt.Sprint(i), Phase: "pending"})
	}
	v.Errors = []string{strings.Repeat("<", couchmessage.MaxFrameBytes)}
	projected := projectResourceView(v)
	dir, err := os.MkdirTemp("/tmp", "r587-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	socket := filepath.Join(dir, "resource.sock")
	server, err := couchmessage.StartServer(context.Background(), socket, func(context.Context, []byte) ([]byte, error) {
		return json.Marshal(couchmessage.Response{Code: "ok", Resources: &projected})
	})
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	var got couchmessage.Response
	if err := couchmessage.Call(context.Background(), socket, couchmessage.Request{Op: "resources"}, &got); err != nil {
		t.Fatal(err)
	}
	if got.Code != "ok" || got.Resources == nil || len(got.Resources.Slots) != len(v.Slots) {
		t.Fatalf("lost aggregates: %+v", got)
	}
	for i, s := range got.Resources.Slots {
		if s.ProcessCount != 32 || s.Slot != v.Slots[i].Slot {
			t.Fatal("aggregate changed")
		}
	}
	if len(got.Resources.Processes) >= len(v.Processes) || len(got.Resources.Errors) == 0 {
		t.Fatal("expected explicit bounded detail")
	}
	// The sweep is the operator's evidence for a reap, so the wire keeps a
	// bounded prefix of it and says how much it dropped rather than the section.
	if len(got.Resources.OrphanNvim) == 0 || len(got.Resources.OrphanNvim) >= len(v.OrphanNvim) {
		t.Fatalf("sweep rows on the wire: %d of %d", len(got.Resources.OrphanNvim), len(v.OrphanNvim))
	}
	if !strings.Contains(strings.Join(got.Resources.Errors, " "), "orphan nvim omitted") {
		t.Fatalf("no omission notice for the sweep: %v", got.Resources.Errors)
	}
	if v.Slots[0].Reason != strings.Repeat("<", 4096) {
		t.Fatal("projection mutated source")
	}
}
