package layoutcmd

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/notifytransport"
	"github.com/xianxu/pair/cmd/internal/pairlifecycletest"
	"github.com/xianxu/pair/cmd/internal/testpair"
	"github.com/xianxu/pair/cmd/internal/workbenchshortcut"
	"github.com/xianxu/pair/cmd/internal/zellijpane"
	"github.com/xianxu/pair/cmd/internal/zellijreply"
)

// This deliberately distinguishes finding noise from qualifying a protected
// refusal. A fixed sample without protected bad evidence is inconclusive.
type nativeReplyEvent struct {
	Completed         bool     `json:"completed,omitempty"`
	CollectionUnknown bool     `json:"collection_unknown,omitempty"`
	Kind              string   `json:"kind"`
	At                string   `json:"at,omitempty"`
	Args              []string `json:"args,omitempty"`
	Unknown           bool     `json:"unknown,omitempty"`
	Error             string   `json:"error,omitempty"`
	Stdout            string   `json:"stdout,omitempty"`
	Stderr            string   `json:"stderr,omitempty"`
	StdoutBytes       int      `json:"stdout_bytes,omitempty"`
	StderrBytes       int      `json:"stderr_bytes,omitempty"`
}
type nativeReplyOutcome struct {
	AdmissionExpired, CollectionUnknown                                                 bool
	Events                                                                              []nativeReplyEvent `json:"events"`
	Refused, ProtectedBad, Broker, RecordUnchanged, Observed, WrongTarget, ActionFailed bool
}

func nativeReplyClassification(o nativeReplyOutcome) string {
	unknown, effects := false, 0
	for _, e := range o.Events {
		if e.Unknown || e.Kind == "refusal" {
			unknown = true
		}
		if e.Kind == "mutation" || e.Kind == "record" || e.Kind == "nudge" {
			effects++
			if unknown {
				return "violation"
			}
		}
	}
	if o.WrongTarget {
		return "violation"
	}
	if o.ActionFailed || !o.Observed || o.AdmissionExpired || o.CollectionUnknown {
		return "stopped"
	}
	if o.ProtectedBad && o.Refused {
		if effects > 0 {
			return "stopped"
		}
		if !o.RecordUnchanged {
			return "violation"
		}
		if !o.Broker {
			return "stopped"
		}
		return "qualified"
	}
	if o.Refused {
		return "stopped"
	}
	return "inconclusive"
}
func TestNativeReplyQualificationClassifier(t *testing.T) {
	good := nativeReplyOutcome{Events: []nativeReplyEvent{{Kind: "query", Unknown: true}, {Kind: "refusal"}}, Refused: true, ProtectedBad: true, Broker: true, RecordUnchanged: true, Observed: true}
	for _, tc := range []struct {
		name   string
		change func(*nativeReplyOutcome)
		want   string
	}{
		{"protected refusal", func(*nativeReplyOutcome) {}, "qualified"},
		{"no protected fault", func(o *nativeReplyOutcome) { o.ProtectedBad = false; o.Refused = false }, "inconclusive"},
		{"unclassified refusal", func(o *nativeReplyOutcome) { o.ProtectedBad = false }, "stopped"},
		{"raw noise alone", func(o *nativeReplyOutcome) { o.ProtectedBad = false; o.Refused = false }, "inconclusive"},
		{"effect after unknown", func(o *nativeReplyOutcome) { o.Events = append(o.Events, nativeReplyEvent{Kind: "mutation"}) }, "violation"},
		{"store after unknown", func(o *nativeReplyOutcome) { o.Events = append(o.Events, nativeReplyEvent{Kind: "record"}) }, "violation"},
		{"wrong target", func(o *nativeReplyOutcome) { o.WrongTarget = true }, "violation"},
		{"record changed", func(o *nativeReplyOutcome) { o.RecordUnchanged = false }, "violation"},
		{"missing broker", func(o *nativeReplyOutcome) { o.Broker = false }, "stopped"},
		{"missing independent oracle", func(o *nativeReplyOutcome) { o.Observed = false }, "stopped"},
		{"native action failure", func(o *nativeReplyOutcome) { o.ActionFailed = true }, "stopped"},
		{"admission expired", func(o *nativeReplyOutcome) { o.AdmissionExpired = true }, "stopped"},
		{"query ownership unknown", func(o *nativeReplyOutcome) { o.CollectionUnknown = true }, "stopped"},
		{"completed prefix", func(o *nativeReplyOutcome) {
			o.Events = append([]nativeReplyEvent{{Kind: "mutation"}}, o.Events...)
			o.RecordUnchanged = false
		}, "stopped"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			o := good
			o.Events = append([]nativeReplyEvent(nil), good.Events...)
			tc.change(&o)
			if got := nativeReplyClassification(o); got != tc.want {
				t.Fatalf("classification=%s want %s", got, tc.want)
			}
		})
	}
}

// nativeReplyNoise is the retained #640 diagnostic, not a query proxy. It
// connects only to this fixture's known socket and is cancelled and joined
// before any final outcome observation or session shutdown.
type nativeReplyNoise struct {
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	mu         sync.Mutex
	closing    bool
	conns      map[net.Conn]bool
	Total, Bad int
	BadFrames  []nativeReplyEvent
}

