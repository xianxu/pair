package couchcmd

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

func TestSwitchAgentReceiptBindsCompleteIntentInBothAdmissionPaths(t *testing.T) {
	caller := couchmessage.Binding{Scope: "scope", Tag: "hub"}
	base := switchRequest(caller, "intent", "pair:2")
	for _, source := range []string{`{}`, `{"AgentArgs":[]}`, `{"AgentArgs":["--model","a b",""]}`, `{"InterruptFirst":true}`} {
		original := base
		if err := json.Unmarshal([]byte(source), &original); err != nil {
			t.Fatal(err)
		}
		for _, changed := range []string{`{}`, `{"AgentArgs":[]}`, `{"AgentArgs":["--model","a b",""]}`, `{"InterruptFirst":true}`} {
			t.Run(source+"/"+changed, func(t *testing.T) {
				ops := newSlotOperations(nil, func(context.Context) ([]couchcore.RepositoryName, error) {
					return []couchcore.RepositoryName{{Key: "/repo/.git", Dir: "pair"}}, nil
				}, time.Now)
				queued := 0
				run := func(string, func(), couchcore.OperationProgress, func(any, error)) error { queued++; return nil }
				first := ops.admit(context.Background(), caller, original, run)
				if first.Code != "accepted" {
					t.Fatalf("initial admission: %+v", first)
				}
				replay := base
				if err := json.Unmarshal([]byte(changed), &replay); err != nil {
					t.Fatal(err)
				}
				want := "id-conflict"
				if source == changed {
					want = "accepted"
				}
				early, held := ops.existingSwitch(caller, replay)
				if !held || early.Code != want {
					t.Fatalf("early replay: %+v held=%v want=%s", early, held, want)
				}
				admitted := ops.admit(context.Background(), caller, replay, run)
				if admitted.Code != want || queued != 1 {
					t.Fatalf("queue replay: %+v queued=%d want=%s", admitted, queued, want)
				}
			})
		}
	}
}

func TestSwitchAgentSocketQueuesExactExplicitArguments(t *testing.T) {
	for _, agent := range []string{"claude", "codex"} {
		for _, argv := range [][]string{{}, {"--model", "a b", ""}} {
			t.Run(agent+"/"+strings.Join(argv, "|"), func(t *testing.T) {
				rig, queue, hub, worker, _ := leanSwitchRig(t)
				rig.s.switches.Load().prepare = func(_ context.Context, address couchcore.ThreadAddress, target string, explicit *[]string) (couchcore.PreparedAgentSwitch, error) {
					if target != agent || explicit == nil || !reflect.DeepEqual(*explicit, argv) {
						return couchcore.PreparedAgentSwitch{}, fmt.Errorf("explicit preparation lost intent: %s %v", target, explicit)
					}
					return couchcore.PreparedAgentSwitch{Address: address, Profile: couchcore.LaunchProfile{Agent: target, Argv: argv}, Fingerprint: "explicit-config"}, nil
				}
				request := switchRequest(hub, "explicit", worker.Slot)
				request.Agent = agent
				var err error
				request.AgentArgs, err = couchmessage.NewSwitchArguments(argv)
				if err != nil {
					t.Fatal(err)
				}
				got := switchCall(t, rig, request)
				if got.Code != "accepted" || queue.count() != 1 {
					t.Fatalf("admission: %+v queued=%d", got, queue.count())
				}
				call, err := queue.job(0).prepare(context.Background())
				if err != nil {
					t.Fatal(err)
				}
				encoded, _ := json.Marshal(argv)
				if call.Args["argv"] != string(encoded) || call.Args["known-config"] != "" || call.Args["agent"] != agent || call.Args["fingerprint"] != "explicit-config" {
					t.Fatalf("explicit argv lost in queue: %+v", call.Args)
				}
			})
		}
	}
}
