package webcast

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/cdp"
	"github.com/xianxu/pair/cmd/internal/webcast/board"
	"github.com/xianxu/pair/cmd/internal/webcast/notes"
)

// Exact receipts exercise the serialized worker rather than just its pure
// permission reducer. C and 2C contexts get one delivery each per generation;
// accepted contexts get none on unchanged refreshes, regardless of C.
func TestOperatorOwnerPermissionDeliveryCost(t *testing.T) {
	for _, count := range []int{4, 8} {
		t.Run(strconv.Itoa(count)+"Contexts", func(t *testing.T) {
			c, p := startPeer(t, func(p *peer) { p.ignore["Runtime.evaluate"] = true })
			b := board.New(notes.Structural{}, nil).WithAnnotationCleanup(func(context.Context, uint64) error { return nil })
			defer b.Close()
			contexts := make(map[string]int64, count)
			done := make(chan struct{})
			c.do(func(s *loopState) {
				for index := 1; index <= count; index++ {
					session, target := fmt.Sprintf("S%d", index), fmt.Sprintf("T%d", index)
					contexts[session] = int64(100 + index)
					s.contexts[session] = contexts[session]
					s.sessions[session] = target
					s.tracker.Apply(TargetCreated{ID: target})
				}
				close(done)
			})
			<-done
			active := fmt.Sprintf("S%d", count) // most recently created target
			revisionRE := regexp.MustCompile(`const revision="([0-9]+)"`)
			hookRE := regexp.MustCompile(`hook\((true|false),"([0-9]+)"\)`)
			lastRevision := make(map[string]uint64, count)
			totalDeliveries, totalReceipts := 0, 0
			var held *cdp.Message
			release := func() {
				if held != nil {
					p.send(fmt.Sprintf(`{"id":%d,"result":{}}`, held.ID))
					held = nil
				}
			}
			defer release()
			// Hold the final ordinary view reply in each whole worker pass. All
			// contexts' permission checks precede this barrier, so a subsequent
			// authority change cannot race an unfinished permission iteration.
			pass := func(generation uint64, wantDeliveries int) {
				t.Helper()
				seenViews, seenPermissions := map[string]bool{}, map[string]bool{}
				deadline := time.After(5 * time.Second)
				for len(seenViews) < count {
					var message cdp.Message
					select {
					case message = <-p.calls:
					case <-deadline:
						t.Fatalf("generation%d: only %d/%d view barriers, permissions=%v", generation, len(seenViews), count, seenPermissions)
					}
					if message.Method != "Runtime.evaluate" {
						continue // startup/close calls are answered by the existing peer
					}
					var params struct {
						Context    int64  `json:"contextId"`
						Expression string `json:"expression"`
					}
					if err := json.Unmarshal(message.Params, &params); err != nil {
						t.Fatal(err)
					}
					if context, ok := contexts[message.SessionID]; !ok || params.Context != context {
						t.Fatalf("delivery escaped registered context: %s %s", message.SessionID, message.Params)
					}
					if strings.Contains(params.Expression, "__couchcastAnnotationAllowed") {
						totalDeliveries++
						if seenPermissions[message.SessionID] {
							t.Fatalf("generation%d repeated accepted context %s", generation, message.SessionID)
						}
						seenPermissions[message.SessionID] = true
						revision, hook := revisionRE.FindStringSubmatch(params.Expression), hookRE.FindStringSubmatch(params.Expression)
						if len(revision) != 2 || len(hook) != 3 || hook[2] != strconv.FormatUint(generation, 10) || hook[1] != strconv.FormatBool(message.SessionID == active) {
							t.Fatalf("wrong owner generation/active-target permission: %s %s", message.SessionID, params.Expression)
						}
						nextRevision, err := strconv.ParseUint(revision[1], 10, 64)
						if err != nil || nextRevision <= lastRevision[message.SessionID] {
							t.Fatalf("revision did not advance for %s: %s", message.SessionID, revision[1])
						}
						lastRevision[message.SessionID] = nextRevision
						p.send(fmt.Sprintf(`{"id":%d,"result":{"result":{"value":{"revision":%q,"generation":%q,"allowed":%s}}}}`, message.ID, revision[1], hook[2], hook[1]))
						totalReceipts++
						continue
					}
					if !strings.Contains(params.Expression, "__couchcastOperatorView") || seenViews[message.SessionID] {
						t.Fatalf("unexpected view pass delivery: %s %s", message.SessionID, params.Expression)
					}
					seenViews[message.SessionID] = true
					if len(seenViews) == count {
						held = &message
					} else {
						p.send(fmt.Sprintf(`{"id":%d,"result":{}}`, message.ID))
					}
				}
				if len(seenPermissions) != wantDeliveries {
					t.Fatalf("generation%d permission sends=%d, want%d for %d contexts", generation, len(seenPermissions), wantDeliveries, count)
				}
			}
			refresh := func(generation uint64) {
				t.Helper()
				c.refreshOperator()
				release()
				pass(generation, 0)
			}
			c.startOperator(b)
			pass(0, count)
			refresh(0)
			refresh(0)
			for _, allowed := range []bool{true, false} {
				if err := b.SetAnnotationAllowed(context.Background(), allowed); err != nil {
					t.Fatal(err)
				}
				generation := b.AnnotationCapability().Generation
				release()
				pass(generation, count)
				refresh(generation)
				refresh(generation)
			}
			if totalDeliveries != 3*count || totalReceipts != totalDeliveries {
				t.Fatalf("cost sends=%d exact receipt checks=%d, want%d", totalDeliveries, totalReceipts, 3*count)
			}
		})
	}
}