func startNativeReplyNoise(parent context.Context, path string) *nativeReplyNoise {
	ctx, cancel := context.WithCancel(parent)
	n := &nativeReplyNoise{cancel: cancel, conns: map[net.Conn]bool{}}
	query, _ := hex.DecodeString("4a0b0a07920604280130012001")
	for worker := 0; worker < 8; worker++ {
		n.wg.Add(1)
		go func(worker int) {
			defer n.wg.Done()
			for sample := 0; sample < 250; sample++ {
				if ctx.Err() != nil {
					return
				}
				probe := (sample+worker)%2 == 0
				payload := query
				if probe {
					payload = []byte{0x6a, 0}
				}
				c, err := (&net.Dialer{Timeout: time.Second}).DialContext(ctx, "unix", path)
				var reply []byte
				if err == nil {
					n.mu.Lock()
					if n.closing {
						n.mu.Unlock()
						_ = c.Close()
						return
					}
					n.conns[c] = true
					n.mu.Unlock()
					_ = c.SetDeadline(time.Now().Add(time.Second))
					err = nativeReplySend(c, payload)
					if err == nil {
						var header [4]byte
						_, err = io.ReadFull(c, header[:])
						if err == nil {
							size := binary.LittleEndian.Uint32(header[:])
							if size > 1048576 {
								err = fmt.Errorf("oversize native frame %d", size)
							} else {
								reply = make([]byte, size)
								_, err = io.ReadFull(c, reply)
							}
						}
					}
					if !probe {
						_ = nativeReplySend(c, []byte{0x5a, 0})
					}
					_ = c.Close()
					n.mu.Lock()
					delete(n.conns, c)
					n.mu.Unlock()
				}
				if ctx.Err() != nil {
					return
				}
				correct := err == nil && len(reply) > 0 && ((probe && reply[0] == 0x22) || (!probe && reply[0] == 0x2a))
				n.mu.Lock()
				n.Total++
				if !correct {
					n.Bad++
					n.BadFrames = append(n.BadFrames, nativeReplyEvent{At: nativeReplyTimestamp(), Kind: "raw-first-frame", Args: []string{strconv.Itoa(worker), strconv.Itoa(sample), strconv.FormatBool(probe)}, Stdout: hex.EncodeToString(reply[:min(len(reply), 200)]), StdoutBytes: len(reply), Error: fmt.Sprint(err)})
				}
				n.mu.Unlock()
			}
		}(worker)
	}
	return n
}
func nativeReplySend(c net.Conn, payload []byte) error {
	frame := make([]byte, 4+len(payload))
	binary.LittleEndian.PutUint32(frame, uint32(len(payload)))
	copy(frame[4:], payload)
	for len(frame) > 0 {
		written, err := c.Write(frame)
		if err != nil {
			return err
		}
		if written == 0 {
			return io.ErrShortWrite
		}
		frame = frame[written:]
	}
	return nil
}
func (n *nativeReplyNoise) stop() {
	n.cancel()
	n.mu.Lock()
	n.closing = true
	for c := range n.conns {
		_ = c.Close()
	}
	n.mu.Unlock()
	n.wg.Wait()
}

type nativeReplyStore struct {
	workbenchshortcut.FullscreenStore
	rt *nativeReplyRuntime
}

func (s nativeReplyStore) Write(value string) error {
	s.rt.record(nativeReplyEvent{Kind: "record", Args: []string{"write", value}})
	return s.FullscreenStore.Write(value)
}
func (s nativeReplyStore) Clear() error {
	s.rt.record(nativeReplyEvent{Kind: "record", Args: []string{"clear"}})
	return s.FullscreenStore.Clear()
}

type nativeReplyRuntime struct {
	*fullscreenLiveRuntime
	events                    []nativeReplyEvent
	tabID                     int
	target, source            string
	wrongTarget, actionFailed bool
}

