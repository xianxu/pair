package couchscratch

import (
	"context"
	"strings"
	"sync"
	"time"
)

// Options supplies observations and scheduling without global environment edits.
type Options struct {
	Probe  func() Probe
	Now    func() time.Time
	Notify func(string)
	Ticks  <-chan time.Time
}

// Manager prepares slot scratch and serializes age sweeps.
type Manager struct {
	options      Options
	mu           sync.Mutex
	beforeRemove func()
}

// New constructs a manager with system defaults for omitted observations.
func New(o Options) *Manager {
	if o.Probe == nil {
		o.Probe = func() Probe { return ReadProbe(DefaultRoot) }
	}
	if o.Now == nil {
		o.Now = time.Now
	}
	return &Manager{options: o}
}

// note reports an optional scratch fallback diagnostic.
func (m *Manager) note(s string) {
	if m.options.Notify != nil {
		s = strings.ReplaceAll(strings.ReplaceAll(s, "\n", " "), "\r", " ")
		if len(s) > 512 {
			s = s[:509] + "..."
		}
		m.options.Notify(s)
	}
}

// Run performs one startup sweep and hourly sweeps until cancellation. It returns
// only after its current sweep stops; each manager runs sweeps serially.
func (m *Manager) Run(ctx context.Context) {
	m.Sweep(ctx)
	m.Maintain(ctx)
}

// Maintain performs hourly sweeps until cancellation without repeating startup
// cleanup. It joins each sweep synchronously and creates no background worker.
func (m *Manager) Maintain(ctx context.Context) {
	ticks := m.options.Ticks
	var timer *time.Ticker
	if ticks == nil {
		timer = time.NewTicker(time.Hour)
		defer timer.Stop()
		ticks = timer.C
	}
	for {
		select {
		case <-ctx.Done():
			return
		case _, ok := <-ticks:
			if !ok {
				return
			}
			m.Sweep(ctx)
		}
	}
}
