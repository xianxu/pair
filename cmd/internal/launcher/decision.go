package launcher

import "strconv"

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
}

// DecideLaunch decides the launch action without touching zellij, fzf, or disk.
func DecideLaunch(args LaunchArgs, snap SessionSnapshot) (LaunchDecision, error) {
	if args.SelectedTag != "" {
		return createDecision(args.SelectedTag, sessionNameForTag(snap, args.SelectedTag), false), nil
	}
	if args.ForcedTag != "" {
		name := sessionNameForTag(snap, args.ForcedTag)
		if sessionBlocksReuse(snap, name) {
			return LaunchDecision{Action: ActionAttach, Tag: args.ForcedTag, SessionName: name}, nil
		}
		return createDecision(args.ForcedTag, name, false), nil
	}
	if args.Agent != "" && args.AgentArgsExplicit {
		tag := snap.BaseTag
		if tag == "" {
			tag = "pair"
		}
		tag = nextFreeTag(tag, snap)
		return createDecision(tag, sessionNameForTag(snap, tag), true), nil
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
		return sess.State == SessionAttached || sess.State == SessionDetached
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
