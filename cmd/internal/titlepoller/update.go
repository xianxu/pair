package titlepoller

// TitleUpdateTarget names one title surface in an exact launch generation.
// Generation is opaque to this package; callers must include all launch identity
// fields and distinguish frame and workspace surfaces. PaneID distinguishes
// panes within that generation (and may be empty for a workspace).
type TitleUpdateTarget struct {
	Generation string
	PaneID     string
}

// TitleUpdate is the desired title of a current target.
type TitleUpdate struct {
	Target TitleUpdateTarget
	Title  string
}

// TitleUpdateEffect requests a write. Return this exact value with its outcome
// to confirm it; merely requesting a write never confirms the title.
type TitleUpdateEffect struct {
	ID     uint64
	Target TitleUpdateTarget
	Title  string
}

// TitleUpdateResult acknowledges a write. False includes an unknown outcome.
type TitleUpdateResult struct {
	Effect  TitleUpdateEffect
	Success bool
}

type TitleUpdateEventKind uint8

const (
	// RefreshTitleUpdates supplies the complete current desired-title snapshot.
	// Missing targets are removed, including all their pending confirmations.
	RefreshTitleUpdates TitleUpdateEventKind = iota
	// CompleteTitleUpdate acknowledges the latest write for a current target.
	CompleteTitleUpdate
)

type TitleUpdateEvent struct {
	Kind   TitleUpdateEventKind
	Titles []TitleUpdate
	Result TitleUpdateResult
}

// TitleUpdateState keeps at most one record per current target and no history.
// Its zero value is ready to use. A single owner must serialize events.
type TitleUpdateState struct {
	targets map[TitleUpdateTarget]titleUpdateRecord
	nextID  uint64
}

type titleUpdateRecord struct {
	effect    TitleUpdateEffect
	confirmed bool
}

// AdvanceTitleUpdate is the IO-free title update decision boundary. A refresh
// retries every unconfirmed title, including writes with missing completions.
// Completions for an older request or a removed/replaced target are ignored.
func AdvanceTitleUpdate(state *TitleUpdateState, event TitleUpdateEvent) []TitleUpdateEffect {
	switch event.Kind {
	case CompleteTitleUpdate:
		result := event.Result
		record, ok := state.targets[result.Effect.Target]
		if ok && record.effect == result.Effect {
			record.confirmed = result.Success
			state.targets[result.Effect.Target] = record
		}
		return nil
	case RefreshTitleUpdates:
		// Replacing the map bounds retained state to this snapshot. If callers
		// accidentally repeat a target, the final desired title wins.
		desired := make(map[TitleUpdateTarget]string, len(event.Titles))
		for _, title := range event.Titles {
			desired[title.Target] = title.Title
		}
		next := make(map[TitleUpdateTarget]titleUpdateRecord, len(desired))
		var effects []TitleUpdateEffect
		for _, title := range event.Titles {
			if _, seen := next[title.Target]; seen {
				continue
			}
			title.Title = desired[title.Target]
			record := state.targets[title.Target]
			if !record.confirmed || record.effect.Title != title.Title {
				state.nextID++
				record = titleUpdateRecord{effect: TitleUpdateEffect{
					ID: state.nextID, Target: title.Target, Title: title.Title,
				}}
				effects = append(effects, record.effect)
			}
			next[title.Target] = record
		}
		state.targets = next
		return effects
	default:
		return nil
	}
}
