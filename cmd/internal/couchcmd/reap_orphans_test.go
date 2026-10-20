package couchcmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/launcher"
	"github.com/xianxu/pair/cmd/internal/procutil"
)

// fakeOrphanTable is the reap's host: the rows a snapshot names, the identity
// each pid holds now, and the signals it delivered. diesOn models a process that
// ends on one signal, so a test can hold a clean SIGTERM and a survivor that
// needs SIGKILL apart.
type fakeOrphanTable struct {
	rows    []launcher.ProcessRow
	ids     map[int]string
	diesOn  map[int]syscall.Signal
	signals map[int][]syscall.Signal
	snapErr error
}

func (f *fakeOrphanTable) Snapshot(context.Context) ([]launcher.ProcessRow, error) {
	if f.snapErr != nil {
		return nil, f.snapErr
	}
	return append([]launcher.ProcessRow(nil), f.rows...), nil
}

func (f *fakeOrphanTable) Identity(pid int) string { return f.ids[pid] }

func (f *fakeOrphanTable) Command(int) ([]string, bool) { return nil, false }

func (f *fakeOrphanTable) Signal(pid int, sig syscall.Signal) error {
	if _, alive := f.ids[pid]; !alive {
		return fmt.Errorf("no such pid %d", pid)
	}
	if f.signals == nil {
		f.signals = map[int][]syscall.Signal{}
	}
	f.signals[pid] = append(f.signals[pid], sig)
	if fatal, ok := f.diesOn[pid]; ok && fatal == sig {
		delete(f.ids, pid)
		rows := f.rows[:0]
		for _, row := range f.rows {
			if row.PID != pid {
				rows = append(rows, row)
			}
		}
		f.rows = rows
	}
	return nil
}

func fastOrphanReaper(table launcher.ProcessTable) launcher.Reaper {
	return launcher.Reaper{Table: table, TermWait: time.Millisecond, KillWait: time.Millisecond, Poll: time.Millisecond}
}

func orphanRow(pid int, identity string) couchcore.OrphanNvim {
	rss := uint64(22 << 30)
	return couchcore.OrphanNvim{
		Key:      couchcore.ResourceProcessKey{PID: pid, Identity: identity},
		Age:      13 * 24 * time.Hour,
		Cwd:      "/fleet/worktree/pair-slot1/pair/cmd",
		Root:     "/fleet/worktree/pair-slot1/pair",
		RSSBytes: &rss,
		Reason:   "unattributed embedded nvim",
	}
}

// The verb takes no slot -- the sweep it confirms is fleet-wide -- so its whole
// surface is the two switches, each once (#583).
func TestReapOrphansParsesConfirmationAndJSONOnly(t *testing.T) {
	for _, c := range []struct {
		args              []string
		confirmed, asJSON bool
	}{
		{[]string{"--reap-orphans"}, false, false},
		{[]string{"--reap-orphans", "--confirm"}, true, false},
		{[]string{"--reap-orphans", "--json"}, false, true},
		{[]string{"--reap-orphans", "--confirm", "--json"}, true, true},
		{[]string{"--reap-orphans", "--json", "--confirm"}, true, true},
	} {
		inv, err := ParseCLI(c.args, nil)
		if err != nil || inv.messageOp != orphanReapOp || inv.confirmed != c.confirmed || inv.jsonOutput != c.asJSON {
			t.Fatalf("%v: %+v %v", c.args, inv, err)
		}
	}
	for _, args := range [][]string{
		{"--reap-orphans", "pair:1"},
		{"--reap-orphans", "--confirm", "--confirm"},
		{"--reap-orphans", "--json", "--json"},
		{"--reap-orphans", "--force-unknown"},
		{"--reap-orphans", "--confirm", "extra"},
	} {
		if inv, err := ParseCLI(args, nil); err == nil {
			t.Fatalf("accepted %v: %+v", args, inv)
		}
	}
}

