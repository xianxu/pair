package launcher

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/zellijreply"
)

// projectSessionOwnerReply retains actual command/identity evidence, excluding
// volatile geometry and titles which cannot establish a storage owner.
func projectSessionOwnerReply(expected artifactpath.StorageOwner, raw []byte) (string, error) {
	panes, err := zellijreply.ParsePanes(raw, zellijreply.PaneRequirements{})
	if err != nil {
		return "", err
	}
	classified := ClassifySessionOwner(expected, panes)
	if classified.State == SessionOwnerUnknown {
		return "", fmt.Errorf("unknown pane owner: %s", classified.Diagnostic)
	}
	type command struct {
		ID   string
		Args []string
	}
	var evidence []command
	for _, p := range panes {
		if p.IsPlugin {
			continue
		}
		args, err := splitObservedCommand(p.PaneCommand)
		if err != nil {
			return "", err
		}
		evidence = append(evidence, command{p.ID, args})
	}
	sort.Slice(evidence, func(i, j int) bool { return evidence[i].ID < evidence[j].ID })
	out, err := json.Marshal(evidence)
	return string(out), err
}
