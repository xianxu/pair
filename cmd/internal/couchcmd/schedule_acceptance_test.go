package couchcmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"syscall"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchschedule"
	"github.com/xianxu/pair/cmd/internal/operationalrun"
	"github.com/xianxu/pair/cmd/internal/procutil"
)

// This qualifies the installed command's private owner dispatch, not a test
// binary impersonating the owner. All authority, permits and tasks are disposable.
func TestScheduleNativeManualAcceptance(t *testing.T) {
	base, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	defer cancel()
	binary := filepath.Join(base, "couch")
	build := exec.CommandContext(ctx, "go", "build", "-o", binary, "../../couch")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build real Couch: %v\n%s", err, output)
	}
	root := filepath.Join(base, "hub")
	taskDir := filepath.Join(root, "scripts", "scheduled", "report")
	if err := os.MkdirAll(taskDir, 0700); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"go.mod": "module example.invalid/scheduled-acceptance\n\ngo 1.24\n",
		"scripts/scheduled/report/main.go": `package main
import("encoding/json";"fmt";"io";"net";"os")
func main(){
 cwd,err:=os.Getwd();if err!=nil{panic(err)}
 json.NewEncoder(os.Stdout).Encode(struct{Args []string;Cwd string;PID int}{os.Args[1:],cwd,os.Getpid()})
 if len(os.Args)>1&&os.Args[1]=="--help"{fmt.Println("report fixture help");return}
 if len(os.Args)>1&&os.Args[1]=="wait"{
  conn,err:=net.Dial("tcp",os.Args[2]);if err!=nil{panic(err)}
  json.NewEncoder(conn).Encode(struct{PID int}{os.Getpid()})
  io.Copy(io.Discard,conn);os.Exit(3)
 }
 os.Exit(10)
}
`,
	}
	for name, raw := range files {
		if err := os.WriteFile(filepath.Join(root, name), []byte(raw), 0600); err != nil {
			t.Fatal(err)
		}
	}
	namespace := filepath.Join(base, "namespace")
	store := couchschedule.StateStore{Dir: filepath.Join(namespace, "schedule")}
	key := couchschedule.TaskKey{Hub: root, Task: "report"}
	due := time.Now().UTC().Add(48 * time.Hour)
	setDeclaration := func(timeout string) {
		t.Helper()
		raw := fmt.Sprintf(`{"version":1,"cron":"0 7 * * *","timeout":%q,"script_run":true}`, timeout)
		if err := os.WriteFile(filepath.Join(taskDir, "schedule"), []byte(raw), 0600); err != nil {
			t.Fatal(err)
		}
		decls, err := (couchschedule.Catalog{}).Discover(ctx, []string{root})
		if err != nil || len(decls) != 1 || !decls[0].Allowed {
			t.Fatalf("fixture catalog: %+v %v", decls, err)
		}
		if err := updateScheduleFixture(store, ctx, func(l *couchschedule.Ledger) error {
			state, _ := couchschedule.Advance(l.Tasks[key.ID()], couchschedule.Event{Kind: couchschedule.CatalogChanged, At: time.Now(), Declaration: &decls[0], Location: time.UTC})
			state.NextDue = due
			l.Tasks[key.ID()] = state
			return nil
		}); err != nil {
			t.Fatal(err)
		}
	}
	setDeclaration("45s")
	id := "native-help"
	runner := manualTaskRunner{namespace: namespace, permitsDir: filepath.Join(base, "permits"), executor: couchschedule.Executor{Program: binary}, now: time.Now, newID: func() (string, error) { return id, nil }}
	checkSaved := func(runID, outcome string, result couchschedule.ExecutionResult) {
		t.Helper()
		ledger, err := store.View(ctx)
		if err != nil {
			t.Fatal(err)
		}
		state := ledger.Tasks[key.ID()]
		if state.Active != nil || state.Last == nil || state.Last.ID != runID || state.Last.Source != "manual" || !state.NextDue.Equal(due) {
			t.Fatalf("completion/recurrence: %+v", state)
		}
		record, output, err := (operationalrun.Store{Namespace: namespace}).Read(ctx, operationalrun.RunKey{Hub: key.HubID(), Task: key.Task, ID: runID})
		if err != nil || record.Source != "manual" || record.Outcome != outcome || record.Notice != "manual" || record.EndedAt.IsZero() || !bytes.Equal(output, result.Output) {
			t.Fatalf("retained bundle: %+v output=%q error=%v", record, output, err)
		}
		if result.Exit >= 0 && (record.Exit == nil || *record.Exit != result.Exit) {
			t.Fatalf("lost exit code: %+v", record)
		}
	}
	help, err := runner.run(ctx, root, "report", []string{"--help"})
	if err != nil || help.Exit != 0 || help.Outcome != "quiet" || !bytes.Contains(help.Output, []byte("report fixture help")) {
		t.Fatalf("real help: %+v error=%v", help, err)
	}
	checkSaved(id, "quiet", help)
	if help.BuildKey == "" {
		t.Fatal("real help did not report its build cache key")
	}
	cacheBinary := filepath.Join(namespace, "schedule", "cache", help.BuildKey, "task")
	first, err := os.Stat(cacheBinary)
	if err != nil {
		t.Fatal(err)
	}
	id = "native-run"
	marker := filepath.Join(root, "shell-expanded")
	args := []string{"script-run", "space value", "$(touch " + marker + ")", "", "--json", "--", "'quoted'"}
	result, err := runner.run(ctx, root, "report", args)
	if err != nil || result.Exit != 10 || result.Outcome != "wake" {
		t.Fatalf("real exit-10 run: %+v error=%v", result, err)
	}
	var report struct {
		Args []string
		Cwd  string
		PID  int
	}
	if err := json.Unmarshal(bytes.TrimSpace(result.Output), &report); err != nil || !reflect.DeepEqual(report.Args, args) || report.Cwd != root {
		t.Fatalf("literal argv/cwd: %+v output=%q error=%v", report, result.Output, err)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("argv was shell-expanded: %v", err)
	}
	checkSaved(id, "wake", result)
	second, err := os.Stat(cacheBinary)
	if err != nil || result.BuildKey != help.BuildKey || !os.SameFile(first, second) || !first.ModTime().Equal(second.ModTime()) {
		t.Fatalf("same-source call rebuilt/replaced executable: keys=%q/%q error=%v", help.BuildKey, result.BuildKey, err)
	}

	// The task announces its actual process over a local connection, then waits
	// for EOF. The declared timeout must stop/join that process before publication.
	setDeclaration("4s")
	id = "native-timeout"
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	type completion struct {
		result couchschedule.ExecutionResult
		err    error
	}
	completed := make(chan completion, 1)
	go func() {
		result, err := runner.run(ctx, root, "report", []string{"wait", listener.Addr().String()})
		completed <- completion{result, err}
	}()
	type accepted struct {
		conn net.Conn
		err  error
	}
	connections := make(chan accepted, 1)
	go func() { conn, err := listener.Accept(); connections <- accepted{conn, err} }()
	var conn net.Conn
	select {
	case done := <-completed:
		t.Fatalf("task never announced readiness: %+v error=%v", done.result, done.err)
	case accepted := <-connections:
		if accepted.err != nil {
			t.Fatal(accepted.err)
		}
		conn = accepted.conn
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	defer conn.Close()
	if err := conn.SetReadDeadline(time.Now().Add(10 * time.Second)); err != nil {
		t.Fatal(err)
	}
	var ready struct{ PID int }
	if err := json.NewDecoder(conn).Decode(&ready); err != nil {
		t.Fatal(err)
	}
	identity := procutil.StrictIdentity(strconv.Itoa(ready.PID))
	if ready.PID <= 0 || identity == "" {
		t.Fatalf("invalid native readiness: %+v", ready)
	}
	t.Cleanup(func() {
		if procutil.StrictIdentity(strconv.Itoa(ready.PID)) == identity {
			_ = syscall.Kill(ready.PID, syscall.SIGKILL)
		}
	})
	var done completion
	select {
	case done = <-completed:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	if procutil.StrictIdentity(strconv.Itoa(ready.PID)) == identity {
		t.Fatal("timeout returned with the exact task process still alive")
	}
	if done.err != nil || done.result.Outcome != "timeout" {
		t.Fatalf("manual declared timeout: outcome=%q exit=%d detail=%q error=%v", done.result.Outcome, done.result.Exit, done.result.Error, done.err)
	}
	if done.result.BuildKey != help.BuildKey {
		t.Fatalf("declaration timeout changed cached source identity: %q != %q", done.result.BuildKey, help.BuildKey)
	}
	checkSaved(id, "timeout", done.result)
}
