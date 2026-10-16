package wrapcmd

import (
	"os"
	"path/filepath"
)

// groupHasLiveMember reports whether the owned group still has a running
// member, so cleanup ends as soon as it is empty instead of always spending
// the grace. Linux lets kill(-pgid, 0) succeed while our unreaped zombie leader
// is in the group, so it reads /proc instead: any member not in state Z counts.
// An unreadable /proc counts as live, which only means waiting out the grace.
func groupHasLiveMember(pgid int) bool {
	stats, err := filepath.Glob("/proc/[0-9]*/stat")
	if err != nil {
		return true
	}
	for _, path := range stats {
		raw, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		if procStatLiveInGroup(raw, pgid) {
			return true
		}
	}
	return false
}
