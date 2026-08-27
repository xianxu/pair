package launcher

import (
	"fmt"
	"strings"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/zellijpane"
)

type layoutRecordSnapshot struct {
	raw     string
	present bool
}

func layoutRecordPath(dataDir, tag string) string {
	paths, err := artifactpath.ResolveScoped(dataDir, tag)
	if err != nil {
		return ""
	}
	return paths.WorkbenchLayout()
}

func readLayoutSelection(rt FSOps, dataDir, tag string, request LayoutRequest) (LayoutResolution, layoutRecordSnapshot) {
	raw, err := rt.ReadFile(layoutRecordPath(dataDir, tag))
	snapshot := layoutRecordSnapshot{raw: raw, present: err == nil}
	recorded, valid := ParseLayoutMode(raw)
	return ResolveLayout(request, recorded, valid), snapshot
}

func writeLayoutRecord(rt FSOps, dataDir, tag string, mode LayoutMode) error {
	return rt.WriteAtomic(layoutRecordPath(dataDir, tag), string(mode)+"\n")
}

func restoreLayoutRecord(rt FSOps, dataDir, tag string, snapshot layoutRecordSnapshot) {
	path := layoutRecordPath(dataDir, tag)
	if snapshot.present {
		_ = rt.WriteAtomic(path, snapshot.raw)
		return
	}
	rt.Remove(path)
}

// ClassifyLiveLayout recognizes the two supported workbench pane signatures.
// Layout3's signature is a right terminal in the tiled tree (#123 pivot); the
// pre-pivot signature — invisible filler covered by a floating terminal — is
// still recognized so probing a live session started by an older binary
// doesn't misclassify it as Layout2.
func ClassifyLiveLayout(panes []zellijpane.Pane) (LayoutMode, bool) {
	var agent, draft, filler, floatingTerminal, tiledTerminal bool
	for _, pane := range panes {
		command := pane.TerminalCommand
		switch {
		case strings.Contains(command, "pair wrap"):
			agent = true
		case pane.Title == "draft" || strings.Contains(command, "draft-") && strings.Contains(command, "nvim"):
			draft = true
		}
		if pane.Title == "terminal-filler" || strings.Contains(command, "tail -f /dev/null") {
			filler = true
		}
		if pane.Title == "terminal" || strings.HasPrefix(pane.Title, "[terminal") || strings.Contains(command, "pair term") {
			if pane.IsFloating {
				floatingTerminal = true
			} else {
				tiledTerminal = true
			}
		}
	}
	if agent && draft && (tiledTerminal || (filler && floatingTerminal)) {
		return Layout3, true
	}
	if agent && draft && !tiledTerminal && !filler && !floatingTerminal {
		return Layout2, true
	}
	return "", false
}

func resolveLiveLayout(rt Runtime, dataDir, tag, session string, request LayoutRequest) (LayoutResolution, error) {
	raw, err := rt.ReadFile(layoutRecordPath(dataDir, tag))
	if recorded, valid := ParseLayoutMode(raw); err == nil && valid {
		return ResolveLayout(request, recorded, true), nil
	}
	recorded, err := rt.ProbeLiveLayout(session)
	if err != nil {
		return LayoutResolution{}, fmt.Errorf("cannot detect live workbench layout: %w", err)
	}
	if err := writeLayoutRecord(rt, dataDir, tag, recorded); err != nil {
		return LayoutResolution{}, fmt.Errorf("cannot record detected live workbench layout: %w", err)
	}
	return ResolveLayout(request, recorded, true), nil
}

func sessionStillPresent(sessions []Session, name string) bool {
	for _, session := range sessions {
		if session.Name == name {
			return true
		}
	}
	return false
}
