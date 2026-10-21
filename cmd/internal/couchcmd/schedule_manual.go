package couchcmd

import (
	"context"
	"errors"
	"path/filepath"

	"github.com/xianxu/pair/cmd/internal/couchschedule"
)

func manualDeclaration(ctx context.Context, root, task string) (couchschedule.Declaration, error) {
	declarations, err := (couchschedule.Catalog{}).Discover(ctx, []string{root})
	if err != nil {
		return couchschedule.Declaration{}, err
	}
	for _, declaration := range declarations {
		if declaration.Key.Hub == root && declaration.Key.Task == task {
			return declaration, nil
		}
	}
	return couchschedule.Declaration{}, errors.New("scheduled task not found in the current hub")
}

func schedulePermitsDir(rt Runtime) (string, error) {
	if isolated := rt.Getenv("COUCH_ISOLATED_ROOT"); isolated != "" {
		root, err := canonicalFuture(isolated)
		if err != nil {
			return "", err
		}
		return filepath.Join(root, "schedule-permits"), nil
	}
	lookup := localAccountHome
	if production, ok := rt.(OSRuntime); ok && production.accountHome != nil {
		lookup = production.accountHome
	}
	home, err := lookup()
	if err != nil {
		return "", err
	}
	root, err := canonicalFuture(home)
	if err != nil {
		return "", err
	}
	return filepath.Join(root, ".local", "share", "pair-host", "schedule-permits"), nil
}

func sameReservedAttempt(current, reserved couchschedule.TaskState) bool {
	a, b := current.Active, reserved.Active
	return a != nil && b != nil && a.Stage == "reserved" && a.ControllerPID == 0 && a.ControllerStart == "" && current.Key == reserved.Key && current.Revision == reserved.Revision && current.Spec == reserved.Spec && current.Allowed && a.ID == b.ID && a.Revision == b.Revision && a.Source == b.Source && a.StartedAt.Equal(b.StartedAt) && a.Deadline.Equal(b.Deadline) && a.DueAt.Equal(b.DueAt)
}

func executionNotice(source, outcome string) string {
	if source == "manual" {
		return "manual"
	}
	if outcome == "quiet" {
		return "none"
	}
	if outcome == "unknown" {
		return "uncertain"
	}
	return "pending"
}