// The CLI's --confirm is presentation. A request that reaches the supervisor
// without it is refused there, so a forged or hand-written call cannot reap.
func TestReapOrphansConfirmationIsAProtocolGate(t *testing.T) {
	if err := couchmessage.ValidateRequest(couchmessage.Request{Op: orphanReapOp}); err == nil {
		t.Fatal("an unconfirmed reap passed validation")
	}
	if err := couchmessage.ValidateRequest(couchmessage.Request{Op: orphanReapOp, Confirmed: true}); err != nil {
		t.Fatal(err)
	}
	typ := reflect.TypeOf(couchmessage.Request{})
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if field.Name == "Op" || field.Name == "Confirmed" {
			continue
		}
		t.Run(field.Name, func(t *testing.T) {
			r := couchmessage.Request{Op: orphanReapOp, Confirmed: true}
			v := reflect.ValueOf(&r).Elem().Field(i)
			switch v.Kind() {
			case reflect.String:
				v.SetString("extra")
			case reflect.Bool:
				v.SetBool(true)
			case reflect.Int:
				v.SetInt(1)
			case reflect.Pointer:
				v.Set(reflect.New(v.Type().Elem()))
			default:
				t.Fatalf("unhandled request field %s", field.Name)
			}
			if err := couchmessage.ValidateRequest(r); err == nil {
				t.Fatalf("accepted %s", field.Name)
			}
		})
	}
}

// An unchanged world is signalled exactly once; a pid that no longer holds the
// identity the sweep named is not signalled at all, and the operator is told why
// rather than left to infer it from a count.
func TestReapOrphanNvimFencesEveryRowOnItsIdentity(t *testing.T) {
	cases := []struct {
		name      string
		table     *fakeOrphanTable
		wantState []syscall.Signal
		status    string
	}{
		{
			name:      "clean exit on SIGTERM",
			table:     &fakeOrphanTable{rows: []launcher.ProcessRow{{PID: 4242, PPID: 1, Identity: "gen-1", Name: "nvim"}}, ids: map[int]string{4242: "gen-1"}, diesOn: map[int]syscall.Signal{4242: syscall.SIGTERM}},
			wantState: []syscall.Signal{syscall.SIGTERM},
			status:    orphanReaped,
		},
		{
			name:      "a survivor escalates to SIGKILL",
			table:     &fakeOrphanTable{rows: []launcher.ProcessRow{{PID: 4242, PPID: 1, Identity: "gen-1", Name: "nvim"}}, ids: map[int]string{4242: "gen-1"}, diesOn: map[int]syscall.Signal{4242: syscall.SIGKILL}},
			wantState: []syscall.Signal{syscall.SIGTERM, syscall.SIGKILL},
			status:    orphanReaped,
		},
		{
			name:      "the pid was recycled into another generation",
			table:     &fakeOrphanTable{rows: []launcher.ProcessRow{{PID: 4242, PPID: 1, Identity: "gen-2", Name: "nvim"}}, ids: map[int]string{4242: "gen-2"}},
			wantState: nil,
			status:    orphanSkipped,
		},
		{
			name:      "the process already exited",
			table:     &fakeOrphanTable{},
			wantState: nil,
			status:    orphanSkipped,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			outcomes := reapOrphanNvim(context.Background(), fastOrphanReaper(c.table), []couchcore.OrphanNvim{orphanRow(4242, "gen-1")})
			if len(outcomes) != 1 || outcomes[0].Status != c.status {
				t.Fatalf("outcomes %+v", outcomes)
			}
			if got := c.table.signals[4242]; !reflect.DeepEqual(got, c.wantState) {
				t.Fatalf("signals %v, want %v", got, c.wantState)
			}
			if c.status == orphanSkipped && outcomes[0].Detail == "" {
				t.Fatal("a skip owed the operator no reason")
			}
		})
	}
}