func (r *nativeReplyRuntime) observeQuery(ctx context.Context, tab bool) ([]byte, error) {
	var captured nativeReplyEvent
	captured.Kind = "query"
	previous := r.queryLog
	r.queryLog = func(_ string, fields ...any) {
		captured = nativeReplyEvent{Kind: "query"}
		captured.Args = append([]string(nil), fields[0].([]string)...)
		captured.Stdout = string(fields[1].([]byte))
		captured.StdoutBytes = fields[2].(int)
		captured.Stderr = string(fields[3].([]byte))
		captured.StderrBytes = fields[4].(int)
		if fields[5] != nil {
			captured.Error = fmt.Sprint(fields[5])
		}
	}
	defer func() { r.queryLog = previous }()
	var raw []byte
	var err error
	if tab {
		raw, err = r.fullscreenLiveRuntime.CurrentTabJSONContext(ctx)
	} else {
		raw, err = r.fullscreenLiveRuntime.ListPanesJSONContext(ctx)
	}
	var validation error
	if tab {
		_, validation = zellijreply.ParseTab(raw)
	} else {
		_, validation = zellijreply.ParsePanes(raw, zellijreply.PaneRequirements{Command: true, State: true, Geometry: true, Tab: true})
	}
	captured.Unknown = err != nil || validation != nil || ctx.Err() != nil
	if err != nil {
		_, directExit := err.(*exec.ExitError)
		captured.CollectionUnknown = !directExit
	}
	r.record(captured)
	return raw, err
}
func (r *nativeReplyRuntime) ListPanesJSONContext(ctx context.Context) ([]byte, error) {
	return r.observeQuery(ctx, false)
}
func (r *nativeReplyRuntime) CurrentTabJSONContext(ctx context.Context) ([]byte, error) {
	return r.observeQuery(ctx, true)
}
func (r *nativeReplyRuntime) FullscreenStore() workbenchshortcut.FullscreenStore {
	return nativeReplyStore{r.store, r}
}
func (r *nativeReplyRuntime) RunZellijAction(args ...string) error {
	r.record(nativeReplyEvent{Kind: "mutation", Args: append([]string(nil), args...)})
	if !nativeReplyAllowedTarget(args, r.source, r.target, r.tabID) {
		r.wrongTarget = true
	}
	err := r.fullscreenLiveRuntime.RunZellijAction(args...)
	completed := err == nil
	if err != nil {
		desired := len(args) > 1 && args[0] == "focus-pane-id" && IsAlreadyFocused(err, args[1])
		desired = desired || (len(args) > 0 && args[0] == "show-floating-panes" && exitCode(err) == 2)
		completed = desired
		if !desired {
			r.actionFailed = true
		}
	}
	r.record(nativeReplyEvent{Kind: "mutation-result", Completed: completed, Args: append([]string(nil), args...), Error: fmt.Sprint(err)})
	return err
}
func nativeReplyAllowedTarget(args []string, source, target string, tab int) bool {
	if len(args) == 0 {
		return false
	}
	canonical := func(id string) string { return strings.TrimPrefix(id, "terminal_") }
	if args[0] == "focus-pane-id" {
		return len(args) == 2 && (canonical(args[1]) == source || canonical(args[1]) == target)
	}
	paneAction, tabAction := false, false
	switch args[0] {
	case "toggle-pane-embed-or-floating", "change-floating-pane-coordinates", "toggle-fullscreen", "move-pane":
		paneAction = true
	case "show-floating-panes", "hide-floating-panes", "next-swap-layout", "previous-swap-layout":
		tabAction = true
	default:
		return false
	}
	paneCount, tabCount := 0, 0
	for i := 1; i < len(args); i++ {
		if args[i] == "--pane-id" {
			paneCount++
			if i+1 >= len(args) || canonical(args[i+1]) != target {
				return false
			}
		}
		if args[i] == "--tab-id" {
			tabCount++
			if i+1 >= len(args) || args[i+1] != strconv.Itoa(tab) {
				return false
			}
		}
	}
	return (paneAction && paneCount == 1 && tabCount == 0) || (tabAction && tabCount == 1 && paneCount == 0)
}

func (r *nativeReplyRuntime) NotifyRefusal(message string) error {
	r.record(nativeReplyEvent{Kind: "refusal"})
	return (OSRuntime{}).NotifyRefusal(message)
}
func (r *nativeReplyRuntime) NudgeWrap() error {
	r.record(nativeReplyEvent{Kind: "nudge"})
	return nil
}

type nativeReplyAttempt struct {
	Number        int    `json:"number"`
	Status        int    `json:"status"`
	Diagnostic    string `json:"diagnostic"`
	Before, After string
	ExpectedFocus string             `json:"expected_focus"`
	Outcome       nativeReplyOutcome `json:"outcome"`
}
type nativeReplyResult struct {
	Classification, Reason, Version, Session string
	NoiseRequests, NoiseBad                  int
	Noise                                    []nativeReplyEvent
	Attempts                                 []nativeReplyAttempt
	Oracle                                   []nativeReplyEvent
	Initial, Final                           map[string]zellijpane.Pane
	Receivers                                map[string]string
}

