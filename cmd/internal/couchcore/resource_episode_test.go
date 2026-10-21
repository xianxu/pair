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
func TestResourceEpisodeDeliveryOutcomesAreGenerationFenced(t *testing.T) {
	e := ResourceEpisode{Phase: "pending", Effect: &ResourceEffect{ID: "current"}}
	for _, outcome := range []ResourceDeliveryOutcome{ResourceDeliveryRefused, ResourceDeliveryUnknown, ResourceDeliveryAdmitted} {
		next, effect := AdvanceResourceEpisode(e, ResourceEpisodeEvent{Delivery: outcome, EffectID: "older"})
		if next.Phase != "pending" || next.Delivery != "" || effect != nil {
			t.Fatalf("late old receipt changed new episode: %+v", next)
		}
	}
	for _, outcome := range []ResourceDeliveryOutcome{ResourceDeliveryRefused, ResourceDeliveryUnknown, ResourceDeliveryAdmitted} {
		next, effect := AdvanceResourceEpisode(e, ResourceEpisodeEvent{Delivery: outcome, EffectID: "current"})
		want := "pending"
		if outcome == ResourceDeliveryAdmitted {
			want = "notified"
		}
		if next.Phase != want || next.Delivery != outcome || effect != nil || e.Phase != "pending" || e.Delivery != "" {
			t.Fatalf("outcome %s: %+v", outcome, next)
		}
	}
	admitted, _ := AdvanceResourceEpisode(e, ResourceEpisodeEvent{Delivery: ResourceDeliveryAdmitted, EffectID: "current"})
	late, _ := AdvanceResourceEpisode(admitted, ResourceEpisodeEvent{Delivery: ResourceDeliveryRefused, EffectID: "current"})
	if late.Phase != "notified" || late.Delivery != ResourceDeliveryAdmitted {
		t.Fatal("late refusal undid admission")
	}
	clear, _ := AdvanceResourceEpisode(admitted, ResourceEpisodeEvent{Known: true})
	if clear.Phase != "clear" || clear.Delivery != "" {
		t.Fatal("recovery retained old delivery")
	}
}
func TestResourceEpisodeDeliveryDoesNotActAsUnknownMetric(t *testing.T) {
	e := ResourceEpisode{Phase: "accumulating", Samples: 3, Since: time.Now()}
	next, _ := AdvanceResourceEpisode(e, ResourceEpisodeEvent{Delivery: ResourceDeliveryUnknown, EffectID: "old"})
	if next.Phase != e.Phase || next.Samples != 3 {
		t.Fatal("receipt reset metric accumulation")
	}
}

func TestResourceEpisodeSuppressedIsTerminalWithoutClaimingDelivery(t *testing.T) {
	at := time.Unix(1, 0)
	initial, effect := AdvanceResourceEpisode(ResourceEpisode{Key: "cpu/worker", Metric: "idle-cpu"}, ResourceEpisodeEvent{At: at, Known: true, High: true, RequiredSamples: 1})
	if effect == nil {
		t.Fatal("no initial incident")
	}
	suppressed := ResourceDeliveryOutcome("suppressed")
	episode, _ := AdvanceResourceEpisode(initial, ResourceEpisodeEvent{Delivery: suppressed, EffectID: effect.ID})
	if episode.Phase != "suppressed" || episode.Delivery != suppressed || episode.Effect.ID != effect.ID {
		t.Fatalf("suppression did not retire occurrence: %+v", episode)
	}
	// No uncertain metrics, ongoing high observation, stale receipt, or later
	// admission can turn this consumed occurrence back into pending work.
	for _, event := range []ResourceEpisodeEvent{
		{At: at.Add(time.Minute)},
		{At: at.Add(2 * time.Minute), Known: true, High: true, RequiredSamples: 1},
		{Delivery: ResourceDeliveryAdmitted, EffectID: effect.ID},
		{Delivery: ResourceDeliveryRefused, EffectID: effect.ID},
		{Delivery: suppressed, EffectID: "stale"},
	} {
		var next *ResourceEffect
		episode, next = AdvanceResourceEpisode(episode, event)
		if next != nil || episode.Phase != "suppressed" || episode.Delivery != suppressed {
			t.Fatalf("consumed incident replayed: %+v", episode)
		}
	}
	episode, _ = AdvanceResourceEpisode(episode, ResourceEpisodeEvent{At: at.Add(3 * time.Minute), Known: true})
	episode, next := AdvanceResourceEpisode(episode, ResourceEpisodeEvent{At: at.Add(4 * time.Minute), Known: true, High: true, RequiredSamples: 1})
	if next == nil || next.ID == effect.ID || episode.Phase != "pending" || episode.Delivery != "" {
		t.Fatalf("fresh incident after recovery lost: %+v", episode)
	}
}

func TestResourceEpisodeStaleSuppressionCannotRetireReplacement(t *testing.T) {
	initial := ResourceEpisode{Phase: "pending", Effect: &ResourceEffect{ID: "current"}}
	next, effect := AdvanceResourceEpisode(initial, ResourceEpisodeEvent{Delivery: ResourceDeliveryOutcome("suppressed"), EffectID: "older"})
	if next.Phase != "pending" || next.Delivery != "" || effect != nil {
		t.Fatalf("stale suppression changed replacement: %+v", next)
	}
}
