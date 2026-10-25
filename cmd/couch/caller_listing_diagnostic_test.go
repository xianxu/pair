package main

import (
	"context"
	"debug/buildinfo"
	"os"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/testcouch"
)

// One opt-in Claude sample; a success does not classify the original incident.
func TestDisposableCallerListingDiagnostic(t *testing.T) {
	if os.Getenv("PAIR_NATIVE_CALLER_LISTING_DIAGNOSTIC") != "1" {
		t.Skip("set PAIR_NATIVE_CALLER_LISTING_DIAGNOSTIC=1 for the authorized single sample")
	}
	binaries := testcouch.Build(t)
	t.Logf("candidate evidence: %v", binaries.Evidence)
	info, err := buildinfo.ReadFile(binaries.Couch)
	if err != nil {
		t.Fatal(err)
	}
	settings := map[string]string{}
	for _, setting := range info.Settings {
		settings[setting.Key] = setting.Value
	}
	t.Logf("actual Couch candidate: go=%s vcs.revision=%s vcs.modified=%s", info.GoVersion, settings["vcs.revision"], settings["vcs.modified"])
	if settings["vcs.modified"] != "false" || settings["vcs.revision"] != binaries.Evidence["source-head"] {
		t.Fatal("native candidate must be built from the clean recorded source head")
	}
	fleet := testcouch.New(t, binaries, testcouch.Options{Workers: 5, AgentKind: "claude", RuntimeBudget: 120 * time.Second, RetainStartupServerLog: true})
	setup, stopSetup := context.WithTimeout(t.Context(), 180*time.Second)
	err = fleet.Prepare(setup)
	stopSetup()
	if err != nil {
		t.Fatal(err)
	}
	ctx, stop := context.WithTimeout(t.Context(), 120*time.Second)
	defer stop()
	if err = fleet.Start(ctx); err != nil {
		t.Fatal(err)
	}
	actors, err := fleet.Actors(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var caller couchmessage.Binding
	count := 0
	for _, actor := range actors.Actors {
		if actor.Binding.Slot == "fixture:0" {
			caller = actor.Binding
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected one genuine caller: %+v", actors)
	}
	if err = caller.Validate(); err != nil {
		t.Fatal(err)
	}
	t.Logf("frozen genuine public caller: %+v", caller)
	for n := 1; n <= 5; n++ {
		if err = fleet.StartWorker(ctx, n); err != nil {
			t.Logf("explicit caller before cleanup after worker%d failure: %s", n, fleet.CaptureCallerEvidence(ctx, caller))
			t.Fatalf("worker%d: %v", n, err)
		}
	}
	t.Logf("worker5 reached; later caller observation before cleanup: %s", fleet.CaptureCallerEvidence(ctx, caller))
}
