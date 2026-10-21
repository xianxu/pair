package workbenchshortcut

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/xianxu/pair/cmd/internal/adapt"
	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/diagnosticlog"
	"github.com/xianxu/pair/cmd/internal/procutil"
)

type terminalOwnerState uint8

const (
	terminalOwnerUnavailable terminalOwnerState = iota
	terminalOwnerPair
	terminalOwnerOther
)

type terminalOwnerEvidence struct {
	state  terminalOwnerState
	reason string
}

func terminalOwnerArgs(args []string) bool {
	return len(args) == 2 && args[1] == "term" && (filepath.Base(args[0]) == "pair" || filepath.Base(args[0]) == "pair-go")
}

func inspectTerminalOwner(source procutil.Source, pid int) terminalOwnerEvidence {
	before := source.Identity(pid)
	if before == "" {
		return terminalOwnerEvidence{terminalOwnerUnavailable, "identity-before-unavailable"}
	}
	args, ok := source.Command(pid)
	after := source.Identity(pid)
	if after == "" {
		return terminalOwnerEvidence{terminalOwnerUnavailable, "identity-after-unavailable"}
	}
	if after != before {
		return terminalOwnerEvidence{terminalOwnerOther, "identity-changed"}
	}
	if !ok || len(args) == 0 {
		return terminalOwnerEvidence{terminalOwnerUnavailable, "argv-unavailable"}
	}
	if terminalOwnerArgs(args) {
		return terminalOwnerEvidence{state: terminalOwnerPair}
	}
	return terminalOwnerEvidence{state: terminalOwnerOther}
}

// LiveOwnerIDs excludes confirmed native shell handoffs from live registrations.
// Unavailable inspection preserves the existing liveness policy and records why.
func (r TerminalPaneRegistry) LiveOwnerIDs(alive func(int) bool, source procutil.Source) ([]string, error) {
	return r.liveOwnerIDs(alive, source, r.logOwnerUnavailable)
}

func (r TerminalPaneRegistry) liveOwnerIDs(alive func(int) bool, source procutil.Source, report func(string)) ([]string, error) {
	reasons := map[string]int{}
	ids, err := r.LiveIDs(func(pid int) bool {
		if !alive(pid) {
			return false
		}
		evidence := inspectTerminalOwner(source, pid)
		if evidence.state == terminalOwnerUnavailable {
			reasons[evidence.reason]++
		}
		return evidence.state != terminalOwnerOther
	})
	if len(reasons) > 0 && report != nil {
		var detail []string
		for reason, count := range reasons {
			detail = append(detail, fmt.Sprintf("%s=%d", reason, count))
		}
		sort.Strings(detail)
		report(strings.Join(detail, ","))
	}
	return ids, err
}

func (r TerminalPaneRegistry) logOwnerUnavailable(detail string) {
	tag := r.Tag
	if tag == "" {
		tag = "pair"
	}
	paths, err := artifactpath.ResolveScoped(r.DataDir, tag)
	if err != nil {
		return
	}
	opts := diagnosticlog.EnvironmentOptions(func(key string) string {
		if key == "PAIR_DATA_DIR" {
			return paths.ScopeDir()
		}
		return ""
	})
	w, err := diagnosticlog.Open(paths.AdaptLog(), opts)
	if err != nil {
		return
	}
	defer w.Close()
	adapt.New(w, "terminal-owner", "").Log(0, "inspection-unavailable", adapt.Bypass, detail)
}
