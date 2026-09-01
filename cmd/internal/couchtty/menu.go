package couchtty

import (
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/xianxu/pair/cmd/internal/couchcore"
)

// MenuControl is one operator-entered switcher surface. README checks consume
// this inventory so a new key cannot ship undocumented.
type MenuControl struct {
	Keys   string
	Action string
}

var menuControls = []MenuControl{
	{Keys: "typeahead", Action: "filter"},
	{Keys: "↑↓", Action: "select"},
	{Keys: "Enter", Action: "switch/resume"},
	{Keys: "Tab", Action: "actions"},
	{Keys: "Left", Action: "back"},
	{Keys: "Right", Action: "forward"},
	{Keys: "Ctrl-Space", Action: "start"},
	{Keys: "Alt+x", Action: "park/leave"},
	{Keys: "Escape", Action: "clear/back"},
}

// MenuControls returns the shared, immutable-by-copy key inventory.
func MenuControls() []MenuControl {
	return append([]MenuControl(nil), menuControls...)
}

const (
	menuFilterLimit = 1024
	menuNameLimit   = 1024
	menuTextLimit   = 4096
)

// MenuFrameKind is a fail-safe frame vocabulary; zero is never interactive.
type MenuFrameKind uint8

const (
	MenuFrameUnknown MenuFrameKind = iota
	MenuFrameRoot
	MenuFrameActions
	MenuFrameConfirmation
	MenuFrameText
	MenuFrameStart
)

// MenuFrame owns the navigation state for exactly one menu level.
type MenuFrame struct {
	Instance             uint64
	Kind                 MenuFrameKind
	Filter               string
	SelectedAddress      couchcore.ThreadAddress
	SelectedItem         string
	Thread               couchcore.ThreadAddress
	Action               string
	Input                string
	FormField            MenuFormField
	Path                 string
	Agent                string
	AgentSticky          bool
	Generation           uint64
	PreviewPending       uint64
	PreviewAccepted      uint64
	PreviewToken         couchcore.StartGrantToken
	PreviewPath          string
	PreviewAgent         string
	PreviewAgentSource   couchcore.AgentSource
	PreviewArgvSource    couchcore.ArgvSource
	SubmitGeneration     uint64
	CompletionRequest    CompletionIdentity
	CompletionPath       string
	CompletionPending    bool
	CompletionCandidates []string
	CompletionSelected   int
	CompletionTruncated  bool
}

type MenuFormField uint8

const (
	MenuFieldUnknown MenuFormField = iota
	MenuFieldPath
	MenuFieldAgent
)

type MenuNoticeLevel uint8

const (
	MenuNoticeUnknown MenuNoticeLevel = iota
	MenuNoticeInfo
	MenuNoticeProgress
	MenuNoticeError
)

type MenuNotice struct {
	Level MenuNoticeLevel
	Text  string
	Owner MenuProgressOwner
}

type MenuProgressOwner struct {
	PreviewGeneration uint64
	OperationAttempt  uint64
	Completion        CompletionIdentity
}

func infoMenuNotice(text string) MenuNotice { return MenuNotice{Level: MenuNoticeInfo, Text: text} }
func progressMenuNotice(text string) MenuNotice {
	return MenuNotice{Level: MenuNoticeProgress, Text: text}
}
func errorMenuNotice(text string) MenuNotice { return MenuNotice{Level: MenuNoticeError, Text: text} }

// MenuState is immutable-by-copy reducer state. Frames retain identities and
// text; the inventory remains one separately owned slice.
type MenuState struct {
	Inventory         []couchcore.ActionableThreadSummary
	InventoryReady    bool
	RefreshPending    bool
	ProjectionPending bool
	// ProjectionAfterGeneration is the newest refresh already admitted when a
	// mutation committed. Only a later snapshot can represent that mutation.
	ProjectionAfterGeneration uint64
	Frames                    []MenuFrame
	ActiveAddress             couchcore.ThreadAddress
	Agents                    []string
	RootAgent                 string
	Attention                 map[couchcore.ThreadAddress][]AttentionMessage
	InFlight                  MenuOperationOrigin
	PreviewSequence           uint64
	CompletionSequence        uint64
	OperationSequence         uint64
	FrameSequence             uint64
	SpinnerPhase              uint8
	Notice                    MenuNotice
}

// MenuOperationOrigin captures the exact frame that emitted asynchronous
// work, so completion does not depend on whichever frame is visible later.
type MenuOperationOrigin struct {
	Operation        string
	Attempt          uint64
	Address          couchcore.ThreadAddress
	FrameInstance    uint64
	FrameKind        MenuFrameKind
	Depth            int
	AttentionCapture AttentionCapture
}

type MenuEventKind uint8

const (
	MenuEventUnknown MenuEventKind = iota
	MenuEventKey
	MenuEventRefreshStarted
	MenuEventInventory
	MenuEventOperationResult
	MenuEventPreviewResult
	MenuEventCompletionResult
	MenuEventParkHotkey
	MenuEventTick
)

type MenuEvent struct {
	Kind         MenuEventKind
	Key          PanelKey
	Address      couchcore.ThreadAddress
	Inventory    []couchcore.ActionableThreadSummary
	InventorySet bool
	Operation    string
	Attempt      uint64
	Success      bool
	Error        string
	Generation   uint64
	// ProjectionAfterGeneration records the newest inventory generation that
	// predates a committed operation mutation.
	ProjectionAfterGeneration uint64
	Prepared                  *couchcore.PreparedStart
	Completion                *CompletionResult
}

// MenuEffect is an operation request for the thin Console shell.
type MenuEffect struct {
	Operation  string
	Attempt    uint64
	Args       map[string]string
	Preview    *PreviewRequest
	Completion *CompletionRequest
}

