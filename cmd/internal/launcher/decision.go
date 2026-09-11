package launcher

import (
	"fmt"
	"strconv"
)

// LaunchAction is the guarded prototype's next launcher action.
type LaunchAction string

const (
	ActionAttach LaunchAction = "attach"
	ActionCreate LaunchAction = "create"
	ActionPick   LaunchAction = "pick"
)

// LaunchDecision is a pure create/attach/pick decision. Tag is canonical bare
// form; SessionName is the assigned public zellij session name when known.
type LaunchDecision struct {
	Action       LaunchAction
	Tag          string
	SessionName  string
	PromptName   bool
	LegacyImport bool
	ContinueDoc  string
	ContinueText string
	SourceAgent  string
}

// launchShape is which family of branch DecideLaunch takes for an args shape.
// It is the ONE switch both DecideLaunch and runOnce read (pair#228): runOnce
// takes a liveness snapshot -- no list-clients -- exactly when the shape's
// branch reads nothing but liveness, and only shapeBare reads attach state
// (hasDetached, then the picker). Two conditions that merely agree would drift;
// DecideLaunch's branches are this switch's arms.
type launchShape uint8

const (
	shapeBare launchShape = iota
	shapeSelected
	shapeForced
	shapeExplicitArgs
)

// launchShapeOf tests in the order DecideLaunch always has. The explicit-args
// arm keys on AgentArgsExplicit, NOT AgentExplicit: `pair -- x` defaults the
// agent but types the args, and must create rather than pick.
func launchShapeOf(args LaunchArgs) launchShape {
	switch {
	case args.SelectedTag != "":
		return shapeSelected
	case args.ForcedTag != "":
		return shapeForced
	case args.Agent != "" && args.AgentArgsExplicit:
		return shapeExplicitArgs
	default:
		return shapeBare
	}
}

// decisionNeedsAttachState reports whether DecideLaunch's branch for args
// distinguishes attached from detached sessions -- the only reason to pay for
// a full snapshot's list-clients calls.
func decisionNeedsAttachState(args LaunchArgs) bool { return launchShapeOf(args) == shapeBare }

// DecideLaunch decides the launch action without touching zellij, fzf, or disk.
func DecideLaunch(args LaunchArgs, snap SessionSnapshot) (LaunchDecision, error) {
	switch launchShapeOf(args) {
	case shapeSelected:
		return createDecision(args.SelectedTag, sessionNameForTag(snap, args.SelectedTag), false), nil
	case shapeForced:
		name := sessionNameForTag(snap, args.ForcedTag)
		if sessionBlocksReuse(snap, name) {
			return LaunchDecision{Action: ActionAttach, Tag: args.ForcedTag, SessionName: name}, nil
		}
		return createDecision(args.ForcedTag, name, false), nil
	case shapeExplicitArgs:
		tag := snap.BaseTag
		if tag == "" {
			tag = "pair"
		}
		tag = nextFreeTag(tag, snap)
		return createDecision(tag, sessionNameForTag(snap, tag), true), nil
	}
	// shapeBare: the only branch that reads attach state. A snapshot that never
	// asked would read "not detached", skip the picker, and mint a new session.
	// runOnce takes a full snapshot for this shape, so this fires only on drift.
	if err := RequireAttachState(snap.Sessions); err != nil {
		return LaunchDecision{}, fmt.Errorf("launch decision: %w", err)
	}
	if hasDetached(snap) || len(snap.Historical) > 0 {
		return LaunchDecision{Action: ActionPick}, nil
	}
	tag := snap.BaseTag
	if tag == "" {
		tag = "pair"
	}
	tag = nextFreeTag(tag, snap)
	return createDecision(tag, sessionNameForTag(snap, tag), true), nil
}

func createDecision(tag, session string, prompt bool) LaunchDecision {
	return LaunchDecision{Action: ActionCreate, Tag: tag, SessionName: session, PromptName: prompt}
}

func sessionName(tag string) string {
	// Legacy degraded fallback. sessionNameForTag prefers snap.SessionNames,
	// which assignLaunchSessionNames fills with composed names; this path is
	// reached for tags outside that set (nextFreeTag's base-2/base-3 probes) and
	// when RepoScope resolution failed. SessionSnapshot carries no scope, so
	// there is nothing to compose from — and the value is only ever COMPARED
	// against the snapshot, never minted as a real session (#130).
	return legacySessionPrefix + tag
}

func sessionNameForTag(snap SessionSnapshot, tag string) string {
	if snap.SessionNames != nil {
		if name := snap.SessionNames[tag]; name != "" {
			return name
		}
	}
	return sessionName(tag)
}

func hasDetached(snap SessionSnapshot) bool {
	for _, sess := range snap.Sessions {
		if sess.State == SessionDetached {
			return true
		}
	}
	return false
}

func sessionBlocksReuse(snap SessionSnapshot, name string) bool {
	for _, sess := range snap.Sessions {
		if sess.Name != name {
			continue
		}
		// SessionLive is "not exited, attach state not asked": it blocks reuse
		// exactly as attached or detached do, which is all this reads.
		return sess.State == SessionAttached || sess.State == SessionDetached || sess.State == SessionLive
	}
	return false
}

func nextFreeTag(base string, snap SessionSnapshot) string {
	for i := 1; i <= 100; i++ {
		tag := base
		if i > 1 {
			tag = base + "-" + strconv.Itoa(i)
		}
		if !sessionBlocksReuse(snap, sessionNameForTag(snap, tag)) && !isHistorical(snap, tag) {
			return tag
		}
	}
	return base
}

func isHistorical(snap SessionSnapshot, tag string) bool {
	for _, hist := range snap.Historical {
		if hist.Tag == tag {
			return true
		}
	}
	return false
}
