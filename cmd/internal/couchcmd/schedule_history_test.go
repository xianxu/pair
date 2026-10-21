package couchcmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchschedule"
	"github.com/xianxu/pair/cmd/internal/operationalrun"
)

func TestScheduleHistoryReadOnlyEmpty(t *testing.T) {
	namespace := filepath.Join(t.TempDir(), "absent")
	var out, stderr bytes.Buffer
	if code := runScheduleHistory(t.Context(), &scheduleArgs{verb: "history", task: "one", json: true}, namespace, "/hub", &out, &stderr); code != 0 {
		t.Fatalf("code=%d %s", code, &stderr)
	}
	if _, err := os.Stat(namespace); !os.IsNotExist(err) {
		t.Fatalf("history created namespace: %v", err)
	}
	if !strings.Contains(out.String(), `"records":[]`) {
		t.Fatalf("empty history: %s", &out)
	}
}

func TestScheduleHistoryIdentityAndSanitizedOutput(t *testing.T) {
	namespace, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	key := operationalrun.RunKey{Hub: (couchschedule.TaskKey{Hub: "/hub", Task: "one"}).HubID(), Task: "one", ID: "run-1"}
	exit := 10
	record := operationalrun.Record{Version: 1, Key: key, DeclarationRevision: "rev\x1b[2J", Source: "manual\nforged", StartedAt: time.Now().UTC(), EndedAt: time.Now().UTC(), Exit: &exit, Outcome: "wake\x0e", Notice: "pending\rforged"}
	if err := (operationalrun.Store{Namespace: namespace}).Publish(t.Context(), key, record, []byte("first\x1b[2J\nsecond\x0e\r\n")); err != nil {
		t.Fatal(err)
	}
	for _, jsonMode := range []bool{false, true} {
		var out, stderr bytes.Buffer
		a := &scheduleArgs{verb: "history", task: "one", runID: "run-1", json: jsonMode}
		if code := runScheduleHistory(t.Context(), a, namespace, "/hub", &out, &stderr); code != 0 {
			t.Fatalf("read code=%d: %s", code, &stderr)
		}
		if jsonMode {
			var payload struct {
				Record operationalrun.Record `json:"record"`
				Output string                `json:"output"`
			}
			if err := json.Unmarshal(out.Bytes(), &payload); err != nil || payload.Record.Key.ID != key.ID || !strings.Contains(payload.Output, "second") {
				t.Fatalf("json: %+v %v", payload, err)
			}
		} else {
			if strings.ContainsAny(out.String(), "\x1b\x0e\r") {
				t.Fatalf("unsafe text: %q", out.String())
			}
			if !strings.Contains(out.String(), "first\nsecond") {
				t.Fatalf("lost output lines: %q", out.String())
			}
		}
	}
	var out, stderr bytes.Buffer
	if code := runScheduleHistory(t.Context(), &scheduleArgs{verb: "history", task: "one", runID: "run-1"}, namespace, "/other-hub", &out, &stderr); code == 0 {
		t.Fatal("read another hub's run")
	}
	out.Reset()
	stderr.Reset()
	if code := runScheduleHistory(t.Context(), &scheduleArgs{verb: "history", task: "one", runID: "../run-1"}, namespace, "/hub", &out, &stderr); code == 0 {
		t.Fatal("accepted traversal run")
	}
	// The directory name cannot authorize a different record identity.
	record.Key.ID = "other-run"
	raw, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(namespace, "schedule", key.Hub, key.Task, "runs", key.ID, "record.json"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	stderr.Reset()
	if code := runScheduleHistory(t.Context(), &scheduleArgs{verb: "history", task: "one", runID: key.ID}, namespace, "/hub", &out, &stderr); code == 0 {
		t.Fatal("accepted mismatched run metadata")
	}
}