func NewMenuState(inventory []couchcore.ActionableThreadSummary, active couchcore.ThreadAddress) MenuState {
	owned := append([]couchcore.ActionableThreadSummary(nil), inventory...)
	root := MenuFrame{Instance: 1, Kind: MenuFrameRoot}
	if len(owned) > 0 {
		root.SelectedAddress = owned[0].Address
	}
	return MenuState{Inventory: owned, InventoryReady: inventory != nil, Frames: []MenuFrame{root}, ActiveAddress: active, FrameSequence: 1}
}

func (s MenuState) CurrentFrame() MenuFrame {
	if len(s.Frames) == 0 {
		return MenuFrame{}
	}
	return s.Frames[len(s.Frames)-1]
}

// VisibleMenuThreads applies the same exact-over-fuzzy field rule as operation
// resolution, but only to the already-present in-memory inventory.
func VisibleMenuThreads(state MenuState) []couchcore.ActionableThreadSummary {
	frame := state.CurrentFrame()
	if frame.Kind != MenuFrameRoot {
		return nil
	}
	return visibleRootThreads(state.Inventory, frame)
}

func visibleRootThreads(inventory []couchcore.ActionableThreadSummary, frame MenuFrame) []couchcore.ActionableThreadSummary {
	if frame.Filter == "" {
		return append([]couchcore.ActionableThreadSummary(nil), inventory...)
	}
	fields := make([]couchcore.ThreadReferenceFields, len(inventory))
	for i, thread := range inventory {
		fields[i] = couchcore.ThreadReferenceFields{
			Address:     thread.Address,
			Name:        thread.Name,
			WorkingPath: thread.WorkingPath,
		}
	}
	addresses, err := couchcore.MatchThreadReferenceFields(fields, frame.Filter)
	if errors.Is(err, couchcore.ErrThreadReferenceNotFound) {
		return []couchcore.ActionableThreadSummary{}
	}
	if err != nil {
		return []couchcore.ActionableThreadSummary{}
	}
	wanted := make(map[couchcore.ThreadAddress]bool, len(addresses))
	for _, address := range addresses {
		wanted[address] = true
	}
	visible := make([]couchcore.ActionableThreadSummary, 0, len(addresses))
	for _, thread := range inventory {
		if wanted[thread.Address] {
			visible = append(visible, thread)
		}
	}
	return visible
}

// ReduceMenu is the single total transition authority for menu input and
// asynchronous completions. The initial slice ownership is cloned before any
// transition so callers can retain prior states safely.
func ReduceMenu(state MenuState, event MenuEvent) (MenuState, []MenuEffect) {
	next := cloneMenuState(state)
	if event.Kind == MenuEventTick {
		if menuProgressMatches(next.Notice, event) {
			next.SpinnerPhase = (next.SpinnerPhase + 1) % 4
		}
		return next, nil
	}
	if event.Kind == MenuEventOperationResult && !menuOperationMatches(next.InFlight, event) {
		return next, nil
	}
	if event.Kind == MenuEventCompletionResult && !completionResultMatches(next, event.Completion) {
		return next, nil
	}
	if next.Notice.Level != MenuNoticeProgress && event.Kind != MenuEventOperationResult && event.Kind != MenuEventPreviewResult && event.Kind != MenuEventCompletionResult {
		next.Notice = MenuNotice{}
	}
	if event.Kind == MenuEventParkHotkey {
		return reduceParkHotkey(next, event), nil
	}
	if event.Kind == MenuEventRefreshStarted {
		next.RefreshPending = true
		if !next.InventoryReady && next.Notice.Level != MenuNoticeProgress {
			next.Notice = infoMenuNotice("thread inventory unavailable")
		}
		return next, nil
	}
	if event.Kind == MenuEventInventory {
		next.RefreshPending = false
		if event.Error != "" {
			next.Notice = errorMenuNotice("thread inventory unavailable: " + event.Error)
			return next, nil
		}
		if !next.ProjectionPending || event.Generation > next.ProjectionAfterGeneration {
			next.ProjectionPending = false
			next.ProjectionAfterGeneration = 0
		}
		previous := append([]couchcore.ActionableThreadSummary(nil), next.Inventory...)
		next.Inventory = append([]couchcore.ActionableThreadSummary(nil), event.Inventory...)
		next.InventoryReady = true
		return reconcileMenuFrames(next, previous), nil
	}
	if event.Kind == MenuEventOperationResult {
		if event.InventorySet {
			next.ProjectionPending = false
			next.ProjectionAfterGeneration = 0
			previous := append([]couchcore.ActionableThreadSummary(nil), next.Inventory...)
			next.Inventory = append([]couchcore.ActionableThreadSummary(nil), event.Inventory...)
			next = reconcileMenuFrames(next, previous)
		}
		return reduceOperationResult(next, event), nil
	}
	if event.Kind == MenuEventPreviewResult {
		return reducePreviewResult(next, event)
	}
	if event.Kind == MenuEventCompletionResult {
		return reduceCompletionResult(next, *event.Completion), nil
	}
	if len(next.Frames) == 0 || event.Kind != MenuEventKey {
		return next, nil
	}
	if event.Key.Kind == KeyCtrlSpace {
		return openStartForm(next)
	}
	switch next.CurrentFrame().Kind {
	case MenuFrameRoot:
		return reduceRootKey(next, event.Key)
	case MenuFrameActions:
		return reduceActionKey(next, event.Key)
	case MenuFrameConfirmation:
		return reduceConfirmationKey(next, event.Key)
	case MenuFrameText:
		return reduceTextKey(next, event.Key)
	case MenuFrameStart:
		return reduceStartKey(next, event.Key)
	default:
		return next, nil
	}
}

