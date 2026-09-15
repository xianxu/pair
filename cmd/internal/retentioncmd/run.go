// Package retentioncmd exposes the internal managed-use protocol for non-Go
// writers. It has no collection or deletion operations.
package retentioncmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"

	"github.com/xianxu/pair/cmd/internal/storagegc"
)

type command struct {
	action, id, role, target   string
	pid                        int
	parent                     storagegc.ProcessIdentity
	unregisteredChildrenAbsent bool
}

func parse(args []string) (command, error) {
	var cmd command
	if len(args) == 0 {
		return cmd, errors.New("expected register, release, begin, complete, unchanged or resolve-start")
	}
	cmd.action = args[0]
	allowed := map[string]bool{}
	switch cmd.action {
	case "resolve-start":
		allowed["--id"] = true
		allowed["--parent-pid"] = true
		allowed["--parent-birth"] = true
		allowed["--unregistered-children-absent"] = true
	case "register":
		allowed["--pid"] = true
		allowed["--role"] = true
		allowed["--target"] = true
	case "begin":
		allowed["--pid"] = true
		allowed["--target"] = true
	case "release", "complete", "unchanged":
		allowed["--id"] = true
	default:
		return cmd, fmt.Errorf("unknown retention action %q", cmd.action)
	}
	values := map[string]string{}
	for i := 1; i < len(args); i += 2 {
		flag := args[i]
		if !allowed[flag] || i+1 >= len(args) {
			return cmd, fmt.Errorf("invalid argument %q", flag)
		}
		if _, ok := values[flag]; ok {
			return cmd, fmt.Errorf("duplicate argument %q", flag)
		}
		if args[i+1] == "" {
			return cmd, fmt.Errorf("empty %s", flag)
		}
		values[flag] = args[i+1]
	}
	for flag := range allowed {
		if cmd.action == "register" && flag == "--target" {
			continue
		}
		if _, ok := values[flag]; !ok {
			return cmd, fmt.Errorf("missing %s", flag)
		}
	}
	cmd.id, cmd.role, cmd.target = values["--id"], values["--role"], values["--target"]
	if raw, ok := values["--pid"]; ok {
		pid, err := strconv.Atoi(raw)
		if err != nil || pid <= 0 {
			return cmd, errors.New("--pid must name a positive actual writer PID")
		}
		cmd.pid = pid
	}
	if cmd.action == "resolve-start" {
		pid, err := strconv.Atoi(values["--parent-pid"])
		if err != nil || pid <= 0 {
			return cmd, errors.New("--parent-pid must name the recorded launch parent")
		}
		if values["--unregistered-children-absent"] != "yes" {
			return cmd, errors.New("--unregistered-children-absent requires explicit yes")
		}
		cmd.parent = storagegc.ProcessIdentity{PID: pid, Birth: values["--parent-birth"]}
		cmd.unregisteredChildrenAbsent = true
	}
	return cmd, nil
}

// Run requires explicit owner environment and registers the requested process,
// never this short-lived helper. IDs on stdout are opaque operation tokens.
func Run(args []string, getenv func(string) string, stdout, stderr io.Writer) int {
	fail := func(err error) int { fmt.Fprintf(stderr, "pair retention: %v\n", err); return 1 }
	cmd, err := parse(args)
	if err != nil {
		return fail(err)
	}
	if getenv == nil {
		return fail(errors.New("explicit environment is required"))
	}
	dataDir, tag := getenv("PAIR_DATA_DIR"), getenv("PAIR_TAG")
	if dataDir == "" || tag == "" {
		return fail(errors.New("PAIR_DATA_DIR and PAIR_TAG are required"))
	}
	owner, err := storagegc.SelectedOwner(dataDir, getenv("PAIR_SCOPE_KEY"), tag)
	if err != nil {
		return fail(err)
	}
	coordinator, err := storagegc.NewCoordinator(owner.DataDir)
	if err != nil {
		return fail(err)
	}
	var process storagegc.ProcessIdentity
	if cmd.pid != 0 {
		process, err = storagegc.CurrentProcessIdentity(cmd.pid)
		if err != nil {
			return fail(err)
		}
	}
	ctx := context.Background()
	var id string
	switch cmd.action {
	case "register":
		id, err = coordinator.AcquireRoleProcessTarget(ctx, owner, process, cmd.role, getenv("PAIR_RETENTION_START_ID"), cmd.target)
	case "release":
		err = coordinator.ReleaseProcess(ctx, owner, cmd.id)
	case "begin":
		id, err = coordinator.BeginUse(ctx, owner, process, cmd.target)
	case "complete":
		err = coordinator.CompleteUse(ctx, owner, cmd.id)
	case "unchanged":
		err = coordinator.CancelUnchangedUse(ctx, owner, cmd.id)
	case "resolve-start":
		err = coordinator.ResolveAbandonedStart(ctx, owner, cmd.id, cmd.parent, cmd.unregisteredChildrenAbsent)
	}
	if err != nil {
		return fail(err)
	}
	if id != "" {
		if _, err := fmt.Fprintln(stdout, id); err != nil {
			return fail(err)
		}
	}
	return 0
}
