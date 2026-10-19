package couchcmd

import (
	"encoding/json"
	"fmt"
	"unicode/utf8"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

// projectResourceView bounds cached samples and fresh request diagnostics.
// Keep every bounded slot aggregate; divide the remaining wire budget among
// detail sections so a large process list cannot crowd out orphan evidence.
// The actual JSON encoder accounts for escaping, not just source string bytes.
func projectResourceView(v couchcore.ResourceView) couchcore.ResourceView {
	out := v
	out.Slots = append([]couchcore.ResourceSlotView(nil), v.Slots...)
	shortened := 0
	clip := func(s string) string {
		if len(s) <= 32 {
			return s
		}
		shortened++
		s = s[:29]
		for !utf8.ValidString(s) {
			s = s[:len(s)-1]
		}
		return s + "..."
	}
	for i := range out.Slots {
		s := &out.Slots[i]
		s.Slot = clip(s.Slot)
		s.Thread.RepoScope = clip(s.Thread.RepoScope)
		s.Thread.Tag = couchcore.ThreadTag(clip(string(s.Thread.Tag)))
		s.Reason = clip(s.Reason)
	}
	out.Processes = nil
	out.Orphans = nil
	out.Unattributed = nil
	out.OrphanNvim = nil
	out.LeftRunning = nil
	out.Episodes = nil
	out.Errors = nil
	if v.HostedIngest != nil {
		hosted := *v.HostedIngest
		hosted.Clients = nil
		out.HostedIngest = &hosted
	}
	base, _ := json.Marshal(couchmessage.Response{Code: "ok", Resources: &out})
	// Reserve space for omission notices and the final measured cost counters.
	budget := (couchmessage.MaxFrameBytes - len(base) - 4096) / 7
	var omitted int
	out.Processes, omitted = resourceDetailPrefix(v.Processes, budget)
	notices := []string{}
	note := func(label string, n int) {
		if n > 0 {
			notices = append(notices, fmt.Sprintf("response detail limit: %d %s omitted", n, label))
		}
	}
	note("processes", omitted)
	out.Orphans, omitted = resourceDetailPrefix(v.Orphans, budget)
	note("orphans", omitted)
	out.Unattributed, omitted = resourceDetailPrefix(v.Unattributed, budget)
	note("unattributed processes", omitted)
	out.OrphanNvim, omitted = resourceDetailPrefix(v.OrphanNvim, budget)
	note("orphan nvim", omitted)
	out.LeftRunning, omitted = resourceDetailPrefix(v.LeftRunning, budget)
	note("left-running processes", omitted)
	out.Episodes, omitted = resourceDetailPrefix(v.Episodes, budget)
	note("episodes", omitted)
	out.Errors, omitted = resourceDetailPrefix(v.Errors, budget)
	note("diagnostics", omitted)
	if v.HostedIngest != nil {
		out.HostedIngest.Clients, omitted = resourceDetailPrefix(v.HostedIngest.Clients, budget)
		out.HostedIngest.Omitted += omitted
		note("hosted clients", out.HostedIngest.Omitted)
	}
	if shortened > 0 {
		notices = append(notices, fmt.Sprintf("response detail limit: %d slot labels or reasons shortened", shortened))
	}
	out.Errors = append(out.Errors, notices...)
	return out
}

func resourceDetailPrefix[T any](rows []T, budget int) ([]T, int) {
	used := 2
	for i, row := range rows {
		encoded, err := json.Marshal(row)
		if err != nil || used+len(encoded)+1 > budget {
			return append([]T(nil), rows[:i]...), len(rows) - i
		}
		used += len(encoded) + 1
	}
	return append([]T(nil), rows...), 0
}