func menuProgressMatches(notice MenuNotice, event MenuEvent) bool {
	if notice.Level != MenuNoticeProgress {
		return false
	}
	if notice.Owner.OperationAttempt != 0 {
		return event.Attempt == notice.Owner.OperationAttempt && event.Generation == 0
	}
	if notice.Owner.PreviewGeneration != 0 {
		return event.Generation == notice.Owner.PreviewGeneration && event.Attempt == 0
	}
	return false
}

func reduceRootKey(state MenuState, key PanelKey) (MenuState, []MenuEffect) {
	key = hierarchyNavigationKey(key, KeyTab)
	frame := &state.Frames[len(state.Frames)-1]
	switch key.Kind {
	case KeyRune:
		if key.Rune != utf8.RuneError && utf8.ValidRune(key.Rune) && utf8.RuneLen(key.Rune) > 0 {
			candidate := frame.Filter + string(key.Rune)
			if len(candidate) <= menuFilterLimit {
				frame.Filter = candidate
			}
		}
		reconcileRootSelection(&state, frame.SelectedAddress)
	case KeyBackspace:
		frame.Filter = removeLastRune(frame.Filter)
		reconcileRootSelection(&state, frame.SelectedAddress)
	case KeyUp:
		moveRootSelection(&state, -1)
	case KeyDown:
		moveRootSelection(&state, 1)
	case KeyEnter:
		thread, ok := selectedMenuThread(state)
		if !ok {
			state.Notice = errorMenuNotice("no selection")
			return state, nil
		}
		operation := "resume"
		if thread.Live() {
			operation = "switch"
		}
		return dispatchThreadOperation(state, operation, thread.Address)
	case KeyTab:
		thread, ok := selectedMenuThread(state)
		if !ok {
			state.Notice = errorMenuNotice("no selection")
			return state, nil
		}
		items := menuActionItems(thread)
		appendMenuFrame(&state, MenuFrame{
			Kind: MenuFrameActions, Thread: thread.Address, SelectedItem: items[0],
		})
	case KeyEscape:
		if frame.Filter != "" {
			selected := frame.SelectedAddress
			frame.Filter = ""
			reconcileRootSelection(&state, selected)
			return state, nil
		}
		active, ok := findMenuThread(state.Inventory, state.ActiveAddress)
		if !ok || !active.Live() {
			state.Notice = errorMenuNotice("no live thread can receive focus")
			return state, nil
		}
		return dispatchThreadOperation(state, "switch", active.Address)
	}
	return state, nil
}

func reduceParkHotkey(state MenuState, event MenuEvent) MenuState {
	if event.Operation != "park" && event.Operation != "leave" {
		state.Notice = errorMenuNotice("park action is unavailable")
		return state
	}
	thread, ok := findMenuThread(state.Inventory, event.Address)
	if !ok || !thread.Live() {
		state.Notice = errorMenuNotice("active thread is no longer actionable")
		return state
	}
	state.Frames = state.Frames[:1]
	state.Frames[0].SelectedAddress = event.Address
	if !appendMenuFrame(&state, MenuFrame{
		Kind: MenuFrameConfirmation, Thread: event.Address, Action: event.Operation, SelectedItem: "cancel",
	}) {
		return state
	}
	return state
}

func reduceActionKey(state MenuState, key PanelKey) (MenuState, []MenuEffect) {
	key = hierarchyNavigationKey(key, KeyEnter)
	frame := &state.Frames[len(state.Frames)-1]
	thread, ok := findMenuThread(state.Inventory, frame.Thread)
	if !ok {
		return discardThreadFrames(state, frame.Thread, "thread is no longer actionable"), nil
	}
	items := menuActionItems(thread)
	switch key.Kind {
	case KeyRune:
		candidate := frame.Filter + string(key.Rune)
		if key.Rune != utf8.RuneError && utf8.ValidRune(key.Rune) && utf8.RuneLen(key.Rune) > 0 && len(candidate) <= menuFilterLimit {
			frame.Filter = candidate
		}
		reconcileItemSelection(frame, filterMenuItems(items, frame.Filter))
	case KeyBackspace:
		frame.Filter = removeLastRune(frame.Filter)
		reconcileItemSelection(frame, filterMenuItems(items, frame.Filter))
	case KeyUp:
		moveItemSelection(frame, filterMenuItems(items, frame.Filter), -1)
	case KeyDown:
		moveItemSelection(frame, filterMenuItems(items, frame.Filter), 1)
	case KeyEscape:
		state.Frames = state.Frames[:len(state.Frames)-1]
	case KeyEnter:
		if !containsMenuItem(items, frame.SelectedItem) {
			state.Notice = errorMenuNotice("no selection")
			return state, nil
		}
		switch frame.SelectedItem {
		case "park":
			appendMenuFrame(&state, MenuFrame{
				Kind: MenuFrameConfirmation, Thread: thread.Address, Action: "park", SelectedItem: "cancel",
			})
		case "resume":
			return dispatchThreadOperation(state, "resume", thread.Address)
		case "name", "describe":
			appendMenuFrame(&state, MenuFrame{
				Kind: MenuFrameText, Thread: thread.Address, Action: frame.SelectedItem,
			})
		}
	}
	return state, nil
}

