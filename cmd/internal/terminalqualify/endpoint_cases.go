package terminalqualify

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/xianxu/pair/cmd/internal/terminal"
	"github.com/xianxu/pair/cmd/internal/ttyio"
)

// EndpointCases measure actual production endpoints and their FIFO transport.
// Product focus/notification policy and wrapper/native acceptance remain separate
// required obligations; endpoint success is not evidence those consumers migrated.
func EndpointCases() []Case {
	const source = "workshop/plans/000255-terminal-abstraction-plan.md#chunk-2-m2-repaired-core-endpoint-and-presenter"
	cases := []Case{
		{ID: "device-attributes", Capability: "local endpoint profile and capability replies", Expected: Observation{"term": "pair-vt-256color", "replies": "\x1b[?1;2c\x1b[>0;1;0c\x1bP1+r544e=706169722d76742d323536636f6c6f72;436f=323536\x1b\\\x1bP0+r\x1b\\"}, Integration: observeEndpointProfile},
		{ID: "endpoint-input", Capability: "actual input decoder to negotiated endpoint writer", Expected: Observation{"replies-filtered": "1", "wire": "\x1b[13;5:3u\x1b[<32;3;2M\x1b[I\x1b[200~a\x1b[A\x1b[201~"}, Integration: observeEndpointInput},
		{ID: "sync-publication", Capability: "immutable synchronized endpoint publication", Expected: Observation{"held": "A", "same-write-held": "A", "released": "C", "owned": "C"}, Integration: observeEndpointSync},
		{ID: "sync-recovery", Capability: "bounded synchronized endpoint recovery", Expected: Observation{"before-deadline": "A", "at-deadline": "B", "after-timeout": "C"}, Integration: observeEndpointRecovery},
		{ID: "reply-backpressure-routing", Capability: "origin-bound FIFO replies, paste and keys under blocked transport", Expected: Observation{"blocked-a": "", "independent-b": "\x1b[3;4R", "ordered-a": "X\x1b[2;3R\x1b[200~p\x1b[A\x1b[201~q", "bounded": "true", "blocked-started": "true", "joined": "true"}, Integration: observeEndpointTransport},
		{ID: "endpoint-effects", Capability: "typed once-only effects and deterministic clipboard policy", Expected: Observation{"kinds": "0,1,2,3,4", "origin": "effects", "positions": "true", "sequences": "1,2,3,4,5", "clipboard": "hi", "notification": "hello", "repeated": "0", "query": "\x1b]52;c;\x1b\\", "oversize-effects": "0"}, Integration: observeEndpointEffects},
		{ID: "endpoint-resize", Capability: "geometry epoch only after acknowledged PTY outcome", Expected: Observation{"invalid-applied": "false", "failure-preserved": "true", "success": "10x5", "epoch-advanced": "true", "applied": "10x5"}, Integration: observeEndpointResize},
	}
	for i := range cases {
		cases[i].Target = "endpoint"
		cases[i].Source = source
	}
	return cases
}

func observeEndpointProfile(ctx context.Context) (Observation, error) {
	e, out, err := scenarioEndpoint("profile")
	if err != nil {
		return nil, err
	}
	defer e.Close()
	if _, err = e.Feed([]byte("\x1b[c\x1b[>c\x1bP+q544e;436f\x1b\\\x1bP+q736978656c\x1b\\"), time.Unix(100, 0)); err != nil {
		return nil, err
	}
	if err = e.Flush(ctx); err != nil {
		return nil, err
	}
	return Observation{"term": terminal.DefaultProfile().TERM, "replies": string(out.Bytes())}, nil
}

func observeEndpointInput(ctx context.Context) (Observation, error) {
	e, out, err := scenarioEndpoint("decoded-input")
	if err != nil {
		return nil, err
	}
	defer e.Close()
	if _, err = e.Feed([]byte("\x1b[>3u\x1b[?1002;1006;1004;2004h"), time.Unix(100, 0)); err != nil {
		return nil, err
	}
	var decoder terminal.Decoder
	events, err := decoder.Feed([]byte("\x1b[13;5:3u\x1b[<32;3;2M\x1b[I\x1b[200~a\x1b[A\x1b[201~\x1b[?1;2c"))
	if err != nil {
		return nil, err
	}
	replies := 0
	for _, event := range events {
		if event.Reply {
			replies++
			continue
		}
		if err = e.Send(event.Event); err != nil {
			return nil, err
		}
	}
	if err = e.Flush(ctx); err != nil {
		return nil, err
	}
	return Observation{"replies-filtered": strconv.Itoa(replies), "wire": string(out.Bytes())}, nil
}

