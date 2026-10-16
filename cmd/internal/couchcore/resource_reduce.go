package couchcore

import (
	"fmt"
	"github.com/xianxu/pair/cmd/internal/procutil"
	"math"
	"sort"
	"strings"
	"time"
)

type ResourcePrevious struct {
	At          time.Time
	Observation ResourceUsageObservation
}
type ResourceSlotTotal struct {
	Slot           string        `json:"slot"`
	Thread         ThreadAddress `json:"thread"`
	CPUTimeNS      uint64        `json:"cpu_time_ns"`
	DiskWriteBytes uint64        `json:"disk_write_bytes"`
}
type ResourceState struct {
	SlotTotals             []ResourceSlotTotal                     `json:"slot_totals"`
	Provenance             []ResourceProvenance                    `json:"provenance"`
	Episodes               []ResourceEpisode                       `json:"episodes"`
	ObservedCPUTimeNS      uint64                                  `json:"observed_cpu_time_ns"`
	ObservedDiskWriteBytes uint64                                  `json:"observed_disk_write_bytes"`
	Previous               map[ResourceProcessKey]ResourcePrevious `json:"-"`
}

func ResetResourceBaselines(s ResourceState) ResourceState {
	s.Previous = nil
	s.SlotTotals = append([]ResourceSlotTotal{}, s.SlotTotals...)
	s.Provenance = append([]ResourceProvenance{}, s.Provenance...)
	s.Episodes = append([]ResourceEpisode{}, s.Episodes...)
	for i, e := range s.Episodes {
		if e.Phase == "accumulating" {
			e.Phase = "clear"
			e.Samples = 0
			e.Since = time.Time{}
			e.Last = time.Time{}
			s.Episodes[i] = e
		}
	}
	return s
}
func ValidateResourceState(s ResourceState) error {
	if len(s.SlotTotals) > ResourceMaxSlots || len(s.Provenance) > ResourceMaxProcesses || len(s.Episodes) > ResourceMaxEpisodes {
		return fmt.Errorf("resource state exceeds bounds")
	}
	totals := map[string]bool{}
	for _, t := range s.SlotTotals {
		k := t.Slot + "/" + t.Thread.RepoScope + "/" + string(t.Thread.Tag)
		if t.Slot == "" || len(k) > 1024 || totals[k] {
			return fmt.Errorf("invalid slot totals")
		}
		totals[k] = true
	}
	keys := map[ResourceProcessKey]bool{}
	for _, p := range s.Provenance {
		if p.Key.PID <= 0 || p.Key.Identity == "" || len(p.Key.Identity) > 256 || keys[p.Key] || p.Root.PID <= 0 || p.Root.Identity == "" || len(p.Slot) > 256 || p.Slot == "" {
			return fmt.Errorf("invalid resource provenance")
		}
		keys[p.Key] = true
		if err := p.Workspace.validate(); err != nil {
			return fmt.Errorf("resource workspace: %w", err)
		}
	}
	eps := map[string]bool{}
	for _, e := range s.Episodes {
		if e.Key == "" || len(e.Key) > 1024 || eps[e.Key] {
			return fmt.Errorf("invalid episode key")
		}
		eps[e.Key] = true
		switch e.Metric {
		case "process-rss", "orphan-rss", "slot-write", "idle-cpu":
		default:
			return fmt.Errorf("invalid resource metric")
		}
		if e.Samples < 0 || e.Samples > 1000000 {
			return fmt.Errorf("invalid sample count")
		}
		switch e.Phase {
		case "clear", "accumulating", "pending", "notified":
		default:
			return fmt.Errorf("invalid episode phase")
		}
		switch e.Delivery {
		case "": // Legacy checkpoints predate explicit delivery observations.
		case ResourceDeliveryAdmitted:
			if e.Phase != "notified" {
				return fmt.Errorf("admitted delivery requires notified episode")
			}
		case ResourceDeliveryRefused, ResourceDeliveryUnknown:
			if e.Phase != "pending" {
				return fmt.Errorf("unadmitted delivery requires pending episode")
			}
		default:
			return fmt.Errorf("invalid resource delivery outcome")
		}
		if (e.Phase == "pending" || e.Phase == "notified") && (e.Effect == nil || e.Effect.ID == "" || len(e.Effect.ID) > 256) {
			return fmt.Errorf("missing episode effect")
		}
		if e.Effect != nil {
			f := e.Effect
			if len(f.Name) > 1024 || f.Metric != e.Metric || f.Slot == "" || len(f.Slot) > 256 || f.Duration < 0 || math.IsNaN(f.Value) || math.IsInf(f.Value, 0) || f.Value < 0 {
				return fmt.Errorf("invalid resource effect")
			}
			if err := f.Workspace.validate(); err != nil {
				return fmt.Errorf("effect workspace: %w", err)
			}
			if (e.Metric == "process-rss" || e.Metric == "orphan-rss") && (f.Process.PID <= 0 || f.Process.Identity == "") {
				return fmt.Errorf("invalid effect generation")
			}
		}
	}
	return nil
}
func resourceFloat(v float64) *float64 { return &v }
func AdvanceResources(state ResourceState, s ResourceSample, p ResourcePolicy) (ResourceState, ResourceView, []ResourceEffect) {
	sel := SelectResourceProcesses(s.Table, s.Slots, state.Provenance, s.TableKnown, s.InventoryComplete && s.InventoryOmitted == 0)
	next := ResourceState{Provenance: sel.Processes, Previous: map[ResourceProcessKey]ResourcePrevious{}, ObservedCPUTimeNS: state.ObservedCPUTimeNS, ObservedDiskWriteBytes: state.ObservedDiskWriteBytes}
	v := ResourceView{SampledAt: s.At, Slots: []ResourceSlotView{}, Processes: []ResourceProcessView{}, Orphans: []ResourceProcessView{}, Errors: append(append([]string{}, s.Errors...), sel.Errors...), Cost: s.Cost}
	obs := map[ResourceProcessKey]ResourceUsageObservation{}
	for _, o := range s.Observations {
		obs[o.Key] = o
	}
	names := map[ResourceProcessKey]string{}
	for _, r := range s.Table {
		names[resourceKey(r)] = r.Name
	}
	episodes := map[string]ResourceEpisode{}
	currentSlots := map[string]ResourceSlot{}
	for _, slot := range s.Slots {
		currentSlots[slot.Slot] = slot
	}
	priorEpisodes := append([]ResourceEpisode(nil), state.Episodes...)
	sort.Slice(priorEpisodes, func(i, j int) bool {
		a, b := priorEpisodes[i], priorEpisodes[j]
		ap := a.Phase == "pending" || a.Phase == "notified"
		bp := b.Phase == "pending" || b.Phase == "notified"
		if ap != bp {
			return ap
		}
		return a.Key < b.Key
	})
	episodeLimitReported := false
	episodeLimit := func() {
		if !episodeLimitReported {
			v.Errors = append(v.Errors, "episode limit exceeded; new alerts unavailable until capacity recovers")
			episodeLimitReported = true
		}
	}
	for _, e := range priorEpisodes {
		if slot, ok := currentSlots[e.Slot]; ok && slot.OwnershipKnown && (e.Metric == "slot-write" || e.Metric == "idle-cpu") {
			prefix := "write/"
			if e.Metric == "idle-cpu" {
				prefix = "cpu/"
			}
			if e.Key != prefix+slot.Slot+"/"+string(slot.Thread.Tag) {
				continue
			}
		}
		if len(episodes) >= ResourceMaxEpisodes {
			episodeLimit()
			continue
		}
		episodes[e.Key] = e
	}
	totals := map[string]ResourceSlotTotal{}
	totalKey := func(slot string, thread ThreadAddress) string {
		return slot + "/" + thread.RepoScope + "/" + string(thread.Tag)
	}
	for _, t := range state.SlotTotals {
		totals[totalKey(t.Slot, t.Thread)] = t
	}
	var effects []ResourceEffect
	visited := map[string]bool{}
	advance := func(key, metric string, known, high bool, value float64, d time.Duration, n int, owner ResourceRoot) {
		e, exists := episodes[key]
		if !exists && len(episodes) >= ResourceMaxEpisodes {
			episodeLimit()
			return
		}
		e.Key = key
		e.Process = owner.Key
		e.Slot = owner.Slot
		e.Metric = metric
		event := ResourceEpisodeEvent{At: s.At, Known: known, High: high, RequiredDuration: d, RequiredSamples: n, MaxGap: p.Cadence * 2, Effect: ResourceEffect{Name: names[owner.Key], Slot: owner.Slot, Process: owner.Key, Value: value, Workspace: owner.Workspace}}
		e, f := AdvanceResourceEpisode(e, event)
		if e.Phase == "clear" {
			delete(episodes, key)
		} else {
			episodes[key] = e
		}
		visited[key] = true
		if f != nil {
			effects = append(effects, *f)
		}
	}
	for _, pv := range sel.Processes {
		o, ok := obs[pv.Key]
		r := ResourceProcessView{ResourceProvenance: pv, Name: names[pv.Key], Reason: o.Reason}
		if !ok {
			r.Reason = "observation unavailable"
		}
		if ok && o.RSSKnown {
			rss := o.RSSBytes
			r.RSSBytes = &rss
		}
		old, has := state.Previous[pv.Key]
		delta := s.At.Sub(old.At)
		valid := has && delta > 0 && delta <= p.Cadence*2 && s.TableKnown && len(sel.Errors) == 0
		if valid && o.CPUKnown && old.Observation.CPUKnown && o.CPUTimeNS >= old.Observation.CPUTimeNS {
			d := o.CPUTimeNS - old.Observation.CPUTimeNS
			r.CPUPercent = resourceFloat(float64(d) / float64(delta) * 100)
			next.ObservedCPUTimeNS = resourceSaturatingAdd(next.ObservedCPUTimeNS, d)
			k := totalKey(pv.Slot, pv.Thread)
			t := totals[k]
			t.Slot = pv.Slot
			t.Thread = pv.Thread
			t.CPUTimeNS = resourceSaturatingAdd(t.CPUTimeNS, d)
			totals[k] = t
		}
		if valid && o.DiskKnown && old.Observation.DiskKnown && o.DiskWriteBytes >= old.Observation.DiskWriteBytes {
			d := o.DiskWriteBytes - old.Observation.DiskWriteBytes
			r.WriteBytesPerSecond = resourceFloat(float64(d) / delta.Seconds())
			next.ObservedDiskWriteBytes = resourceSaturatingAdd(next.ObservedDiskWriteBytes, d)
			k := totalKey(pv.Slot, pv.Thread)
			t := totals[k]
			t.Slot = pv.Slot
			t.Thread = pv.Thread
			t.DiskWriteBytes = resourceSaturatingAdd(t.DiskWriteBytes, d)
			totals[k] = t
		}
		if ok && s.TableKnown && len(sel.Errors) == 0 {
			next.Previous[pv.Key] = ResourcePrevious{s.At, o}
		}
		if r.CPUPercent == nil && r.Reason == "" {
			r.Reason = "warming-up or incomplete interval"
		}
		v.Processes = append(v.Processes, r)
		if pv.Orphan {
			v.Orphans = append(v.Orphans, r)
		}
		key := fmt.Sprintf("%d/%s", pv.Key.PID, pv.Key.Identity)
		advance("rss/"+key, "process-rss", ok && o.RSSKnown, o.RSSBytes > p.ProcessRSSBytes, float64(o.RSSBytes), 0, 1, pv.ResourceRoot)
		advance("orphan/"+key, "orphan-rss", ok && o.RSSKnown && pv.OrphanKnown, pv.Orphan && o.RSSBytes > p.OrphanRSSBytes, float64(o.RSSBytes), 0, p.OrphanSamples, pv.ResourceRoot)
	}
	slots := append([]ResourceSlot(nil), s.Slots...)
	sort.Slice(slots, func(i, j int) bool { return slots[i].Slot < slots[j].Slot })
	if len(slots) > ResourceMaxSlots {
		slots = slots[:ResourceMaxSlots]
	}
	for _, slot := range slots {
		sv := ResourceSlotView{Slot: slot.Slot, Thread: slot.Thread, IdleKnown: slot.IdleKnown, Idle: slot.Idle}
		cpu, write := 0.0, 0.0
		var cpuKey, writeKey ResourceProcessKey
		maxCPU, maxWrite := -1.0, -1.0
		rss := uint64(0)
		ck, wk, rk := s.TableKnown && slot.OwnershipKnown && len(sel.Errors) == 0, s.TableKnown && slot.OwnershipKnown && len(sel.Errors) == 0, s.TableKnown && slot.OwnershipKnown && len(sel.Errors) == 0
		for _, r := range v.Processes {
			if r.Slot != slot.Slot || r.Thread != slot.Thread || r.Orphan {
				continue
			}
			sv.ProcessCount++
			if r.CPUPercent == nil {
				ck = false
			} else {
				cpu += *r.CPUPercent
				if *r.CPUPercent > maxCPU || *r.CPUPercent == maxCPU && r.Key.PID < cpuKey.PID {
					maxCPU = *r.CPUPercent
					cpuKey = r.Key
				}
			}
			if r.WriteBytesPerSecond == nil {
				wk = false
			} else {
				write += *r.WriteBytesPerSecond
				if *r.WriteBytesPerSecond > maxWrite || *r.WriteBytesPerSecond == maxWrite && r.Key.PID < writeKey.PID {
					maxWrite = *r.WriteBytesPerSecond
					writeKey = r.Key
				}
			}
			if r.RSSBytes == nil {
				rk = false
			} else {
				rss += *r.RSSBytes
			}
		}
		if sv.ProcessCount == 0 {
			ck = false
			wk = false
			rk = false
		}
		if ck {
			sv.CPUPercent = &cpu
		}
		if wk {
			sv.WriteBytesPerSecond = &write
		}
		if rk {
			sv.RSSBytes = &rss
		}
		if !ck || !wk || !rk {
			sv.Reason = "partial or warming-up"
		}
		if slot.Reason != "" {
			sv.Reason = slot.Reason
		}
		total := totals[totalKey(slot.Slot, slot.Thread)]
		sv.ObservedCPUTimeNS = total.CPUTimeNS
		sv.ObservedDiskWriteBytes = total.DiskWriteBytes
		v.Slots = append(v.Slots, sv)
		owner := ResourceRoot{Slot: slot.Slot, Thread: slot.Thread, Workspace: slot.Workspace}
		suffix := slot.Slot + "/" + string(slot.Thread.Tag)
		owner.Key = writeKey
		advance("write/"+suffix, "slot-write", wk, write > p.WriteBytesPerSecond, write, p.WriteDuration, 1, owner)
		owner.Key = cpuKey
		advance("cpu/"+suffix, "idle-cpu", ck && slot.IdleKnown, slot.Idle && cpu > p.IdleCPUPercent, cpu, p.IdleCPUDuration, 1, owner)
	}
	deathRows := map[int]procutil.Process{}
	ambiguousPID := map[int]bool{}
	for _, row := range s.Table {
		if _, ok := deathRows[row.PID]; ok {
			ambiguousPID[row.PID] = true
		}
		deathRows[row.PID] = row
	}
	for key, e := range episodes {
		if !visited[key] {
			dead := false
			if s.TableKnown && len(s.Table) <= ResourceMaxTableRows && (strings.HasPrefix(key, "rss/") || strings.HasPrefix(key, "orphan/")) && e.Process.PID > 0 {
				row, exists := deathRows[e.Process.PID]
				dead = !ambiguousPID[e.Process.PID] && (!exists || row.Zombie || row.Identity != "" && row.Identity != e.Process.Identity)
			}

			if dead || s.TableKnown && e.Phase == "clear" {
				continue
			}
			e, _ = AdvanceResourceEpisode(e, ResourceEpisodeEvent{At: s.At})
		}
		next.Episodes = append(next.Episodes, e)
	}
	sort.Slice(next.Episodes, func(i, j int) bool { return next.Episodes[i].Key < next.Episodes[j].Key })
	v.Unattributed = ClassifyUnattributed(s.Table, sel.Processes, s.Observations, s.Commands, p.OrphanRSSBytes)
	totalKeys := make([]string, 0, len(totals))
	for k := range totals {
		totalKeys = append(totalKeys, k)
	}
	activeTotals := map[string]bool{}
	for _, slot := range s.Slots {
		activeTotals[totalKey(slot.Slot, slot.Thread)] = true
	}
	sort.Slice(totalKeys, func(i, j int) bool {
		a, b := activeTotals[totalKeys[i]], activeTotals[totalKeys[j]]
		if a != b {
			return a
		}
		return totalKeys[i] < totalKeys[j]
	})
	for _, k := range totalKeys {
		if len(next.SlotTotals) >= ResourceMaxSlots {
			v.Errors = append(v.Errors, "slot totals limit exceeded")
			break
		}
		next.SlotTotals = append(next.SlotTotals, totals[k])
	}
	v.ObservedCPUTimeNS = next.ObservedCPUTimeNS
	v.ObservedDiskWriteBytes = next.ObservedDiskWriteBytes
	v.Episodes = append([]ResourceEpisode{}, next.Episodes...)
	return next, v, effects
}

func resourceSaturatingAdd(a, b uint64) uint64 {
	if ^uint64(0)-a < b {
		return ^uint64(0)
	}
	return a + b
}