func reduceConfirmationKey(state MenuState, key PanelKey) (MenuState, []MenuEffect) {
	key = hierarchyNavigationKey(key, KeyEnter)
	frame := &state.Frames[len(state.Frames)-1]
	thread, ok := findMenuThread(state.Inventory, frame.Thread)
	if !ok {
		return discardThreadFrames(state, frame.Thread, "thread is no longer actionable"), nil
	}
	items := confirmationMenuItems(frame.Action, thread)
	visible := filterMenuItems(items, frame.Filter)
	switch key.Kind {
	case KeyRune:
		candidate := frame.Filter + string(key.Rune)
		if key.Rune != utf8.RuneError && utf8.ValidRune(key.Rune) && utf8.RuneLen(key.Rune) > 0 && len(candidate) <= menuFilterLimit {
			frame.Filter = candidate
		}
		reconcileItemSelection(frame, filterMenuItems(items, frame.Filter))
	case KeyBackspace:
		frame.Filter = removeLastRune(frame.Filter)
		reconcileItemSelection(frame, filterMenuItems(items, frame.Filter))
	case KeyUp:
		moveItemSelection(frame, visible, -1)
	case KeyDown:
		moveItemSelection(frame, visible, 1)
	case KeyEscape:
		state.Frames = state.Frames[:len(state.Frames)-1]
	case KeyEnter:
		if !containsMenuItem(visible, frame.SelectedItem) {
			state.Notice = errorMenuNotice("no selection")
			return state, nil
		}
		if frame.SelectedItem == "cancel" {
			state.Frames = state.Frames[:len(state.Frames)-1]
			return state, nil
		}
		if frame.SelectedItem != frame.Action || (frame.Action != "park" && frame.Action != "leave") || !thread.Live() {
			return discardThreadFrames(state, frame.Thread, "thread action is no longer applicable"), nil
		}
		return dispatchThreadOperation(state, frame.Action, thread.Address)
	}
	return state, nil
}

func reduceTextKey(state MenuState, key PanelKey) (MenuState, []MenuEffect) {
	if key.Kind == KeyLeft {
		key.Kind = KeyEscape
	}
	frame := &state.Frames[len(state.Frames)-1]
	thread, ok := findMenuThread(state.Inventory, frame.Thread)
	if !ok {
		return discardThreadFrames(state, frame.Thread, "thread is no longer actionable"), nil
	}
	limit := menuTextLimit
	if frame.Action == "name" {
		limit = menuNameLimit
	}
	switch key.Kind {
	case KeyRune:
		candidate := frame.Input + string(key.Rune)
		if key.Rune != utf8.RuneError && utf8.ValidRune(key.Rune) && utf8.RuneLen(key.Rune) > 0 && len(candidate) <= limit {
			frame.Input = candidate
		}
	case KeyBackspace:
		frame.Input = removeLastRune(frame.Input)
	case KeyEscape:
		state.Frames = state.Frames[:len(state.Frames)-1]
	case KeyEnter:
		args := map[string]string{
			"repo-scope": thread.Address.RepoScope,
			"ref":        string(thread.Address.Tag),
		}
		if frame.Action == "name" {
			args["name"] = frame.Input
			return dispatchMenuOperation(state, MenuEffect{Operation: "name", Args: args}, thread.Address)
		}
		if frame.Action == "describe" {
			args["description"] = frame.Input
			return dispatchMenuOperation(state, MenuEffect{Operation: "describe", Args: args}, thread.Address)
		}
	}
	return state, nil
}

func openStartForm(state MenuState) (MenuState, []MenuEffect) {
	current := state.CurrentFrame().Kind
	if current == MenuFrameStart || current == MenuFrameText {
		return state, nil
	}
	if current != MenuFrameRoot && current != MenuFrameActions && current != MenuFrameConfirmation {
		return state, nil
	}
	agent := ""
	for _, candidate := range state.Agents {
		if candidate == state.RootAgent {
			agent = candidate
			break
		}
	}
	if agent == "" && len(state.Agents) > 0 {
		agent = state.Agents[0]
	}
	generation, ok := nextPreviewGeneration(&state)
	if !ok {
		return state, nil
	}
	if !appendMenuFrame(&state, MenuFrame{
		Kind: MenuFrameStart, FormField: MenuFieldPath, Agent: agent, Generation: generation,
	}) {
		return state, nil
	}
	return state, nil
}

