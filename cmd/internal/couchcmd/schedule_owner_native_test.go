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

	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/couchschedule"
	"github.com/xianxu/pair/cmd/internal/operationalrun"
	"github.com/xianxu/pair/cmd/internal/procutil"
	"github.com/xianxu/pair/cmd/internal/strictjson"
	"github.com/xianxu/pair/cmd/internal/testwait"
)

// Exercise the production actor, private Unix transport, CLI polling/output and
// native controller. No legacy manual runner or fake executor participates.
func TestScheduleOwnerNativeManualTransportAcceptance(t *testing.T) {
	base, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ctx, stop := context.WithTimeout(t.Context(), 2*time.Minute)
	defer stop()
	binary := filepath.Join(base, "couch")
	if output, err := exec.CommandContext(ctx, "go", "build", "-o", binary, "../../couch").CombinedOutput(); err != nil {
		t.Fatalf("build native Couch: %v\n%s", err, output)
	}
	root := filepath.Join(base, "hub")
	dir := filepath.Join(root, "scripts", "scheduled", "report")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"go.mod": "module example.invalid/owner-native\n\ngo 1.24\n",
		"scripts/scheduled/report/main.go": `package main
import("encoding/json";"fmt";"io";"net";"os";"os/exec";"os/signal";"syscall")
func main(){
 if len(os.Args)>1&&(os.Args[1]=="wait"||os.Args[1]=="child"){
  role:=os.Args[1]
  if role=="child"{signal.Ignore(syscall.SIGTERM)}else{
   child:=exec.Command(os.Args[0],"child",os.Args[2]);child.Stdout=os.Stdout;child.Stderr=os.Stderr
   if err:=child.Start();err!=nil{panic(err)}
  }
  c,err:=net.Dial("tcp",os.Args[2]);if err!=nil{panic(err)}
  json.NewEncoder(c).Encode(struct{PID int;Role string}{os.Getpid(),role})
  io.Copy(io.Discard,c);return
 }
 cwd,err:=os.Getwd();if err!=nil{panic(err)}
 json.NewEncoder(os.Stdout).Encode(struct{Args []string;Cwd string}{os.Args[1:],cwd})
 if len(os.Args)>1&&os.Args[1]=="--help"{fmt.Println("owner native help");return}
 os.Exit(10)
}
`,
	}
	for name, raw := range files {
		if err := os.WriteFile(filepath.Join(root, name), []byte(raw), 0600); err != nil {
			t.Fatal(err)
		}
	}
	for _, mode := range []string{"quiet", "wake", "timeout", "cancel"} {
		t.Run(mode, func(t *testing.T) {
			timeout := "45s"
			if mode == "timeout" {
				timeout = "10s"
			}
			raw := fmt.Sprintf(`{"version":1,"cron":"0 7 * * *","timeout":%q,"script_run":true}`, timeout)
			if err := os.WriteFile(filepath.Join(dir, "schedule"), []byte(raw), 0600); err != nil {
				t.Fatal(err)
			}
			namespace := filepath.Join(base, mode)
			c := newScheduleCoordinator(namespace, []string{root}, time.UTC, time.Now)
			o := newScheduleOwner(c)
			o.configure(scheduleExecutionWorker{namespace: namespace, permitsDir: filepath.Join(base, "permits"), executor: couchschedule.Executor{Program: binary}, now: time.Now}, offlineScheduleNotices{})
			ownerCtx, cancelOwner := context.WithCancel(ctx)
			joined := make(chan struct{})
			go func() { defer close(joined); o.run(ownerCtx, nativeScheduleClock{}) }()
			defer func() {
				cancelOwner()
				select {
				case <-joined:
				case <-time.After(10 * time.Second):
					t.Error("native owner did not join")
				}
			}()
			testwait.Until(t, "native owner catalogue", func() bool { return len(c.view().Tasks) == 1 && c.view().Error == "" })
			key := couchschedule.TaskKey{Hub: root, Task: "report"}
			ledger, err := c.store.View(ctx)
			if err != nil {
				t.Fatal(err)
			}
			due := ledger.Tasks[key.ID()].NextDue
			s := &messageService{}
			s.schedule.Store(c)
			s.scheduler.Store(o)
			socket, err := couchmessage.SocketPath(namespace, "owner-native-test")
			if err != nil {
				t.Fatal(err)
			}
			submitted := make(chan couchschedule.ManualIntent, 1)
			server, err := couchmessage.StartServer(ctx, socket, func(callCtx context.Context, raw []byte) ([]byte, error) {
				var request couchmessage.Request
				if err := strictjson.Decode(raw, &request); err != nil {
					return nil, err
				}
				if request.Op == "schedule-run" {
					submitted <- request.Manual.Intent
				}
				return json.Marshal(s.handle(callCtx, request))
			})
			if err != nil {
				t.Fatal(err)
			}
			defer server.Close()
			runCtx, cancelRun := context.WithCancel(ctx)
			defer cancelRun()
			args := []string{"--help"}
			if mode == "wake" {
				args = []string{"script-run", "space value", "$(touch marker)", "", "--", "'quoted'"}
			}
			var listener net.Listener
			if mode == "timeout" || mode == "cancel" {
				listener, err = net.Listen("tcp", "127.0.0.1:0")
				if err != nil {
					t.Fatal(err)
				}
				defer listener.Close()
				if err = listener.(*net.TCPListener).SetDeadline(time.Now().Add(30 * time.Second)); err != nil {
					t.Fatal(err)
				}
				args = []string{"wait", listener.Addr().String()}
			}
			var output, errorsOut bytes.Buffer
			completed := make(chan int, 1)
			go func() {
				completed <- runScheduleRemote(runCtx, &scheduleArgs{verb: "run", task: "report", args: args}, root, socket, &output, &errorsOut, couchmessage.Call)
			}()
			var intent couchschedule.ManualIntent
			select {
			case intent = <-submitted:
			case <-ctx.Done():
				t.Fatal(ctx.Err())
			}
			identities := map[int]string{}
			if listener != nil {
				roles := map[string]bool{}
				for i := 0; i < 2; i++ {
					conn, err := listener.Accept()
					if err != nil {
						t.Fatal(err)
					}
					defer conn.Close()
					if err := conn.SetReadDeadline(time.Now().Add(10 * time.Second)); err != nil {
						t.Fatal(err)
					}
					var ready struct {
						PID  int
						Role string
					}
					if err := json.NewDecoder(conn).Decode(&ready); err != nil {
						t.Fatal(err)
					}
					birth := procutil.StrictIdentity(strconv.Itoa(ready.PID))
					if ready.PID <= 0 || birth == "" {
						t.Fatalf("missing task identity: %+v", ready)
					}
					identities[ready.PID] = birth
					roles[ready.Role] = true
					t.Cleanup(func() {
						if procutil.StrictIdentity(strconv.Itoa(ready.PID)) == birth {
							_ = syscall.Kill(ready.PID, syscall.SIGKILL)
						}
					})
				}
				if !roles["wait"] || !roles["child"] {
					t.Fatalf("missing native process group roles: %v", roles)
				}
				ledger, err = c.store.View(ctx)
				if err != nil {
					t.Fatal(err)
				}
				active := ledger.Tasks[key.ID()].Active
				if active == nil || active.ID != intent.ID || active.Stage != "running" || active.ControllerPID <= 0 || active.ControllerStart == "" {
					t.Fatalf("worker started without durable controller: %+v", active)
				}
				identities[active.ControllerPID] = active.ControllerStart
				if mode == "cancel" {
					cancelRun()
				}
			}
			var exit int
			select {
			case exit = <-completed:
			case <-ctx.Done():
				t.Fatal(ctx.Err())
			}
			expectedExit, expectedOutcome := 0, "quiet"
			switch mode {
			case "wake":
				expectedExit, expectedOutcome = 10, "wake"
			case "timeout":
				expectedExit, expectedOutcome = 1, "timeout"
			case "cancel":
				expectedExit, expectedOutcome = 130, "cancelled"
			}
			if exit != expectedExit {
				t.Fatalf("CLI exit=%d want=%d output=%q errors=%q", exit, expectedExit, output.String(), errorsOut.String())
			}
			// A cancellation acknowledgement may precede cleanup. Wait through the
			// transport until native completion is retained and authority is settled.
			var record operationalrun.Record
			var saved []byte
			testwait.Until(t, "native terminal settlement", func() bool {
				var response couchmessage.Response
				request := couchmessage.Request{Op: "schedule-run-status", Manual: &couchschedule.ManualRequest{Intent: couchschedule.ManualIntent{ID: intent.ID, Hub: root, Task: "report"}}}
				if couchmessage.Call(ctx, socket, request, &response) != nil || response.Code != "ok" || response.Manual == nil || !response.Manual.Terminal() {
					return false
				}
				ledger, err = c.store.View(ctx)
				if err != nil || ledger.Tasks[key.ID()].Active != nil {
					return false
				}
				record, saved, err = c.runs.Read(ctx, operationalrun.RunKey{Hub: key.HubID(), Task: key.Task, ID: intent.ID})
				return err == nil
			})
			state := ledger.Tasks[key.ID()]
			if record.Outcome != expectedOutcome || record.Source != "manual" || record.Notice != "manual" || record.ManualIntentDigest != intent.Digest() || record.EndedAt.IsZero() || state.Last == nil || state.Last.ID != intent.ID || !state.NextDue.Equal(due) {
				t.Fatalf("native history/recurrence: record=%+v state=%+v", record, state)
			}
			if mode == "quiet" || mode == "wake" {
				if record.Exit == nil || *record.Exit != expectedExit || !bytes.Equal(saved, output.Bytes()) {
					t.Fatalf("native output/exit mismatch: record=%+v saved=%q output=%q", record, saved, output.Bytes())
				}
				if mode == "quiet" && !bytes.Contains(saved, []byte("owner native help")) {
					t.Fatalf("missing help output: %q", saved)
				}
				if mode == "wake" {
					var report struct {
						Args []string
						Cwd  string
					}
					if err := json.Unmarshal(saved, &report); err != nil || !reflect.DeepEqual(report.Args, args) || report.Cwd != root {
						t.Fatalf("literal argv/cwd: %+v output=%q error=%v", report, saved, err)
					}
				}
			}
			for pid, birth := range identities {
				if procutil.StrictIdentity(strconv.Itoa(pid)) == birth {
					t.Errorf("settled %s while exact native process %d survived", mode, pid)
				}
			}
		})
	}
}
