package couchtty

import (
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/xianxu/pair/cmd/internal/couchcore"
)

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
	Kind             MenuFrameKind
	Filter           string
	SelectedAddress  couchcore.ThreadAddress
	SelectedItem     string
	Thread           couchcore.ThreadAddress
	Action           string
	Input            string
	FormField        MenuFormField
	Path             string
	Agent            string
	AgentSticky      bool
	Generation       uint64
	PreviewPending   uint64
	PreviewAccepted  uint64
	PreviewToken     couchcore.StartGrantToken
	PreviewPath      string
	PreviewAgent     string
	SubmitGeneration uint64
}

type MenuFormField uint8

const (
	MenuFieldUnknown MenuFormField = iota
	MenuFieldPath
	MenuFieldAgent
)

// MenuState is immutable-by-copy reducer state. Frames retain identities and
// text; the inventory remains one separately owned slice.
type MenuState struct {
	Inventory     []couchcore.ActionableThreadSummary
	Frames        []MenuFrame
	ActiveAddress couchcore.ThreadAddress
	Agents        []string
	RootAgent     string
	Bells         map[couchcore.ThreadAddress]bool
	Notice        string
}

type MenuEventKind uint8

const (
	MenuEventUnknown MenuEventKind = iota
	MenuEventKey
	MenuEventBell
	MenuEventInventory
	MenuEventOperationResult
	MenuEventPreviewResult
)

type MenuEvent struct {
	Kind         MenuEventKind
	Key          PanelKey
	Address      couchcore.ThreadAddress
	Bell         bool
	Inventory    []couchcore.ActionableThreadSummary
	InventorySet bool
	Operation    string
	Success      bool
	Error        string
	Generation   uint64
	Prepared     *couchcore.PreparedStart
}

// MenuEffect is an operation request for the thin Console shell.
type MenuEffect struct {
	Operation string
	Args      map[string]string
	Preview   *PreviewRequest
}

func NewMenuState(inventory []couchcore.ActionableThreadSummary, active couchcore.ThreadAddress) MenuState {
	owned := append([]couchcore.ActionableThreadSummary(nil), inventory...)
	root := MenuFrame{Kind: MenuFrameRoot}
	if len(owned) > 0 {
		root.SelectedAddress = owned[0].Address
	}
	return MenuState{Inventory: owned, Frames: []MenuFrame{root}, ActiveAddress: active}
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
	next.Notice = ""
	if event.Kind == MenuEventInventory {
		next.Inventory = append([]couchcore.ActionableThreadSummary(nil), event.Inventory...)
		return reconcileMenuFrames(next), nil
	}
	if event.Kind == MenuEventOperationResult {
		if event.InventorySet {
			next.Inventory = append([]couchcore.ActionableThreadSummary(nil), event.Inventory...)
			next = reconcileMenuFrames(next)
		}
		return reduceOperationResult(next, event), nil
	}
	if event.Kind == MenuEventPreviewResult {
		return reducePreviewResult(next, event)
	}
	if event.Kind == MenuEventBell {
		if next.Bells == nil {
			next.Bells = make(map[couchcore.ThreadAddress]bool)
		}
		if event.Bell {
			next.Bells[event.Address] = true
		} else {
			delete(next.Bells, event.Address)
		}
		return next, nil
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

func reduceRootKey(state MenuState, key PanelKey) (MenuState, []MenuEffect) {
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
			state.Notice = "no selection"
			return state, nil
		}
		operation := "resume"
		if thread.Live() {
			operation = "switch"
			delete(state.Bells, thread.Address)
		}
		return state, []MenuEffect{threadEffect(operation, thread.Address)}
	case KeyTab:
		thread, ok := selectedMenuThread(state)
		if !ok {
			state.Notice = "no selection"
			return state, nil
		}
		items := menuActionItems(thread)
		state.Frames = append(state.Frames, MenuFrame{
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
			state.Notice = "no live thread can receive focus"
			return state, nil
		}
		return state, []MenuEffect{threadEffect("switch", active.Address)}
	}
	return state, nil
}

func reduceActionKey(state MenuState, key PanelKey) (MenuState, []MenuEffect) {
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
			state.Notice = "no selection"
			return state, nil
		}
		switch frame.SelectedItem {
		case "park":
			state.Frames = append(state.Frames, MenuFrame{
				Kind: MenuFrameConfirmation, Thread: thread.Address, Action: "park", SelectedItem: "cancel",
			})
		case "resume":
			return state, []MenuEffect{threadEffect("resume", thread.Address)}
		case "name", "describe":
			state.Frames = append(state.Frames, MenuFrame{
				Kind: MenuFrameText, Thread: thread.Address, Action: frame.SelectedItem,
			})
		}
	}
	return state, nil
}

