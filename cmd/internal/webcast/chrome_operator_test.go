package webcast

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/webcast/board"
	"github.com/xianxu/pair/cmd/internal/webcast/notes"
)

func TestOperatorTargetGuardSurvivesAnUncertainInput(t *testing.T) {
	c, p := startPeer(t, nil)
	_ = p
	if err := c.uiGuard("S1", "acquire"); err != nil {
		t.Fatal(err)
	}
	if err := c.uiGuard("S1", "unknown"); err != nil {
		t.Fatal(err)
	}
	if err := c.uiGuard("S1", "acquire"); err == nil {
		t.Fatal("uncertain input allowed another batch")
	}
	waitState(t, c, "recovery instruction", func(s State) bool { return s.Recovery != "" })
}

func TestOperatorModeRejectsInvalidOrUnavailableWithoutFocus(t *testing.T) {
	c, _ := startPeer(t, nil)
	if err := c.SetAnnotationMode("bogus"); err == nil {
		t.Fatal("invalid mode accepted")
	}
	if err := c.SetAnnotationMode("point"); err == nil {
		t.Fatal("unrecorded annotation accepted")
	}
}

func TestOperatorWorkerDeliversOwnerAvailabilityWithViewersOffToExactContext(t *testing.T) {
	c, p := startPeer(t, nil)
	b := board.New(notes.Structural{}, nil)
	defer b.Close()
	done := make(chan struct{})
	c.do(func(s *loopState) {
		s.contexts["S1"] = 17
		s.sessions["S1"] = "T1"
		s.tracker.Apply(TargetCreated{ID: "T1"})
		close(done)
	})
	<-done
	c.startOperator(b)
	call := p.expect("S1", "Runtime.evaluate")
	if !strings.Contains(string(call.Params), "hook(true,") || !strings.Contains(string(call.Params), `"contextId":17`) {
		t.Fatalf("missing exact-context permission: %s", call.Params)
	}
}

func TestOperatorPermissionReceiptRejectsLateAndRepeatedDelivery(t *testing.T) {
	on := operatorPermissionExpression(9007199254740993, 9007199254740993, true)
	off := operatorPermissionExpression(9007199254740994, 9007199254740994, false)
	newer := operatorPermissionExpression(9007199254740995, 9007199254740995, true)
	runOperatorJS(t, `let observed;globalThis.__couchcastAnnotationAllowed=(allowed,generation)=>observed=generation;`+on+`;if(observed!=='9007199254740993'||globalThis.__couchcastAnnotationDelivery.generation!==observed)throw Error('authority generation precision lost');`)
	runOperatorJS(t, `let calls=[];globalThis.__couchcastAnnotationAllowed=v=>calls.push(v);`+on+`;`+off+`;`+on+`;`+off+`;if(JSON.stringify(calls)!=='[true,false]')throw Error('late or repeated delivery changed permission');`+newer+`;if(JSON.stringify(calls)!=='[true,false,true]')throw Error('new permission missing');`)
	runOperatorJS(t, on+`;if(globalThis.__couchcastAnnotationDelivery)throw Error('absent hook consumed revision');let calls=[];globalThis.__couchcastAnnotationAllowed=v=>calls.push(v);`+on+`;if(JSON.stringify(calls)!=='[true]')throw Error('ready retry missing');`)
}

func TestOperatorPermissionTargetSwitchAndCoalescedGeneration(t *testing.T) {
	var revision uint64
	next := func() uint64 { revision++; return revision }
	p := &operatorPermission{}
	p.want(1, true, true, next)
	if p.revision != 1 || !p.allowed() {
		t.Fatal(p)
	}
	p.phase = operatorPermissionReady
	p.want(1, true, true, next)
	if p.revision != 1 || p.phase != operatorPermissionReady {
		t.Fatal("view refresh reset permission")
	}
	p.want(1, false, false, next)
	if p.revision != 2 || p.allowed() {
		t.Fatal("same-generation target departure ignored")
	}
	p.want(1, true, true, next)
	if p.revision != 3 || !p.allowed() {
		t.Fatal("same-generation target return ignored")
	}
	p.want(3, true, true, next)
	if p.revision != 4 {
		t.Fatal("coalesced Off/On ignored")
	}
}