func reduceStartKey(state MenuState, key PanelKey) (MenuState, []MenuEffect) {
	frame := &state.Frames[len(state.Frames)-1]
	switch key.Kind {
	case KeyRune:
		if frame.FormField != MenuFieldPath || key.Rune == utf8.RuneError || !utf8.ValidRune(key.Rune) {
			return state, nil
		}
		candidate := frame.Path + string(key.Rune)
		if utf8.RuneLen(key.Rune) > 0 && len(candidate) <= menuTextLimit {
			invalidateStartCompletion(&state, frame)
			frame.Path = candidate
			invalidateStartPreview(&state, frame)
		}
	case KeyBackspace:
		if frame.FormField == MenuFieldPath {
			before := frame.Path
			frame.Path = removeLastRune(frame.Path)
			if frame.Path != before {
				invalidateStartCompletion(&state, frame)
				invalidateStartPreview(&state, frame)
			}
		}
	case KeyTab:
		if frame.FormField == MenuFieldPath {
			if len(frame.CompletionCandidates) > 0 {
				frame.CompletionSelected = (frame.CompletionSelected + 1) % len(frame.CompletionCandidates)
				return state, nil
			}
			return requestPathCompletion(state)
		}
	case KeyUp:
		if len(frame.CompletionCandidates) > 0 {
			frame.CompletionSelected = (frame.CompletionSelected - 1 + len(frame.CompletionCandidates)) % len(frame.CompletionCandidates)
		} else if frame.FormField == MenuFieldAgent {
			frame.FormField = MenuFieldPath
			invalidateStartCompletion(&state, frame)
		}
	case KeyDown:
		if len(frame.CompletionCandidates) > 0 {
			frame.CompletionSelected = (frame.CompletionSelected + 1) % len(frame.CompletionCandidates)
		} else if frame.FormField == MenuFieldPath {
			frame.FormField = MenuFieldAgent
			invalidateStartCompletion(&state, frame)
			return requestStartPreview(state)
		}
	case KeyLeft:
		if frame.FormField == MenuFieldAgent {
			if selectStartAgent(frame, state.Agents, -1) && invalidateStartPreview(&state, frame) {
				return requestStartPreview(state)
			}
		}
	case KeyRight:
		if frame.FormField == MenuFieldAgent {
			if selectStartAgent(frame, state.Agents, 1) && invalidateStartPreview(&state, frame) {
				return requestStartPreview(state)
			}
		}
	case KeyEnter:
		if len(frame.CompletionCandidates) > 0 {
			path := frame.CompletionCandidates[frame.CompletionSelected]
			invalidateStartCompletion(&state, frame)
			frame.Path = path
			invalidateStartPreview(&state, frame)
			return state, nil
		}
		if frame.PreviewAccepted == frame.Generation && frame.PreviewToken != "" {
			return dispatchMenuOperation(state, startMenuEffect(*frame), couchcore.ThreadAddress{})
		}
		frame.SubmitGeneration = frame.Generation
		state.SpinnerPhase = 0
		state.Notice = MenuNotice{
			Level: MenuNoticeProgress,
			Text:  "resolving",
			Owner: MenuProgressOwner{PreviewGeneration: frame.Generation},
		}
		if frame.PreviewPending == frame.Generation {
			return state, nil
		}
		return requestStartPreview(state)
	case KeyEscape:
		if len(frame.CompletionCandidates) > 0 {
			invalidateStartCompletion(&state, frame)
			return state, nil
		}
		if state.Notice.Level == MenuNoticeProgress && state.Notice.Owner.PreviewGeneration == frame.Generation {
			state.Notice = MenuNotice{}
		}
		state.Frames = state.Frames[:len(state.Frames)-1]
	}
	return state, nil
}

func selectStartAgent(frame *MenuFrame, agents []string, delta int) bool {
	if len(agents) == 0 {
		return false
	}
	index := 0
	for i, agent := range agents {
		if agent == frame.Agent {
			index = i
			break
		}
	}
	index = (index + delta + len(agents)) % len(agents)
	if frame.Agent != agents[index] || !frame.AgentSticky {
		frame.Agent = agents[index]
		frame.AgentSticky = true
		return true
	}
	return false
}

func invalidateStartPreview(state *MenuState, frame *MenuFrame) bool {
	if state.Notice.Level == MenuNoticeProgress && state.Notice.Owner.PreviewGeneration == frame.Generation {
		state.Notice = MenuNotice{}
	}
	generation, ok := nextPreviewGeneration(state)
	clearStartPreview(frame)
	if ok {
		frame.Generation = generation
	} else {
		frame.Generation = 0
	}
	return ok
}

func requestPathCompletion(state MenuState) (MenuState, []MenuEffect) {
	frame := &state.Frames[len(state.Frames)-1]
	if frame.CompletionPending && frame.CompletionPath == frame.Path {
		return state, nil
	}
	query := SplitCompletionPath(frame.Path)
	if query.Immediate != "" {
		invalidateStartCompletion(&state, frame)
		frame.Path = query.Immediate
		invalidateStartPreview(&state, frame)
		return state, nil
	}
	if state.CompletionSequence == ^uint64(0) {
		state.Notice = errorMenuNotice("path completion identity exhausted")
		return state, nil
	}
	state.CompletionSequence++
	identity := CompletionIdentity{FrameInstance: frame.Instance, Generation: state.CompletionSequence}
	invalidateStartCompletion(&state, frame)
	frame.CompletionRequest = identity
	frame.CompletionPath = frame.Path
	frame.CompletionPending = true
	request := CompletionRequest{Identity: identity, Path: frame.Path}
	return state, []MenuEffect{{Completion: &request}}
}

func invalidateStartCompletion(state *MenuState, frame *MenuFrame) {
	identity := frame.CompletionRequest
	frame.CompletionRequest = CompletionIdentity{}
	frame.CompletionPath = ""
	frame.CompletionPending = false
	frame.CompletionCandidates = nil
	frame.CompletionSelected = 0
	frame.CompletionTruncated = false
	if state.Notice.Owner.Completion == identity && identity != (CompletionIdentity{}) {
		state.Notice = MenuNotice{}
	}
}

func completionResultMatches(state MenuState, result *CompletionResult) bool {
	if result == nil || result.Identity.FrameInstance == 0 || result.Identity.Generation == 0 || state.CurrentFrame().Kind != MenuFrameStart {
		return false
	}
	frame := state.CurrentFrame()
	return frame.Instance == result.Identity.FrameInstance && frame.CompletionRequest == result.Identity
}

func reduceCompletionResult(state MenuState, result CompletionResult) MenuState {
	frame := &state.Frames[len(state.Frames)-1]
	frame.CompletionPending = false
	frame.CompletionCandidates = nil
	frame.CompletionSelected = 0
	frame.CompletionTruncated = false
	owner := MenuProgressOwner{Completion: result.Identity}
	if result.Error != "" {
		state.Notice = MenuNotice{Level: MenuNoticeError, Text: result.Error, Owner: owner}
		return state
	}
	paths := append([]string(nil), result.Matches.Paths...)
	if len(paths) == 0 {
		state.Notice = MenuNotice{Level: MenuNoticeInfo, Text: "no matching directories", Owner: owner}
		return state
	}
	if len(paths) == 1 {
		frame.Path = paths[0]
		invalidateStartCompletion(&state, frame)
		invalidateStartPreview(&state, frame)
		return state
	}
	frame.CompletionCandidates = paths
	frame.CompletionTruncated = result.Matches.Truncated
	return state
}