func reduceConfirmationKey(state MenuState, key PanelKey) (MenuState, []MenuEffect) {
	frame := &state.Frames[len(state.Frames)-1]
	thread, ok := findMenuThread(state.Inventory, frame.Thread)
	if !ok {
		return discardThreadFrames(state, frame.Thread, "thread is no longer actionable"), nil
	}
	items := []string{"cancel", "park"}
	switch key.Kind {
	case KeyUp:
		moveItemSelection(frame, items, -1)
	case KeyDown:
		moveItemSelection(frame, items, 1)
	case KeyEscape:
		state.Frames = state.Frames[:len(state.Frames)-1]
	case KeyEnter:
		if frame.SelectedItem == "cancel" {
			state.Frames = state.Frames[:len(state.Frames)-1]
			return state, nil
		}
		if frame.SelectedItem != "park" || !thread.Live() {
			return discardThreadFrames(state, frame.Thread, "thread action is no longer applicable"), nil
		}
		return state, []MenuEffect{threadEffect("park", thread.Address)}
	}
	return state, nil
}

func reduceTextKey(state MenuState, key PanelKey) (MenuState, []MenuEffect) {
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
			return state, []MenuEffect{{Operation: "name", Args: args}}
		}
		if frame.Action == "describe" {
			args["description"] = frame.Input
			return state, []MenuEffect{{Operation: "describe", Args: args}}
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
	state.Frames = append(state.Frames, MenuFrame{
		Kind: MenuFrameStart, FormField: MenuFieldPath, Agent: agent, Generation: 1,
	})
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
			frame.Path = candidate
			frame.Generation++
			clearStartPreview(frame)
		}
	case KeyBackspace:
		if frame.FormField == MenuFieldPath {
			before := frame.Path
			frame.Path = removeLastRune(frame.Path)
			if frame.Path != before {
				frame.Generation++
				clearStartPreview(frame)
			}
		}
	case KeyTab:
		if frame.FormField == MenuFieldPath {
			frame.FormField = MenuFieldAgent
			return requestStartPreview(state)
		} else {
			frame.FormField = MenuFieldPath
		}
	case KeyLeft:
		if frame.FormField == MenuFieldAgent {
			if selectStartAgent(frame, state.Agents, -1) {
				return requestStartPreview(state)
			}
		}
	case KeyRight:
		if frame.FormField == MenuFieldAgent {
			if selectStartAgent(frame, state.Agents, 1) {
				return requestStartPreview(state)
			}
		}
	case KeyEnter:
		if frame.PreviewAccepted == frame.Generation && frame.PreviewToken != "" {
			return state, []MenuEffect{startMenuEffect(*frame)}
		}
		frame.SubmitGeneration = frame.Generation
		state.Notice = "resolving"
		if frame.PreviewPending == frame.Generation {
			return state, nil
		}
		return requestStartPreview(state)
	case KeyEscape:
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
		frame.Generation++
		clearStartPreview(frame)
		return true
	}
	return false
}

func clearStartPreview(frame *MenuFrame) {
	frame.PreviewPending = 0
	frame.PreviewAccepted = 0
	frame.PreviewToken = ""
	frame.PreviewPath = ""
	frame.PreviewAgent = ""
	frame.SubmitGeneration = 0
}

func requestStartPreview(state MenuState) (MenuState, []MenuEffect) {
	frame := &state.Frames[len(state.Frames)-1]
	if frame.PreviewPending == frame.Generation {
		return state, nil
	}
	path := frame.Path
	if path == "" {
		path = "."
	}
	request := PreviewRequest{Generation: frame.Generation, Path: path, Agent: frame.Agent}
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
		state.Notice = event.Error
		if state.Notice == "" {
			state.Notice = "start preview failed"
		}
		return state, nil
	}
	frame.PreviewAccepted = event.Generation
	frame.PreviewToken = event.Prepared.Token
	frame.PreviewPath = event.Prepared.Resolution.CanonicalPath
	frame.PreviewAgent = event.Prepared.Resolution.Profile.Agent
	if frame.SubmitGeneration != event.Generation {
		return state, nil
	}
	frame.SubmitGeneration = 0
	return state, []MenuEffect{startMenuEffect(*frame)}
}

