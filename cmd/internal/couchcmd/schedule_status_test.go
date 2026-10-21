package couchcmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/couchschedule"
)

type scheduleStatusRT struct {
	Runtime
	dir string
}

func (r scheduleStatusRT) StoreDir() string { return r.dir }

func TestScheduleStatusOnlyUsesSnapshot(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "not-created")
	snapshot := couchschedule.StatusSnapshot{Version: 1, Hubs: []string{"/hub"}, Tasks: []couchschedule.TaskStatus{{Hub: "/hub", Task: "one", Scope: "machine", Cron: "0 7 * * *", Location: "UTC", State: "waiting", NextDue: time.Date(2026, 1, 1, 7, 0, 0, 0, time.UTC)}}}
	for _, wantJSON := range []bool{false, true} {
		var out, stderr bytes.Buffer
		calls := 0
		call := func(ctx context.Context, socket string, request, response any) error {
			calls++
			if !reflect.DeepEqual(request, couchmessage.Request{Op: "schedule-status"}) {
				t.Fatalf("request: %+v", request)
			}
			deadline, ok := ctx.Deadline()
			if !ok || time.Until(deadline) > 2*time.Second {
				t.Fatal("query lacks bounded deadline")
			}
			*response.(*couchmessage.Response) = couchmessage.Response{Code: "ok", Schedule: &snapshot}
			return nil
		}
		code := runScheduleCLI(&scheduleArgs{verb: "status", json: wantJSON}, scheduleStatusRT{dir: missing}, &out, &stderr, call)
		if code != 0 || calls != 1 || stderr.Len() != 0 {
			t.Fatalf("status code=%d calls=%d err=%s", code, calls, &stderr)
		}
		if wantJSON {
			var got couchschedule.StatusSnapshot
			if err := json.Unmarshal(out.Bytes(), &got); err != nil || !reflect.DeepEqual(got, snapshot) {
				t.Fatalf("json: %+v %v", got, err)
			}
		} else {
			for _, field := range []string{"/hub", "one", "machine", "0 7 * * *", "UTC", "waiting", "2026-01-01"} {
				if !strings.Contains(out.String(), field) {
					t.Errorf("text missing %s: %s", field, &out)
				}
			}
		}
	}
	if _, err := os.Stat(missing); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("status created namespace: %v", err)
	}
}

func TestScheduleStatusRejectsMalformedAndBlockedManual(t *testing.T) {
	valid := couchschedule.StatusSnapshot{Version: 1, Hubs: []string{"/hub"}, Tasks: []couchschedule.TaskStatus{}}
	for _, tc := range []struct {
		name     string
		args     *scheduleArgs
		response couchmessage.Response
		callErr  error
	}{
		{name: "nil args"},
		{name: "transport", args: &scheduleArgs{verb: "status"}, callErr: errors.New("offline")},
		{name: "nil payload", args: &scheduleArgs{verb: "status"}, response: couchmessage.Response{Code: "ok"}},
		{name: "refused", args: &scheduleArgs{verb: "status"}, response: couchmessage.Response{Code: "refused", Error: "broken\x1b[2J"}},
		{name: "bad payload", args: &scheduleArgs{verb: "status"}, response: couchmessage.Response{Code: "ok", Schedule: &couchschedule.StatusSnapshot{Version: 2}}},
		{name: "blocked run", args: &scheduleArgs{verb: "run", task: "one"}, response: couchmessage.Response{Code: "ok", Schedule: &couchschedule.StatusSnapshot{Version: 1, Error: "repair state", Hubs: valid.Hubs, Tasks: valid.Tasks}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var out, stderr bytes.Buffer
			code := runScheduleCLI(tc.args, scheduleStatusRT{dir: filepath.Join(t.TempDir(), "missing")}, &out, &stderr, func(_ context.Context, _ string, _ any, result any) error {
				*result.(*couchmessage.Response) = tc.response
				return tc.callErr
			})
			if code == 0 || stderr.Len() == 0 || bytes.Contains(stderr.Bytes(), []byte{27}) {
				t.Fatalf("code=%d err=%q", code, stderr.String())
			}
		})
	}
}

