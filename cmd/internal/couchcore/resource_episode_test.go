package couchcore

import (
	"testing"
	"time"
)

func TestResourceEpisodeUnknownDoesNotRecover(t *testing.T) {
	at := time.Now()
	e := ResourceEpisode{Key: "rss/p", Metric: "process-rss"}
	e, f := AdvanceResourceEpisode(e, ResourceEpisodeEvent{At: at, Known: true, High: true, RequiredSamples: 1})
	if f == nil || e.Phase != "pending" {
		t.Fatal(e)
	}
	id := e.Effect.ID
	e, _ = AdvanceResourceEpisode(e, ResourceEpisodeEvent{At: at.Add(time.Minute), Admitted: true})
	e, _ = AdvanceResourceEpisode(e, ResourceEpisodeEvent{At: at.Add(2 * time.Minute)})
	if e.Phase != "notified" || e.Effect.ID != id {
		t.Fatal(e)
	}
	e, f = AdvanceResourceEpisode(e, ResourceEpisodeEvent{At: at.Add(3 * time.Minute), Known: true, High: true})
	if f != nil {
		t.Fatal("duplicate effect")
	}
	e, _ = AdvanceResourceEpisode(e, ResourceEpisodeEvent{At: at.Add(4 * time.Minute), Known: true})
	if e.Phase != "clear" {
		t.Fatal(e)
	}
	e, f = AdvanceResourceEpisode(e, ResourceEpisodeEvent{At: at.Add(5 * time.Minute), Known: true, High: true, RequiredSamples: 1})
	if f == nil || f.ID == id {
		t.Fatal("recovery did not renew")
	}
}
func TestResourceEpisodeGapAndUnknownBreakAccumulation(t *testing.T) {
	at := time.Now()
	e := ResourceEpisode{Key: "w", Metric: "slot-write"}
	ev := ResourceEpisodeEvent{At: at, Known: true, High: true, RequiredDuration: 2 * time.Minute, RequiredSamples: 1, MaxGap: 2 * time.Minute}
	e, _ = AdvanceResourceEpisode(e, ev)
	ev.At = at.Add(time.Minute)
	e, _ = AdvanceResourceEpisode(e, ev)
	e, _ = AdvanceResourceEpisode(e, ResourceEpisodeEvent{At: at.Add(2 * time.Minute)})
	ev.At = at.Add(3 * time.Minute)
	e, f := AdvanceResourceEpisode(e, ev)
	if f != nil || e.Samples != 1 {
		t.Fatal("unknown bridged high duration")
	}
	ev.At = at.Add(6 * time.Minute)
	e, f = AdvanceResourceEpisode(e, ev)
	if f != nil || e.Samples != 1 {
		t.Fatal("gap bridged high duration")
	}
}
