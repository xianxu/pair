package sessionsync

import (
	"encoding/json"
	"errors"
	"os"
	"time"
)

// Gap is a slot's record that Couch failed to append a delivery: a row is
// missing for good, so "nothing owed" can't be trusted for any file whose
// horizon reaches back past it. A restart does not clear it (a restart
// recovers no row); each file acknowledges it with --accept-gap, after the
// agent has checked its recent messages by other means. It expires keepFor
// after the last failure, when every row it could hide would have been
// trimmed anyway.
type Gap struct {
	First, Last time.Time
	Cause       string
}

// MarkGap records a failed append to slot.
func (l *Ledger) MarkGap(slot string, cause error) error {
	p, err := l.path(slot, ".gap")
	if err != nil {
		return err
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	g, err := readGap(p)
	if err != nil {
		return err
	}
	now := l.now()
	if g == nil {
		g = &Gap{First: now}
	}
	g.Last, g.Cause = now, cause.Error()
	raw, err := json.Marshal(g)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(l.Dir, 0o700); err != nil {
		return err
	}
	return writeAtomic(p, raw)
}

// Gap returns slot's unexpired gap, or nil.
func (l *Ledger) Gap(slot string) (*Gap, error) {
	p, err := l.path(slot, ".gap")
	if err != nil {
		return nil, err
	}
	g, err := readGap(p)
	if err != nil || g == nil || l.now().Sub(g.Last) > keepFor {
		return nil, err
	}
	return g, nil
}

// expireGap removes an expired gap record; callers hold l.mu.
func (l *Ledger) expireGap(slot string) error {
	p, err := l.path(slot, ".gap")
	if err != nil {
		return err
	}
	g, err := readGap(p)
	if err != nil || g == nil || l.now().Sub(g.Last) <= keepFor {
		return err
	}
	if err := os.Remove(p); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func readGap(p string) (*Gap, error) {
	raw, err := os.ReadFile(p)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var g Gap
	if err := json.Unmarshal(raw, &g); err != nil {
		// An unreadable gap record is still a gap.
		return &Gap{Cause: "unreadable gap record: " + err.Error(), Last: time.Now()}, nil
	}
	return &g, nil
}
