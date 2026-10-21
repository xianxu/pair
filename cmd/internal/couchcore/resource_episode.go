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
	// Age and Root are the orphan sweep's evidence: how long the process has
	// outlived its launcher, and the managed checkout it was found in. Both are
	// bound when the effect fires and never move afterwards (#583).
	Age  time.Duration `json:"age,omitempty"`
	Root string        `json:"root,omitempty"`
}
type ResourceDeliveryOutcome string

// ResourceMetricOrphanNvim is the episode metric for the untagged embedded-nvim
// sweep. It is a constant because the alert body is chosen from it in couchcmd
// (#583); the other metrics are read only where they are written.
const ResourceMetricOrphanNvim = "orphan-nvim"

const (
	ResourceDeliveryAdmitted   ResourceDeliveryOutcome = "admitted"
	ResourceDeliveryRefused    ResourceDeliveryOutcome = "refused"
	ResourceDeliveryUnknown    ResourceDeliveryOutcome = "unknown"
	ResourceDeliverySuppressed ResourceDeliveryOutcome = "suppressed"
)

type ResourceEpisode struct {
	// AutomationEpoch binds a pending publisher occurrence to volatile policy.
	// It is not persisted metric evidence and clears only on metric recovery.
	AutomationEpoch *uint64                 `json:"-"`
	Delivery        ResourceDeliveryOutcome `json:"delivery,omitempty"`
	Process         ResourceProcessKey      `json:"process"`
	Slot            string                  `json:"slot"`
	Key             string                  `json:"key"`
	Metric          string                  `json:"metric"`
	Phase           string                  `json:"phase"`
	Since           time.Time               `json:"since"`
	Last            time.Time               `json:"last"`
	Samples         int                     `json:"samples"`
	Effect          *ResourceEffect         `json:"effect,omitempty"`
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

// Unknown breaks accumulation but cannot clear a pending, admitted, or
// explicitly suppressed incident. Suppression consumes the occurrence until
// positive metric recovery, without claiming delivery.
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
		case ResourceDeliverySuppressed:
			e.Phase = "suppressed"
			e.Delivery = outcome
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
		e.AutomationEpoch = nil
		return e, nil
	}
	if e.Phase == "pending" || e.Phase == "notified" || e.Phase == "suppressed" {
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
