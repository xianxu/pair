package couchcmd

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/zellijtest"
)

func TestSlotOperationLostListingStaysUncertainWithoutResend(t *testing.T) {
	for _, lost := range []string{"pair-other [Created 1s ago]\n", "pair-caller [Created 1s ago] (EXITED - attach to resurrect)\n"} {
		t.Run(strings.TrimSpace(lost), func(t *testing.T) {
			fixture := zellijtest.New(t)
			for _, entry := range fixture.Env() {
				key, value, _ := strings.Cut(entry, "=")
				t.Setenv(key, value)
			}
			fixture.SetModel(zellijtest.Model{Sessions: map[string]zellijtest.Session{"pair-caller": {}}})
			fixture.SetFaults(zellijtest.Fault{Ordinal: 2, Reply: &lost})
			rt := newRT(t, "/repo")
			record := seedDetachedThread(t, rt, "/repo")
			c, err := rt.NewCouch()
			if err != nil {
				t.Fatal(err)
			}
			c.Artifacts = couchcore.ScopedThreadArtifactCollisionChecker{GlobalDataDir: t.TempDir(), Zellij: launcher.ZellijSource{Path: fixture.Path()}}
			poller := launcher.NewListingPoller(launcher.ZellijSource{Path: fixture.Path()}, time.Hour, time.Minute)
			poller.Refresh(t.Context())
			resolve := messageThreadResolver(c, poller)
			binding := couchmessage.Binding{Scope: record.Address.RepoScope, Tag: string(record.Address.Tag), Session: "pair-caller"}
			var proofError error
			run := runSlotScript(t, cliInvocation{kind: cliMessage, messageOp: "reboot", ref: "pair:1"}, messageRuntime(), func(n int, request couchmessage.Request) (couchmessage.Response, error) {
				if n == 0 {
					if _, err := resolve(t.Context(), binding); err != nil {
						t.Fatalf("admission must start listed: %v", err)
					}
					return receiptResponse("accepted", couchmessage.ReceiptQueued, func(r *couchmessage.OperationReceipt) { r.ID, r.Op = request.ID, "reboot" }), nil
				}
				poller.Refresh(t.Context())
				_, proofError = resolve(t.Context(), binding)
				if proofError == nil {
					t.Fatal("lost caller was authorized")
				}
				poller.Refresh(t.Context()) // a later good listing must not replace the judged evidence
				response := couchmessage.Response{Code: "unavailable", Error: proofError.Error()}
				wire, err := json.Marshal(response)
				if err != nil || len(wire) > couchmessage.MaxFrameBytes {
					t.Fatalf("response exceeds transport: %d %v", len(wire), err)
				}
				var decoded couchmessage.Response
				if err := json.Unmarshal(wire, &decoded); err != nil {
					t.Fatal(err)
				}
				return decoded, nil
			})
			if run.code == 0 || len(run.calls) != 2 || run.calls[0].Op != "reboot" || run.calls[1].Op != "operation-status" || run.calls[0].ID != run.calls[1].ID || !strings.Contains(run.stderr, "outcome uncertain") || !strings.Contains(run.stderr, run.calls[0].ID) {
				t.Fatalf("uncertainty or no-resend contract changed: %+v", run)
			}
			_, raw, ok := strings.Cut(proofError.Error(), "listing_evidence=")
			var evidence couchcore.SelectedListingEvidence
			if !ok || json.Unmarshal([]byte(raw), &evidence) != nil || string(evidence.Listing.Raw) != lost || evidence.Listing.At.IsZero() || evidence.Name != "pair-caller" || evidence.Claims != 1 {
				t.Fatalf("lost exact judged evidence: %v", proofError)
			}
			if len(fixture.Queries()) != 3 || len(fixture.Effects()) != 0 {
				t.Fatalf("auth spawned queries or mutations: queries=%v effects=%v", fixture.Queries(), fixture.Effects())
			}
		})
	}
}
