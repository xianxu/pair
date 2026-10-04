package couchcore

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func readFleetGolden(t *testing.T, name string) []byte {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read golden %s: %v", name, err)
	}
	return raw
}

// TestDecodeFleetInventoryGolden decodes a trimmed capture of real
// `sdlc fleet inventory --json` output (pair#367 Task 1.1).
func TestDecodeFleetInventoryGolden(t *testing.T) {
	inv, err := DecodeFleetInventory(readFleetGolden(t, "sdlc_fleet_inventory_v1.json"))
	if err != nil {
		t.Fatalf("decode golden: %v", err)
	}
	if inv.Machine.State != "present" || inv.Machine.Fingerprint == "" {
		t.Fatalf("machine = %+v, want present with a fingerprint", inv.Machine)
	}
	if len(inv.Slots) == 0 {
		t.Fatal("golden decoded with no slots")
	}
	if len(inv.DanglingClaims) != 1 || inv.DanglingClaims[0].Ref != "pair#000315" || inv.DanglingClaims[0].Claimant.Worktree != "/fleet/worktree/pair-slot5/pair" {
		t.Fatalf("dangling = %+v, want exactly the pair#000315 claim on pair-slot5", inv.DanglingClaims)
	}
	var depClaim *FleetClaim
	for i := range inv.Rows {
		if inv.Rows[i].TreePath == "/fleet/worktree/pair-slot1/ariadne" && len(inv.Rows[i].Claims) == 1 {
			depClaim = &inv.Rows[i].Claims[0]
		}
	}
	if depClaim == nil || depClaim.Claimant.Workspace != "" {
		t.Fatalf("dependency-member claim = %+v, want one with an empty workspace", depClaim)
	}
	verdicts := map[string]bool{}
	for _, slot := range inv.Slots {
		for _, m := range slot.Members {
			verdicts[m.Verdict] = true
			for _, r := range m.Reasons {
				if !knownFleetReason(r) {
					t.Errorf("golden reason %q is outside the grammar", r)
				}
			}
		}
	}
	for _, want := range []string{"ready", "holds-work", "needs-recovery", "missing"} {
		if !verdicts[want] {
			t.Errorf("golden lacks a %s member", want)
		}
	}
	if len(inv.Diagnostics) == 0 {
		t.Error("golden lacks a diagnostics entry")
	}
}

func TestDecodeFleetInventoryRejectsUnsupported(t *testing.T) {
	golden := readFleetGolden(t, "sdlc_fleet_inventory_v1.json")
	edit := func(f func(map[string]any)) []byte {
		var doc map[string]any
		if err := json.Unmarshal(golden, &doc); err != nil {
			t.Fatal(err)
		}
		f(doc)
		raw, err := json.Marshal(doc)
		if err != nil {
			t.Fatal(err)
		}
		return raw
	}
	for _, tc := range []struct {
		name string
		raw  []byte
	}{
		{"schema 2", edit(func(d map[string]any) { d["schema_version"] = 2 })},
		{"schema missing", edit(func(d map[string]any) { delete(d, "schema_version") })},
		{"schema string", edit(func(d map[string]any) { d["schema_version"] = "1" })},
		{"no slots", edit(func(d map[string]any) { delete(d, "slots") })},
		{"no machine", edit(func(d map[string]any) { delete(d, "machine") })},
		{"no dangling_claims", edit(func(d map[string]any) { delete(d, "dangling_claims") })},
		{"row without claims_state", edit(func(d map[string]any) {
			delete(d["rows"].([]any)[0].(map[string]any), "claims_state")
		})},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := DecodeFleetInventory(tc.raw); !errors.Is(err, ErrFleetSchemaUnsupported) {
				t.Fatalf("err = %v, want ErrFleetSchemaUnsupported", err)
			}
		})
	}
	for _, tc := range []struct {
		name string
		raw  string
	}{
		{"array", `[]`},
		{"empty", ``},
		{"duplicate key", `{"schema_version":1,"schema_version":1,"machine":{"state":"present"},"rows":[],"slots":[],"dangling_claims":[]}`},
		{"trailing value", `{"schema_version":1,"machine":{"state":"present"},"rows":[],"slots":[],"dangling_claims":[]} {}`},
	} {
		t.Run("malformed "+tc.name, func(t *testing.T) {
			_, err := DecodeFleetInventory([]byte(tc.raw))
			if err == nil || errors.Is(err, ErrFleetSchemaUnsupported) {
				t.Fatalf("err = %v, want a malformed-input error that is not ErrFleetSchemaUnsupported", err)
			}
		})
	}
	// Additive v1 fields are allowed: an unknown key must not refuse.
	if _, err := DecodeFleetInventory(edit(func(d map[string]any) { d["future_field"] = true })); err != nil {
		t.Fatalf("additive v1 field refused: %v", err)
	}
}