func TestOperatorPermissionUncertainEnableFencesBeforeRetry(t *testing.T) {
	c, p := startPeer(t, func(p *peer) { p.ignore["Runtime.evaluate"] = true })
	var revision uint64
	next := func() uint64 { revision++; return revision }
	permission := &operatorPermission{}
	permission.want(1, true, true, next)
	target := operatorTarget{session: "S1", context: 17, active: true}
	deliver := func() chan struct{} {
		done := make(chan struct{})
		go func() { c.deliverOperatorPermission(target, permission, next); close(done) }()
		return done
	}
	done := deliver()
	first := p.expect("S1", "Runtime.evaluate")
	p.send(fmt.Sprintf(`{"id":%d,"error":{"code":-32000,"message":"execution result lost"}}`, first.ID))
	<-done
	if permission.phase != operatorPermissionFencing || permission.allowed() || permission.revision != 2 {
		t.Fatalf("enable uncertainty not fenced: %+v", permission)
	}
	done = deliver()
	fence := p.expect("S1", "Runtime.evaluate")
	if !strings.Contains(string(fence.Params), "hook(false,") {
		t.Fatalf("not a false fence: %s", fence.Params)
	}
	p.send(fmt.Sprintf(`{"id":%d,"result":{"result":{"value":{"revision":"2","generation":"1","allowed":false}}}}`, fence.ID))
	<-done
	if permission.phase != operatorPermissionPending || permission.revision != 3 || !permission.allowed() {
		t.Fatalf("acknowledged fence did not allow fresh enable: %+v", permission)
	}
	done = deliver()
	last := p.expect("S1", "Runtime.evaluate")
	p.send(fmt.Sprintf(`{"id":%d,"result":{"result":{"value":{"revision":"3","generation":"1","allowed":true}}}}`, last.ID))
	<-done
	if permission.phase != operatorPermissionReady {
		t.Fatal("receipt not accepted")
	}
	done = deliver()
	<-done
	p.never("Runtime.evaluate", 20*time.Millisecond)
}

func TestOperatorPermissionMissingHookDoesNotAcknowledge(t *testing.T) {
	c, p := startPeer(t, func(p *peer) { p.ignore["Runtime.evaluate"] = true })
	var revision uint64
	next := func() uint64 { revision++; return revision }
	permission := &operatorPermission{}
	permission.want(1, true, true, next)
	done := make(chan struct{})
	go func() {
		c.deliverOperatorPermission(operatorTarget{session: "S1", context: 17}, permission, next)
		close(done)
	}()
	call := p.expect("S1", "Runtime.evaluate")
	p.send(fmt.Sprintf(`{"id":%d,"result":{"result":{"value":null}}}`, call.ID))
	<-done
	if permission.phase != operatorPermissionPending || permission.revision != 1 {
		t.Fatal("missing hook consumed permission")
	}
}

func TestOperatorModeChecksClosedBoardAuthorityBeforeBrowserDispatch(t *testing.T) {
	c, p := startPeer(t, nil)
	b := board.New(notes.Structural{}, nil)
	defer b.Close()
	c.opts.Record = &RecordOptions{Operator: b}
	b.Close()
	done := make(chan struct{})
	c.do(func(s *loopState) {
		s.contexts["S1"] = 17
		s.sessions["S1"] = "T1"
		s.tracker.Apply(TargetCreated{ID: "T1"})
		close(done)
	})
	<-done
	if err := c.SetAnnotationMode("note"); !errors.Is(err, board.ErrClosed) {
		t.Fatalf("off mode: %v", err)
	}
	p.never("Runtime.evaluate", 20*time.Millisecond)
}

