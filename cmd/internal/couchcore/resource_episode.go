package couchcore

import (
	"crypto/sha256"
	"fmt"
	"time"
)

type ResourceEffect struct {
	Name      string             `json:"name"`
	ID        string             `json:"id"`
	Metric    string             `json:"metric"`
	Slot      string             `json:"slot"`
	Process   ResourceProcessKey `json:"process"`
	Value     float64            `json:"value"`
	Duration  time.Duration      `json:"duration"`
	Workspace WorkspaceIdentity  `json:"workspace"`
}
type ResourceDeliveryOutcome string

const (
	ResourceDeliveryAdmitted ResourceDeliveryOutcome = "admitted"
	ResourceDeliveryRefused  ResourceDeliveryOutcome = "refused"
	ResourceDeliveryUnknown  ResourceDeliveryOutcome = "unknown"
)

type ResourceEpisode struct {
	Delivery ResourceDeliveryOutcome `json:"delivery,omitempty"`
	Process  ResourceProcessKey      `json:"process"`
	Slot     string                  `json:"slot"`
	Key      string                  `json:"key"`
	Metric   string                  `json:"metric"`
	Phase    string                  `json:"phase"`
	Since    time.Time               `json:"since"`
	Last     time.Time               `json:"last"`
	Samples  int                     `json:"samples"`
	Effect   *ResourceEffect         `json:"effect,omitempty"`
}
type ResourceEpisodeEvent struct {
	Delivery              ResourceDeliveryOutcome
	EffectID              string
	At                    time.Time
	Known, High, Admitted bool
	RequiredDuration      time.Duration
	RequiredSamples       int
	MaxGap                time.Duration
	Effect                ResourceEffect
}

// Unknown breaks accumulation but cannot clear a pending/admitted incident.
func AdvanceResourceEpisode(e ResourceEpisode, v ResourceEpisodeEvent) (ResourceEpisode, *ResourceEffect) {
	if e.Effect != nil {
		x := *e.Effect
		e.Effect = &x
	}
	if e.Phase == "" {
		e.Phase = "clear"
	}
	outcome := v.Delivery
	if v.Admitted {
		outcome = ResourceDeliveryAdmitted
	}
	if outcome != "" {
		// Delivery observations belong to one immutable effect. A late receipt
		// cannot admit its recovered episode's replacement or change metric time.
		if e.Effect == nil || v.EffectID != "" && v.EffectID != e.Effect.ID || e.Phase != "pending" && e.Phase != "notified" {
			return e, nil
		}
		if e.Phase == "notified" {
			return e, nil
		}
		switch outcome {
		case ResourceDeliveryAdmitted:
			e.Phase = "notified"
			e.Delivery = outcome
		case ResourceDeliveryRefused, ResourceDeliveryUnknown:
			e.Delivery = outcome
		}
		return e, nil
	}
	if !v.Known {
		if e.Phase == "accumulating" {
			e.Phase = "clear"
			e.Samples = 0
			e.Since = time.Time{}
		}
		return e, nil
	}
	if !v.High {
		e.Phase = "clear"
		e.Samples = 0
		e.Since = time.Time{}
		e.Last = v.At
		e.Effect = nil
		e.Delivery = ""
		return e, nil
	}
	if e.Phase == "pending" || e.Phase == "notified" {
		return e, nil
	}
	if e.Phase == "clear" || !v.At.After(e.Last) || v.MaxGap > 0 && v.At.Sub(e.Last) > v.MaxGap {
		e.Since = v.At
		e.Samples = 0
	}
	e.Phase = "accumulating"
	e.Last = v.At
	e.Samples++
	if e.Samples >= v.RequiredSamples && v.At.Sub(e.Since) >= v.RequiredDuration {
		x := v.Effect
		x.Metric = e.Metric
		x.Duration = v.At.Sub(e.Since)
		x.ID = fmt.Sprintf("resource-%x", sha256.Sum256([]byte(e.Key+"/"+e.Since.UTC().Format(time.RFC3339Nano))))
		e.Effect = &x
		e.Phase = "pending"
		e.Delivery = ""
		return e, &x
	}
	return e, nil
}