// A pre-#288/#289 v1 build carries schema_version 1 without the slot and claim
// sections; reading it as "no slots" would erase every slot (pair#367).
func TestDecodeFleetInventoryPre288IsUnsupportedNotEmpty(t *testing.T) {
	if _, err := DecodeFleetInventory(readFleetGolden(t, "sdlc_fleet_inventory_v1_pre288.json")); !errors.Is(err, ErrFleetSchemaUnsupported) {
		t.Fatalf("err = %v, want ErrFleetSchemaUnsupported", err)
	}
}

func TestKnownFleetReasonGrammar(t *testing.T) {
	for reason, want := range map[string]bool{
		"dirty": true, "detached": true, "missing": true, "unlanded-commits": true,
		"operation:rebase-merge": true, "open-issue:pair#000367": true, "claimed:pair#000367": true, "probe:base": true,
		"operation:": false, "claimed:": false, "": false, "dirty:x": false, "stale": false, "probe": false,
	} {
		if got := knownFleetReason(reason); got != want {
			t.Errorf("knownFleetReason(%q) = %v, want %v", reason, got, want)
		}
	}
}

// TestFleetInventoryLiveConformance pins the decoder and the fake's
// vocabulary to the real producer: the installed sdlc's own output must decode
// and use only verdicts and reasons the fake can produce. It needs network for
// sdlc's tracker fetches; a sandboxed run still decodes (claims go stale).
func TestFleetInventoryLiveConformance(t *testing.T) {
	if testing.Short() {
		t.Skip("live sdlc conformance is skipped in -short mode")
	}
	if _, err := exec.LookPath("sdlc"); err != nil {
		t.Skip("no sdlc on PATH")
	}
	vantage, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	raw, err := OSProvisionIO{}.Run(ctx, ProvisionCommand{Dir: vantage, Program: "sdlc", Args: []string{"fleet", "inventory", "--json", "--path", vantage}, Timeout: 90 * time.Second})
	if err != nil {
		t.Fatalf("sdlc fleet inventory: %v", err)
	}
	inv, err := DecodeFleetInventory(raw)
	if err != nil {
		t.Fatalf("decode live inventory: %v", err)
	}
	for _, slot := range inv.Slots {
		if !knownFleetVerdicts[slot.Verdict] {
			t.Errorf("slot %s verdict %q is unknown to the decoder", slot.Address, slot.Verdict)
		}
		for _, m := range slot.Members {
			if !knownFleetVerdicts[m.Verdict] {
				t.Errorf("member %s verdict %q is unknown to the decoder", m.Path, m.Verdict)
			}
			for _, r := range m.Reasons {
				if !knownFleetReason(r) {
					t.Errorf("member %s reason %q is outside the grammar", m.Path, r)
				}
			}
		}
	}
	for _, row := range inv.Rows {
		if !knownClaimsStates[row.ClaimsState] {
			t.Errorf("row %s claims_state %q is unknown to the decoder", row.TreePath, row.ClaimsState)
		}
	}
	if strings.TrimSpace(inv.Machine.State) == "" {
		t.Error("live inventory has an empty machine state")
	}
}

// knownClaimsStates is sdlc's claim-read quality vocabulary; the live
// conformance test pins the producer to it.
var knownClaimsStates = map[string]bool{
	FleetClaimsPresent: true, FleetClaimsStale: true, FleetClaimsPartial: true,
	FleetClaimsUnknown: true, FleetClaimsAbsent: true,
}