// This opt-in sample qualifies an observed protected refusal, never universal
// protocol safety. No fault, or raw diagnostic faults alone, is INCONCLUSIVE.
// Run only after the hub grants the native lane. No Pair binary is built.
func TestFullscreenReplyRefusalInstalledZellijLive(t *testing.T) {
	if os.Getenv("PAIR_LIVE_ZELLIJ") != "1" {
		t.Skip("set PAIR_LIVE_ZELLIJ=1 only with the native lane grant")
	}
	//lint:allow tmp-literal — the installed native session appends s/contract_version_1 and its session name; notification sockets also append digest/PID suffixes
	root, err := os.MkdirTemp("/tmp", "p647nr-")
	if err != nil {
		t.Fatal(err)
	}
	result := nativeReplyResult{Classification: "stopped", Reason: "fixture setup did not complete"}
	t.Logf("retained native reply evidence: %s", root)
	defer func() {
		result.Receivers = map[string]string{}
		for _, name := range []string{"agent", "draft", "terminal-top", "terminal-bottom"} {
			raw, _ := os.ReadFile(filepath.Join(root, name))
			result.Receivers[name] = string(raw[:min(len(raw), 4096)])
		}
		events := append([]nativeReplyEvent(nil), result.Oracle...)
		events = append(events, result.Noise...)
		for _, attempt := range result.Attempts {
			events = append(events, attempt.Outcome.Events...)
		}
		sort.SliceStable(events, func(i, j int) bool { return events[i].At < events[j].At })
		var jsonl bytes.Buffer
		encoder := json.NewEncoder(&jsonl)
		for _, event := range events {
			if err := encoder.Encode(event); err != nil {
				t.Errorf("encode native event: %v", err)
			}
		}
		if err := os.WriteFile(filepath.Join(root, "events.jsonl"), jsonl.Bytes(), 0600); err != nil {
			t.Errorf("retain native events: %v", err)
		}
		data, encodeErr := json.MarshalIndent(result, "", "  ")
		if encodeErr != nil {
			t.Errorf("encode native result: %v", encodeErr)
			return
		}
		if err := os.WriteFile(filepath.Join(root, "result.json"), append(data, '\n'), 0600); err != nil {
			t.Errorf("retain native result: %v", err)
		}
		t.Logf("native sample classification=%s reason=%s raw=%d bad=%d attempts=%d result=%s", result.Classification, result.Reason, result.NoiseRequests, result.NoiseBad, len(result.Attempts), filepath.Join(root, "result.json"))
	}()
	native, err := exec.LookPath("zellij")
	if err != nil {
		t.Fatal(err)
	}
	version, err := exec.Command(native, "--version").Output()
	if err != nil {
		t.Fatal(err)
	}
	result.Version = strings.TrimSpace(string(version))
	if result.Version != "zellij 0.45.1" {
		result.Classification = "inconclusive"
		result.Reason = "fixture pins installed Zellij 0.45.1"
		t.Skip(result.Reason)
	}
	nativeReplySetEnvironment(t, root, native)
	binding := filepath.Join(root, "wrapper-pid")
	broker, err := notifytransport.Start(binding, os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := broker.Close(); err != nil {
			result.Classification = "stopped"
			result.Reason += "; broker shutdown: " + err.Error()
			t.Error(err)
		}
	}()
	t.Setenv("PAIR_PAIR_WRAP_PID_PATH", binding)
	config := filepath.Join(root, "config.kdl")
	if err := os.WriteFile(config, []byte("pane_frames false\nshow_startup_tips false\nshow_release_notes false\n"), 0600); err != nil {
		t.Fatal(err)
	}
	names := []string{"agent", "draft", "terminal-top", "terminal-bottom"}
	pane := func(name string) string {
		script := `printf ready > "$1.ready"; while IFS= read -r line; do printf '%s\n' "$line" >> "$1"; done`
		role := "pair term"
		if name == "agent" {
			role = "pair wrap"
		}
		if name == "draft" {
			role = "nvim -u /fixture/nvim/init.lua"
		}
		return fmt.Sprintf("pane name=%s command=\"/bin/sh\" { args \"-c\" %s \"receiver\" %s %s; }\n", strconv.Quote(name), strconv.Quote(script), strconv.Quote(filepath.Join(root, name)), strconv.Quote(role))
	}
	layout := filepath.Join(root, "layout.kdl")
	body := "layout {\n pane split_direction=\"vertical\" {\n pane {\n" + pane("agent") + pane("draft") + "}\n pane {\n" + pane("terminal-top") + pane("terminal-bottom") + "}\n }\n}\n"
	if err := os.WriteFile(layout, []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	defer cancel()
	session := "pair-r-" + filepath.Base(root)
	result.Session = session
	fixture, err := pairlifecycletest.StartControlledZellijWithOptions(ctx, session, pairlifecycletest.ZellijOptions{ConfigFile: config, LayoutFile: layout})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := fixture.Close(); err != nil {
			result.Classification = "stopped"
			result.Reason += "; shutdown: " + err.Error()
			t.Errorf("owned fixture shutdown: %v", err)
		}
	}()
	// These are state/file birth waits, never retries of a failed native query.
	fullscreenLiveWait(t, ctx, "four owned receiver births", func() bool {
		for _, name := range names {
			if _, err := os.Stat(filepath.Join(root, name+".ready")); err != nil {
				return false
			}
		}
		return true
	})
	rt := &nativeReplyRuntime{fullscreenLiveRuntime: &fullscreenLiveRuntime{ctx: ctx, session: session, store: workbenchshortcut.FullscreenReturnStore{DataDir: filepath.Join(root, "d"), Tag: "native"}}}
	initial, initialEvidence, err := nativeReplySnapshot(ctx, rt.fullscreenLiveRuntime, names, false)
	result.Oracle = append(result.Oracle, initialEvidence...)
	if err != nil {
		result.Reason = "initial one-shot observation: " + err.Error()
		t.Fatal(result.Reason)
	}
	result.Initial = initial
	for _, name := range names {
		p, ok := initial[name]
		if !ok || p.TabID == nil || p.TabPosition == nil || p.IsFloating || p.IsFullscreen == nil || *p.IsFullscreen || p.Columns <= 0 || p.Rows <= 0 {
			t.Fatalf("invalid initial owned pane %s: %+v", name, p)
		}
	}
	rt.source = initial["agent"].ID
	rt.target = initial["terminal-bottom"].ID
	rt.tabID = *initial["agent"].TabID
	rt.caller = rt.source
	rt.last = rt.target
	rt.terminals = []string{initial["terminal-top"].ID, rt.target}
	if initial["terminal-top"].X <= initial["draft"].X || initial["terminal-bottom"].Y <= initial["terminal-top"].Y {
		t.Fatalf("not original split layout: %+v", initial)
	}
	// Establish the real client before testing CLI focus; do not count this setup
	// mutation as a production action or hide a failure by retrying it.
	if err := nativeReplyReceiverWitness(t, ctx, fixture, root, names, "agent", "prime"); err != nil {
		t.Fatal(err)
	}
	if err := rt.fullscreenLiveRuntime.RunZellijAction("focus-pane-id", rt.source); err != nil && !IsAlreadyFocused(err, rt.source) {
		t.Fatal(err)
	}
	socket := filepath.Join(root, "s", "contract_version_1", session)
	info, err := os.Stat(socket)
	if err != nil || info.Mode()&os.ModeSocket == 0 {
		t.Fatalf("owned native socket %s: %v %v", socket, info, err)
	}
	noise := startNativeReplyNoise(ctx, socket)
	defer noise.stop()
	expectedFocus := "agent"
	expected := nativeReplyExpectedState{ExactTiling: true, Focus: rt.source}
	expectedKnown := true
	stopReason := "fixed eight-decision sample completed without protected refusal"
	for sample := 0; sample < 8; sample++ {
		rt.events = nil
		rt.actionFailed = false
		rt.wrongTarget = false
		rt.caller = initial[expectedFocus].ID
		before, err := rt.store.Read()
		if err != nil {
			stopReason = err.Error()
			break
		}
		admission, done := zellijreply.AdmissionContext(ctx)
		var diagnostic bytes.Buffer
		status := RunToggleFocusedContext(admission, nil, rt, &diagnostic)
		admissionExpired := admission.Err() != nil
		done()
		after, readErr := rt.store.Read()
		outcome := nativeReplyOutcome{AdmissionExpired: admissionExpired, Events: append([]nativeReplyEvent(nil), rt.events...), Refused: status != 0, RecordUnchanged: readErr == nil && before == after, WrongTarget: rt.wrongTarget, ActionFailed: rt.actionFailed}
		for _, e := range rt.events {
			if e.Unknown {
				outcome.ProtectedBad = true
			}
			if e.CollectionUnknown {
				outcome.CollectionUnknown = true
			}
		}
		// Valid-but-disagreeing decisions are only known at the production selector's
		// refusal, not by comparing arbitrary raw JSON/cosmetic fields here.
		if status != 0 && (strings.Contains(diagnostic.String(), "list panes:") || strings.Contains(diagnostic.String(), "tab info:")) {
			outcome.ProtectedBad = true
		}
		nextExpected, prefixErr := nativeReplyExpectedPrefix(expected, rt.events, rt.source, rt.target, rt.tabID)
		if prefixErr != nil {
			expectedKnown = false
			outcome.ActionFailed = true
			stopReason = "completed-prefix outcome unknown: " + prefixErr.Error()
		} else {
			expected = nextExpected
			expectedFocus = ""
			for name, p := range initial {
				if p.ID == expected.Focus {
					expectedFocus = name
				}
			}
		}

		attempt := nativeReplyAttempt{Number: sample + 1, Status: status, Diagnostic: diagnostic.String(), Before: before, After: after, ExpectedFocus: expectedFocus, Outcome: outcome}
		if status != 0 {
			select {
			case message := <-broker.Messages():
				attempt.Outcome.Broker = message == "Pair action refused: terminal context could not be confirmed."
			case <-time.After(time.Second):
			}
		}
		result.Attempts = append(result.Attempts, attempt)
		if status != 0 || rt.actionFailed || readErr != nil || prefixErr != nil {
			if prefixErr == nil {
				stopReason = "production attempt stopped: " + diagnostic.String()
			}
			break
		}
		if expectedFocus == "" {
			stopReason = "successful decision lacks completed focus expectation"
			expectedKnown = false
			break
		}
		if err := nativeReplyReceiverWitness(t, ctx, fixture, root, names, expectedFocus, fmt.Sprintf("sample-%d", sample)); err != nil {
			stopReason = err.Error()
			rt.wrongTarget = true
			result.Attempts[len(result.Attempts)-1].Outcome.WrongTarget = true
			break
		}
	}
	// Join every raw worker and close every owned connection before clean outcome
	// queries or shutdown. No native failed operation is replayed.
	noise.stop()
	result.NoiseRequests = noise.Total
	result.NoiseBad = noise.Bad
	result.Noise = append([]nativeReplyEvent(nil), noise.BadFrames...)
	final, oracle, oracleErr := nativeReplySnapshot(ctx, rt.fullscreenLiveRuntime, names, true)
	result.Final = final
	result.Oracle = append(result.Oracle, oracle...)
	observed := oracleErr == nil && expectedKnown
	if observed {
		observedFocus, err := nativeReplyCheckExpected(initial, final, expected, oracle)
		if err != nil {
			observed = false
			stopReason = "independent completed-prefix outcome: " + err.Error()
		} else {
			expectedFocus = observedFocus
		}
	}

	if observed {
		if err := nativeReplyReceiverWitness(t, ctx, fixture, root, names, expectedFocus, "final"); err != nil {
			observed = false
			stopReason = err.Error()
		}
	}
	if len(result.Attempts) == 0 {
		result.Reason = stopReason
		t.Fatal(stopReason)
	}
	last := &result.Attempts[len(result.Attempts)-1]
	last.Outcome.Observed = observed
	// All earlier successes must have retained their independently witnessed focus.
	result.Classification = nativeReplyClassification(last.Outcome)
	result.Reason = stopReason
	if oracleErr != nil {
		result.Classification = "stopped"
		result.Reason = "one-shot final oracle failed: " + oracleErr.Error()
	}
	if result.Classification == "qualified" {
		result.Reason = "observed protected native refusal; zero effects, unchanged record, fixed broker receipt and independent state/input witnesses"
		return
	}
	if result.Classification == "inconclusive" {
		t.Skip("INCONCLUSIVE: " + result.Reason)
	}
	t.Fatalf("native safety sample %s: %s (retained %s)", result.Classification, result.Reason, root)
}

