package storagegc

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
)

func policyFixture(t *testing.T) (time.Time, Evidence) {
	t.Helper()
	now := time.Date(2026, 9, 13, 0, 0, 0, 0, time.UTC)
	owner, err := artifactpath.NewStorageOwner("/pair-data", "scope", "thread")
	if err != nil {
		t.Fatal(err)
	}
	return now, Evidence{Owner: owner, Complete: true, Live: ProcessDead, Activity: &ActivityRecord{Version: 1, Owner: owner, Incarnation: "abc", InitializedAt: now.Add(-RetentionPeriod), LastUse: now.Add(-RetentionPeriod)}}
}

func TestDecideRetentionBoundary(t *testing.T) {
	now, e := policyFixture(t)
	for _, delta := range []time.Duration{-time.Nanosecond, 0, time.Nanosecond} {
		d := Decide(now.Add(delta), e)
		want := Eligible
		if delta < 0 {
			want = Grace
		}
		if d.State != want {
			t.Errorf("delta %v: got %s want %s", delta, d.State, want)
		}
	}
}

func TestDecideProtectionDominatesExpiry(t *testing.T) {
	now, e := policyFixture(t)
	e.Protected = true
	if d := Decide(now, e); d.State != Protected {
		t.Fatal(d)
	}
	e.Protected = false
	e.Live = ProcessAlive
	if d := Decide(now, e); d.State != Live {
		t.Fatal(d)
	}
	e.Live = ProcessUnknown
	if d := Decide(now, e); d.State != Blocked {
		t.Fatal(d)
	}
	e.Live = ProcessDead
	e.Complete = false
	if d := Decide(now, e); d.State != Blocked {
		t.Fatal(d)
	}
}

func TestDecideMissingAndInvalidClocks(t *testing.T) {
	now, e := policyFixture(t)
	e.Activity = nil
	if d := Decide(now, e); d.State != Untracked {
		t.Fatal(d)
	}
	now, e = policyFixture(t)
	e.ArchiveTimes = []time.Time{{}}
	if d := Decide(now, e); d.State != Untracked {
		t.Fatal(d)
	}
	e.ArchiveTimes = []time.Time{now.Add(-time.Hour)}
	if d := Decide(now, e); d.State != Grace {
		t.Fatal(d)
	}
	e.Activity.LastUse = now.Add(time.Hour)
	if d := Decide(now, e); d.State != Blocked {
		t.Fatal(d)
	}
}

func TestDecodeActivityRejectsInvalidEvidence(t *testing.T) {
	_, e := policyFixture(t)
	raw, err := json.Marshal(e.Activity)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := DecodeActivity(raw, e.Owner); err != nil {
		t.Fatal(err)
	}
	for _, b := range [][]byte{raw[:len(raw)-1], append(append([]byte{}, raw...), []byte(" {}")...), []byte(`{"version":99}`), []byte(`null`)} {
		if _, err := DecodeActivity(b, e.Owner); err == nil {
			t.Fatalf("accepted %s", b)
		}
	}
	wrong := e.Owner
	wrong.Tag = "other"
	if _, err := DecodeActivity(raw, wrong); err == nil {
		t.Fatal("accepted foreign owner")
	}
}

func FuzzDecodeActivity(f *testing.F) {
	f.Add([]byte(`{}`))
	f.Add([]byte(`null`))
	f.Fuzz(func(t *testing.T, b []byte) {
		owner, err := artifactpath.NewStorageOwner("/data", "scope", "tag")
		if err != nil {
			t.Fatal(err)
		}
		a, err := DecodeActivity(b, owner)
		if err == nil {
			if a.Owner != owner || a.Version != 1 || a.Incarnation == "" || a.InitializedAt.IsZero() {
				t.Fatal("invalid accepted activity")
			}
		}
	})
}

func FuzzDecideProtectionMonotonic(f *testing.F) {
	f.Add(int64(0))
	f.Add(int64(86400 * 61))
	f.Fuzz(func(t *testing.T, seconds int64) {
		now, e := policyFixture(t)
		now = now.Add(time.Duration(seconds%10000000) * time.Second)
		e.Protected = true
		if d := Decide(now, e); d.State == Eligible {
			t.Fatal("protection lost")
		}
	})
}