// A descendant of the leak is part of the leak: killing only the editor reparents
// its language servers to launchd and manufactures the next #583.
func TestReapOrphanNvimEndsTheWholeTree(t *testing.T) {
	table := &fakeOrphanTable{
		rows: []launcher.ProcessRow{
			{PID: 4242, PPID: 1, Identity: "gen-1", Name: "nvim"},
			{PID: 4300, PPID: 4242, Identity: "gen-3", Name: "lua-language-server"},
		},
		ids:    map[int]string{4242: "gen-1", 4300: "gen-3"},
		diesOn: map[int]syscall.Signal{4242: syscall.SIGTERM, 4300: syscall.SIGTERM},
	}
	outcomes := reapOrphanNvim(context.Background(), fastOrphanReaper(table), []couchcore.OrphanNvim{orphanRow(4242, "gen-1")})
	if outcomes[0].Status != orphanReaped {
		t.Fatalf("%+v", outcomes)
	}
	for _, pid := range []int{4242, 4300} {
		if len(table.signals[pid]) != 1 || table.signals[pid][0] != syscall.SIGTERM {
			t.Fatalf("pid %d signalled %v", pid, table.signals[pid])
		}
	}
}

// A snapshot the host refuses is a failed reap, never a silent one: the operator
// confirmed a kill and is owed the outcome of every row they confirmed.
func TestReapOrphanNvimReportsARefusedSnapshot(t *testing.T) {
	table := &fakeOrphanTable{snapErr: fmt.Errorf("sysctl denied")}
	outcomes := reapOrphanNvim(context.Background(), fastOrphanReaper(table), []couchcore.OrphanNvim{orphanRow(4242, "gen-1")})
	if outcomes[0].Status != orphanFailed || !strings.Contains(outcomes[0].Detail, "sysctl denied") {
		t.Fatalf("%+v", outcomes)
	}
	if len(table.signals) != 0 {
		t.Fatal("signalled without a snapshot")
	}
}

func reapCLIStub(t *testing.T, view couchcore.ResourceView, sent *[]couchmessage.Request) messageCall {
	t.Helper()
	return func(_ context.Context, _ string, req any, resp any) error {
		r, ok := req.(couchmessage.Request)
		if !ok {
			t.Fatalf("request %T", req)
		}
		*sent = append(*sent, r)
		out, ok := resp.(*couchmessage.Response)
		if !ok {
			t.Fatalf("response %T", resp)
		}
		v := view
		out.Code, out.Resources = "ok", &v
		return nil
	}
}

// Without --confirm the verb is a report: it names what a reap would end and
// signals nothing. An operator who has not decided yet must be able to look
// (#583's Done-when).
func TestReapOrphansCLIWithoutConfirmationNamesAndRefuses(t *testing.T) {
	table := &fakeOrphanTable{rows: []launcher.ProcessRow{{PID: 4242, PPID: 1, Identity: "gen-1", Name: "nvim"}}, ids: map[int]string{4242: "gen-1"}}
	var sent []couchmessage.Request
	view := couchcore.ResourceView{SampledAt: time.Unix(1760100000, 0), OrphanNvim: []couchcore.OrphanNvim{orphanRow(4242, "gen-1"), orphanRow(4300, "gen-2")}}
	var out, errw bytes.Buffer
	code := runReapOrphansCLI(cliInvocation{messageOp: orphanReapOp}, testRT{dir: t.TempDir()}, &out, &errw, reapCLIStub(t, view, &sent), fastOrphanReaper(table))
	if code == 0 {
		t.Fatal("an unconfirmed reap succeeded")
	}
	if len(sent) != 1 || sent[0].Op != "resources" || sent[0].Confirmed {
		t.Fatalf("an unconfirmed reap asked for %v", sent)
	}
	if len(table.signals) != 0 {
		t.Fatalf("signalled %v without confirmation", table.signals)
	}
	text := out.String() + errw.String()
	for _, want := range []string{"--confirm", orphanNvimSection, "pid=4242", "pid=4300"} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q in:\n%s", want, text)
		}
	}
}