func nativeReplySnapshot(ctx context.Context, rt *fullscreenLiveRuntime, names []string, full bool) (map[string]zellijpane.Pane, []nativeReplyEvent, error) {
	var evidence []nativeReplyEvent
	previous := rt.queryLog
	rt.queryLog = func(_ string, fields ...any) {
		e := nativeReplyEvent{At: nativeReplyTimestamp(), Kind: "oracle", Args: append([]string(nil), fields[0].([]string)...), Stdout: string(fields[1].([]byte)), StdoutBytes: fields[2].(int), Stderr: string(fields[3].([]byte)), StderrBytes: fields[4].(int)}
		if fields[5] != nil {
			e.Error = fmt.Sprint(fields[5])
		}
		evidence = append(evidence, e)
	}
	defer func() { rt.queryLog = previous }()
	raw, err := rt.ListPanesJSONContext(ctx)
	if err != nil {
		return nil, evidence, err
	}
	rows, err := zellijreply.ParsePanes(raw, zellijreply.PaneRequirements{Command: true, State: true, Geometry: true, Tab: true})
	if err != nil {
		return nil, evidence, err
	}
	found := map[string]zellijpane.Pane{}
	for _, p := range rows {
		for _, name := range names {
			if p.Title == name && !p.IsPlugin {
				if _, duplicate := found[name]; duplicate {
					return nil, evidence, fmt.Errorf("duplicate owned receiver %s", name)
				}
				found[name] = p
			}
		}
	}
	if len(found) != len(names) {
		return found, evidence, fmt.Errorf("owned receiver set incomplete: %d/%d", len(found), len(names))
	}
	if !full {
		return found, evidence, nil
	}
	raw, err = rt.CurrentTabJSONContext(ctx)
	if err != nil {
		return found, evidence, err
	}
	tab, err := zellijreply.ParseTab(raw)
	if err != nil {
		return found, evidence, err
	}
	source := found["agent"]
	if tab.ID == nil || tab.Position == nil || source.TabID == nil || source.TabPosition == nil || *tab.ID != *source.TabID || *tab.Position != *source.TabPosition {
		return found, evidence, errors.New("oracle tab does not match known source tab")
	}
	raw, err = rt.queryContext(ctx, "are-floating-panes-visible", "--tab-id", strconv.Itoa(*tab.ID))
	diagnostic := evidence[len(evidence)-1].Stderr
	if nativeReplyVisibilityKnown(string(raw), diagnostic, err) {
		return found, evidence, nil
	}
	return found, evidence, fmt.Errorf("oracle visibility unknown: stdout=%q stderr=%q error=%v", raw, diagnostic, err)
}
func nativeReplyGeometry(initial, final map[string]zellijpane.Pane, focus bool, oracle ...[]nativeReplyEvent) error {
	if len(initial) != 4 || len(final) != 4 {
		return errors.New("owned geometry set is incomplete")
	}
	for name, want := range initial {
		got, ok := final[name]
		if !ok || got.ID != want.ID || got.TabID == nil || want.TabID == nil || *got.TabID != *want.TabID || got.TabPosition == nil || want.TabPosition == nil || *got.TabPosition != *want.TabPosition || got.IsFullscreen == nil || *got.IsFullscreen {
			return fmt.Errorf("owned pane identity/fullscreen changed: %s", name)
		}
		if !focus {
			if got.IsFloating || got.X != want.X || got.Y != want.Y || got.Columns != want.Columns || got.Rows != want.Rows {
				return fmt.Errorf("original tiling not restored: %s got=%+v want=%+v", name, got, want)
			}
		} else if got.IsFloating != (name == "terminal-bottom") {
			return fmt.Errorf("unexpected floating pane: %s", name)
		}
	}
	if !focus {
		return nil
	}
	if len(oracle) != 1 {
		return errors.New("Focus geometry lacks display observation")
	}
	var columns, rows int
	var visible bool
	for _, e := range oracle[0] {
		if len(e.Args) > 0 && e.Args[0] == "current-tab-info" {
			var display struct {
				Columns int `json:"display_area_columns"`
				Rows    int `json:"display_area_rows"`
			}
			if err := json.Unmarshal([]byte(e.Stdout), &display); err != nil {
				return err
			}
			columns, rows = display.Columns, display.Rows
		}
		if len(e.Args) > 0 && e.Args[0] == "are-floating-panes-visible" {
			visible = e.Error == "" && strings.TrimSpace(e.Stdout) == "true"
		}
	}
	near := func(a, b int) bool { return a >= b-1 && a <= b+1 }
	p := final["terminal-bottom"]
	if columns <= 0 || rows <= 0 || !visible || !near(p.Columns, columns*3/4) || !near(p.X, columns/8) || !near(p.Rows, rows) {
		return fmt.Errorf("Focus is not centered75%%/full-height: pane=%+v display=%dx%d visible=%v", p, columns, rows, visible)
	}
	return nil
}
func nativeReplyReceiverWitness(t *testing.T, ctx context.Context, fixture *pairlifecycletest.ControlledZellij, root string, names []string, expected, marker string) error {
	t.Helper()
	marker = "p647-" + marker
	if _, err := fixture.WriteInput([]byte(marker + "\n")); err != nil {
		return err
	}
	var observed error
	fullscreenLiveWait(t, ctx, "actual PTY receiver "+marker, func() bool {
		matched := false
		for _, name := range names {
			raw, err := os.ReadFile(filepath.Join(root, name))
			if err != nil && !errors.Is(err, os.ErrNotExist) {
				observed = err
				return true
			}
			if strings.Contains(string(raw), marker+"\n") {
				matched = true
				if name != expected {
					observed = fmt.Errorf("actual input reached %s, wanted %s", name, expected)
				}
			}
		}
		return matched
	})
	t.Logf("actual PTY marker=%s expected_receiver=%s error=%v", marker, expected, observed)
	return observed
}

