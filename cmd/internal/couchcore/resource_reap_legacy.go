package couchcore

import (
	"fmt"

	"github.com/xianxu/pair/cmd/internal/procutil"
)

// LegacyOrphanTarget names one frozen tree member and its originally admitted
// legacy root. Neither identity nor family is an authority without fresh proof.
type LegacyOrphanTarget struct {
	Family string             `json:"family"`
	Root   ResourceProcessKey `json:"root"`
	Key    ResourceProcessKey `json:"key"`
}

func (t LegacyOrphanTarget) Validate() error {
	if t.Family != "orphan-nvim" && t.Family != "left-running" {
		return fmt.Errorf("unknown legacy orphan family")
	}
	if t.Root.PID <= 0 || t.Root.Identity == "" || t.Key.PID <= 0 || t.Key.Identity == "" {
		return fmt.Errorf("missing exact legacy process generation")
	}
	return nil
}

// DecideLegacyOrphanSignal permits only a currently eligible, wholly unowned
// tree's exact member. It discovers no new effect targets. The caller must still
// hold the revocation/ownership gates and recheck both native identities.
func DecideLegacyOrphanSignal(target LegacyOrphanTarget, proof ProvenOrphanProof, view ResourceView) (string, string) {
	if err := target.Validate(); err != nil {
		return "refused", err.Error()
	}
	if !proof.TableKnown || !proof.InventoryComplete || proof.IncompleteReason != "" || len(proof.Table) > ResourceMaxTableRows || len(proof.Selection.Processes) > ResourceMaxProcesses || len(proof.Selection.Errors) != 0 || len(proof.UnknownDetached) != 0 {
		return "refused", "complete current legacy ownership proof unavailable"
	}
	rows := make(map[int]procutil.Process, len(proof.Table))
	children := make(map[int][]int, len(proof.Table))
	for _, row := range proof.Table {
		if _, duplicate := rows[row.PID]; duplicate || row.PID <= 0 || !row.Zombie && row.Identity == "" {
			return "refused", "current process identity is ambiguous or unavailable"
		}
		rows[row.PID] = row
		children[row.PPID] = append(children[row.PPID], row.PID)
	}
	// A complete census cannot contain an ancestry cycle. Each row becomes
	// finished once, so rejecting even an unrelated cycle stays linear in P.
	ancestry := make(map[int]uint8, len(rows))
	for pid := range rows {
		var path []int
		for {
			row, present := rows[pid]
			if !present || ancestry[pid] == 2 {
				break
			}
			if ancestry[pid] == 1 {
				return "refused", "current process ancestry is cyclic"
			}
			ancestry[pid] = 1
			path = append(path, pid)
			pid = row.PPID
		}
		for _, pid := range path {
			ancestry[pid] = 2
		}
	}
	process, present := rows[target.Key.PID]
	if !present || process.Zombie || process.Identity != target.Key.Identity {
		return "skipped", "original legacy process is gone or changed"
	}
	root, present := rows[target.Root.PID]
	if !present || root.Zombie || root.Identity != target.Root.Identity {
		return "refused", "legacy root is gone or changed; surviving child has no permit"
	}
	eligible := 0
	switch target.Family {
	case "orphan-nvim":
		for _, row := range view.OrphanNvim {
			if row.Key.PID == target.Root.PID {
				if row.Key != target.Root {
					return "refused", "legacy root family identity changed"
				}
				eligible++
			}
		}
	case "left-running":
		for _, row := range view.LeftRunning {
			if row.Process.PID == target.Root.PID {
				if row.Process.Identity != target.Root.Identity {
					return "refused", "legacy root family identity changed"
				}
				eligible++
			}
		}
	}
	if eligible != 1 {
		return "refused", "exact root is not uniquely eligible in its current legacy family"
	}
	protected := map[int]bool{}
	for _, p := range proof.Selection.Processes {
		protected[p.Key.PID] = true
	}
	for _, p := range view.Processes {
		protected[p.Key.PID] = true
	}
	for _, p := range view.Orphans {
		protected[p.Key.PID] = true
	}
	// Walk the complete target ancestry, including above the admitted root. A
	// newly owned ancestor or a cycle cannot be treated as proof of detachment.
	seen := map[int]bool{}
	reachesRoot := false
	for cur := process; ; {
		if seen[cur.PID] || protected[cur.PID] || cur.Identity == "" || cur.Zombie {
			return "refused", "legacy ancestry is protected, cyclic or unknown"
		}
		seen[cur.PID] = true
		if resourceKey(cur) == target.Root {
			reachesRoot = true
		}
		parent, ok := rows[cur.PPID]
		if !ok || cur.PPID <= 0 {
			break
		}
		cur = parent
	}
	if !reachesRoot {
		return "refused", "legacy member no longer descends from its exact root"
	}
	// A protected child must also veto a root/ancestor signal: ownership cannot
	// be bypassed by selecting a different member of the same frozen tree.
	seen = map[int]bool{}
	queue := []int{root.PID}
	for len(queue) > 0 {
		pid := queue[0]
		queue = queue[1:]
		if seen[pid] || protected[pid] {
			return "refused", "legacy tree contains protected ownership or cyclic ancestry"
		}
		seen[pid] = true
		queue = append(queue, children[pid]...)
	}
	return "sent", ""
}
