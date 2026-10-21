package couchcmd

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/xianxu/pair/cmd/internal/couchschedule"
	"github.com/xianxu/pair/cmd/internal/operationalrun"
	"github.com/xianxu/pair/cmd/internal/rowtext"
)

func scheduledRunID(s couchschedule.TaskState) string {
	h := sha256.Sum256([]byte(s.Key.ID() + "\x00" + s.Revision + "\x00" + s.NextDue.UTC().Format(time.RFC3339Nano)))
	return hex.EncodeToString(h[:16])
}

func scheduleRunKey(s couchschedule.TaskState) operationalrun.RunKey {
	return operationalrun.RunKey{Hub: s.Key.HubID(), Task: s.Key.Task, ID: s.Last.ID}
}

func retainedUnknownSummary(s couchschedule.TaskState) bool {
	return s.Active != nil && s.Active.Stage == "uncertain" && s.Last != nil && s.Last.ID == s.Active.ID && s.Last.Outcome == "unknown"
}

func interleaveScheduleHubs(keys []string, tasks map[string]couchschedule.TaskState, cursor int) []string {
	var queues [][]string
	for _, key := range keys {
		if len(queues) == 0 || tasks[queues[len(queues)-1][0]].Key.Hub != tasks[key].Key.Hub {
			queues = append(queues, []string{key})
		} else {
			queues[len(queues)-1] = append(queues[len(queues)-1], key)
		}
	}
	if len(queues) == 0 {
		return nil
	}
	n := cursor % len(queues)
	queues = append(queues[n:], queues[:n]...)
	out := make([]string, 0, len(keys))
	for len(queues) > 0 {
		var remaining [][]string
		for _, q := range queues {
			out = append(out, q[0])
			if len(q) > 1 {
				remaining = append(remaining, q[1:])
			}
		}
		queues = remaining
	}
	return out
}

func containsScheduleRoot(roots []string, root string) bool {
	i := sort.SearchStrings(roots, root)
	return i < len(roots) && roots[i] == root
}

func scheduleNoticeBody(s couchschedule.TaskState, output []byte) string {
	body := fmt.Sprintf("Scheduled task %s is due. Run couch --schedule run %s -- --help for its task interface.", s.Key.Task, s.Key.Task)
	if s.Spec.ScriptRun && s.Last != nil {
		body = fmt.Sprintf("Scheduled task %s finished: %s. Inspect couch --schedule history %s --run %s for output.", s.Key.Task, s.Last.Outcome, s.Key.Task, s.Last.ID)
	}
	if s.Last != nil {
		body += "\nDue: " + s.Last.DueAt.UTC().Format(time.RFC3339)
	}
	if s.Spec.ScriptRun && len(output) > 0 {
		text := rowtext.Sanitize(strings.ToValidUTF8(string(output), "�"))
		if len(text) > 6000 {
			text = text[len(text)-6000:]
			for !utf8.ValidString(text) {
				text = text[1:]
			}
		}
		body += "\nOutput:\n" + text
	}
	body = rowtext.Sanitize(body)
	if len(body) > 8192 {
		body = body[:8192]
		for !utf8.ValidString(body) {
			body = body[:len(body)-1]
		}
	}
	return body
}

// admitNotice uses one persisted budget for queue/gate/authority/broker waits.
// The adapter takes the hub-effect gate first, then this callback holds the
// scheduler authority lock until SendSystem has returned. No path may take
// those locks in reverse. Receipt reconciliation happens after both release.

func sameNoticeAuthority(state, want couchschedule.TaskState) bool {
	at := want.Active
	return at != nil && state.Allowed && state.Revision == at.Revision && state.Active != nil && state.Active.ID == at.ID && state.Active.Stage == "delivery" && state.Active.ReceiptID == at.ReceiptID && state.Active.NoticeBody == at.NoticeBody && string(state.Active.NoticeBinding) == string(at.NoticeBinding) && state.Active.NoticeDeadline.Equal(at.NoticeDeadline)
}

// scheduleTimer is a single cancelable clock seam; tests drive exact instants.

type scheduleTimer interface {
	C() <-chan time.Time
	Stop()
}

type scheduleClock interface {
	Now() time.Time
	NewTimer(time.Duration) scheduleTimer
}

type nativeScheduleClock struct{}

func (nativeScheduleClock) Now() time.Time { return time.Now() }

func (nativeScheduleClock) NewTimer(d time.Duration) scheduleTimer {
	return nativeScheduleTimer{time.NewTimer(d)}
}

type nativeScheduleTimer struct{ *time.Timer }

func (t nativeScheduleTimer) C() <-chan time.Time { return t.Timer.C }

func (t nativeScheduleTimer) Stop() { t.Timer.Stop() }
