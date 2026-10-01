package couchmessage

import (
	"errors"
	"testing"
	"time"
)

func TestRouting(t *testing.T) {
	now := time.Unix(100, 0)
	ready := Candidate{Binding: binding("pair:1"), Resting: true, Known: true, LastActivity: now.Add(-30 * time.Second), Remaining: 8, Supported: true}
	cases := []struct {
		name, target string
		change       func(*Candidate)
		want         error
	}{
		{"boundary", "pair", nil, nil}, {"exact busy work", "pair:1", func(c *Candidate) { c.Resting = false; c.LastActivity = now }, nil},
		{"too recent", "pair", func(c *Candidate) { c.LastActivity = c.LastActivity.Add(time.Nanosecond) }, ErrNoRecipient},
		{"issue branch", "pair", func(c *Candidate) { c.Resting = false }, ErrNoRecipient},
		{"unknown", "pair", func(c *Candidate) { c.Known = false }, ErrNoRecipient},
		{"pending", "pair:1", func(c *Candidate) { c.Pending = true }, ErrRecipientBusy},
		{"budget", "pair:1", func(c *Candidate) { c.Remaining = 0 }, ErrBudgetExhausted},
		{"unsupported", "pair:1", func(c *Candidate) { c.Supported = false }, ErrUnsupported},
		{"missing", "pair:2", nil, ErrUnavailable}, {"bad target", "pair:01", nil, ErrInvalidTarget},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := ready
			if tc.change != nil {
				tc.change(&c)
			}
			_, err := ResolveRecipient(Route{Target: tc.target}, []Candidate{c}, nil, now)
			if !errors.Is(err, tc.want) {
				t.Fatalf("%v want %v", err, tc.want)
			}
		})
	}
	zero := ready
	zero.Binding = binding("pair:0")
	got, err := ResolveRecipient(Route{Target: "pair"}, []Candidate{ready, zero}, nil, now)
	if err != nil || got.Slot != "pair:0" {
		t.Fatalf("lowest slot %v %v", got, err)
	}
	duplicate := ready
	duplicate.Binding.Repository = "/other/.git"
	if _, err := ResolveRecipient(Route{Target: "pair"}, []Candidate{ready, duplicate}, nil, now); !errors.Is(err, ErrAmbiguous) {
		t.Fatalf("ambiguous %v", err)
	}
}