func TestOperatorWorkerReplacementDoesNotInheritDelayedEnable(t *testing.T) {
	c, p := startPeer(t, func(p *peer) { p.ignore["Runtime.evaluate"] = true })
	b := board.New(notes.Structural{}, nil).WithAnnotationCleanup(func(context.Context, uint64) error { return nil })
	defer b.Close()
	if err := b.SetAnnotationAllowed(context.Background(), true); err != nil {
		t.Fatal(err)
	}
	update := func(id int64) {
		done := make(chan struct{})
		c.do(func(s *loopState) {
			s.contexts["S1"] = id
			s.sessions["S1"] = "T1"
			s.tracker.Apply(TargetCreated{ID: "T1"})
			close(done)
		})
		<-done
	}
	update(17)
	c.startOperator(b)
	first := p.expect("S1", "Runtime.evaluate")
	var before struct {
		Context    int64  `json:"contextId"`
		Expression string `json:"expression"`
	}
	if err := json.Unmarshal(first.Params, &before); err != nil {
		t.Fatal(err)
	}
	if before.Context != 17 || !strings.Contains(before.Expression, "hook(true,") {
		t.Fatalf("initial permission: %s", first.Params)
	}
	if err := b.SetAnnotationAllowed(context.Background(), false); err != nil {
		t.Fatal(err)
	}
	update(18)
	c.refreshOperator()
	// The old enable settles only after its context was replaced and Off won.
	p.send(fmt.Sprintf(`{"id":%d,"result":{"result":{"value":{"revision":"1","generation":"1","allowed":true}}}}`, first.ID))
	for {
		call := p.expect("S1", "Runtime.evaluate")
		var params struct {
			Context    int64  `json:"contextId"`
			Expression string `json:"expression"`
		}
		if err := json.Unmarshal(call.Params, &params); err != nil {
			t.Fatal(err)
		}
		if strings.Contains(params.Expression, "__couchcastAnnotationAllowed") {
			if params.Context != 18 || !strings.Contains(params.Expression, "hook(true,") {
				t.Fatalf("replacement missing fresh owner permission: %s", call.Params)
			}
			p.send(fmt.Sprintf(`{"id":%d,"result":{"result":{"value":{"revision":"2","generation":"2","allowed":true}}}}`, call.ID))
			break
		}
		p.send(fmt.Sprintf(`{"id":%d,"result":{}}`, call.ID))
	}
	// Release the ordinary view call so test shutdown does not wait its timeout.
	call := p.expect("S1", "Runtime.evaluate")
	p.send(fmt.Sprintf(`{"id":%d,"result":{}}`, call.ID))
}

func TestOperatorModeUsesOwnerAuthorityWithViewersOff(t *testing.T) {
	c, p := startPeer(t, func(p *peer) { p.ignore["Runtime.evaluate"] = true })
	b := board.New(notes.Structural{}, nil)
	defer b.Close()
	c.opts.Record = &RecordOptions{Operator: b}
	done := make(chan struct{})
	c.do(func(s *loopState) {
		s.contexts["S1"] = 17
		s.sessions["S1"] = "T1"
		s.tracker.Apply(TargetCreated{ID: "T1"})
		close(done)
	})
	<-done
	result := make(chan error, 1)
	go func() { result <- c.SetAnnotationMode("note") }()
	call := p.expect("S1", "Runtime.evaluate")
	if !strings.Contains(string(call.Params), `"contextId":17`) {
		t.Fatalf("wrong context: %s", call.Params)
	}
	p.send(fmt.Sprintf(`{"id":%d,"result":{"result":{"value":true}}}`, call.ID))
	if err := <-result; err != nil {
		t.Fatal(err)
	}
	if _, err := b.AdmitAnnotation(); !errors.Is(err, board.ErrDisabled) {
		t.Fatalf("viewer acquired authority: %v", err)
	}
}
