package couchcmd

import (
	"fmt"

	"github.com/xianxu/pair/cmd/internal/couchschedule"
)

type scheduleArgs struct {
	verb  string
	task  string
	runID string
	args  []string
	json  bool
}

const scheduleUsage = "couch --schedule --status [--json]\ncouch --schedule run TASK -- [ARGS...]\ncouch --schedule history TASK [--run ID] [--json]"

// parseScheduleCLI runs before layout parsing: everything after run's separator
// belongs to the task, including flags understood by Couch itself.
func parseScheduleCLI(args []string) (cliInvocation, error) {
	invalid := func() (cliInvocation, error) {
		return cliInvocation{}, fmt.Errorf("usage:\n%s", scheduleUsage)
	}
	if len(args) < 2 {
		return invalid()
	}
	a := &scheduleArgs{}
	switch args[1] {
	case "--status":
		if len(args) > 3 || len(args) == 3 && args[2] != "--json" {
			return invalid()
		}
		a.verb, a.json = "status", len(args) == 3
	case "run":
		if len(args) < 4 || !validScheduleTask(args[2]) || args[3] != "--" {
			return invalid()
		}
		a.verb, a.task = "run", args[2]
		a.args = append([]string{}, args[4:]...)
	case "history":
		if len(args) < 3 || !validScheduleTask(args[2]) {
			return invalid()
		}
		a.verb, a.task = "history", args[2]
		for i := 3; i < len(args); i++ {
			switch args[i] {
			case "--json":
				if a.json {
					return invalid()
				}
				a.json = true
			case "--run":
				if a.runID != "" || i+1 == len(args) || !validScheduleRunID(args[i+1]) {
					return invalid()
				}
				i++
				a.runID = args[i]
			default:
				return invalid()
			}
		}
	default:
		return invalid()
	}
	return cliInvocation{kind: cliSchedule, schedule: a}, nil
}

func validScheduleTask(s string) bool {
	return couchschedule.ValidTaskName(s)
}

func validScheduleRunID(s string) bool {
	return validScheduleTask(s)
}