func nextPreviewGeneration(state *MenuState) (uint64, bool) {
	if state.PreviewSequence == ^uint64(0) {
		state.Notice = errorMenuNotice("start preview identity exhausted")
		return 0, false
	}
	state.PreviewSequence++
	return state.PreviewSequence, true
}

func clearStartPreview(frame *MenuFrame) {
	frame.PreviewPending = 0
	frame.PreviewAccepted = 0
	frame.PreviewToken = ""
	frame.PreviewPath = ""
	frame.PreviewAgent = ""
	frame.PreviewAgentSource = ""
	frame.PreviewArgvSource = ""
	frame.SubmitGeneration = 0
}

func requestStartPreview(state MenuState) (MenuState, []MenuEffect) {
	frame := &state.Frames[len(state.Frames)-1]
	if frame.Generation == 0 {
		return state, nil
	}
	if frame.PreviewPending == frame.Generation ||
		(frame.PreviewAccepted == frame.Generation && frame.PreviewToken != "") {
		return state, nil
	}
	path := frame.Path
	if path == "" {
		path = "."
	}
	request := PreviewRequest{Generation: frame.Generation, Path: path}
	if frame.AgentSticky {
		request.Agent = frame.Agent
	}
	frame.PreviewPending = frame.Generation
	return state, []MenuEffect{{Preview: &request}}
}

func reducePreviewResult(state MenuState, event MenuEvent) (MenuState, []MenuEffect) {
	if state.CurrentFrame().Kind != MenuFrameStart {
		return state, nil
	}
	frame := &state.Frames[len(state.Frames)-1]
	if event.Generation != frame.Generation || event.Generation != frame.PreviewPending {
		return state, nil
	}
	frame.PreviewPending = 0
	if event.Error != "" || event.Prepared == nil || event.Prepared.Token == "" {
		frame.SubmitGeneration = 0
		state.Notice = errorMenuNotice(event.Error)
		if state.Notice.Text == "" {
			state.Notice = errorMenuNotice("start preview failed")
		}
		return state, nil
	}
	frame.PreviewAccepted = event.Generation
	frame.PreviewToken = event.Prepared.Token
	frame.PreviewPath = event.Prepared.Resolution.CanonicalPath
	frame.PreviewAgent = event.Prepared.Resolution.Profile.Agent
	frame.Agent = event.Prepared.Resolution.Profile.Agent
	frame.PreviewAgentSource = event.Prepared.Resolution.AgentSource
	frame.PreviewArgvSource = event.Prepared.Resolution.ArgvSource
	if frame.SubmitGeneration != event.Generation {
		return state, nil
	}
	frame.SubmitGeneration = 0
	return dispatchMenuOperation(state, startMenuEffect(*frame), couchcore.ThreadAddress{})
}

func startMenuEffect(frame MenuFrame) MenuEffect {
	return MenuEffect{Operation: "start", Args: map[string]string{
		"token": string(frame.PreviewToken),
	}}
}

func menuActionItems(thread couchcore.ActionableThreadSummary) []string {
	first := "resume"
	if thread.Live() {
		first = "park"
	}
	return []string{first, "name", "describe"}
}

func confirmationMenuItems(action string, thread couchcore.ActionableThreadSummary) []string {
	if action == "leave" {
		return []string{"cancel", "leave couch"}
	}
	return []string{"cancel", "park " + thread.Label()}
}

func filterMenuItems(items []string, query string) []string {
	if query == "" {
		return append([]string(nil), items...)
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		if strings.Contains(strings.ToLower(menuItemLabel(item)), strings.ToLower(query)) {
			out = append(out, item)
		}
	}
	return out
}

func menuItemLabel(item string) string {
	if item == "name" {
		return "rename"
	}
	return item
}

func menuItemID(item string) string {
	if before, _, found := strings.Cut(item, " "); found {
		return before
	}
	return item
}

func reconcileItemSelection(frame *MenuFrame, items []string) {
	if containsMenuItem(items, frame.SelectedItem) {
		return
	}
	frame.SelectedItem = ""
	if len(items) > 0 {
		frame.SelectedItem = menuItemID(items[0])
	}
}

func moveItemSelection(frame *MenuFrame, items []string, delta int) {
	if len(items) == 0 {
		frame.SelectedItem = ""
		return
	}
	index := 0
	for i, item := range items {
		if menuItemID(item) == frame.SelectedItem {
			index = i
			break
		}
	}
	index += delta
	if index < 0 {
		index = 0
	}
	if index >= len(items) {
		index = len(items) - 1
	}
	frame.SelectedItem = menuItemID(items[index])
}

func containsMenuItem(items []string, want string) bool {
	for _, item := range items {
		if menuItemID(item) == want {
			return true
		}
	}
	return false
}

func discardThreadFrames(state MenuState, address couchcore.ThreadAddress, notice string) MenuState {
	state.Frames = state.Frames[:1]
	state.Frames[0].SelectedAddress = address
	reconcileRootSelection(&state, address)
	state.Notice = errorMenuNotice(notice)
	return state
}

func reconcileRootSelection(state *MenuState, preferred couchcore.ThreadAddress) {
	visible := visibleRootThreads(state.Inventory, state.Frames[0])
	frame := &state.Frames[0]
	frame.SelectedAddress = couchcore.ThreadAddress{}
	for _, thread := range visible {
		if thread.Address == preferred {
			frame.SelectedAddress = preferred
			return
		}
	}
	if len(visible) > 0 {
		frame.SelectedAddress = visible[0].Address
	}
}

