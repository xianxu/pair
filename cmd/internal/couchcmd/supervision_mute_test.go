package couchcmd

import (
	"os"
	"testing"
	"time"
)

func TestSupervisionMutedSlotSuppressesReminders(t *testing.T) {
	w := newWaitRig(t, true, true, nil)
	var notices []string
	installTestSupervision(w, func(s string) { notices = append(notices, s) })
	if err := os.WriteFile(w.watcher.optOut, []byte(w.worker.Slot+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if r := supervisionReport(w, "muted-report", "Landed #530"); r.Code != "accepted" {
		t.Fatal(r)
	}
	w.tickAt(0)
	if got := w.tickAt(31 * time.Minute); len(got) != 1 || len(notices) != 0 {
		t.Fatalf("muted slot reminded: deliveries=%v notices=%v", got, notices)
	}
}
