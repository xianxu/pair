package testcouch

import (
	"fmt"
	"strconv"

	"github.com/xianxu/pair/cmd/internal/procutil"
)

// selectOwnedCohort takes the least closure of positive current ownership under
// same-table ancestry and controlling-terminal membership. Missing environment
// markers are never negative evidence: native shell/sleep descendants may expose
// no environment. A terminal is private only when a live owned generation in this
// snapshot witnesses it. Recycled PIDs cannot create that witness.
//
// Children and TTY buckets are indexed once, and each selected row and witnessed
// TTY bucket is expanded once. The returned resources follow census order; no IO,
// sorting, mutation or per-member census is needed.
func selectOwnedCohort(rows []procutil.Process, known OwnedRun, marked map[int]bool) ([]Resource, error) {
	knownPIDs := make(map[int]bool, len(known.Resources))
	exact := make(map[string]Resource, len(known.Resources))
	for _, resource := range known.Resources {
		pid, err := strconv.Atoi(resource.ID)
		if err != nil || pid <= 0 || resource.Generation == "" {
			continue
		}
		knownPIDs[pid] = true
		exact[strconv.Itoa(pid)+"\x00"+resource.Generation] = resource
	}
	children := make(map[int][]int, len(rows))
	terminals := make(map[string][]int)
	countPID := make(map[int]int, len(rows))
	for i, row := range rows {
		if row.Zombie || row.PID <= 0 {
			continue
		}
		children[row.PPID] = append(children[row.PPID], i)
		if row.TTY != "" {
			terminals[row.TTY] = append(terminals[row.TTY], i)
		}
		countPID[row.PID]++
	}
	selected := make([]bool, len(rows))
	queue := make([]int, 0, len(rows))
	enqueue := func(i int) error {
		if selected[i] {
			return nil
		}
		row := rows[i]
		if row.Identity == "" {
			return fmt.Errorf("identity unavailable for relevant owned cohort pid %d", row.PID)
		}
		if countPID[row.PID] != 1 {
			return fmt.Errorf("ambiguous process census for owned cohort pid %d", row.PID)
		}
		selected[i] = true
		queue = append(queue, i)
		return nil
	}
	for i, row := range rows {
		if row.Zombie || row.PID <= 0 {
			continue
		}
		_, matches := exact[strconv.Itoa(row.PID)+"\x00"+row.Identity]
		// Unknown identity at a retained PID is uncertainty, not a confirmed reuse.
		if marked[row.PID] || matches || (knownPIDs[row.PID] && row.Identity == "") {
			if err := enqueue(i); err != nil {
				return nil, err
			}
		}
	}
	witnessedTTY := make(map[string]bool, len(terminals))
	for cursor := 0; cursor < len(queue); cursor++ {
		row := rows[queue[cursor]]
		for _, child := range children[row.PID] {
			if err := enqueue(child); err != nil {
				return nil, err
			}
		}
		if row.TTY == "" || witnessedTTY[row.TTY] {
			continue
		}
		witnessedTTY[row.TTY] = true
		for _, peer := range terminals[row.TTY] {
			if err := enqueue(peer); err != nil {
				return nil, err
			}
		}
	}
	var resources []Resource
	for i, row := range rows {
		if !selected[i] {
			continue
		}
		key := strconv.Itoa(row.PID) + "\x00" + row.Identity
		if resource, ok := exact[key]; ok {
			resources = append(resources, resource)
		} else {
			resources = append(resources, resourceFor(row))
		}
	}
	return resources, nil
}