func TestScheduleHistoryUsesConfiguredHubEvenWithBlockedEmptySnapshot(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if output, err := exec.CommandContext(t.Context(), "git", "init", "-q", root).CombinedOutput(); err != nil {
		t.Fatalf("git init: %s %v", output, err)
	}
	t.Chdir(root)
	namespace := filepath.Join(root, "absent-state")
	call := func(_ context.Context, _ string, _ any, response any) error {
		*response.(*couchmessage.Response) = couchmessage.Response{Code: "ok", Schedule: &couchschedule.StatusSnapshot{Version: 1, Error: "repair local state", Hubs: []string{root}, Tasks: []couchschedule.TaskStatus{}}}
		return nil
	}
	var out, stderr bytes.Buffer
	if code := runScheduleCLI(&scheduleArgs{verb: "history", task: "one", json: true}, scheduleStatusRT{dir: namespace}, &out, &stderr, call); code != 0 {
		t.Fatalf("history blocked by ledger: code %d %s", code, &stderr)
	}
	if !strings.Contains(out.String(), `"records":[]`) {
		t.Fatalf("history: %s", &out)
	}
	if _, err := os.Stat(namespace); !os.IsNotExist(err) {
		t.Fatalf("created state: %v", err)
	}
}

func TestScheduleInternalOwnerDispatchIsExactAndBypassesRuntime(t *testing.T) {
	var out, stderr bytes.Buffer
	if code := RunWithRuntime([]string{"--internal-schedule-owner"}, strings.NewReader(""), &out, &stderr, scheduleStatusRT{}); code != 2 || stderr.Len() != 0 {
		t.Fatalf("owner dispatch: code %d %s", code, &stderr)
	}
	if code := RunWithRuntime([]string{"--internal-schedule-owner", "extra"}, strings.NewReader(""), &out, &stderr, scheduleStatusRT{}); code != 2 || stderr.Len() == 0 {
		t.Fatalf("owner extra args: code %d %s", code, &stderr)
	}
}

func TestScheduleStatusSanitizesSnapshotText(t *testing.T) {
	tainted := "bad\x1b[2J\x0e\r\nlabel"
	snapshot := couchschedule.StatusSnapshot{Version: 1, Error: tainted, Hubs: []string{"/hub"}, Tasks: []couchschedule.TaskStatus{{Hub: "/hub", Task: "one", Scope: tainted, Owner: tainted, Cron: tainted, Location: tainted, State: "waiting", Reason: tainted, Last: &couchschedule.RunSummary{ID: tainted, Source: tainted, Outcome: tainted, Notice: tainted}}}}
	var out, stderr bytes.Buffer
	call := func(_ context.Context, _ string, _ any, response any) error {
		*response.(*couchmessage.Response) = couchmessage.Response{Code: "ok", Schedule: &snapshot}
		return nil
	}
	if code := runScheduleCLI(&scheduleArgs{verb: "status"}, scheduleStatusRT{dir: filepath.Join(t.TempDir(), "absent")}, &out, &stderr, call); code != 0 {
		t.Fatalf("code %d %s", code, &stderr)
	}
	if strings.ContainsAny(out.String(), "\x1b\x0e\r") || strings.Count(out.String(), "\n") != 2 {
		t.Fatalf("unsafe status rows: %q", out.String())
	}
}

type scheduleHubGit struct {
	root  string
	err   error
	calls int
}

func (g *scheduleHubGit) RunContext(_ context.Context, _ string, args ...string) (string, error) {
	g.calls++
	if strings.Join(args, " ") != "rev-parse --show-toplevel" {
		return "", errors.New("unexpected Git")
	}
	return g.root, g.err
}
func TestScheduleHubRequiresExactConfiguredCheckout(t *testing.T) {
	hub, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	worker, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, root string
		hubs       []string
		allowed    bool
	}{{"hub", hub, []string{hub}, true}, {"worker", worker, []string{hub}, false}, {"empty catalog", hub, nil, false}, {"another hub", hub, []string{worker}, false}, {"relative response", "relative", []string{hub}, false}} {
		t.Run(tc.name, func(t *testing.T) {
			git := &scheduleHubGit{root: tc.root}
			got, err := resolveScheduleHub(t.Context(), filepath.Join(hub, "subdir"), tc.hubs, git)
			if (err == nil) != tc.allowed || tc.allowed && got != hub || git.calls != 1 {
				t.Fatalf("root=%q err=%v calls=%d", got, err, git.calls)
			}
		})
	}
}
