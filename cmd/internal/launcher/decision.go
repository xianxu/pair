package launcher

// LaunchAction is the guarded prototype's next launcher action.
type LaunchAction string

const (
	ActionAttach LaunchAction = "attach"
	ActionCreate LaunchAction = "create"
	ActionPick   LaunchAction = "pick"
)

// LaunchDecision is a pure create/attach/pick decision. Tag is canonical bare
// form; SessionName is derived as pair-<tag> when a zellij session is named.
type LaunchDecision struct {
	Action      LaunchAction
	Tag         string
	SessionName string
	PromptName  bool
}

// DecideLaunch decides the launch action without touching zellij, fzf, or disk.
func DecideLaunch(args LaunchArgs, snap SessionSnapshot) (LaunchDecision, error) {
	if args.SelectedTag != "" {
		return createDecision(args.SelectedTag, false), nil
	}
	if args.ForcedTag != "" {
		if sessionBlocksReuse(snap, sessionName(args.ForcedTag)) {
			return LaunchDecision{Action: ActionAttach, Tag: args.ForcedTag, SessionName: sessionName(args.ForcedTag)}, nil
		}
		return createDecision(args.ForcedTag, false), nil
	}
	if hasDetached(snap) || len(snap.Historical) > 0 {
		return LaunchDecision{Action: ActionPick}, nil
	}
	tag := snap.BaseTag
	if tag == "" {
		tag = "pair"
	}
	return createDecision(nextFreeTag(tag, snap), true), nil
}

func createDecision(tag string, prompt bool) LaunchDecision {
	return LaunchDecision{Action: ActionCreate, Tag: tag, SessionName: sessionName(tag), PromptName: prompt}
}

func sessionName(tag string) string {
	return "pair-" + tag
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
			tag = base + "-" + itoa(i)
		}
		if !sessionBlocksReuse(snap, sessionName(tag)) && !isHistorical(snap, tag) {
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

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