func TestReapOrphansCLIWithConfirmationReapsAndReports(t *testing.T) {
	table := &fakeOrphanTable{
		rows: []launcher.ProcessRow{
			{PID: 4242, PPID: 1, Identity: "gen-1", Name: "nvim"},
			{PID: 4300, PPID: 1, Identity: "gen-9", Name: "nvim"},
		},
		ids:    map[int]string{4242: "gen-1", 4300: "gen-9"},
		diesOn: map[int]syscall.Signal{4242: syscall.SIGTERM},
	}
	var sent []couchmessage.Request
	view := couchcore.ResourceView{SampledAt: time.Unix(1760100000, 0), OrphanNvim: []couchcore.OrphanNvim{orphanRow(4242, "gen-1"), orphanRow(4300, "gen-2")}}
	var out, errw bytes.Buffer
	code := runReapOrphansCLI(cliInvocation{messageOp: orphanReapOp, confirmed: true}, testRT{dir: t.TempDir()}, &out, &errw, reapCLIStub(t, view, &sent), fastOrphanReaper(table))
	if len(sent) != 1 || sent[0].Op != orphanReapOp || !sent[0].Confirmed {
		t.Fatalf("confirmed reap sent %v", sent)
	}
	if len(table.signals[4242]) != 1 || table.signals[4242][0] != syscall.SIGTERM {
		t.Fatalf("the named orphan was not ended: %v", table.signals)
	}
	if len(table.signals[4300]) != 0 {
		t.Fatalf("a recycled pid was signalled: %v", table.signals[4300])
	}
	text := out.String() + errw.String()
	for _, want := range []string{"reaped pid=4242", "skipped pid=4300"} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q in:\n%s", want, text)
		}
	}
	// A skip is the world having moved, not a failure of the verb; only a
	// process that survived SIGKILL is.
	if code != 0 {
		t.Fatalf("exit %d on a skipped row: %s", code, text)
	}
}

func TestReapOrphansCLIFailsARowThatSurvivedSigkill(t *testing.T) {
	table := &fakeOrphanTable{
		rows:   []launcher.ProcessRow{{PID: 4242, PPID: 1, Identity: "gen-1", Name: "nvim"}},
		ids:    map[int]string{4242: "gen-1"},
		diesOn: map[int]syscall.Signal{},
	}
	var sent []couchmessage.Request
	view := couchcore.ResourceView{OrphanNvim: []couchcore.OrphanNvim{orphanRow(4242, "gen-1")}}
	var out, errw bytes.Buffer
	code := runReapOrphansCLI(cliInvocation{messageOp: orphanReapOp, confirmed: true}, testRT{dir: t.TempDir()}, &out, &errw, reapCLIStub(t, view, &sent), fastOrphanReaper(table))
	if code == 0 {
		t.Fatal("a survivor exited 0")
	}
	if !strings.Contains(out.String()+errw.String(), "failed pid=4242") {
		t.Fatalf("no failure named: %s%s", out.String(), errw.String())
	}
}

func TestReapOrphansCLISaysSoWhenThereIsNothingToReap(t *testing.T) {
	table := &fakeOrphanTable{}
	var sent []couchmessage.Request
	var out, errw bytes.Buffer
	code := runReapOrphansCLI(cliInvocation{messageOp: orphanReapOp, confirmed: true}, testRT{dir: t.TempDir()}, &out, &errw, reapCLIStub(t, couchcore.ResourceView{}, &sent), fastOrphanReaper(table))
	if code != 0 {
		t.Fatalf("exit %d: %s", code, errw.String())
	}
	if !strings.Contains(out.String(), "nothing to reap") {
		t.Fatalf("an empty sweep did not say it looked: %q", out.String())
	}
}

