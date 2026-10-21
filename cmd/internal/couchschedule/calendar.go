package couchschedule

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/robfig/cron/v3"
)

// Calendar computes future due instants in one injected machine location.
// A repeated local time is eligible only during its first occurrence.
type Calendar struct{ schedule *cron.SpecSchedule }

// NewCalendar accepts five cron fields, without descriptors or timezone overrides.
func NewCalendar(expression string, location *time.Location) (Calendar, error) {
	if location == nil {
		return Calendar{}, fmt.Errorf("calendar location is required")
	}
	if len(expression) > maxTaskSpecBytes || !utf8.ValidString(expression) || len(strings.Fields(expression)) != 5 {
		return Calendar{}, fmt.Errorf("cron requires exactly five fields")
	}
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	schedule, err := parser.Parse(expression)
	if err != nil {
		return Calendar{}, err
	}
	spec, ok := schedule.(*cron.SpecSchedule)
	if !ok {
		return Calendar{}, fmt.Errorf("cron must be a five-field calendar")
	}
	spec.Location = location
	return Calendar{schedule: spec}, nil
}

// Next returns the first eligible instant strictly after after, or zero when
// the schedule has no matching future date within the cron parser's horizon.
func (c Calendar) Next(after time.Time) time.Time {
	if c.schedule == nil {
		return time.Time{}
	}
	for {
		// cron treats time.Local as the input's location. Normalize explicitly
		// so callers supplying UTC still use the injected machine location.
		next := c.schedule.Next(after.In(c.schedule.Location))
		if next.IsZero() {
			return next
		}
		// ZoneBounds identifies a rollback even when the caller starts inside
		// the second fold, without storing prior calls or reading ambient time.
		local := next.In(c.schedule.Location)
		start, _ := local.ZoneBounds()
		if !start.IsZero() {
			_, offset := local.Zone()
			_, previous := start.Add(-time.Nanosecond).In(c.schedule.Location).Zone()
			if rollback := time.Duration(previous-offset) * time.Second; rollback > 0 && next.Before(start.Add(rollback)) {
				after = start.Add(rollback).Add(-time.Nanosecond)
				continue
			}
		}
		return next
	}
}