func moveRootSelection(state *MenuState, delta int) {
	visible := visibleRootThreads(state.Inventory, state.Frames[0])
	if len(visible) == 0 {
		state.Frames[0].SelectedAddress = couchcore.ThreadAddress{}
		return
	}
	current := 0
	for i, thread := range visible {
		if thread.Address == state.Frames[0].SelectedAddress {
			current = i
			break
		}
	}
	current += delta
	if current < 0 {
		current = 0
	}
	if current >= len(visible) {
		current = len(visible) - 1
	}
	state.Frames[0].SelectedAddress = visible[current].Address
}

func reconcileMenuFrames(state MenuState, previous ...[]couchcore.ActionableThreadSummary) MenuState {
	priorInventory := state.Inventory
	if len(previous) > 0 {
		priorInventory = previous[0]
	}
	if len(state.Frames) == 0 || state.Frames[0].Kind != MenuFrameRoot {
		state.Frames = nil
		appendMenuFrame(&state, MenuFrame{Kind: MenuFrameRoot})
		return state
	}
	original := append([]MenuFrame(nil), state.Frames...)
	root := original[0]
	state.Frames = []MenuFrame{root}
	reconcileRootSelection(&state, root.SelectedAddress)

	invalidThreadFrame := false
	var bound couchcore.ThreadAddress
	for _, frame := range original[1:] {
		if frame.Kind == MenuFrameStart {
			state.Frames = append(state.Frames, frame)
			continue
		}
		if invalidThreadFrame {
			continue
		}
		thread, ok := findMenuThread(state.Inventory, frame.Thread)
		if !ok {
			invalidThreadFrame = true
			state.Notice = errorMenuNotice(hiddenThreadNotice(priorInventory, frame.Thread))
			continue
		}
		switch frame.Kind {
		case MenuFrameActions:
			if bound != (couchcore.ThreadAddress{}) {
				invalidThreadFrame = true
				state.Notice = errorMenuNotice("thread action is no longer applicable")
				continue
			}
			reconcileItemSelection(&frame, filterMenuItems(menuActionItems(thread), frame.Filter))
			bound = frame.Thread
		case MenuFrameConfirmation:
			if (bound != (couchcore.ThreadAddress{}) && bound != frame.Thread) ||
				(frame.Action != "park" && frame.Action != "leave") || !thread.Live() {
				invalidThreadFrame = true
				state.Notice = errorMenuNotice("thread action is no longer applicable")
				continue
			}
			reconcileItemSelection(&frame, filterMenuItems(confirmationMenuItems(frame.Action, thread), frame.Filter))
		case MenuFrameText:
			if bound != frame.Thread || (frame.Action != "name" && frame.Action != "describe") {
				invalidThreadFrame = true
				state.Notice = errorMenuNotice("thread input is no longer applicable")
				continue
			}
		default:
			invalidThreadFrame = true
			state.Notice = errorMenuNotice("menu frame is no longer valid")
			continue
		}
		state.Frames = append(state.Frames, frame)
	}
	return state
}

func hiddenThreadNotice(previous []couchcore.ActionableThreadSummary, address couchcore.ThreadAddress) string {
	label := string(address.Tag)
	if thread, found := findMenuThread(previous, address); found {
		label = thread.Label()
	}
	return "thread " + label + " (" + address.RepoScope + "/" + string(address.Tag) + ") is no longer actionable"
}

func reduceOperationResult(state MenuState, event MenuEvent) MenuState {
	origin := state.InFlight
	if !menuOperationMatches(origin, event) {
		return state
	}
	state.InFlight = MenuOperationOrigin{}
	if origin.Address != (couchcore.ThreadAddress{}) {
		if _, stillActionable := findMenuThread(state.Inventory, origin.Address); !stillActionable {
			return state
		}
	}
	originFrame, originVisible := menuOperationOriginFrame(state, origin)
	if !event.Success {
		if (event.Operation == "park" || event.Operation == "leave") && origin.FrameKind == MenuFrameConfirmation && originVisible {
			state = restoreMenuPrefixPreservingStart(state, origin.Depth-1, origin)
		}
		state.Notice = errorMenuNotice(event.Error)
		if state.Notice.Text == "" {
			state.Notice = errorMenuNotice(event.Operation + " failed")
		}
		state.Notice.Owner = MenuProgressOwner{OperationAttempt: origin.Attempt}
		return state
	}
	if state.Notice.Level == MenuNoticeProgress && state.Notice.Owner.OperationAttempt == origin.Attempt {
		state.Notice = MenuNotice{}
	}
	if !event.InventorySet && operationNeedsProjectionRefresh(event.Operation) {
		state.ProjectionPending = true
		state.ProjectionAfterGeneration = event.ProjectionAfterGeneration
	}

	switch event.Operation {
	case "switch":
	case "name", "describe":
		if origin.FrameKind == MenuFrameText && originVisible && originFrame.Thread == origin.Address && originFrame.Action == event.Operation {
			state.Frames = state.Frames[:origin.Depth-1]
		}
	case "start":
		if originVisible {
			state = restoreMenuPrefixPreservingStart(state, 1, origin)
			state.Frames[0].SelectedAddress = event.Address
		}
	case "park", "resume", "leave":
		state = restoreMenuPrefixPreservingStart(state, 1, origin)
		state.Frames[0].SelectedAddress = event.Address
		reconcileRootSelection(&state, event.Address)
	}
	return state
}

// operationNeedsProjectionRefresh is the exhaustive projection policy for all
// switcher operations. Mutations whose result does not carry an actionable
// inventory must visibly defer to the next provider refresh. Switch only moves
// terminal focus; leave terminates the console and has no next frame to update.
func operationNeedsProjectionRefresh(operation string) bool {
	switch operation {
	case "start", "park", "resume", "name", "describe":
		return true
	case "switch", "leave":
		return false
	default:
		// Fail safe: a new successful operation may have changed the inventory.
		return true
	}
}

