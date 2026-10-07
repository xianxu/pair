package launcher

import (
	"fmt"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/zellijpane"
)

type SessionOwnerState uint8

const (
	SessionOwnerUnknown SessionOwnerState = iota
	SessionOwnerAbsent
	SessionOwnerOwned
	SessionOwnerForeign
	// SessionOwnerOrphaned: the exact server is alive but its socket is gone,
	// so nothing can reach it -- and its agent may still be writing (#399).
	SessionOwnerOrphaned
)

// OrphanDiagnostic is the one sentence every surface shows for an orphan.
func OrphanDiagnostic(session string, pid int) string {
	return fmt.Sprintf("%s: server PID %d lost its socket — Tab → recover", session, pid)
}

// SessionServerIdentity witnesses one server generation, not just its name.
// Socket is the path from the server's own argv: the only evidence that tells a
// reachable server from an orphaned one (#399).
type SessionServerIdentity struct {
	PID               int
	Identity, Session string
	Socket            string
}

type SessionOwnerObservation struct {
	State      SessionOwnerState
	Name       string
	Owner      artifactpath.StorageOwner
	Server     SessionServerIdentity
	Diagnostic string
}

func sessionExpectedOwner(root, scope, tag string) (artifactpath.StorageOwner, error) {
	return artifactpath.NewStorageOwner(root, scope, tag)
}

// ClassifySessionOwner requires positive live-command evidence. Layout templates,
// pane titles and cached sidecars are deliberately not ownership authority.
func ClassifySessionOwner(expected artifactpath.StorageOwner, panes []zellijpane.Pane) SessionOwnerObservation {
	unknown := func(why string) SessionOwnerObservation { return SessionOwnerObservation{Diagnostic: why} }
	var found *artifactpath.StorageOwner
	for _, pane := range panes {
		if pane.IsPlugin {
			continue
		}
		args, err := splitObservedCommand(pane.PaneCommand)
		if err != nil || len(args) == 0 {
			return unknown("pane has no unambiguous actual command")
		}
		if args[0] == "exec" {
			args = args[1:]
		}
		if len(args) == 0 {
			return unknown("empty pane executable")
		}
		var paths []string
		switch filepath.Base(args[0]) {
		case "nvim":
			paths = editorOwnerPaths(args[1:])
		case "pair-wrap":
			paths = wrapperOwnerPaths(args[1:])
		case "pair":
			if len(args) > 1 && args[1] == "wrap" {
				paths = wrapperOwnerPaths(args[2:])
			}
		}
		for _, path := range paths {
			owner, ok := artifactpath.SessionArtifactOwner(expected.DataDir, path, AgentInventory())
			if !ok {
				continue
			}
			// Editors only witness drafts, wrappers only witness their raw capture.
			p, err := artifactpath.ResolveScoped(owner.Directory(), owner.Tag)
			if err != nil {
				continue
			}
			if filepath.Base(args[0]) == "nvim" && path != p.Draft() {
				continue
			}
			if filepath.Base(args[0]) != "nvim" && path == p.Draft() {
				continue
			}
			if found != nil && *found != owner {
				return unknown("live panes name conflicting Pair owners")
			}
			copy := owner
			found = &copy
		}
	}
	if found == nil {
		return unknown("no live Pair wrapper or draft owner evidence")
	}
	state := SessionOwnerForeign
	if *found == expected {
		state = SessionOwnerOwned
	}
	return SessionOwnerObservation{State: state, Owner: *found}
}

func editorOwnerPaths(args []string) []string {
	var out []string
	options := true
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if options {
			switch arg {
			case "--":
				options = false
				continue
			case "-u", "-U", "-i", "-c", "--cmd", "-s", "-S", "-w", "-W", "--startuptime", "--listen", "-t", "-q":
				i++
				if i >= len(args) {
					return nil
				}
				continue
			}
			if strings.HasPrefix(arg, "-") || strings.HasPrefix(arg, "+") {
				continue
			}
		}
		if filepath.IsAbs(arg) {
			out = append(out, arg)
		}
	}
	return out
}

func wrapperOwnerPaths(args []string) []string {
	var out []string
	for len(args) > 0 && strings.HasPrefix(args[0], "-") {
		switch args[0] {
		case "--scrollback-log":
			if len(args) < 2 {
				return nil
			}
			// The wrapper replaces this setting if repeated.
			out = []string{args[1]}
			args = args[2:]
		case "--from-launch-env":
			args = args[1:]
		case "--":
			return out
		default:
			return nil
		}
	}
	return out
}

// splitObservedCommand decodes quoting without evaluating shell expressions.
// Operators and expansions are ambiguous observations, never executable input.
func splitObservedCommand(raw string) ([]string, error) {
	var args []string
	var word strings.Builder
	var quote rune
	escaped, started := false, false
	flush := func() {
		if started {
			args = append(args, word.String())
			word.Reset()
			started = false
		}
	}
	for _, r := range raw {
		if escaped {
			if quote == '"' && !strings.ContainsRune("$`\"\\\n", r) {
				word.WriteRune('\\')
			}
			if r == '\n' {
				escaped = false
				continue
			}
			word.WriteRune(r)
			escaped = false
			started = true
			continue
		}
		if quote == '\'' {
			if r == '\'' {
				quote = 0
			} else {
				word.WriteRune(r)
			}
			continue
		}
		if r == '\\' {
			escaped = true
			started = true
			continue
		}
		if quote == '"' {
			if r == '"' {
				quote = 0
			} else if r == '$' || r == '`' {
				return nil, fmt.Errorf("unresolved expansion")
			} else {
				word.WriteRune(r)
			}
			continue
		}
		if r == '\'' || r == '"' {
			quote = r
			started = true
			continue
		}
		if unicode.IsSpace(r) {
			flush()
			continue
		}
		if strings.ContainsRune(";|&<>`$", r) {
			return nil, fmt.Errorf("ambiguous command operator")
		}
		word.WriteRune(r)
		started = true
	}
	if quote != 0 || escaped {
		return nil, fmt.Errorf("incomplete command quoting")
	}
	flush()
	return args, nil
}
