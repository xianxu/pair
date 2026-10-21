package couchschedule

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/procutil"
	"golang.org/x/sys/unix"
)

func TestExecutionOwnerHelper(t *testing.T) {
	if os.Getenv("COUCH_SCHEDULE_TEST_OWNER") != "1" {
		return
	}
	os.Exit(RunOwner(os.Stdin, os.Stdout, os.Stderr))
}

func TestExecutionParentDeathHelper(t *testing.T) {
	if os.Getenv("COUCH_SCHEDULE_TEST_PARENT") == "" {
		return
	}
	var r ExecutionRequest
	if json.Unmarshal([]byte(os.Getenv("COUCH_SCHEDULE_TEST_REQUEST")), &r) != nil {
		os.Exit(92)
	}
	marker := os.Getenv("COUCH_SCHEDULE_TEST_READY")
	go func() {
		for {
			if _, err := os.Stat(r.Args[0]); err == nil {
				os.Exit(33)
			}
			time.Sleep(time.Millisecond)
		}
	}()
	e := Executor{PrefixArgs: []string{"-test.run=^TestExecutionOwnerHelper$", "--"}, OnOwner: func(o OwnerIdentity) error { raw, _ := json.Marshal(o); return os.WriteFile(marker, raw, 0600) }}
	_, _ = e.Run(context.Background(), r)
	os.Exit(93)
}

func TestExecutionLostResultHelper(t *testing.T) {
	if os.Getenv("COUCH_SCHEDULE_TEST_OWNER") != "1" {
		return
	}
	frame, err := readOwnerFrame(os.Stdin)
	if err != nil || frame.Request == nil {
		os.Exit(91)
	}
	owner := OwnerIdentity{PID: os.Getpid(), Start: procutil.StrictIdentity(strconv.Itoa(os.Getpid()))}
	if writeOwnerFrame(os.Stdout, ownerFrame{Kind: "ready", Owner: &owner}) != nil {
		os.Exit(92)
	}
	ack, err := readOwnerFrame(os.Stdin)
	if err != nil || ack.Kind != "authorize" {
		os.Exit(93)
	}
	if os.WriteFile(frame.Request.Args[0], []byte("effect"), 0600) != nil {
		os.Exit(94)
	}
	os.Exit(44)
}

func TestExecutionUncertainStreamHelper(t *testing.T) {
	if os.Getenv("COUCH_SCHEDULE_TEST_OWNER") != "1" {
		return
	}
	f, err := readOwnerFrame(os.Stdin)
	if err != nil || f.Request == nil {
		os.Exit(91)
	}
	owner := OwnerIdentity{PID: os.Getpid(), Start: procutil.StrictIdentity(strconv.Itoa(os.Getpid()))}
	if writeOwnerFrame(os.Stdout, ownerFrame{Kind: "ready", Owner: &owner}) != nil {
		os.Exit(92)
	}
	ack, err := readOwnerFrame(os.Stdin)
	if err != nil || ack.Kind != "authorize" {
		os.Exit(93)
	}
	stream, err := inheritedOwnerOutput(f.StreamOutput)
	if err != nil || stream == nil {
		os.Exit(94)
	}
	_, _ = stream.Write([]byte("prefix"))
	stream.Close()
	r := ExecutionResult{Exit: -1, Outcome: "uncertain", Output: []byte("prefix"), OutputBytes: 6, Error: "retained uncertain work group"}
	if writeOwnerFrame(os.Stdout, ownerFrame{Kind: "result", Result: &r}) != nil {
		os.Exit(95)
	}
	// Keep a native event wait alive without intercepting the parent's TERM/KILL.
	hold := make(chan os.Signal, 1)
	signal.Notify(hold, syscall.SIGUSR1)
	defer signal.Stop(hold)
	<-hold
}

func TestExecutionManualOwnedRun(t *testing.T) {
	e, r := executionFixture(t, "package main\nimport (\"fmt\";\"os\")\nfunc main(){fmt.Print(os.Args[1]);os.Exit(10)}\n")
	r.Args = []string{"literal argument"}
	var owner OwnerIdentity
	e.OnOwner = func(o OwnerIdentity) error { owner = o; return nil }
	result, err := e.Run(context.Background(), r)
	if err != nil || result.Exit != 10 || result.Outcome != "wake" || string(result.Output) != "literal argument" {
		t.Fatalf("result %#v err %v", result, err)
	}
	if owner.PID <= 0 || owner.Start == "" {
		t.Fatalf("no owned controller: %#v", owner)
	}
	if _, err = os.Stat(filepath.Join(r.PermitsDir, "permit0.lock")); err != nil {
		t.Fatal(err)
	}
}

