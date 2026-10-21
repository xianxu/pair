package couchcmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/couchschedule"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

type manualRPCFake struct {
	intent                     couchschedule.ManualIntent
	submits, observes, cancels int
	output                     []byte
}

func (f *manualRPCFake) Submit(_ context.Context, m couchschedule.ManualIntent) (couchschedule.ManualStatus, error) {
	f.submits++
	if f.intent.ID != "" && f.intent.Digest() != m.Digest() {
		return couchschedule.ManualStatus{}, errors.New("conflicting intent")
	}
	f.intent = m
	return f.status(0), nil
}
func (f *manualRPCFake) status(offset int) couchschedule.ManualStatus {
	exit := 10
	end := min(offset+couchschedule.MaxManualOutputPage, len(f.output))
	return couchschedule.ManualStatus{ID: f.intent.ID, Hub: f.intent.Hub, Task: f.intent.Task, Stage: "done", Exit: &exit, Output: f.output[offset:end], Offset: offset, Total: len(f.output)}
}
func (f *manualRPCFake) Observe(_ context.Context, hub, task, id string, offset int) (couchschedule.ManualStatus, error) {
	f.observes++
	return f.status(offset), nil
}
func (f *manualRPCFake) Cancel(_ context.Context, hub, task, id string) (couchschedule.ManualStatus, error) {
	f.cancels++
	return f.status(0), nil
}
func TestScheduleRPCRejectsForeignHubBeforeOwner(t *testing.T) {
	f := &manualRPCFake{}
	r := couchmessage.Request{Op: "schedule-run", Manual: &couchschedule.ManualRequest{Intent: couchschedule.ManualIntent{ID: "run-1", Hub: "/foreign", Task: "task"}}}
	if response := handleScheduleRun(context.Background(), r, []string{"/hub"}, f); response.Code == "ok" || f.submits != 0 {
		t.Fatalf("response=%#v submits=%d", response, f.submits)
	}
}
func TestScheduleRPCClientLostSubmitNeverResubmits(t *testing.T) {
	n := 0
	call := func(_ context.Context, _ string, request, result any) error { n++; return errors.New("reply lost") }
	var out, stderr bytes.Buffer
	code := runScheduleRemote(context.Background(), &scheduleArgs{task: "task"}, "/hub", "socket", &out, &stderr, call)
	if code != 1 || n != 1 || !strings.Contains(stderr.String(), "unknown admission") || !strings.Contains(stderr.String(), "run ID") {
		t.Fatalf("code=%d calls=%d error=%s", code, n, &stderr)
	}
}
func TestScheduleRPCClientCompletionPages(t *testing.T) {
	f := &manualRPCFake{output: bytes.Repeat([]byte("x"), 40000)}
	call := func(ctx context.Context, _ string, request, result any) error {
		r := request.(couchmessage.Request)
		*result.(*couchmessage.Response) = handleScheduleRun(ctx, r, []string{"/hub"}, f)
		return nil
	}
	var out, stderr bytes.Buffer
	if code := runScheduleRemote(context.Background(), &scheduleArgs{task: "task"}, "/hub", "socket", &out, &stderr, call); code != 10 || !bytes.Equal(out.Bytes(), f.output) || f.submits != 1 || f.observes != 2 {
		t.Fatalf("code=%d submits=%d observes=%d output=%d stderr=%s", code, f.submits, f.observes, out.Len(), &stderr)
	}
}

