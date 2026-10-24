package couchcmd

import (
	"context"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchcore"
)

func TestFreshPrimaryUsesInitialEvidenceOnly(t *testing.T) {
	for _, kind := range []couchcore.StartupKind{"fresh", couchcore.StartupNormal, couchcore.StartupFullShutdown, couchcore.StartupUnknown} {
		t.Run(string(kind), func(t *testing.T) {
			rt := newRT(t, "/repo")
			c, err := rt.NewCouch()
			if err != nil {
				t.Fatal(err)
			}
			initial := couchcore.StartupObservation{Kind: kind, Hubs: map[string]couchcore.StartupHubObservation{"/repo": {}}}
			got, err := startFreshPrimary(context.Background(), c, initial, "/repo", couchcore.StartResult{})
			if err != nil {
				t.Fatal(err)
			}
			if want := kind == "fresh"; (got.Handle != nil) != want {
				t.Fatalf("kind %s result %+v", kind, got)
			}
			if kind != "fresh" && len(rt.runner.Ops) != 0 {
				t.Fatalf("nonfresh startup launched: %v", rt.runner.Ops)
			}
			if startupAutomationOn(&initial, "/repo") {
				t.Fatal("initial absent hub gained automation")
			}
		})
	}
}

func TestFreshExtraHubsCarryOpenIntentOncePerAdoption(t *testing.T) {
	_, defs, _ := moveFixture(t)
	for _, kind := range []couchcore.StartupKind{"fresh", couchcore.StartupNormal, couchcore.StartupFullShutdown, couchcore.StartupUnknown} {
		var calls []couchcore.OperationCall
		var completed []func(any, error)
		queueExplicitHubs(defs[1:], func(_ string, _ string, prepare func(context.Context) (couchcore.OperationCall, error), _ func(), finish func(any, error)) error {
			call, err := prepare(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			calls = append(calls, call)
			completed = append(completed, finish)
			return nil
		}, func(string, error) {}, couchcore.StartupObservation{Kind: kind})
		if len(calls) != 1 {
			t.Fatalf("eager launch count %d", len(calls))
		}
		want := ""
		if kind == "fresh" {
			want = string(couchcore.StartOpen)
		}
		if calls[0].Args["action"] != want {
			t.Fatalf("kind%s action%q want%q", kind, calls[0].Args["action"], want)
		}
		completed[0](nil, nil)
		if len(calls) != 2 || calls[1].Args["action"] != want {
			t.Fatalf("serial policy lost: %+v", calls)
		}
	}
}
