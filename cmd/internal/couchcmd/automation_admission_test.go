package couchcmd

import (
	"context"
	"errors"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

func TestAutomaticInputPolicyChangedInsideBrokerRouting(t *testing.T) {
	for _, change := range []string{"off", "off-on"} {
		t.Run(change, func(t *testing.T) {
			w := newWaitRig(t, true, true, nil)
			setTestAutomation(t, w.s, "/repo1", "on")
			permit := w.s.captureHubInputPermit(w.hub)
			w.s.broker.SetFamilies(func(context.Context) (map[string]string, error) {
				setTestAutomation(t, w.s, "/repo1", "off")
				if change == "off-on" {
					setTestAutomation(t, w.s, "/repo1", "on")
				}
				return nil, nil
			})
			const id = "automation-policy-changed-in-routing"
			receipt, err := w.s.sendAutomaticSystem(context.Background(), w.worker, w.hub, id, "generated notice", permit)
			if !errors.Is(err, errHubAutomationOff) {
				t.Fatalf("policy changed before mailbox commit: receipt=%+v err=%v", receipt, err)
			}
			if _, known, _ := w.s.broker.ReceiptSnapshot(id); known {
				t.Fatal("invalidated occurrence entered mailbox")
			}
		})
	}
}

func TestAutomaticInputAlreadyAdmittedRemainsCanonicalAfterOff(t *testing.T) {
	w := newWaitRig(t, true, true, nil)
	setTestAutomation(t, w.s, "/repo1", "on")
	permit := w.s.captureHubInputPermit(w.hub)
	const id = "automation-admission-precedes-off"
	first, err := w.s.sendAutomaticSystem(context.Background(), w.worker, w.hub, id, "generated notice", permit)
	if err != nil || first.Message.ID != id {
		t.Fatalf("first admission: %+v %v", first, err)
	}
	setTestAutomation(t, w.s, "/repo1", "off")
	retained, err := w.s.sendAutomaticSystem(context.Background(), w.worker, w.hub, id, "generated notice", permit)
	if err != nil || retained.Message != first.Message {
		t.Fatalf("off retracted canonical prior admission: %+v %v", retained, err)
	}
	if retained.Message.Origin != couchmessage.OriginSystem {
		t.Fatal("canonical origin changed")
	}
}

func TestAutomationAdmissionGuardOwnsOnlyTheCommitLock(t *testing.T) {
	w := newWaitRig(t, true, true, nil)
	setTestAutomation(t, w.s, "/repo1", "on")
	permit := w.s.captureHubInputPermit(w.hub)
	guard := w.s.automationAdmissionGuard(w.hub, permit)
	// Building the guard must not retain automation across broker routing or IO.
	if !w.s.automation.mu.TryLock() {
		t.Fatal("automation locked before mailbox commit")
	}
	w.s.automation.mu.Unlock()
	release, err := guard(w.hub)
	if err != nil || release == nil {
		t.Fatalf("current permit refused: %v", err)
	}
	if w.s.automation.mu.TryLock() {
		w.s.automation.mu.Unlock()
		release()
		t.Fatal("commit permit did not own automation mutex")
	}
	release()
	if !w.s.automation.mu.TryLock() {
		t.Fatal("commit release retained automation mutex")
	}
	w.s.automation.mu.Unlock()
	setTestAutomation(t, w.s, "/repo1", "off")
	setTestAutomation(t, w.s, "/repo1", "on")
	if release, err := guard(w.hub); !errors.Is(err, errHubAutomationOff) || release != nil {
		if release != nil {
			release()
		}
		t.Fatalf("off/on revived original commit permit: %v", err)
	}
}

func TestAutomationAdmissionGuardPinsRecipientRoleAndHub(t *testing.T) {
	for _, fault := range []string{"replacement", "worker-permit-to-hub", "foreign-hub", "initially-off", "unknown-target"} {
		t.Run(fault, func(t *testing.T) {
			w := newWaitRig(t, true, true, nil)
			setTestAutomation(t, w.s, "/repo1", "on")
			permit := w.s.captureHubInputPermit(w.hub)
			actual := w.hub
			switch fault {
			case "replacement":
				actual.Nonce = "replacement"
			case "worker-permit-to-hub":
				permit = w.s.captureHubInputPermit(w.worker)
			case "foreign-hub":
				permit.root = "/other-hub"
			case "initially-off":
				setTestAutomation(t, w.s, "/repo1", "off")
				permit = w.s.captureHubInputPermit(w.hub)
				setTestAutomation(t, w.s, "/repo1", "on")
			case "unknown-target":
				w.s.mu.Lock()
				delete(w.s.workspaces, w.hub)
				w.s.mu.Unlock()
			}
			guard := w.s.automationAdmissionGuard(w.hub, permit)
			if release, err := guard(actual); !errors.Is(err, errHubAutomationOff) || release != nil {
				if release != nil {
					release()
				}
				t.Fatalf("invalid commit permit accepted: %v", err)
			}
		})
	}
}