func executionFixture(t *testing.T, body string) (Executor, ExecutionRequest) {
	t.Helper()
	t.Setenv("COUCH_SCHEDULE_TEST_OWNER", "1")
	root := t.TempDir()
	writeBuildFixture(t, root, "go.mod", "module fixture.local/task\n\ngo 1.23\n")
	writeBuildFixture(t, root, "scripts/scheduled/task/main.go", body)
	writeBuildFixture(t, root, "scripts/scheduled/task/schedule", `{"version":1,"cron":"* * * * *","timeout":"1m"}`)
	declarations, err := (Catalog{}).Discover(t.Context(), []string{root})
	if err != nil || len(declarations) != 1 || !declarations[0].Allowed {
		t.Fatalf("fixture catalogue: %#v %v", declarations, err)
	}
	r := ExecutionRequest{Namespace: t.TempDir(), HubRoot: root, Task: "task", RunID: "run1", Revision: "revision1", CacheDir: t.TempDir(), PermitsDir: t.TempDir(), Spec: TaskSpec{Timeout: time.Minute}, Deadline: time.Now().Add(time.Minute)}
	r.Revision = declarations[0].Revision
	r.Spec = declarations[0].Spec
	e := Executor{PrefixArgs: []string{"-test.run=^TestExecutionOwnerHelper$", "--"}, OnOwner: func(OwnerIdentity) error { return nil }}
	return e, r
}

func TestExecutionOwnerMustBeDurableBeforeEffects(t *testing.T) {
	e, r := executionFixture(t, "package main\nimport \"os\"\nfunc main(){os.WriteFile(os.Args[1],[]byte(\"effect\"),0600)}\n")
	marker := filepath.Join(t.TempDir(), "effect")
	r.Args = []string{marker}
	var owner OwnerIdentity
	e.OnOwner = func(o OwnerIdentity) error {
		owner = o
		if _, err := os.Stat(marker); !os.IsNotExist(err) {
			t.Fatal("task started before acknowledgement")
		}
		return errors.New("durability refused")
	}
	if _, err := e.Run(t.Context(), r); err == nil {
		t.Fatal("undurable owner executed")
	} else {
		var executionError *ExecutionError
		if !errors.As(err, &executionError) || executionError.MayHaveStarted {
			t.Fatalf("pre-ack refusal has wrong uncertainty: %v", err)
		}
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatal("task effect after refused acknowledgement")
	}
	if procutil.StrictIdentity(strconv.Itoa(owner.PID)) == owner.Start {
		t.Fatal("refused controller survived")
	}
}

func TestExecutionLostResultRetainsStartedAuthority(t *testing.T) {
	e, r := executionFixture(t, "package main\nfunc main(){}\n")
	r.Args = []string{filepath.Join(t.TempDir(), "effect")}
	e.PrefixArgs = []string{"-test.run=^TestExecutionLostResultHelper$", "--"}
	_, err := e.Run(t.Context(), r)
	var lost *ExecutionError
	if !errors.As(err, &lost) || !lost.MayHaveStarted || lost.Owner.PID <= 0 || lost.Owner.Start == "" {
		t.Fatalf("lost result lacks authority: %#v %v", lost, err)
	}
	if _, err = os.Stat(r.Args[0]); err != nil {
		t.Fatalf("negative control did not execute an effect: %v", err)
	}
}

