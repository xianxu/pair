package couchcmd

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/diagnosticlog"
)

func TestSupervisionFactStorageAndOfflineReader(t *testing.T) {
	root := t.TempDir()
	namespace := filepath.Join(root, "couch")
	if err := os.Mkdir(namespace, 0700); err != nil {
		t.Fatal(err)
	}
	log := openSupervisionFacts(root, namespace)
	defer log.Close()
	now := log.start
	log.now = func() time.Time { return now }
	subject := log.hash("slot", "ops:0")
	sender := log.hash("slot", "pair:7")
	generated := false
	// Real Go serialization, registration and diagnostic persistence feed the CLI.
	log.offer(supervisionFact{Event: "hub_delivery", Subject: subject, Sender: sender, Message: log.hash("message", "report"), Hint: "landed", Outcome: "confirmed", Generated: &generated})
	now = now.Add(time.Millisecond) // deterministic strictly later observation
	log.offer(supervisionFact{Event: "session_write", Subject: subject, Incarnation: log.hash("incarnation", "writer")})
	log.offer(supervisionFact{Event: "snapshot", Slots: []supervisionFactSlot{}})
	path := filepath.Join(namespace, "supervision.jsonl")
	supervisionEventually(t, "diagnostic rows not persisted", func() bool {
		raw, err := os.ReadFile(path)
		return err == nil && len(raw) >= 2 && string(raw[len(raw)-2:]) == "}\n" && bytesCountLines(raw) == 3
	})
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatalf("private log: %v %v", info, err)
	}
	page, err := diagnosticlog.EnumerateRoot(context.Background(), root, 0, 100)
	if err != nil || len(page.Entries) != 1 {
		t.Fatalf("retention registration missing: %+v %v", page, err)
	}
	python, err := exec.LookPath("python3")
	if err != nil {
		t.Fatal("Python 3 is required for supervision offline label acceptance")
	}
	output, err := exec.Command(python, "../../../scripts/supervision-labels/labels.py", path).CombinedOutput()
	if err != nil {
		t.Fatalf("offline reader: %v %s", err, output)
	}
	var label struct {
		Label *bool `json:"label"`
	}
	if err := json.Unmarshal(output, &label); err != nil || label.Label == nil || !*label.Label {
		t.Fatalf("Go facts do not produce positive offline label: %s %v", output, err)
	}
}
func bytesCountLines(raw []byte) int {
	count := 0
	for _, b := range raw {
		if b == '\n' {
			count++
		}
	}
	return count
}
