package couchcore

import (
	"fmt"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

// LifecycleNoticeKind names only confirmed lifecycle boundaries; attachment exit
// deliberately has no corresponding kind.
type LifecycleNoticeKind string

const (
	LifecycleStarted       LifecycleNoticeKind = "started"
	LifecycleResumed       LifecycleNoticeKind = "resumed"
	LifecycleParked        LifecycleNoticeKind = "parked"
	LifecycleRelaunched    LifecycleNoticeKind = "relaunched"
	LifecycleRecovered     LifecycleNoticeKind = "recovered"
	LifecycleNeedsRelaunch LifecycleNoticeKind = "needs-relaunch"
	// LifecycleExited is the native agent leaving on its own, proved by the
	// wrapper that reaped it (pair#657); Mode carries "code N".
	LifecycleExited LifecycleNoticeKind = "exited"
)

// LifecycleNoticeFact is copied after the authoritative operation completes.
// Observation increases per slot across replacements, not merely per incarnation.
// Incarnation is an opaque exact identity and is never rendered in the notice.
type LifecycleNoticeFact struct {
	Hub, Slot                                                string
	Address                                                  ThreadAddress
	Incarnation                                              string
	Observation                                              uint64
	Kind                                                     LifecycleNoticeKind
	Operation, Agent, Model, Branch, Issue, Initiation, Mode string
}
type LifecycleNoticeEventKind uint8

const (
	LifecycleNoticeConfirmed LifecycleNoticeEventKind = iota + 1
	LifecycleNoticeArm
	LifecycleNoticeDrain
	LifecycleNoticeHubReady
	LifecycleNoticeAdmission
	LifecycleNoticeReceipt
	LifecycleNoticeShutdown
	LifecycleNoticeDiscard
	LifecycleNoticeSuppressHub
)

type LifecycleNoticeEvent struct {
	Kind             LifecycleNoticeEventKind
	Hubs             []string
	Fact             LifecycleNoticeFact
	Hub, Incarnation string
	Ready            bool
	Reason           string
	Token            uint64
	Outcome          string
}
type LifecycleNoticeEffectKind uint8

const (
	LifecycleNoticePublish LifecycleNoticeEffectKind = iota + 1
	LifecycleNoticeDiagnostic
	LifecycleNoticeCancel
)

// Token identifies a single attempt; the shell assigns its broker message ID.
type LifecycleNoticeEffect struct {
	Kind             LifecycleNoticeEffectKind
	Hub, Incarnation string
	Token            uint64
	Body             string
}
type lifecycleNoticeCurrent struct {
	fact    LifecycleNoticeFact
	changes uint32
	retired uint64
}
type lifecycleNoticeRevision struct {
	key         string
	observation uint64
	changes     uint32
}
type lifecycleNoticePart struct {
	body      string
	revisions []lifecycleNoticeRevision
}
type lifecycleNoticeHub struct {
	incarnation                    string
	ready, suppressed, unsupported bool
	parts                          []lifecycleNoticePart
	token                          uint64
	target                         string
}

// LifecycleNoticeState is ephemeral. Its zero value is ready for ordinary facts;
// Arm must precede startup observations to establish the restart drain barrier.
type LifecycleNoticeState struct {
	facts               map[string]lifecycleNoticeCurrent
	hubs                map[string]lifecycleNoticeHub
	collecting, stopped bool
	reason              string
	overflow            uint32
	nextToken           uint64
}

// AdvanceLifecycleNotices is the sole transition. It does no IO and never mutates
// the input state's maps or immutable parts. Unknown outcomes mean uncertainty,
// never permission to replay an attempt.
func AdvanceLifecycleNotices(s LifecycleNoticeState, e LifecycleNoticeEvent) (LifecycleNoticeState, []LifecycleNoticeEffect) {
	if s.stopped {
		return s, nil
	}
	s = cloneLifecycleNotices(s)
	var effects []LifecycleNoticeEffect
	diagnostic := func(hub, message string) {
		effects = append(effects, LifecycleNoticeEffect{Kind: LifecycleNoticeDiagnostic, Hub: hub, Body: noticeText(message, 256)})
	}
	switch e.Kind {
	case LifecycleNoticeShutdown:
		for hub, h := range s.hubs {
			if h.token != 0 {
				effects = append(effects, LifecycleNoticeEffect{Kind: LifecycleNoticeCancel, Hub: hub, Incarnation: h.target, Token: h.token})
			}
		}
		return LifecycleNoticeState{stopped: true}, effects
	case LifecycleNoticeDiscard:
		// The shell retains already-admitted attempts solely for their receipts.
		// None of those tokens may complete or replay this new projection.
		if len(e.Hubs) > 128 {
			return s, nil
		}
		roots := make(map[string]bool, len(e.Hubs))
		for _, hub := range e.Hubs {
			if hub == "" || len(hub) > 4096 {
				return s, nil
			}
			roots[hub] = true
		}
		count := len(s.hubs)
		for hub := range roots {
			if _, exists := s.hubs[hub]; !exists {
				count++
			}
		}
		if count > 128 {
			return s, nil
		}
		s.facts = make(map[string]lifecycleNoticeCurrent)
		s.overflow = 0
		for hub, h := range s.hubs {
			h.parts, h.token, h.target = nil, 0, ""
			h.suppressed = false
			s.hubs[hub] = h
		}
		for hub := range roots {
			h := s.hubs[hub]
			h.parts = []lifecycleNoticePart{{body: "Couch hub changed; earlier lifecycle notices were dropped; check Slots"}}
			s.hubs[hub] = h
		}
	case LifecycleNoticeSuppressHub:
		if e.Hub == "" || len(e.Hub) > 4096 {
			return s, nil
		}
		for key, current := range s.facts {
			if current.fact.Hub == e.Hub {
				current.retired = current.fact.Observation
				current.changes = 0
				s.facts[key] = current
			}
		}
		if h, ok := s.hubs[e.Hub]; ok {
			if h.token != 0 {
				effects = append(effects, LifecycleNoticeEffect{Kind: LifecycleNoticeCancel, Hub: e.Hub, Incarnation: h.target, Token: h.token})
			}
			h.parts, h.token, h.target = nil, 0, ""
			h.suppressed = false
			s.hubs[e.Hub] = h
		}
	case LifecycleNoticeArm:
		s.collecting = true
	case LifecycleNoticeDrain:
		if !s.collecting {
			return s, nil
		}
		s.collecting = false
		s.reason = noticeText(e.Reason, 128)
	case LifecycleNoticeConfirmed:
		f := e.Fact
		if !validLifecycleNoticeFact(f) {
			return s, nil
		}
		key := noticeFactKey(f)
		current, exists := s.facts[key]
		if exists && f.Observation <= current.fact.Observation {
			return s, nil
		}
		if !exists && len(s.facts) >= 128 {
			if s.overflow < ^uint32(0) {
				s.overflow++
			}
			return s, nil
		}
		f.Agent = noticeText(f.Agent, 96)
		f.Model = noticeText(f.Model, 96)
		f.Branch = noticeText(f.Branch, 128)
		f.Issue = noticeText(f.Issue, 64)
		f.Mode = noticeText(f.Mode, 32)
		switch f.Initiation {
		case "operator", "peer", "automatic":
		default:
			f.Initiation = "unknown"
		}
		current.fact = f
		if current.changes < ^uint32(0) {
			current.changes++
		}
		if e.Outcome == "suppressed" {
			current.retired = f.Observation
			current.changes = 0
		}
		s.facts[key] = current
		h := s.hubs[f.Hub]
		h.suppressed = false
		if _, ok := s.hubs[f.Hub]; ok {
			s.hubs[f.Hub] = h
		}
	case LifecycleNoticeHubReady:
		if e.Hub == "" || len(e.Hub) > 4096 || len(e.Incarnation) > 512 {
			return s, nil
		}
		h, exists := s.hubs[e.Hub]
		if !exists && len(s.hubs) >= 128 {
			return s, nil
		}
		changed := h.incarnation != e.Incarnation || h.ready != e.Ready
		replacement := h.incarnation != e.Incarnation
		if replacement {
			h.unsupported = false
		}
		if changed {
			h.suppressed = false
		}
		if replacement && h.token != 0 {
			effects = append(effects, LifecycleNoticeEffect{Kind: LifecycleNoticeCancel, Hub: e.Hub, Incarnation: h.target, Token: h.token})
		}
		h.incarnation = e.Incarnation
		h.ready = e.Ready
		s.hubs[e.Hub] = h
	case LifecycleNoticeAdmission, LifecycleNoticeReceipt:
		h, ok := s.hubs[e.Hub]
		if !ok || h.token == 0 || e.Token != h.token {
			return s, nil
		}
		if e.Kind == LifecycleNoticeAdmission && e.Outcome == "accepted" {
			return s, nil
		}
		changedRecipient := h.target != h.incarnation
		submitted := e.Kind == LifecycleNoticeReceipt && e.Outcome == "submitted"
		knownNoWrite := (e.Kind == LifecycleNoticeAdmission && (e.Outcome == "refused" || e.Outcome == "unavailable" || e.Outcome == "capacity" || e.Outcome == "unsupported")) || (e.Kind == LifecycleNoticeReceipt && (e.Outcome == "expired" || e.Outcome == "cancelled"))
		// The owner-change line has no backing facts to freeze again. Keep it
		// only when the outcome proves no write; uncertainty still retires it.
		preserveInfo := knownNoWrite && len(h.parts[0].revisions) == 0
		if submitted || !knownNoWrite {
			for _, r := range h.parts[0].revisions {
				c := s.facts[r.key]
				if r.observation > c.retired {
					c.retired = r.observation
					if c.changes >= r.changes {
						c.changes -= r.changes
					} else {
						c.changes = 0
					}
					s.facts[r.key] = c
				}
			}
			h.parts = h.parts[1:]
		} else {
			if !preserveInfo {
				h.parts = nil
			}
			h.suppressed = true
		}
		if !submitted && e.Outcome != "suppressed" {
			diagnostic(e.Hub, "Lifecycle notice "+e.Outcome+"; no automatic replay")
		}
		if e.Outcome == "unsupported" {
			h.unsupported = true
		}
		if changedRecipient {
			if !preserveInfo {
				h.parts = nil
			}
			h.suppressed = false
		}
		h.token = 0
		h.target = ""
		s.hubs[e.Hub] = h
	}
	if s.collecting {
		return s, effects
	}
	hubs := make([]string, 0, len(s.hubs))
	for hub := range s.hubs {
		hubs = append(hubs, hub)
	}
	sort.Strings(hubs)
	for _, hub := range hubs {
		h := s.hubs[hub]
		if !h.ready || h.incarnation == "" || h.token != 0 || h.suppressed || h.unsupported {
			continue
		}
		if len(h.parts) == 0 {
			h.parts = freezeLifecycleNotices(s, hub)
		}
		if len(h.parts) == 0 {
			continue
		}
		s.nextToken++
		if s.nextToken == 0 {
			s.nextToken++
		}
		h.token = s.nextToken
		h.target = h.incarnation
		s.hubs[hub] = h
		effects = append(effects, LifecycleNoticeEffect{Kind: LifecycleNoticePublish, Hub: hub, Incarnation: h.target, Token: h.token, Body: h.parts[0].body})
	}
	return s, effects
}

func cloneLifecycleNotices(s LifecycleNoticeState) LifecycleNoticeState {
	facts := make(map[string]lifecycleNoticeCurrent, len(s.facts))
	for k, v := range s.facts {
		facts[k] = v
	}
	s.facts = facts
	hubs := make(map[string]lifecycleNoticeHub, len(s.hubs))
	for k, v := range s.hubs {
		hubs[k] = v
	}
	s.hubs = hubs
	return s
}
func noticeFactKey(f LifecycleNoticeFact) string {
	// Slot is stable across Fresh addresses and owning-hub moves.
	return f.Slot
}
func validLifecycleNoticeFact(f LifecycleNoticeFact) bool {
	if f.Hub == "" || f.Slot == "" || f.Incarnation == "" || f.Observation == 0 || len(f.Hub) > 4096 || len(f.Slot) > 128 || len(f.Address.RepoScope) > 4096 || len(f.Address.Tag) > 128 || len(f.Incarnation) > 512 || len(f.Operation) > 512 {
		return false
	}
	switch f.Kind {
	case LifecycleStarted, LifecycleResumed, LifecycleParked, LifecycleRelaunched, LifecycleRecovered, LifecycleNeedsRelaunch, LifecycleExited:
		return true
	}
	return false
}
func noticeText(s string, limit int) string {
	var b strings.Builder
	for _, r := range s {
		if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) {
			r = ' '
		}
		if r == utf8.RuneError {
			r = ' '
		}
		if b.Len()+utf8.RuneLen(r) > limit {
			break
		}
		b.WriteRune(r)
	}
	return strings.TrimSpace(b.String())
}
func lifecycleNoticeRow(c lifecycleNoticeCurrent) string {
	f := c.fact
	row := noticeText(f.Slot, 128) + ": " + string(f.Kind)
	if f.Mode != "" {
		row += " (" + f.Mode + ")"
	}
	for _, field := range []struct{ name, value string }{{"agent", f.Agent}, {"model", f.Model}, {"branch", f.Branch}, {"issue", f.Issue}, {"initiation", f.Initiation}} {
		if field.value != "" {
			row += "; " + field.name + "=" + field.value
		}
	}
	if c.changes > 1 {
		row += fmt.Sprintf("; %d changes", c.changes)
	}
	return row + "\n"
}
func freezeLifecycleNotices(s LifecycleNoticeState, hub string) []lifecycleNoticePart {
	keys := make([]string, 0, len(s.facts))
	for k, c := range s.facts {
		if c.fact.Hub == hub && c.fact.Observation > c.retired {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	if len(keys) == 0 {
		return nil
	}
	header := "Couch lifecycle update"
	if s.reason != "" {
		header += "; restart pass: " + s.reason
	}
	if s.overflow > 0 {
		header += fmt.Sprintf("; %d additional observations omitted; resnapshot required", s.overflow)
	}
	// Reserve ample room for the bounded metadata and multipart numbering. Rows
	// remain whole; even a maximal row is far smaller than the 8 KiB body limit.
	var parts []lifecycleNoticePart
	part := lifecycleNoticePart{}
	for _, key := range keys {
		c := s.facts[key]
		row := lifecycleNoticeRow(c)
		if len(part.body)+len(row) > 7000 {
			parts = append(parts, part)
			part = lifecycleNoticePart{}
		}
		part.body += row
		part.revisions = append(part.revisions, lifecycleNoticeRevision{key: key, observation: c.fact.Observation, changes: c.changes})
	}
	if len(part.revisions) > 0 {
		parts = append(parts, part)
	}
	for i := range parts {
		prefix := header
		if len(parts) > 1 {
			prefix += fmt.Sprintf("; part %d/%d", i+1, len(parts))
		}
		parts[i].body = prefix + "\n" + parts[i].body
	}
	return parts
}