func scenarioEndpoint(id string) (*terminal.Endpoint, *ttyio.Fake, error) {
	out := ttyio.NewFake()
	e, err := terminal.NewEndpoint(id, terminal.Geometry{Cols: 8, Rows: 4}, out)
	return e, out, err
}
func cellText(f terminal.Frame) string {
	if len(f.Cells) == 0 {
		return "<missing>"
	}
	return f.Cells[0].Content
}

func observeEndpointSync(ctx context.Context) (Observation, error) {
	e, _, err := scenarioEndpoint("sync")
	if err != nil {
		return nil, err
	}
	defer e.Close()
	now := time.Unix(100, 0)
	if _, err = e.Feed([]byte("A"), now); err != nil {
		return nil, err
	}
	// No Snapshot before begin: hidden endpoints also need the complete pre-sync
	// state, not whichever older screen happened to be requested by the UI.
	if _, err = e.Feed([]byte("\x1b[?2026h\rB"), now); err != nil {
		return nil, err
	}
	held, err := e.Snapshot(now.Add(149 * time.Millisecond))
	if err != nil {
		return nil, err
	}
	if _, err = e.Feed([]byte("\x1b[?2026l\rC"), now.Add(149*time.Millisecond)); err != nil {
		return nil, err
	}
	released, err := e.Snapshot(now.Add(149 * time.Millisecond))
	if err != nil {
		return nil, err
	}
	observed := Observation{"held": cellText(held), "released": cellText(released)}
	released.Cells[0].Content = "changed by caller"
	fresh, err := e.Snapshot(now.Add(149 * time.Millisecond))
	if err != nil {
		return nil, err
	}
	observed["owned"] = cellText(fresh)
	second, _, err := scenarioEndpoint("same-write")
	if err != nil {
		return nil, err
	}
	defer second.Close()
	if _, err = second.Feed([]byte("A\x1b[?2026h\rB"), now); err != nil {
		return nil, err
	}
	same, err := second.Snapshot(now.Add(time.Millisecond))
	if err != nil {
		return nil, err
	}
	observed["same-write-held"] = cellText(same)
	return observed, ctx.Err()
}

func observeEndpointRecovery(ctx context.Context) (Observation, error) {
	e, _, err := scenarioEndpoint("recovery")
	if err != nil {
		return nil, err
	}
	defer e.Close()
	now := time.Unix(100, 0)
	if _, err = e.Feed([]byte("A\x1b[?2026h\rB"), now); err != nil {
		return nil, err
	}
	held, err := e.Snapshot(now.Add(terminal.SyncTimeout - time.Nanosecond))
	if err != nil {
		return nil, err
	}
	recovered, err := e.Snapshot(now.Add(terminal.SyncTimeout))
	if err != nil {
		return nil, err
	}
	if _, err = e.Feed([]byte("\rC"), now.Add(terminal.SyncTimeout+time.Nanosecond)); err != nil {
		return nil, err
	}
	after, err := e.Snapshot(now.Add(terminal.SyncTimeout + time.Nanosecond))
	if err != nil {
		return nil, err
	}
	return Observation{"before-deadline": cellText(held), "at-deadline": cellText(recovered), "after-timeout": cellText(after)}, ctx.Err()
}