func TestNativeReplyTargetOracleRejectsEveryWrongAddress(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
		want bool
	}{
		{"focus source", []string{"focus-pane-id", "terminal_1"}, true},
		{"focus target", []string{"focus-pane-id", "7"}, true},
		{"pane target", []string{"toggle-pane-embed-or-floating", "--pane-id", "7"}, true},
		{"tab target", []string{"show-floating-panes", "--tab-id", "2"}, true},
		{"unknown action", []string{"unclassified-action", "--pane-id", "7"}, false},
		{"second pane wrong", []string{"toggle-fullscreen", "--pane-id", "7", "--pane-id", "9"}, false},
		{"second tab wrong", []string{"show-floating-panes", "--tab-id", "2", "--tab-id", "3"}, false},
		{"pane with foreign tab", []string{"move-pane", "--pane-id", "7", "--tab-id", "3"}, false},
		{"unaddressed tab", []string{"show-floating-panes"}, false},
		{"wrong focus", []string{"focus-pane-id", "9"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := nativeReplyAllowedTarget(tc.args, "1", "7", 2); got != tc.want {
				t.Fatalf("allowed=%v want %v argv=%v", got, tc.want, tc.args)
			}
		})
	}
}

func nativeReplySetEnvironment(t *testing.T, root, native string) {
	t.Helper()
	for _, leaf := range []string{"h", "c", "d", "k", "r", "s", "n", "t"} {
		if err := os.MkdirAll(filepath.Join(root, leaf), 0700); err != nil {
			t.Fatal(err)
		}
	}
	testpair.ApplyEnvironment(t, filepath.Join(root, "h"), "PATH="+filepath.Dir(native)+":/usr/bin:/bin", "TMPDIR="+filepath.Join(root, "t"), "SHELL=/bin/sh", "ENV=", "PS1=",
		"XDG_CONFIG_HOME="+filepath.Join(root, "c"), "XDG_DATA_HOME="+filepath.Join(root, "d"), "XDG_CACHE_HOME="+filepath.Join(root, "k"), "XDG_RUNTIME_DIR="+filepath.Join(root, "r"),
		"ZELLIJ_SOCKET_DIR="+filepath.Join(root, "s"), "PAIR_NOTIFY_SOCKET_DIR="+filepath.Join(root, "n"))
}