func TestReapOrphansCLIJSONCarriesEveryOutcome(t *testing.T) {
	table := &fakeOrphanTable{
		rows:   []launcher.ProcessRow{{PID: 4242, PPID: 1, Identity: "gen-1", Name: "nvim"}},
		ids:    map[int]string{4242: "gen-1"},
		diesOn: map[int]syscall.Signal{4242: syscall.SIGTERM},
	}
	var sent []couchmessage.Request
	view := couchcore.ResourceView{SampledAt: time.Unix(1760100000, 0), OrphanNvim: []couchcore.OrphanNvim{orphanRow(4242, "gen-1")}}
	var out, errw bytes.Buffer
	if code := runReapOrphansCLI(cliInvocation{messageOp: orphanReapOp, confirmed: true, jsonOutput: true}, testRT{dir: t.TempDir()}, &out, &errw, reapCLIStub(t, view, &sent), fastOrphanReaper(table)); code != 0 {
		t.Fatalf("exit %d: %s", code, errw.String())
	}
	var report orphanReapReport
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatalf("%v: %s", err, out.String())
	}
	if len(report.Outcomes) != 1 || report.Outcomes[0].Status != orphanReaped || report.Outcomes[0].Row.Key.PID != 4242 {
		t.Fatalf("%+v", report)
	}
	if !report.SampledAt.Equal(time.Unix(1760100000, 0)) {
		t.Fatalf("the report does not say which pass classified it: %v", report.SampledAt)
	}
}

// A supervisor that refuses is reported, not retried: the verb is destructive
// and an operator who cannot tell "nothing matched" from "Couch said no" will
// send it again.
func TestReapOrphansCLIReportsARefusedSweep(t *testing.T) {
	table := &fakeOrphanTable{rows: []launcher.ProcessRow{{PID: 4242, PPID: 1, Identity: "gen-1", Name: "nvim"}}, ids: map[int]string{4242: "gen-1"}}
	call := func(_ context.Context, _ string, _ any, resp any) error {
		out := resp.(*couchmessage.Response)
		out.Code, out.Error = "invalid-request", "reap-orphans ends processes and requires --confirm"
		return nil
	}
	var out, errw bytes.Buffer
	if code := runReapOrphansCLI(cliInvocation{messageOp: orphanReapOp, confirmed: true}, testRT{dir: t.TempDir()}, &out, &errw, call, fastOrphanReaper(table)); code == 0 {
		t.Fatal("a refusal exited 0")
	}
	if len(table.signals) != 0 {
		t.Fatal("signalled after a refusal")
	}
	if !strings.Contains(errw.String(), "--confirm") {
		t.Fatalf("the refusal lost its reason: %q", errw.String())
	}
}

// The plan a kill is decided from comes from a pass the worker ran for the
// request, not from the report the last cadence left behind: the world is
// allowed to have moved in the minute since (#583).
func TestReapOrphansSupervisorClassifiesFresh(t *testing.T) {
	f := &samplerReader{rows: []procutil.Process{{PID: 10, Identity: "owned"}}}
	s := testResourceSampler(f)
	s.sweeps = make(chan chan resourceSweep)
	s.policy.Cadence = time.Hour // the answer must not arrive on the cadence
	sampler := &messageService{}
	sampler.resources.Store(s)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() { s.run(ctx); close(done) }()
	waitFor(t, func() bool { return len(s.view().Processes) == 1 })

	cached := sampler.handle(ctx, couchmessage.Request{Op: "resources"})
	if cached.Code != "ok" || len(cached.Resources.Processes) != 1 {
		t.Fatalf("cached report %+v", cached)
	}
	if refused := sampler.handle(ctx, couchmessage.Request{Op: orphanReapOp}); refused.Code != "invalid-request" {
		t.Fatalf("an unconfirmed reap was answered %s", refused.Code)
	}
	f.setRows([]procutil.Process{{PID: 10, Identity: "owned"}, {PID: 11, PPID: 10, Identity: "child", Name: "sdlc"}})
	fresh := sampler.handle(ctx, couchmessage.Request{Op: orphanReapOp, Confirmed: true})
	if fresh.Code != "ok" || fresh.Resources == nil {
		t.Fatalf("confirmed reap answered %s %s", fresh.Code, fresh.Error)
	}
	if len(fresh.Resources.Processes) != 2 {
		t.Fatalf("the sweep answered from a stale pass: %+v", fresh.Resources.Processes)
	}
	cancel()
	<-done
}