func startMenuEffect(frame MenuFrame) MenuEffect {
	path := frame.PreviewPath
	if path == "" {
		path = frame.Path
		if path == "" {
			path = "."
		}
	}
	agent := frame.PreviewAgent
	if agent == "" {
		agent = frame.Agent
	}
	return MenuEffect{Operation: "start", Args: map[string]string{
		"path": path, "agent": agent, "token": string(frame.PreviewToken),
	}}
}

func menuActionItems(thread couchcore.ActionableThreadSummary) []string {
	first := "resume"
	if thread.Live() {
		first = "park"
	}
	return []string{first, "name", "describe"}
}

func filterMenuItems(items []string, query string) []string {
	if query == "" {
		return append([]string(nil), items...)
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		if strings.Contains(strings.ToLower(item), strings.ToLower(query)) {
			out = append(out, item)
		}
	}
	return out
}

func reconcileItemSelection(frame *MenuFrame, items []string) {
	if containsMenuItem(items, frame.SelectedItem) {
		return
	}
	frame.SelectedItem = ""
	if len(items) > 0 {
		frame.SelectedItem = items[0]
	}
}

func moveItemSelection(frame *MenuFrame, items []string, delta int) {
	if len(items) == 0 {
		frame.SelectedItem = ""
		return
	}
	index := 0
	for i, item := range items {
		if item == frame.SelectedItem {
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
	frame.SelectedItem = items[index]
}

func containsMenuItem(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}

func discardThreadFrames(state MenuState, address couchcore.ThreadAddress, notice string) MenuState {
	state.Frames = state.Frames[:1]
	state.Frames[0].SelectedAddress = address
	reconcileRootSelection(&state, address)
	state.Notice = notice
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

func reconcileMenuFrames(state MenuState) MenuState {
	if len(state.Frames) == 0 || state.Frames[0].Kind != MenuFrameRoot {
		state.Frames = []MenuFrame{{Kind: MenuFrameRoot}}
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
			state.Notice = "thread " + string(frame.Thread.Tag) + " is no longer actionable"
			continue
		}
		switch frame.Kind {
		case MenuFrameActions:
			if bound != (couchcore.ThreadAddress{}) || !containsMenuItem(menuActionItems(thread), frame.SelectedItem) {
				invalidThreadFrame = true
				state.Notice = "thread action is no longer applicable"
				continue
			}
			bound = frame.Thread
		case MenuFrameConfirmation:
			if bound != frame.Thread || frame.Action != "park" || !thread.Live() {
				invalidThreadFrame = true
				state.Notice = "thread action is no longer applicable"
				continue
			}
		case MenuFrameText:
			if bound != frame.Thread || (frame.Action != "name" && frame.Action != "describe") {
				invalidThreadFrame = true
				state.Notice = "thread input is no longer applicable"
				continue
			}
		default:
			invalidThreadFrame = true
			state.Notice = "menu frame is no longer valid"
			continue
		}
		state.Frames = append(state.Frames, frame)
	}
	return state
}

func reduceOperationResult(state MenuState, event MenuEvent) MenuState {
	frame := state.CurrentFrame()
	matched := frame.Thread == event.Address
	if !matched {
		return state
	}
	if !event.Success {
		if event.Operation == "park" && frame.Kind == MenuFrameConfirmation {
			state.Frames = state.Frames[:len(state.Frames)-1]
		}
		state.Notice = event.Error
		if state.Notice == "" {
			state.Notice = event.Operation + " failed"
		}
		return state
	}

	switch event.Operation {
	case "name", "describe":
		if frame.Kind == MenuFrameText && frame.Action == event.Operation {
			state.Frames = state.Frames[:len(state.Frames)-1]
		}
	case "park", "resume":
		state.Frames = state.Frames[:1]
		state.Frames[0].SelectedAddress = event.Address
		reconcileRootSelection(&state, event.Address)
	}
	return state
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
	return MenuEffect{Operation: operation, Args: map[string]string{
		"repo-scope": address.RepoScope,
		"tag":        string(address.Tag),
	}}
}

func cloneMenuState(state MenuState) MenuState {
	next := state
	next.Inventory = append([]couchcore.ActionableThreadSummary(nil), state.Inventory...)
	next.Frames = append([]MenuFrame(nil), state.Frames...)
	next.Agents = append([]string(nil), state.Agents...)
	if state.Bells != nil {
		next.Bells = make(map[couchcore.ThreadAddress]bool, len(state.Bells))
		for address, bell := range state.Bells {
			next.Bells[address] = bell
		}
	}
	return next
}