func TestScheduleRPCRealSocketPagesAndDuplicateConflict(t *testing.T) {
	f := &manualRPCFake{output: bytes.Repeat([]byte{0, 0xff, 'x'}, 20000)}
	socket, err := couchmessage.SocketPath(t.TempDir(), "manual-rpc")
	if err != nil {
		t.Fatal(err)
	}
	server, err := couchmessage.StartServer(context.Background(), socket, func(ctx context.Context, raw []byte) ([]byte, error) {
		var r couchmessage.Request
		if err := json.Unmarshal(raw, &r); err != nil {
			return nil, err
		}
		return json.Marshal(handleScheduleRun(ctx, r, []string{"/hub"}, f))
	})
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	var out, stderr bytes.Buffer
	if code := runScheduleRemote(context.Background(), &scheduleArgs{task: "task"}, "/hub", socket, &out, &stderr, couchmessage.Call); code != 10 || !bytes.Equal(out.Bytes(), f.output) {
		t.Fatalf("code=%d output=%d stderr=%s", code, out.Len(), &stderr)
	}
	original := f.intent
	for n := 0; n < 2; n++ {
		var response couchmessage.Response
		err := couchmessage.Call(context.Background(), socket, couchmessage.Request{Op: "schedule-run", Manual: &couchschedule.ManualRequest{Intent: original}}, &response)
		if err != nil || response.Code != "ok" {
			t.Fatalf("duplicate %v %#v", err, response)
		}
	}
	original.Args = []string{"changed"}
	var response couchmessage.Response
	if err := couchmessage.Call(context.Background(), socket, couchmessage.Request{Op: "schedule-run", Manual: &couchschedule.ManualRequest{Intent: original}}, &response); err != nil || response.Code != "uncertain" {
		t.Fatalf("conflict %v %#v", err, response)
	}
}
func TestScheduleRPCClientCancellationUsesExactID(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var submitted, cancelled string
	var ops []string
	call := func(ctx context.Context, _ string, request, result any) error {
		r := request.(couchmessage.Request)
		ops = append(ops, r.Op)
		m := r.Manual.Intent
		s := couchschedule.ManualStatus{ID: m.ID, Hub: m.Hub, Task: m.Task, Stage: "executing"}
		if r.Op == "schedule-run" {
			submitted = m.ID
			cancel()
		} else if r.Op == "schedule-run-cancel" {
			if ctx.Err() != nil {
				t.Fatal("cleanup reused cancelled context")
			}
			cancelled = m.ID
			s.Stage = "cancelled"
		}
		*result.(*couchmessage.Response) = couchmessage.Response{Code: "ok", Manual: &s}
		return nil
	}
	var out, stderr bytes.Buffer
	if code := runScheduleRemote(ctx, &scheduleArgs{task: "task"}, "/hub", "socket", &out, &stderr, call); code != 130 || submitted == "" || cancelled != submitted || len(ops) != 2 {
		t.Fatalf("code=%d submitted=%q cancelled=%q ops=%v", code, submitted, cancelled, ops)
	}
}
func TestScheduleRPCManualCLIOnlyUsesOwnerTransport(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if raw, err := exec.CommandContext(t.Context(), "git", "init", "-q", root).CombinedOutput(); err != nil {
		t.Fatalf("git init: %s %v", raw, err)
	}
	t.Chdir(root)
	namespace := filepath.Join(root, "absent")
	f := &manualRPCFake{output: []byte("remote completion")}
	call := func(ctx context.Context, _ string, request, result any) error {
		r := request.(couchmessage.Request)
		if r.Op == "schedule-status" {
			*result.(*couchmessage.Response) = couchmessage.Response{Code: "ok", Schedule: &couchschedule.StatusSnapshot{Version: 1, Hubs: []string{root}, Tasks: []couchschedule.TaskStatus{}}}
			return nil
		}
		*result.(*couchmessage.Response) = handleScheduleRun(ctx, r, []string{root}, f)
		return nil
	}
	var out, stderr bytes.Buffer
	if code := runScheduleCLI(&scheduleArgs{verb: "run", task: "task"}, scheduleStatusRT{dir: namespace}, &out, &stderr, call); code != 10 || out.String() != "remote completion" || f.submits != 1 {
		t.Fatalf("code=%d submits=%d output=%s error=%s", code, f.submits, &out, &stderr)
	}
	if _, err := os.Stat(namespace); !os.IsNotExist(err) {
		t.Fatalf("manual CLI wrote namespace: %v", err)
	}
}
func TestScheduleRPCUncertainErrorSanitizesTerminalControl(t *testing.T) {
	call := func(_ context.Context, _ string, _ any, _ any) error { return errors.New("lost\x1b[2J") }
	var out, stderr bytes.Buffer
	runScheduleRemote(context.Background(), &scheduleArgs{task: "task"}, "/hub", "socket", &out, &stderr, call)
	if bytes.ContainsRune(stderr.Bytes(), 27) {
		t.Fatalf("terminal control in diagnostic: %q", stderr.String())
	}
}