func observeEndpointTransport(ctx context.Context) (Observation, error) {
	a, aw, err := scenarioEndpoint("a")
	if err != nil {
		return nil, err
	}
	defer a.Close()
	b, bw, err := scenarioEndpoint("b")
	if err != nil {
		return nil, err
	}
	defer b.Close()
	block := make(chan struct{})
	aw.Enqueue(ttyio.WriteStep{Block: block, Limit: 1})
	if err = a.Send(uv.KeyPressEvent{Code: 'X', Text: "X"}); err != nil {
		return nil, err
	}
	select {
	case <-aw.Started():
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	now := time.Unix(100, 0)
	if _, err = a.Feed([]byte("\x1b[?2004h\x1b[2;3H\x1b[6n"), now); err != nil {
		return nil, err
	}
	if err = a.Send(uv.PasteEvent{Content: "p\x1b[A"}); err != nil {
		return nil, err
	}
	if err = a.Send(uv.KeyPressEvent{Code: 'q', Text: "q"}); err != nil {
		return nil, err
	}
	if _, err = b.Feed([]byte("\x1b[3;4H\x1b[6n"), now); err != nil {
		return nil, err
	}
	if err = b.Flush(ctx); err != nil {
		return nil, err
	}
	observed := Observation{"blocked-a": string(aw.Bytes()), "independent-b": string(bw.Bytes())}
	close(block)
	if err = a.Flush(ctx); err != nil {
		return nil, err
	}
	observed["ordered-a"] = string(aw.Bytes())
	bounded, writer, err := scenarioEndpoint("bounded")
	if err != nil {
		return nil, err
	}
	defer bounded.Close()
	writer.Enqueue(ttyio.WriteStep{Block: make(chan struct{})})
	for i := 0; i < terminal.MaxInputPackets; i++ {
		if err = bounded.Send(uv.KeyPressEvent{Code: 'x', Text: "x"}); err != nil {
			return nil, err
		}
	}
	err = bounded.Send(uv.KeyPressEvent{Code: 'x', Text: "x"})
	observed["bounded"] = strconv.FormatBool(errors.Is(err, terminal.ErrBackpressure))
	select {
	case <-writer.Started():
		observed["blocked-started"] = "true"
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	// Close cancels the blocked production InputWriter and joins it. Subsequent
	// methods observe closure; a detached worker is not a successful teardown.
	bounded.Close()
	observed["joined"] = strconv.FormatBool(bounded.Send(uv.KeyPressEvent{Code: 'x'}) != nil)
	return observed, ctx.Err()
}

func observeEndpointEffects(ctx context.Context) (Observation, error) {
	e, out, err := scenarioEndpoint("effects")
	if err != nil {
		return nil, err
	}
	defer e.Close()
	now := time.Unix(100, 0)
	raw := "\a\x1b]2;title\a\x1b]7;file:///tmp/probe\a\x1b]52;c;aGk=\a\x1b]777;notify;pair;hello\a\x1b]52;c;?\a"
	result, err := e.Feed([]byte(raw), now)
	if err != nil {
		return nil, err
	}
	var kinds, sequences []string
	origin := "effects"
	positions := true
	observed := Observation{}
	for _, effect := range result.Effects {
		kinds = append(kinds, strconv.Itoa(int(effect.Kind)))
		sequences = append(sequences, strconv.FormatUint(effect.Sequence, 10))
		if effect.EndpointID != "effects" {
			origin = effect.EndpointID
		}
		positions = positions && effect.Position == uint64(len(raw))
		if effect.Kind == terminal.ClipboardEffect {
			observed["clipboard"] = string(effect.Data)
		}
		if effect.Kind == terminal.NotificationEffect {
			observed["notification"] = effect.Text
		}
	}
	observed["kinds"] = strings.Join(kinds, ",")
	observed["sequences"] = strings.Join(sequences, ",")
	observed["origin"] = origin
	observed["positions"] = strconv.FormatBool(positions)
	if _, err = e.Snapshot(now); err != nil {
		return nil, err
	}
	if _, err = e.Snapshot(now); err != nil {
		return nil, err
	}
	empty, err := e.Feed(nil, now)
	if err != nil {
		return nil, err
	}
	observed["repeated"] = strconv.Itoa(len(empty.Effects))
	if err = e.Flush(ctx); err != nil {
		return nil, err
	}
	observed["query"] = string(out.Bytes())
	oversized := "\x1b]52;c;" + strings.Repeat("a", terminal.MaxStringBytes+8) + "\a"
	rejected, err := e.Feed([]byte(oversized), now)
	if err != nil {
		return nil, err
	}
	observed["oversize-effects"] = strconv.Itoa(len(rejected.Effects))
	return observed, ctx.Err()
}

func observeEndpointResize(ctx context.Context) (Observation, error) {
	e, _, err := scenarioEndpoint("resize")
	if err != nil {
		return nil, err
	}
	defer e.Close()
	now := time.Unix(100, 0)
	before, err := e.Snapshot(now)
	if err != nil {
		return nil, err
	}
	applied := false
	invalid := e.Resize(terminal.Geometry{Cols: terminal.MaxCells + 1, Rows: 1}, func(terminal.Geometry) error { applied = true; return nil })
	if invalid == nil {
		return Observation{"invalid-applied": "accepted"}, nil
	}
	failure := errors.New("synthetic ioctl failure")
	if err = e.Resize(terminal.Geometry{Cols: 10, Rows: 5}, func(terminal.Geometry) error { return failure }); !errors.Is(err, failure) {
		return nil, fmt.Errorf("resize failure outcome: %v", err)
	}
	after, err := e.Snapshot(now)
	if err != nil {
		return nil, err
	}
	observed := Observation{"invalid-applied": strconv.FormatBool(applied), "failure-preserved": strconv.FormatBool(after.Geometry == before.Geometry && after.GeometryEpoch == before.GeometryEpoch)}
	if err = e.Resize(terminal.Geometry{Cols: 10, Rows: 5}, func(g terminal.Geometry) error {
		observed["applied"] = fmt.Sprintf("%dx%d", g.Cols, g.Rows)
		return nil
	}); err != nil {
		return nil, err
	}
	resized, err := e.Snapshot(now)
	if err != nil {
		return nil, err
	}
	observed["success"] = fmt.Sprintf("%dx%d", resized.Geometry.Cols, resized.Geometry.Rows)
	observed["epoch-advanced"] = strconv.FormatBool(resized.GeometryEpoch > before.GeometryEpoch)
	return observed, ctx.Err()
}
