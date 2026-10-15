package launcher

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/xianxu/pair/cmd/internal/storagegc"
)

type admissionWorld struct {
	mu         sync.Mutex
	selfID     storagegc.ProcessIdentity
	parents    map[int]int
	identities map[int]string
	lives      map[int]storagegc.Liveness
}

func newAdmissionWorld() *admissionWorld {
	return &admissionWorld{selfID: storagegc.ProcessIdentity{PID: 7, Birth: "helper"}, parents: map[int]int{7: 1, 12: 1, 13: 12, 22: 1, 23: 22}, identities: map[int]string{7: "helper", 12: "wrapper", 13: "child", 22: "wrapper2", 23: "child2"}, lives: map[int]storagegc.Liveness{7: storagegc.ProcessAlive, 12: storagegc.ProcessAlive, 13: storagegc.ProcessAlive, 22: storagegc.ProcessAlive, 23: storagegc.ProcessAlive}}
}
func (w *admissionWorld) process(pid int) (storagegc.ProcessIdentity, int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.lives[pid] != storagegc.ProcessAlive {
		return storagegc.ProcessIdentity{}, 0, errors.New("process dead or unknown")
	}
	birth := w.identities[pid]
	if birth == "" {
		return storagegc.ProcessIdentity{}, 0, errors.New("unknown identity")
	}
	return storagegc.ProcessIdentity{PID: pid, Birth: birth}, w.parents[pid], nil
}
func (w *admissionWorld) as(pid int) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.selfID = storagegc.ProcessIdentity{PID: pid, Birth: w.identities[pid]}
}
func (w *admissionWorld) store(home string) NativeAdmissionStore {
	return NativeAdmissionStore{Home: home, io: &nativeAdmissionIO{self: func() (storagegc.ProcessIdentity, error) { w.mu.Lock(); defer w.mu.Unlock(); return w.selfID, nil }, child: w.process, inspect: func(p storagegc.ProcessIdentity) storagegc.Liveness {
		w.mu.Lock()
		defer w.mu.Unlock()
		if w.identities[p.PID] != "" && w.identities[p.PID] != p.Birth {
			return storagegc.ProcessDead
		}
		return w.lives[p.PID]
	}}}
}
func admissionRequest(t *testing.T, id string) NativeAdmissionRequest {
	t.Helper()
	d, e := NativeAdmissionCommandDigest("codex", []string{"codex", "resume", id})
	if e != nil {
		t.Fatal(e)
	}
	return NativeAdmissionRequest{Agent: "codex", NativeID: id, Attempt: "attempt-1", CommandDigest: d}
}
func noLegacy(context.Context, NativeAdmissionOwners) error { return nil }
func reserveAdmission(t *testing.T, s NativeAdmissionStore, r NativeAdmissionRequest) string {
	t.Helper()
	g, e := s.Reserve(context.Background(), r, noLegacy)
	if e != nil {
		t.Fatal(e)
	}
	return g
}
func TestNativeAdmissionSingleConsumptionAndExactPublication(t *testing.T) {
	w := newAdmissionWorld()
	s := w.store(t.TempDir())
	r := admissionRequest(t, "native")
	g := reserveAdmission(t, s, r)
	w.as(12)
	if _, e := s.Claim(context.Background(), g, "claude", r.CommandDigest); e == nil {
		t.Fatal("changed agent claimed")
	}
	if _, e := s.Claim(context.Background(), g, "codex", strings.Repeat("a", 64)); e == nil {
		t.Fatal("changed command claimed")
	}
	c, e := s.Claim(context.Background(), g, r.Agent, r.CommandDigest)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.Claim(context.Background(), g, r.Agent, r.CommandDigest); e == nil {
		t.Fatal("grant replay claimed")
	}
	if e = s.PublishChild(context.Background(), c, 23); e == nil {
		t.Fatal("foreign child published")
	}
	if e = s.PublishChild(context.Background(), c, 13); e != nil {
		t.Fatal(e)
	}
	if e = s.PublishChild(context.Background(), c, 13); e != nil {
		t.Fatal("identical publication", e)
	}
	w.as(22)
	if e = s.PublishChild(context.Background(), c, 23); e == nil {
		t.Fatal("different wrapper completed claim")
	}
	if e = s.CancelBeforeHandoff(context.Background(), g); e == nil {
		t.Fatal("running admission canceled")
	}
}
func TestNativeAdmissionLifetimeNeverExpiresOrReleasesOnReadiness(t *testing.T) {
	for _, phase := range []string{"reserved-helper-dead", "spawning-wrapper-dead", "running", "running-wrapper-dead", "running-child-dead", "running-reparented-child", "running-unknown", "running-both-dead"} {
		t.Run(phase, func(t *testing.T) {
			w := newAdmissionWorld()
			s := w.store(t.TempDir())
			r := admissionRequest(t, "native")
			g := reserveAdmission(t, s, r)
			if !strings.HasPrefix(phase, "reserved") {
				w.as(12)
				c, e := s.Claim(context.Background(), g, r.Agent, r.CommandDigest)
				if e != nil {
					t.Fatal(e)
				}
				if strings.HasPrefix(phase, "running") {
					if e = s.PublishChild(context.Background(), c, 13); e != nil {
						t.Fatal(e)
					}
				}
			}
			switch phase {
			case "reserved-helper-dead":
				w.lives[7] = storagegc.ProcessDead
			case "spawning-wrapper-dead", "running-wrapper-dead":
				w.lives[12] = storagegc.ProcessDead
			case "running-child-dead":
				w.lives[13] = storagegc.ProcessDead
			case "running-reparented-child":
				w.lives[12] = storagegc.ProcessDead
				w.parents[13] = 1
			case "running-unknown":
				w.lives[12] = storagegc.ProcessDead
				w.lives[13] = storagegc.ProcessUnknown
			case "running-both-dead":
				w.lives[12] = storagegc.ProcessDead
				w.lives[13] = storagegc.ProcessDead
			}
			w.as(7)
			r.Attempt = "attempt-2"
			_, e := s.Reserve(context.Background(), r, noLegacy)
			if (e == nil) != (phase == "running-both-dead") {
				t.Fatalf("phase=%s admission err=%v", phase, e)
			}
			other := r
			other.NativeID = "other"
			if _, e = s.Reserve(context.Background(), other, noLegacy); e != nil {
				t.Fatal("unrelated key blocked", e)
			}
		})
	}
}
func TestNativeAdmissionBeforeHandoffCancellationAndStaleGrant(t *testing.T) {
	w := newAdmissionWorld()
	s := w.store(t.TempDir())
	r := admissionRequest(t, "native")
	g := reserveAdmission(t, s, r)
	w.as(22)
	if e := s.CancelBeforeHandoff(context.Background(), g); e == nil {
		t.Fatal("foreign helper canceled")
	}
	w.as(7)
	if e := s.CancelBeforeHandoff(context.Background(), g); e != nil {
		t.Fatal(e)
	}
	r.Attempt = "attempt-2"
	next := reserveAdmission(t, s, r)
	if next == g {
		t.Fatal("reused capability")
	}
	w.as(12)
	if _, e := s.Claim(context.Background(), g, r.Agent, r.CommandDigest); e == nil {
		t.Fatal("canceled stale capability executed")
	}
}
func TestNativeAdmissionSameKeyBarrierAndDifferentKeyProgress(t *testing.T) {
	home := t.TempDir()
	w := newAdmissionWorld()
	r := admissionRequest(t, "native")
	start := make(chan struct{})
	results := make(chan error, 2)
	for i := 0; i < 2; i++ {
		go func() {
			<-start
			s := w.store(home)
			_, e := s.Reserve(context.Background(), r, noLegacy)
			results <- e
		}()
	}
	close(start)
	successes := 0
	for i := 0; i < 2; i++ {
		if <-results == nil {
			successes++
		}
	}
	if successes != 1 {
		t.Fatalf("same native writers admitted=%d", successes)
	}
	r.NativeID = "different"
	if _, e := w.store(home).Reserve(context.Background(), r, noLegacy); e != nil {
		t.Fatal(e)
	}
}
func TestNativeAdmissionCommandDigestPinsArgvBoundariesAndPrompt(t *testing.T) {
	a, _ := NativeAdmissionCommandDigest("codex", []string{"codex", "resume", "a b", "prompt"})
	b, _ := NativeAdmissionCommandDigest("codex", []string{"codex", "resume", "a", "b prompt"})
	c, _ := NativeAdmissionCommandDigest("codex", []string{"codex", "resume", "a b", "other"})
	if len(a) != 64 || a == b || a == c {
		t.Fatal("digest erased boundaries/prompt")
	}
	for _, argv := range [][]string{nil, {""}, {"codex", "bad\x00arg"}} {
		if _, e := NativeAdmissionCommandDigest("codex", argv); e == nil {
			t.Fatal("invalid invocation accepted")
		}
	}
}