func TestExecutionParentCancellationJoinsResistantDescendant(t *testing.T) {
	e, r := executionFixture(t, `package main
import("os";"os/exec";"time")
func main(){c:=exec.Command("sh","-c","trap '' TERM; echo $$ > \"$1\"; while :; do sleep 1; done","sh",os.Args[1]);if c.Start()!=nil{os.Exit(2)};for{time.Sleep(time.Second)}}
`)
	marker := filepath.Join(t.TempDir(), "child")
	r.Args = []string{marker}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	var owner OwnerIdentity
	e.OnOwner = func(o OwnerIdentity) error { owner = o; return nil }
	type outcome struct {
		r   ExecutionResult
		err error
	}
	done := make(chan outcome, 1)
	go func() { res, err := e.Run(ctx, r); done <- outcome{res, err} }()
	child := waitExecutionPID(t, marker)
	childIdentity := procutil.StrictIdentity(strconv.Itoa(child))
	if childIdentity == "" {
		t.Fatal("child never had exact identity")
	}
	cancel()
	select {
	case got := <-done:
		if got.err != nil || got.r.Outcome != "cancelled" {
			t.Fatalf("cancel result %#v %v", got.r, got.err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("cancel did not join controller")
	}
	if procutil.StrictIdentity(strconv.Itoa(child)) == childIdentity {
		t.Fatal("resistant descendant survived cancellation")
	}
	if procutil.StrictIdentity(strconv.Itoa(owner.PID)) == owner.Start {
		t.Fatal("controller survived known cleanup")
	}
	for i := 0; i < 2; i++ {
		release, ok, err := executionLease(filepath.Join(r.PermitsDir, fmt.Sprintf("permit%d.lock", i)))
		if err != nil || !ok {
			t.Fatalf("permit retained after child cleanup: %v %v", ok, err)
		}
		release()
	}
}

func waitExecutionPID(t *testing.T, path string) int {
	t.Helper()
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		raw, err := os.ReadFile(path)
		if err == nil {
			n, err := strconv.Atoi(strings.TrimSpace(string(raw)))
			if err == nil && n > 0 {
				return n
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("native child readiness absent: %s", path)
	return 0
}

func TestExecutionStopOwnerChecksExactIdentity(t *testing.T) {
	if err := StopOwner(OwnerIdentity{PID: os.Getpid(), Start: "different-incarnation"}); err == nil {
		t.Fatal("mismatching owner accepted")
	}
	if err := syscall.Kill(os.Getpid(), 0); err != nil {
		t.Fatal(err)
	}
}

func TestExecutionStopOwnerSignalsJoinedController(t *testing.T) {
	e, r := executionFixture(t, "package main\nimport(\"os\";\"strconv\";\"time\")\nfunc main(){os.WriteFile(os.Args[1],[]byte(strconv.Itoa(os.Getpid())),0600);for{time.Sleep(time.Second)}}\n")
	r.Args = []string{filepath.Join(t.TempDir(), "task")}
	owners := make(chan OwnerIdentity, 1)
	e.OnOwner = func(o OwnerIdentity) error { owners <- o; return nil }
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	type outcome struct {
		r   ExecutionResult
		err error
	}
	done := make(chan outcome, 1)
	go func() { result, err := e.Run(ctx, r); done <- outcome{result, err} }()
	owner := <-owners
	task := waitExecutionPID(t, r.Args[0])
	birth := procutil.StrictIdentity(strconv.Itoa(task))
	if err := StopOwner(owner); err != nil {
		t.Fatal(err)
	}
	select {
	case got := <-done:
		if got.err != nil || got.r.Outcome != "cancelled" {
			t.Fatalf("owner stop %#v %v", got.r, got.err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("owner signal did not join")
	}
	if procutil.StrictIdentity(strconv.Itoa(task)) == birth {
		t.Fatal("StopOwner left task alive")
	}
}

func TestExecutionHeavyRefusesBypassAndSkippedAdmission(t *testing.T) {
	t.Setenv("WF_TEST_LOCK", "off")
	t.Setenv("WF_TEST_LOCK_HELD_BY", "1")
	t.Setenv("WF_TEST_LOCK_FILE", "forged")
	e, r := executionFixture(t, "package main\nimport \"os\"\nfunc main(){os.WriteFile(os.Args[1],[]byte(\"effect\"),0600)}\n")
	marker := filepath.Join(t.TempDir(), "effect")
	r.Args = []string{marker}
	r.Spec.Heavy = true
	refreshExecutionDeclaration(t, &r)
	helper := filepath.Join(t.TempDir(), "admit.py")
	if err := os.WriteFile(helper, []byte("import os\nassert 'WF_TEST_LOCK' not in os.environ\nassert 'WF_TEST_LOCK_HELD_BY' not in os.environ\nassert 'WF_TEST_LOCK_FILE' not in os.environ\nprint('skipped')\n"), 0600); err != nil {
		t.Fatal(err)
	}
	r.TestLockHelper = helper
	result, err := e.Run(t.Context(), r)
	if err != nil || result.Outcome != "failed" || !strings.Contains(result.Error, "admission refused: skipped") {
		t.Fatalf("admission result %#v %v", result, err)
	}
	if _, err = os.Stat(marker); !os.IsNotExist(err) {
		t.Fatal("task ran after skipped admission")
	}
}

func TestExecutionActualParentDeathJoinsBeforeReleasingPermit(t *testing.T) {
	_, r := executionFixture(t, `package main
import("os";"os/exec";"time")
func main(){c:=exec.Command("sh","-c","trap '' TERM; echo $$ > \"$1\"; while :; do sleep 1; done","sh",os.Args[1]);if c.Start()!=nil{os.Exit(2)};for{time.Sleep(time.Second)}}
`)
	r.Args = []string{filepath.Join(t.TempDir(), "child")}
	ready := filepath.Join(t.TempDir(), "owner")
	raw, _ := json.Marshal(r)
	program, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	parent := exec.Command(program, "-test.run=^TestExecutionParentDeathHelper$", "--")
	parent.Env = append(os.Environ(), "COUCH_SCHEDULE_TEST_PARENT=1", "COUCH_SCHEDULE_TEST_REQUEST="+string(raw), "COUCH_SCHEDULE_TEST_READY="+ready)
	if err = parent.Start(); err != nil {
		t.Fatal(err)
	}
	child := waitExecutionPID(t, r.Args[0])
	birth := procutil.StrictIdentity(strconv.Itoa(child))
	if birth == "" {
		t.Fatal("child identity unavailable")
	}
	if err = parent.Wait(); err == nil || parent.ProcessState.ExitCode() != 33 {
		t.Fatalf("fixture parent did not exit at readiness: %v", err)
	}
	ownerRaw, err := os.ReadFile(ready)
	if err != nil {
		t.Fatal(err)
	}
	var owner OwnerIdentity
	if err = json.Unmarshal(ownerRaw, &owner); err != nil {
		t.Fatal(err)
	}
	end := time.Now().Add(10 * time.Second)
	for {
		childAlive := procutil.StrictIdentity(strconv.Itoa(child)) == birth
		controllerAlive := procutil.StrictIdentity(strconv.Itoa(owner.PID)) == owner.Start
		if !childAlive && !controllerAlive {
			break
		}
		if time.Now().After(end) {
			t.Fatalf("parent death left child/controller: child=%v controller=%v", childAlive, controllerAlive)
		}
		if childAlive {
			release, taken, err := executionLease(filepath.Join(r.PermitsDir, "permit0.lock"))
			if err != nil {
				t.Fatal(err)
			}
			if taken {
				release()
				t.Fatal("machine permit released before descendant cleanup")
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	release, taken, err := executionLease(filepath.Join(r.PermitsDir, "permit0.lock"))
	if err != nil || !taken {
		t.Fatalf("permit not released after cleanup: %v %v", taken, err)
	}
	release()
}

func TestExecutionDeclaredTimeout(t *testing.T) {
	e, r := executionFixture(t, "package main\nimport \"time\"\nfunc main(){time.Sleep(time.Minute)}\n")
	artifact, err := (Builder{CacheDir: r.CacheDir}).Build(t.Context(), r.HubRoot, r.Task)
	if err != nil {
		t.Fatal(err)
	}
	artifact.Close()
	r.Spec.Timeout = time.Second
	writeBuildFixture(t, r.HubRoot, "scripts/scheduled/task/schedule", `{"version":1,"cron":"* * * * *","timeout":"1s"}`)
	rows, err := (Catalog{}).Discover(t.Context(), []string{r.HubRoot})
	if err != nil {
		t.Fatal(err)
	}
	r.Revision = rows[0].Revision
	r.Spec = rows[0].Spec
	result, err := e.Run(t.Context(), r)
	if err != nil || result.Outcome != "timeout" {
		t.Fatalf("timeout result %#v %v", result, err)
	}
}

func TestExecutionMatchingParentDeadlineKeepsTimeoutCause(t *testing.T) {
	e, r := executionFixture(t, "package main\nimport(\"os\";\"strconv\";\"time\")\nfunc main(){os.WriteFile(os.Args[1],[]byte(strconv.Itoa(os.Getpid())),0600);for{time.Sleep(time.Second)}}\n")
	r.Spec.Timeout = 2 * time.Second
	refreshExecutionDeclaration(t, &r)
	artifact, err := (Builder{CacheDir: r.CacheDir}).Build(t.Context(), r.HubRoot, r.Task)
	if err != nil {
		t.Fatal(err)
	}
	artifact.Close()
	r.Args = []string{filepath.Join(t.TempDir(), "ready")}
	r.Deadline = time.Now().Add(r.Spec.Timeout)
	ctx, cancel := context.WithDeadline(t.Context(), r.Deadline)
	defer cancel()
	type outcome struct {
		result ExecutionResult
		err    error
	}
	done := make(chan outcome, 1)
	go func() { res, err := e.Run(ctx, r); done <- outcome{res, err} }()
	pid := waitExecutionPID(t, r.Args[0])
	birth := procutil.StrictIdentity(strconv.Itoa(pid))
	got := <-done
	if got.err != nil || got.result.Outcome != "timeout" {
		t.Fatalf("matching parent deadline lost cause: %#v %v", got.result, got.err)
	}
	if procutil.StrictIdentity(strconv.Itoa(pid)) == birth {
		t.Fatal("deadline result preceded task cleanup")
	}
}

func TestExecutionStreamsBeforeCompletionAndEmitsBoundedOutputOnce(t *testing.T) {
	e, r := executionFixture(t, `package main
import("fmt";"os";"strings";"time")
func main(){fmt.Print("READY\n");for{if _,err:=os.Stat(os.Args[1]);err==nil{break};time.Sleep(time.Millisecond)};fmt.Print(strings.Repeat("x",400000));fmt.Print("TAIL")}
`)
	r.Spec.Timeout = 2 * time.Second
	refreshExecutionDeclaration(t, &r)
	ack := filepath.Join(t.TempDir(), "stream-seen")
	r.Args = []string{ack}
	var streamed []byte
	e.OnOutput = func(chunk []byte) {
		streamed = append(streamed, chunk...)
		if bytes.Contains(streamed, []byte("READY\n")) {
			_ = os.WriteFile(ack, []byte("ok"), 0600)
		}
	}
	result, err := e.Run(t.Context(), r)
	if err != nil || result.Outcome != "quiet" {
		t.Fatalf("task needed pre-completion stream: %#v %v", result, err)
	}
	if !bytes.Equal(streamed, result.Output) || len(streamed) > 256<<10 || result.OutputBytes != 400010 {
		t.Fatalf("duplicate/unbounded stream: stream=%d retained=%d total=%d", len(streamed), len(result.Output), result.OutputBytes)
	}
}

func TestExecutionSlowOutputCannotDelayControllerCleanup(t *testing.T) {
	e, r := executionFixture(t, `package main
import("fmt";"os";"strconv";"strings";"time")
func main(){os.WriteFile(os.Args[1],[]byte(strconv.Itoa(os.Getpid())),0600);fmt.Print(strings.Repeat("x",400000));os.WriteFile(os.Args[2],[]byte("drained"),0600);for{time.Sleep(time.Second)}}
`)
	r.Spec.Timeout = 2 * time.Second
	refreshExecutionDeclaration(t, &r)
	r.Args = []string{filepath.Join(t.TempDir(), "task"), filepath.Join(t.TempDir(), "drained")}
	owners := make(chan OwnerIdentity, 1)
	e.OnOwner = func(o OwnerIdentity) error { owners <- o; return nil }
	blocked := make(chan struct{})
	release := make(chan struct{})
	var once sync.Once
	e.OnOutput = func([]byte) { once.Do(func() { close(blocked); <-release }) }
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	type outcome struct {
		result ExecutionResult
		err    error
	}
	done := make(chan outcome, 1)
	go func() { res, err := e.Run(ctx, r); done <- outcome{res, err} }()
	var releaseOnce sync.Once
	unblock := func() { releaseOnce.Do(func() { close(release) }) }
	defer unblock()
	owner := <-owners
	task := waitExecutionPID(t, r.Args[0])
	birth := procutil.StrictIdentity(strconv.Itoa(task))
	<-blocked
	end := time.Now().Add(10 * time.Second)
	for procutil.StrictIdentity(strconv.Itoa(owner.PID)) == owner.Start {
		if time.Now().After(end) {
			t.Fatal("blocked output held controller beyond native cleanup")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if _, err := os.Stat(r.Args[1]); err != nil {
		t.Fatal("full output pipe stopped task draining")
	}
	if procutil.StrictIdentity(strconv.Itoa(task)) == birth {
		t.Fatal("controller exited before task cleanup")
	}
	// Releasing the consumer allows the parent reader to finish publishing its
	// retained suffix; the controller and native task are already gone.
	unblock()
	select {
	case got := <-done:
		if got.err != nil || got.result.Outcome != "timeout" {
			t.Fatalf("slow output result %#v %v", got.result, got.err)
		}
	case <-time.After(time.Second):
		t.Fatal("output reader did not join after release")
	}
}

func TestExecutionUncertainResultJoinsStreamWithoutJoiningController(t *testing.T) {
	e, r := executionFixture(t, "package main\nfunc main(){}\n")
	e.PrefixArgs = []string{"-test.run=^TestExecutionUncertainStreamHelper$", "--"}
	var owner OwnerIdentity
	e.OnOwner = func(o OwnerIdentity) error { owner = o; return nil }
	var output []byte
	e.OnOutput = func(p []byte) { output = append(output, p...) }
	type outcome struct {
		result ExecutionResult
		err    error
	}
	done := make(chan outcome, 1)
	go func() { result, err := e.Run(t.Context(), r); done <- outcome{result, err} }()
	select {
	case got := <-done:
		var uncertain *ExecutionError
		if !errors.As(got.err, &uncertain) || !uncertain.MayHaveStarted || got.result.Outcome != "uncertain" || string(output) != "prefix" {
			t.Fatalf("uncertain stream %#v %q %v", got.result, output, got.err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("stream completion waited for retained controller")
	}
	defer StopOwner(owner)
	if procutil.StrictIdentity(strconv.Itoa(owner.PID)) != owner.Start {
		t.Fatal("uncertain controller did not retain ownership")
	}
	if err := StopOwner(owner); err != nil {
		t.Fatal(err)
	}
	end := time.Now().Add(time.Second)
	for procutil.StrictIdentity(strconv.Itoa(owner.PID)) == owner.Start {
		if time.Now().After(end) {
			t.Fatal("fixture controller was not reaped")
		}
		time.Sleep(time.Millisecond)
	}
}

func TestExecutionParentEndDeadlineBoundary(t *testing.T) {
	due := time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)
	for _, tt := range []struct {
		name   string
		at     time.Time
		cancel bool
	}{
		{"earlier explicit cancel or parent deadline", due.Add(-time.Nanosecond), true},
		{"same deadline", due, false},
		{"EOF after deadline", due.Add(time.Nanosecond), false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := parentEndCancelsOwner(tt.at, due); got != tt.cancel {
				t.Fatalf("parent end cancels=%v want %v", got, tt.cancel)
			}
		})
	}
}

func TestExecutionDeclarationChangedDuringBuildRefusesEffects(t *testing.T) {
	e, r := executionFixture(t, "package main\nimport \"os\"\nfunc main(){os.WriteFile(os.Args[1],[]byte(\"effect\"),0600)}\n")
	r.Args = []string{filepath.Join(t.TempDir(), "effect")}
	goBinary, err := exec.LookPath("go")
	if err != nil {
		t.Fatal(err)
	}
	wrapper := filepath.Join(t.TempDir(), "go-wrapper")
	// The wrapper changes only the declaration once capture is complete and
	// the actual compiler is about to consume the private captured view.
	script := "#!/bin/sh\nif [ \"$1\" = build ]; then printf '%s' '{\"version\":1,\"cron\":\"0 7 * * *\",\"timeout\":\"1m\"}' > '" + filepath.Join(r.HubRoot, "scripts/scheduled/task/schedule") + "'; fi\nexec '" + goBinary + "' \"$@\"\n"
	if err = os.WriteFile(wrapper, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	r.GoBinary = wrapper
	result, err := e.Run(t.Context(), r)
	if err != nil || result.Outcome != "failed" || !strings.Contains(result.Error, "authority changed") {
		t.Fatalf("stale declaration executed: %#v %v", result, err)
	}
	if _, err = os.Stat(r.Args[0]); !os.IsNotExist(err) {
		t.Fatal("task effect after declaration revision changed")
	}
}

func TestExecutionFreshAuthorityRefusesOperatorChanges(t *testing.T) {
	root := catalogRoot(t)
	catalogTask(t, root, "task", `,"scope":"team","owner":"xian-xu"`)
	writeBuildFixture(t, root, "team.md", "## Operators\n- xian-xu (Xian Xu): owns task\n")
	git := &catalogGit{names: map[string]string{root: "Xian Xu"}}
	catalog := Catalog{Git: git}
	rows, err := catalog.Discover(t.Context(), []string{root})
	if err != nil || len(rows) != 1 || !rows[0].Allowed {
		t.Fatalf("catalogue %#v %v", rows, err)
	}
	r := ExecutionRequest{HubRoot: root, Task: "task", Spec: rows[0].Spec, Revision: rows[0].Revision}
	if err = validateExecutionAuthority(t.Context(), catalog, r); err != nil {
		t.Fatal(err)
	}
	git.names[root] = "Other Operator"
	if err = validateExecutionAuthority(t.Context(), catalog, r); err == nil {
		t.Fatal("changed effective operator authorized task")
	}
	git.names[root] = "Xian Xu"
	writeBuildFixture(t, root, "team.md", "## Operators\n- another (Another): owns task\n")
	if err = validateExecutionAuthority(t.Context(), catalog, r); err == nil {
		t.Fatal("changed Operators authorized task")
	}
}

func TestExecutionHeavyNativeLeaseLifetime(t *testing.T) {
	helper := os.Getenv("COUCH_SCHEDULE_TEST_LOCK_HELPER")
	if helper == "" {
		t.Skip("set COUCH_SCHEDULE_TEST_LOCK_HELPER to the real workflow test-lock.py; all lease/queue paths are isolated")
	}
	e, r := executionFixture(t, `package main
import("os";"strconv";"time")
func main(){os.WriteFile(os.Args[1],[]byte(strconv.Itoa(os.Getpid())),0600);for{time.Sleep(time.Second)}}
`)
	r.Spec.Heavy = true
	refreshExecutionDeclaration(t, &r)
	r.TestLockHelper = helper
	r.TestLockFile = filepath.Join(t.TempDir(), "suite.lock")
	r.Args = []string{filepath.Join(t.TempDir(), "task")}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	type outcome struct {
		r   ExecutionResult
		err error
	}
	done := make(chan outcome, 1)
	go func() { result, err := e.Run(ctx, r); done <- outcome{result, err} }()
	task := waitExecutionPID(t, r.Args[0])
	birth := procutil.StrictIdentity(strconv.Itoa(task))
	if birth == "" {
		t.Fatal("task identity unavailable")
	}
	if free, err := nativeExecutionSlotFree(r.TestLockFile); err != nil || free {
		t.Fatalf("heavy task ran without actual lease: free=%v err=%v", free, err)
	}
	cancel()
	select {
	case got := <-done:
		if got.err != nil || got.r.Outcome != "cancelled" {
			t.Fatalf("heavy cleanup %#v %v", got.r, got.err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("heavy owner did not join")
	}
	end := time.Now().Add(10 * time.Second)
	for {
		free, err := nativeExecutionSlotFree(r.TestLockFile)
		if err != nil {
			t.Fatal(err)
		}
		if free {
			if procutil.StrictIdentity(strconv.Itoa(task)) == birth {
				t.Fatal("actual machine lease released before task cleanup")
			}
			break
		}
		if time.Now().After(end) {
			t.Fatal("actual machine lease remained after controller exit")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func nativeExecutionSlotFree(path string) (bool, error) {
	fd, err := unix.Open(path, unix.O_RDWR|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return false, err
	}
	defer unix.Close(fd)
	if err = unix.Flock(fd, unix.LOCK_EX|unix.LOCK_NB); err == unix.EWOULDBLOCK || err == unix.EAGAIN {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	_ = unix.Flock(fd, unix.LOCK_UN)
	return true, nil
}

func refreshExecutionDeclaration(t *testing.T, r *ExecutionRequest) {
	t.Helper()
	wire := map[string]any{"version": 1, "cron": r.Spec.Cron, "timeout": r.Spec.Timeout.String(), "scope": r.Spec.Scope, "missed": r.Spec.Missed, "script_run": r.Spec.ScriptRun, "heavy": r.Spec.Heavy}
	if r.Spec.Owner != "" {
		wire["owner"] = r.Spec.Owner
	}
	raw, err := json.Marshal(wire)
	if err != nil {
		t.Fatal(err)
	}
	writeBuildFixture(t, r.HubRoot, "scripts/scheduled/task/schedule", string(raw))
	rows, err := (Catalog{}).Discover(t.Context(), []string{r.HubRoot})
	if err != nil || len(rows) != 1 || !rows[0].Allowed {
		t.Fatalf("refresh fixture: %#v %v", rows, err)
	}
	r.Spec = rows[0].Spec
	r.Revision = rows[0].Revision
}