func TestNativeReplyFixtureChildEnvironmentIsPrivate(t *testing.T) {
	root := t.TempDir()
	t.Setenv("COUCH_TREE", "operator-context")
	t.Setenv("PAIR_API_TOKEN", "fixture-sentinel")
	nativeReplySetEnvironment(t, root, "/owned/bin/zellij")
	output, err := exec.Command("/usr/bin/env").Output()
	if err != nil {
		t.Fatal(err)
	}
	env := map[string]string{}
	for _, line := range strings.Split(string(output), "\n") {
		k, v, ok := strings.Cut(line, "=")
		if ok {
			env[k] = v
		}
	}
	for key, leaf := range map[string]string{"HOME": "h", "TMPDIR": "t", "ZELLIJ_SOCKET_DIR": "s", "XDG_RUNTIME_DIR": "r", "XDG_CACHE_HOME": "k", "PAIR_NOTIFY_SOCKET_DIR": "n"} {
		if env[key] != filepath.Join(root, leaf) {
			t.Fatalf("%s is not owned private %s", key, leaf)
		}
	}
	if env["COUCH_TREE"] != "" || env["PAIR_API_TOKEN"] != "" {
		t.Fatal("operator context crossed empty-environment boundary")
	}
}

func TestNativeReplyNoiseCancellationJoinsOwnedConnections(t *testing.T) {
	root := testpair.ShortRoot(t)
	listener, err := net.Listen("unix", filepath.Join(root, "socket"))
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	accepted := make(chan struct{}, 8)
	var server sync.WaitGroup
	server.Add(1)
	go func() {
		defer server.Done()
		var readers sync.WaitGroup
		defer readers.Wait()
		for i := 0; i < 8; i++ {
			c, err := listener.Accept()
			if err != nil {
				return
			}
			readers.Add(1)
			go func() { defer readers.Done(); defer c.Close(); accepted <- struct{}{}; _, _ = io.Copy(io.Discard, c) }()
		}
	}()
	noise := startNativeReplyNoise(t.Context(), listener.Addr().String())
	defer noise.stop()
	for i := 0; i < 8; i++ {
		select {
		case <-accepted:
		case <-t.Context().Done():
			t.Fatal(t.Context().Err())
		}
	}
	noise.stop()
	server.Wait()
	noise.mu.Lock()
	defer noise.mu.Unlock()
	if len(noise.conns) != 0 || !noise.closing {
		t.Fatalf("unjoined native connections=%d closing=%v", len(noise.conns), noise.closing)
	}
	if noise.Bad != 0 {
		t.Fatalf("intentional cancellation counted as native misrouting: %d", noise.Bad)
	}
}

