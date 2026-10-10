package wrapcmd

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/orientation"
)

func TestAutomaticInputTransactionsDoNotOverlap(t *testing.T) {
	for _, first := range []string{"orientation", "peer"} {
		t.Run(first, func(t *testing.T) {
			f, d := peerIntegrationFixture(t)
			p := f.proxy
			p.orientation = newOrientationDelivery(orientation.Request{Agent: "claude", Body: "read context"})
			p.orientation.ready = true
			timer := time.NewTimer(time.Hour)
			defer timer.Stop()
			peerIntegrationEnqueue(t, d)
			var out bytes.Buffer
			startOrientation := func() { p.dispatchOrientationObservation(&out, timer, false) }
			if first == "orientation" {
				startOrientation()
			} else {
				p.dispatchPeer(&out)
			}
			original := out.String()
			if original == "" {
				t.Fatal("first automatic transaction did not paste")
			}
			if first == "orientation" {
				p.dispatchPeer(&out)
			} else {
				startOrientation()
			}
			if out.String() != original {
				t.Fatalf("second transaction appended before repaint: %q", out.String())
			}
			// Cancel the active transaction before its paste has repainted. Completion
			// alone cannot release the stale empty snapshot to the competing writer.
			if first == "orientation" {
				p.advanceOrientation(orientation.DeliveryEvent{Kind: orientation.DeadlineElapsed}, &out, timer)
				p.dispatchPeer(&out)
			} else {
				p.advancePeer(couchmessage.PeerDeliveryEvent{Kind: couchmessage.PeerDeadlineElapsed}, &out)
				startOrientation()
			}
			if out.String() != original {
				t.Fatalf("terminal outcome released stale screen: %q", out.String())
			}
			// Even a fresh paint containing the abandoned payload must be preserved.
			f.output("\x1b[21;3Hleftover\x1b[21;11H")
			if first == "orientation" {
				p.dispatchPeer(&out)
			} else {
				startOrientation()
			}
			if out.String() != original {
				t.Fatalf("overwrote abandoned input: %q", out.String())
			}
			// Erase/repaint is fresh source evidence, without a sticky ownership flag.
			f.output("\x1b[21;3H\x1b[K\x1b[21;3H")
			if first == "orientation" {
				p.dispatchPeer(&out)
			} else {
				startOrientation()
			}
			if out.String() == original {
				t.Fatal("empty repaint did not release transaction")
			}
			if strings.Count(out.String(), "\x1b[200~") != 2 {
				t.Fatalf("wrong paste count: %q", out.String())
			}
		})
	}
}

func TestAutomaticInputSubmittedOwnerWaitsForEmptyRepaint(t *testing.T) {
	for _, first := range []string{"orientation", "peer"} {
		t.Run(first, func(t *testing.T) {
			f, d := peerIntegrationFixture(t)
			p := f.proxy
			p.orientation = newOrientationDelivery(orientation.Request{Agent: "claude", Body: "read context"})
			p.orientation.ready = true
			timer := time.NewTimer(time.Hour)
			defer timer.Stop()
			peerIntegrationEnqueue(t, d)
			var out bytes.Buffer
			if first == "orientation" {
				p.dispatchOrientationObservation(&out, timer, false)
				f.output("\x1b[21;3Hread context\x1b[21;15H")
				p.dispatchOrientationObservation(&out, timer, true)
				if p.orientation.state.Phase != orientation.DeliverySubmitted {
					t.Fatal("orientation not submitted")
				}
			} else {
				p.dispatchPeer(&out)
				paint := "\x1b[2J" + claudeBox(5, "❯", "136;136;136", strings.Split(peerEnvelope(d.receipt().Message), "\n")...) + "\x1b[?25h\x1b[7;3H"
				d.observeOutput([]byte(paint))
				f.output(paint)
				// pair#427: submit after the fixed delay, confirm on the clear.
				time.Sleep(PeerSubmitDelay)
				p.dispatchPeer(&out)
				peerRender(f, d, peerEmptyComposer())
				p.dispatchPeer(&out)
				if d.receipt().Status != couchmessage.Submitted {
					t.Fatal("peer not submitted")
				}
			}
			submitted := out.String()
			next := func() {
				if first == "orientation" {
					p.dispatchPeer(&out)
				} else {
					p.dispatchOrientationObservation(&out, timer, false)
				}
			}
			next()
			if out.String() != submitted {
				t.Fatal("next writer ran before post-submit repaint")
			}
			f.output("\x1b[2J" + strings.Replace(claudeLiveComposerPaint(), "alpha", "", 1) + "\x1b[21;3H")
			next()
			if out.String() == submitted {
				t.Fatal("next writer did not resume on empty repaint")
			}
			if strings.Count(out.String(), "\x1b[200~") != 2 {
				t.Fatalf("unexpected pastes: %q", out.String())
			}
		})
	}
}