func restoreMenuPrefixPreservingStart(state MenuState, keep int, origin MenuOperationOrigin) MenuState {
	overlays := make([]MenuFrame, 0, 1)
	for _, frame := range state.Frames {
		if frame.Kind == MenuFrameStart && frame.Instance != origin.FrameInstance {
			overlays = append(overlays, frame)
		}
	}
	if keep < 0 {
		keep = 0
	}
	if keep > len(state.Frames) {
		keep = len(state.Frames)
	}
	frames := append([]MenuFrame(nil), state.Frames[:keep]...)
	for _, overlay := range overlays {
		alreadyPresent := false
		for _, frame := range frames {
			if frame.Instance == overlay.Instance {
				alreadyPresent = true
				break
			}
		}
		if !alreadyPresent {
			frames = append(frames, overlay)
		}
	}
	state.Frames = frames
	return state
}

func menuOperationOriginFrame(state MenuState, origin MenuOperationOrigin) (MenuFrame, bool) {
	if origin.Depth < 1 || origin.Depth > len(state.Frames) {
		return MenuFrame{}, false
	}
	frame := state.Frames[origin.Depth-1]
	if frame.Instance != origin.FrameInstance || frame.Kind != origin.FrameKind {
		return MenuFrame{}, false
	}
	return frame, true
}

func menuOperationMatches(origin MenuOperationOrigin, event MenuEvent) bool {
	if origin.Operation == "" || origin.Attempt == 0 || origin.Attempt != event.Attempt || origin.Operation != event.Operation {
		return false
	}
	if origin.Operation == "start" && origin.Address == (couchcore.ThreadAddress{}) {
		return !event.Success || event.Address != (couchcore.ThreadAddress{})
	}
	return origin.Address == event.Address
}

func dispatchThreadOperation(state MenuState, operation string, address couchcore.ThreadAddress) (MenuState, []MenuEffect) {
	return dispatchMenuOperation(state, threadEffect(operation, address), address)
}

func dispatchMenuOperation(state MenuState, effect MenuEffect, address couchcore.ThreadAddress) (MenuState, []MenuEffect) {
	if effect.Operation == "" || state.InFlight.Operation != "" {
		return state, nil
	}
	if state.OperationSequence == ^uint64(0) {
		state.Notice = errorMenuNotice("operation attempt identity exhausted")
		return state, nil
	}
	state.OperationSequence++
	effect.Attempt = state.OperationSequence
	state.InFlight = MenuOperationOrigin{
		Operation:     effect.Operation,
		Attempt:       effect.Attempt,
		Address:       address,
		FrameInstance: state.CurrentFrame().Instance,
		FrameKind:     state.CurrentFrame().Kind,
		Depth:         len(state.Frames),
	}
	state.SpinnerPhase = 0
	state.Notice = MenuNotice{
		Level: MenuNoticeProgress,
		Text:  menuOperationProgressText(state, effect.Operation, address),
		Owner: MenuProgressOwner{OperationAttempt: effect.Attempt},
	}
	return state, []MenuEffect{effect}
}

func menuOperationProgressText(state MenuState, operation string, address couchcore.ThreadAddress) string {
	label := string(address.Tag)
	if thread, ok := findMenuThread(state.Inventory, address); ok {
		label = thread.Label()
	}
	switch operation {
	case "start":
		return "starting thread"
	case "resume":
		return "resuming " + label
	case "park":
		return "parking " + label
	case "leave":
		return "leaving couch"
	case "name":
		return "renaming " + label
	case "describe":
		return "saving " + label + " description"
	default:
		return operation
	}
}

func appendMenuFrame(state *MenuState, frame MenuFrame) bool {
	if state.FrameSequence == ^uint64(0) {
		state.Notice = errorMenuNotice("menu frame identity exhausted")
		return false
	}
	state.FrameSequence++
	frame.Instance = state.FrameSequence
	state.Frames = append(state.Frames, frame)
	return true
}

func selectedMenuThread(state MenuState) (couchcore.ActionableThreadSummary, bool) {
	return findMenuThread(state.Inventory, state.CurrentFrame().SelectedAddress)
}

func findMenuThread(inventory []couchcore.ActionableThreadSummary, address couchcore.ThreadAddress) (couchcore.ActionableThreadSummary, bool) {
	for _, thread := range inventory {
		if thread.Address == address {
			return thread, true
		}
	}
	return couchcore.ActionableThreadSummary{}, false
}

func threadEffect(operation string, address couchcore.ThreadAddress) MenuEffect {
	if operation == "leave" {
		return MenuEffect{Operation: operation}
	}
	return MenuEffect{Operation: operation, Args: map[string]string{
		"repo-scope": address.RepoScope,
		"tag":        string(address.Tag),
	}}
}

func hierarchyNavigationKey(key PanelKey, forward PanelKeyKind) PanelKey {
	switch key.Kind {
	case KeyLeft:
		key.Kind = KeyEscape
	case KeyRight:
		key.Kind = forward
	}
	return key
}

func cloneMenuState(state MenuState) MenuState {
	next := state
	next.Inventory = append([]couchcore.ActionableThreadSummary(nil), state.Inventory...)
	next.Frames = append([]MenuFrame(nil), state.Frames...)
	for i := range next.Frames {
		next.Frames[i].CompletionCandidates = append([]string(nil), state.Frames[i].CompletionCandidates...)
	}
	next.Agents = append([]string(nil), state.Agents...)
	if state.Attention != nil {
		next.Attention = make(map[couchcore.ThreadAddress][]AttentionMessage, len(state.Attention))
		for address, messages := range state.Attention {
			next.Attention[address] = append([]AttentionMessage(nil), messages...)
		}
	}
	return next
}