func TestNativeReplyGeometryOracleChecksActualState(t *testing.T) {
	zero := 0
	notFullscreen := false
	initial := map[string]zellijpane.Pane{}
	for i, name := range []string{"agent", "draft", "terminal-top", "terminal-bottom"} {
		x, y := 0, (i%2)*12
		if i >= 2 {
			x = 40
		}
		initial[name] = zellijpane.Pane{ID: strconv.Itoa(i + 1), TabID: &zero, TabPosition: &zero, IsFullscreen: &notFullscreen, X: x, Y: y, Columns: 40, Rows: 12}
	}
	oracle := []nativeReplyEvent{{Args: []string{"current-tab-info"}, Stdout: `{"display_area_columns":80,"display_area_rows":24}`}, {Args: []string{"are-floating-panes-visible"}, Stdout: "true"}}
	for _, tc := range []struct {
		name   string
		focus  bool
		change func(map[string]zellijpane.Pane, []nativeReplyEvent)
		bad    bool
	}{
		{"exact normal", false, func(map[string]zellijpane.Pane, []nativeReplyEvent) {}, false},
		{"changed normal geometry", false, func(m map[string]zellijpane.Pane, _ []nativeReplyEvent) { p := m["agent"]; p.Columns++; m["agent"] = p }, true},
		{"centered focus", true, func(map[string]zellijpane.Pane, []nativeReplyEvent) {}, false},
		{"focus width drift", true, func(m map[string]zellijpane.Pane, _ []nativeReplyEvent) {
			p := m["terminal-bottom"]
			p.Columns = 70
			m["terminal-bottom"] = p
		}, true},
		{"focus not centered", true, func(m map[string]zellijpane.Pane, _ []nativeReplyEvent) {
			p := m["terminal-bottom"]
			p.X = 0
			m["terminal-bottom"] = p
		}, true},
		{"focus short height", true, func(m map[string]zellijpane.Pane, _ []nativeReplyEvent) {
			p := m["terminal-bottom"]
			p.Rows = 12
			m["terminal-bottom"] = p
		}, true},
		{"foreign identity", true, func(m map[string]zellijpane.Pane, _ []nativeReplyEvent) { p := m["agent"]; p.ID = "99"; m["agent"] = p }, true},
		{"wrong fullscreen", true, func(m map[string]zellijpane.Pane, _ []nativeReplyEvent) {
			p := m["agent"]
			yes := true
			p.IsFullscreen = &yes
			m["agent"] = p
		}, true},
		{"hidden focus", true, func(_ map[string]zellijpane.Pane, e []nativeReplyEvent) { e[1].Stdout = "false" }, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			final := map[string]zellijpane.Pane{}
			for k, v := range initial {
				final[k] = v
			}
			events := append([]nativeReplyEvent(nil), oracle...)
			if tc.focus {
				p := final["terminal-bottom"]
				p.IsFloating = true
				p.Columns = 60
				p.X = 10
				p.Y = 0
				p.Rows = 24
				final["terminal-bottom"] = p
			}
			tc.change(final, events)
			if err := nativeReplyGeometry(initial, final, tc.focus, events); (err != nil) != tc.bad {
				t.Fatalf("oracle error=%v wantReject=%v", err, tc.bad)
			}
		})
	}
}

func nativeReplyTimestamp() string { return time.Now().UTC().Format("2006-01-02T15:04:05.000000000Z") }
func (r *nativeReplyRuntime) record(e nativeReplyEvent) {
	e.At = nativeReplyTimestamp()
	r.events = append(r.events, e)
}

func nativeReplyVisibilityKnown(raw, diagnostic string, err error) bool {
	visible := strings.TrimSpace(raw)
	diagnostic = strings.TrimSpace(diagnostic)
	if err == nil {
		return visible == "true" && diagnostic == ""
	}
	exit, direct := err.(*exec.ExitError)
	if !direct {
		return false
	}
	return (exit.ExitCode() == 1 && visible == "false" && diagnostic == "") || (exit.ExitCode() == 2 && visible == "" && diagnostic == "false")
}

func TestNativeReplyVisibilityOracleRetainsOwnershipErrors(t *testing.T) {
	exitOne := exec.Command("/bin/sh", "-c", "exit 1").Run()
	exitTwo := exec.Command("/bin/sh", "-c", "exit 2").Run()
	for _, tc := range []struct {
		name, raw, diagnostic string
		err                   error
		known                 bool
	}{
		{"true", "true", "", nil, true},
		{"stdout false", "false", "", exitOne, true},
		{"stderr false", "", "false", exitTwo, true},
		{"false plus cancellation", "", "false", errors.Join(exitTwo, context.Canceled), false},
		{"false plus collection failure", "false", "", errors.Join(exitOne, errors.New("collector failed")), false},
		{"true plus collection failure", "true", "", errors.New("collector failed"), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := nativeReplyVisibilityKnown(tc.raw, tc.diagnostic, tc.err); got != tc.known {
				t.Fatalf("known=%v want %v error=%v", got, tc.known, tc.err)
			}
		})
	}
}