// A pass the sampler's lifetime cut short leaves the cache holding an older
// view. Reporting that as a classification is how a reap would kill from a
// minute-old decision, so it is an error instead.
func TestResourcePassReportsWhetherItClassifiedAnything(t *testing.T) {
	f := &samplerReader{rows: []procutil.Process{{PID: 10, Identity: "owned"}}}
	s := testResourceSampler(f)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if s.pass(ctx) {
		t.Fatal("an aborted pass reported itself fresh")
	}
	if !s.pass(context.Background()) {
		t.Fatal("a completed pass reported itself stale")
	}
}

// A sweep nobody is running must fail loudly. Waiting on a worker that has
// joined would hang the verb until its own deadline and then report a timeout
// that says nothing about the real cause.
func TestResourceSweepWithoutAWorkerFails(t *testing.T) {
	s := testResourceSampler(&samplerReader{})
	if _, err := s.sweep(context.Background()); err == nil {
		t.Fatal("a sweep with no worker succeeded")
	}
	s.sweeps = make(chan chan resourceSweep)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	if _, err := s.sweep(ctx); err == nil {
		t.Fatal("a sweep nobody answered succeeded")
	}
}

func waitFor(t *testing.T, ok func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if ok() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("condition never held")
}

// The registry's left-running rows ride the same verb (pair#695): named and
// refused without --confirm, ended by exact identity with it, and a recycled
// pid is skipped, never signalled.
func TestReapOrphansCLIEndsLeftRunningByExactIdentity(t *testing.T) {
	entry := func(pid int, identity string) couchcore.LeftRunningOrphan {
		return couchcore.LeftRunningOrphan{Origin: "slot:/w/pair", Thread: couchcore.ThreadAddress{RepoScope: "s", Tag: "couch-1"}, Process: couchcore.ProcessIdentity{PID: pid, Identity: identity}, Command: "sdlc", LeftAt: time.Unix(1760100000, 0)}
	}
	view := couchcore.ResourceView{SampledAt: time.Unix(1760100000, 0), LeftRunning: []couchcore.LeftRunningOrphan{entry(5100, "gen-1"), entry(5200, "gen-2")}}
	newTable := func() *fakeOrphanTable {
		return &fakeOrphanTable{
			rows:   []launcher.ProcessRow{{PID: 5100, PPID: 1, Identity: "gen-1", Name: "sdlc"}, {PID: 5200, PPID: 1, Identity: "gen-9", Name: "sdlc"}},
			ids:    map[int]string{5100: "gen-1", 5200: "gen-9"},
			diesOn: map[int]syscall.Signal{5100: syscall.SIGTERM},
		}
	}

	table := newTable()
	var sent []couchmessage.Request
	var out, errw bytes.Buffer
	if code := runReapOrphansCLI(cliInvocation{messageOp: orphanReapOp}, testRT{dir: t.TempDir()}, &out, &errw, reapCLIStub(t, view, &sent), fastOrphanReaper(table)); code == 0 || len(table.signals) != 0 {
		t.Fatalf("unconfirmed: exit %d, signals %v", code, table.signals)
	}
	if text := out.String(); !strings.Contains(text, leftRunningSection) || !strings.Contains(text, "pid=5100 sdlc thread=s/couch-1") {
		t.Fatalf("unconfirmed report:\n%s", text)
	}

	table = newTable()
	sent, out, errw = nil, bytes.Buffer{}, bytes.Buffer{}
	code := runReapOrphansCLI(cliInvocation{messageOp: orphanReapOp, confirmed: true}, testRT{dir: t.TempDir()}, &out, &errw, reapCLIStub(t, view, &sent), fastOrphanReaper(table))
	if len(table.signals[5100]) != 1 || len(table.signals[5200]) != 0 {
		t.Fatalf("signals %v, want only the exact identity ended", table.signals)
	}
	text := out.String()
	for _, want := range []string{"reaped left-running pid=5100", "skipped left-running pid=5200"} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q in:\n%s", want, text)
		}
	}
	if code != 0 {
		t.Fatalf("exit %d: %s", code, text)
	}
}
