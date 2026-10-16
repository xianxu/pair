package launcher

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/procutil"
	"github.com/xianxu/pair/cmd/internal/readiness"
	"path/filepath"
	"slices"
	"strings"
)

type recoveryNativeWrapper struct {
	Process procutil.Process
	Owner   artifactpath.StorageOwner
	Agent   string
}

func recoverySourceAncestor(table []procutil.Process, p procutil.Process, pid int, identity string) bool {
	if pid <= 0 || identity == "" {
		return false
	}
	byPID := map[int]procutil.Process{}
	for _, row := range table {
		byPID[row.PID] = row
	}
	seen := map[int]bool{}
	for p.PID > 0 && !seen[p.PID] {
		if p.PID == pid {
			return p.Identity == identity
		}
		seen[p.PID] = true
		p = byPID[p.PPID]
	}
	return false
}

func recoveryNativeWrappers(root string, snapshot recoveryTerminalSnapshot) ([]recoveryNativeWrapper, error) {
	wrappers := []recoveryNativeWrapper{}
	for _, p := range snapshot.Processes {
		// Metadata selects possible wrappers BEFORE requiring command evidence.
		// Unknown names remain candidates. argv is still the matching authority;
		// unreadable system services and terminated rows are not native writers.
		if p.Zombie || (p.Name != "" && p.Name != "pair" && p.Name != "pair-wrap") {
			continue
		}
		command, ok := snapshot.Commands[p.PID]
		if !ok || p.Identity == "" {
			return nil, fmt.Errorf("terminal stop-set process %d (%s) command ownership is incomplete; retry", p.PID, p.Name)
		}
		args, err := splitObservedCommand(command)
		if err != nil {
			if strings.Contains(command, "pair-wrap") || strings.Contains(command, "pair wrap") {
				return nil, errors.New("wrapper command ownership is ambiguous")
			}
			continue
		}
		if len(args) > 0 && args[0] == "exec" {
			args = args[1:]
		}
		if len(args) == 0 {
			continue
		}
		switch filepath.Base(args[0]) {
		case "pair-wrap":
			args = args[1:]
		case "pair":
			if len(args) < 2 || args[1] != "wrap" {
				continue
			}
			args = args[2:]
		default:
			continue
		}
		paths := wrapperOwnerPaths(args)
		if len(paths) != 1 {
			return nil, errors.New("live wrapper has no exact storage owner")
		}
		owner, ok := artifactpath.SessionArtifactOwner(root, paths[0], AgentInventory())
		if !ok {
			return nil, errors.New("live wrapper storage owner is unavailable")
		}
		resolved, err := artifactpath.ResolveScoped(owner.Directory(), owner.Tag)
		if err != nil {
			return nil, err
		}
		agent := ""
		for _, name := range AgentInventory() {
			if resolved.ScrollbackRaw(name) == paths[0] {
				agent = name
				break
			}
		}
		if agent == "" {
			return nil, errors.New("live wrapper agent ownership is unavailable")
		}
		wrappers = append(wrappers, recoveryNativeWrapper{Process: p, Owner: owner, Agent: agent})
	}
	// Process table ordering is not identity.
	slices.SortFunc(wrappers, func(a, b recoveryNativeWrapper) int { return cmp.Compare(a.Process.PID, b.Process.PID) })
	return wrappers, nil
}

// A tag-owned file is insufficient: its exact agent generation must descend
// from this observed wrapper generation before its launch ordinal is authority.
func recoveryWrapperReady(ctx context.Context, snapshot recoveryTerminalSnapshot, wrapper recoveryNativeWrapper, read func(context.Context, string, int64) ([]byte, error)) (readiness.ReadyRecord, error) {
	raw, err := read(ctx, AgentReadyPath(wrapper.Owner.Directory(), wrapper.Owner.Tag, wrapper.Agent), maxAgentReadyBytes)
	if err != nil {
		return readiness.ReadyRecord{}, fmt.Errorf("live wrapper launch association unavailable: %w", err)
	}
	ready, err := readiness.Decode(string(raw))
	if err != nil {
		return readiness.ReadyRecord{}, fmt.Errorf("live wrapper launch association invalid: %w", err)
	}
	if ready.Tag != wrapper.Owner.Tag || ready.Agent != wrapper.Agent || ready.LaunchOrdinal == 0 {
		return readiness.ReadyRecord{}, errors.New("live wrapper launch association is unresolved")
	}
	if ready.AgentIdentity == "" {
		return ready, errors.New("terminal stop-set readiness has no exact child process generation")
	}
	for _, agent := range snapshot.Processes {
		if !agent.Zombie && agent.PID == ready.PID && agent.Identity == ready.AgentIdentity && agent.PID != wrapper.Process.PID && recoverySourceAncestor(snapshot.Processes, agent, wrapper.Process.PID, wrapper.Process.Identity) {
			return ready, nil
		}
	}
	return readiness.ReadyRecord{}, errors.New("live wrapper readiness does not prove its exact process generation")
}
