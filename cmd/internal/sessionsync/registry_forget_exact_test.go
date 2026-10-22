package sessionsync

import (
	"os"
	"testing"
)

func TestForgetExactRefusesReplacementRegistration(t *testing.T) {
	r := newRig(t)
	first, err := r.sync.Register(r.p.File, "ops:0")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = r.sync.Forget(r.p.File); err != nil {
		t.Fatal(err)
	}
	next, err := r.sync.Register(r.p.File, "ops:0")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = r.sync.ForgetExact(r.p.File, first.Registration.ID); err == nil {
		t.Fatal("stale ID removed replacement")
	}
	got, err := r.sync.Lookup(r.p.File)
	if err != nil || got == nil || got.ID != next.Registration.ID {
		t.Fatalf("replacement lost: %+v %v", got, err)
	}
	if _, err = r.sync.ForgetExact(r.p.File, ""); err == nil {
		t.Fatal("empty ID accepted")
	}
	result, err := r.sync.ForgetExact(r.p.File, next.Registration.ID)
	if err != nil || !result.Changed {
		t.Fatalf("exact forget: %+v %v", result, err)
	}
	result, err = r.sync.ForgetExact(r.p.File, next.Registration.ID)
	if err != nil || result.Changed {
		t.Fatalf("retry: %+v %v", result, err)
	}
}

func TestRegistrationForPathCanForgetMissingDocument(t *testing.T) {
	r := newRig(t)
	first, err := r.sync.Register(r.p.File, "ops:0")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(r.p.File); err != nil {
		t.Fatal(err)
	}
	got, err := r.sync.RegistrationForPath(r.p.File)
	if err != nil || got == nil || got.ID != first.Registration.ID {
		t.Fatalf("missing-file registration: %+v %v", got, err)
	}
	if _, err := r.sync.ForgetExact(r.p.File, got.ID); err != nil {
		t.Fatal(err)
	}
}
