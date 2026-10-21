package couchcmd

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/couchschedule"
	"github.com/xianxu/pair/cmd/internal/strictjson"
)

func TestScheduleMixedRefusalStatusAndHistoryThroughSocket(t *testing.T) {
	c, roots, _ := scheduleServiceFixture(t)
	namespace := filepath.Dir(c.store.Dir)
	for _, name := range []string{"BadTask", "bad.name", "bad\nname", ":entry:ff"} {
		if err := os.Mkdir(filepath.Join(roots[0], "scripts/scheduled", name), 0700); err != nil {
			t.Fatal(err)
		}
	}
	if err := c.refresh(t.Context()); err != nil {
		t.Fatalf("refused entry blocked coordinator: %v", err)
	}
	s := &messageService{}
	s.schedule.Store(c)
	socket, err := couchmessage.SocketPath(namespace, "broker")
	if err != nil {
		t.Fatal(err)
	}
	server, err := couchmessage.StartServer(t.Context(), socket, func(ctx context.Context, raw []byte) ([]byte, error) {
		var request couchmessage.Request
		if err := strictjson.Decode(raw, &request); err != nil {
			return nil, err
		}
		return json.Marshal(s.handle(ctx, request))
	})
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	if raw, err := exec.CommandContext(t.Context(), "git", "init", "-q", roots[0]).CombinedOutput(); err != nil {
		t.Fatalf("git init %s: %v", raw, err)
	}
	t.Chdir(roots[0])
	for _, a := range []*scheduleArgs{{verb: "status"}, {verb: "status", json: true}, {verb: "history", task: "report", json: true}} {
		var out, stderr bytes.Buffer
		if code := runScheduleCLI(a, scheduleStatusRT{dir: namespace}, &out, &stderr, couchmessage.Call); code != 0 {
			t.Fatalf("%s json=%v blocked by refused sibling: code=%d %s", a.verb, a.json, code, &stderr)
		}
		if a.verb == "history" {
			if !strings.Contains(out.String(), `"records":[]`) {
				t.Fatalf("valid history unavailable: %s", &out)
			}
		} else if a.json {
			var snapshot couchschedule.StatusSnapshot
			if err := json.Unmarshal(out.Bytes(), &snapshot); err != nil || snapshot.Validate() != nil {
				t.Fatalf("invalid mixed JSON: %s %v", &out, err)
			}
		} else {
			for _, want := range []string{"report", "BadTask", "bad.name", `bad\nname`, "refused"} {
				if !strings.Contains(out.String(), want) {
					t.Errorf("diagnostic text missing %q: %s", want, &out)
				}
			}
			if bytes.Contains(out.Bytes(), []byte{27}) {
				t.Fatal("raw controls reached status terminal")
			}
		}
	}
	for _, verb := range []string{"run", "history"} {
		var out, stderr bytes.Buffer
		calls := 0
		code := runScheduleCLI(&scheduleArgs{verb: verb, task: `"BadTask"`}, scheduleStatusRT{dir: namespace}, &out, &stderr, func(context.Context, string, any, any) error { calls++; return nil })
		if code != 2 || calls != 0 {
			t.Fatalf("diagnostic label gained %s routing: code=%d calls=%d", verb, code, calls)
		}
	}
}

func TestScheduleValidManualRunIgnoresRefusedSibling(t *testing.T) {
	runner, fake, root := manualRunnerFixture(t)
	if err := os.Mkdir(filepath.Join(root, "scripts/scheduled/BadTask"), 0700); err != nil {
		t.Fatal(err)
	}
	fake.Enqueue(couchschedule.ExecutionResult{Exit: 0, Outcome: "quiet"})
	result, err := runner.run(t.Context(), root, "report", nil)
	if err != nil || result.Exit != 0 || len(fake.Runs()) != 1 {
		t.Fatalf("valid manual task blocked by malformed sibling: %#v %v runs=%d", result, err, len(fake.Runs()))
	}
}
