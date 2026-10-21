package board

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"math"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/xianxu/pair/cmd/internal/broadcast"
	"github.com/xianxu/pair/cmd/internal/strictjson"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

const MaxSketchBody = 16 << 10
const MaxSketchTransfer = 32 << 10

// SketchCommand is a validated closed ingress union. Attribution, source time,
// anchor evidence and the persisted record are assigned by the service.
type SketchCommand struct {
	Kind, Operation string
	Sequence, Epoch uint64
	Sketch, ID      string
	Seg, Color      int
	Node            recording.NodeID
	Box             recording.Box
	Chunk           uint64
	Pts             []recording.SketchPoint
	Truncated       bool
	Label           string
	Source, Asset   string
	Offset, Length  int
}
type sketchCommandWire struct {
	Operation *string           `json:"operation"`
	Sequence  *uint64           `json:"sequence"`
	Epoch     *uint64           `json:"epoch"`
	Sketch    *string           `json:"sketch"`
	ID        *string           `json:"id"`
	Seg       *int              `json:"seg"`
	Color     *int              `json:"color"`
	Node      *recording.NodeID `json:"node"`
	Box       *[]*float64       `json:"box"`
	Chunk     *uint64           `json:"chunk"`
	Pts       *[][]*float64     `json:"pts"`
	Truncated *bool             `json:"truncated"`
	Label     *string           `json:"label"`
	Source    *string           `json:"source"`
	Asset     *string           `json:"asset"`
	Offset    *int              `json:"offset"`
	Length    *int              `json:"length"`
}

func sketchCommandFields(kind string) ([]string, bool) {
	switch kind {
	case "sketch-begin":
		return []string{"seg", "color"}, true
	case "sketch-stroke-begin":
		return []string{"sketch", "id", "node", "box"}, true
	case "sketch-points":
		return []string{"sketch", "id", "chunk", "pts"}, true
	case "sketch-stroke-end", "sketch-undo", "sketch-redo":
		return []string{"sketch", "id"}, true
	case "sketch-label":
		return []string{"sketch", "label"}, true
	case "sketch-status":
		return nil, true
	case "sketch-source":
		return []string{"sketch", "source", "offset", "length"}, true
	case "sketch-asset":
		return []string{"sketch", "source", "asset", "offset", "length"}, true
	}
	return nil, false
}
func sketchIdentifier(s string) bool {
	if len(s) == 0 || len(s) > recording.SketchMaxIDBytes || !utf8.ValidString(s) {
		return false
	}
	for _, c := range s {
		if unicode.IsControl(c) || unicode.Is(unicode.Cf, c) {
			return false
		}
	}
	return true
}
func sketchNumber(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0) && math.Abs(v) <= recording.SketchMaxCoordinate
}
func sketchSafe(n uint64) bool { return n <= recording.MaxSketchInteger }

// ParseSketchCommand refuses every field belonging to another command or the
// server, duplicate members, nulls, unsafe counters and malformed Unicode.
func ParseSketchCommand(kind string, body []byte) (SketchCommand, error) {
	fail := func() (SketchCommand, error) { return SketchCommand{}, ErrShape }
	payload, ok := sketchCommandFields(kind)
	if !ok || len(body) > MaxSketchBody {
		return fail()
	}
	required := append([]string{"operation", "sequence", "epoch"}, payload...)
	known := map[string]bool{}
	for _, k := range required {
		known[k] = true
	}
	if kind == "sketch-stroke-end" {
		known["truncated"] = true
	}
	if kind == "sketch-status" {
		known["sketch"] = true
	}
	owned, unknown, err := strictjson.SplitObject(body, known, nil)
	if err != nil || len(unknown) > 0 {
		return fail()
	}
	for _, k := range required {
		if _, ok := owned[k]; !ok {
			return fail()
		}
	}
	if strictjson.ValidateUnicode(body) != nil {
		return fail()
	}
	var raw sketchCommandWire
	if strictjson.Decode(body, &raw) != nil {
		return fail()
	}
	out := SketchCommand{Kind: kind, Operation: *raw.Operation, Sequence: *raw.Sequence, Epoch: *raw.Epoch}
	if !sketchIdentifier(out.Operation) || !sketchSafe(out.Sequence) || !sketchSafe(out.Epoch) {
		return fail()
	}
	if raw.Sketch != nil {
		out.Sketch = *raw.Sketch
		if !sketchIdentifier(out.Sketch) {
			return fail()
		}
	}
	if raw.ID != nil {
		out.ID = *raw.ID
		if !sketchIdentifier(out.ID) {
			return fail()
		}
	}
	switch kind {
	case "sketch-begin":
		out.Seg = *raw.Seg
		out.Color = *raw.Color
		if out.Sequence != 0 || !segOK(out.Seg) || !colorOK(out.Color) {
			return fail()
		}
	case "sketch-stroke-begin":
		out.Node = *raw.Node
		if out.Node <= 0 || uint64(out.Node) > recording.MaxSketchInteger || len(*raw.Box) != 4 {
			return fail()
		}
		for i, v := range *raw.Box {
			if v == nil || !sketchNumber(*v) {
				return fail()
			}
			out.Box[i] = *v
		}
		if out.Box[2] <= 0 || out.Box[3] <= 0 {
			return fail()
		}
	case "sketch-points":
		out.Chunk = *raw.Chunk
		if !sketchSafe(out.Chunk) || len(*raw.Pts) == 0 || len(*raw.Pts) > recording.SketchMaxPointsPerChunk {
			return fail()
		}
		out.Pts = make([]recording.SketchPoint, len(*raw.Pts))
		for i, tuple := range *raw.Pts {
			if len(tuple) != 3 {
				return fail()
			}
			for j, v := range tuple {
				if v == nil || !sketchNumber(*v) {
					return fail()
				}
				out.Pts[i][j] = *v
			}
			if out.Pts[i][2] < 0 || out.Pts[i][2] > 1 {
				return fail()
			}
		}
	case "sketch-status":
		if out.Sketch == "" && out.Sequence != 0 {
			return fail()
		}
	case "sketch-stroke-end":
		if raw.Truncated != nil {
			out.Truncated = *raw.Truncated
		}
	case "sketch-label":
		out.Label = *raw.Label
		if len(out.Label) > recording.SketchMaxLabelBytes || strings.TrimSpace(out.Label) == "" || !broadcast.PlainText(out.Label) {
			return fail()
		}
	case "sketch-source", "sketch-asset":
		out.Source = *raw.Source
		out.Offset = *raw.Offset
		out.Length = *raw.Length
		if !sketchIdentifier(out.Source) || out.Offset < 0 || uint64(out.Offset) > recording.MaxSketchInteger || out.Length <= 0 || out.Length > MaxSketchTransfer {
			return fail()
		}
		if kind == "sketch-asset" {
			out.Asset = *raw.Asset
			if !recording.ValidHash(out.Asset) {
				return fail()
			}
		}
	}
	return out, nil
}
func ParseSketchBegin(b []byte) (SketchCommand, error) { return ParseSketchCommand("sketch-begin", b) }
func ParseSketchStrokeBegin(b []byte) (SketchCommand, error) {
	return ParseSketchCommand("sketch-stroke-begin", b)
}
func ParseSketchPoints(b []byte) (SketchCommand, error) {
	return ParseSketchCommand("sketch-points", b)
}
func ParseSketchStrokeEnd(b []byte) (SketchCommand, error) {
	return ParseSketchCommand("sketch-stroke-end", b)
}
func ParseSketchUndo(b []byte) (SketchCommand, error)  { return ParseSketchCommand("sketch-undo", b) }
func ParseSketchRedo(b []byte) (SketchCommand, error)  { return ParseSketchCommand("sketch-redo", b) }
func ParseSketchLabel(b []byte) (SketchCommand, error) { return ParseSketchCommand("sketch-label", b) }
func ParseSketchStatus(b []byte) (SketchCommand, error) {
	return ParseSketchCommand("sketch-status", b)
}
func ParseSketchSource(b []byte) (SketchCommand, error) {
	return ParseSketchCommand("sketch-source", b)
}
func ParseSketchAsset(b []byte) (SketchCommand, error) { return ParseSketchCommand("sketch-asset", b) }

// SketchCommandCanonical is normalized ingress identity, not persisted-event
// identity. Epoch and kind participate; false truncation and numeric -0 do not.
func SketchCommandCanonical(c SketchCommand) ([]byte, error) {
	if _, ok := sketchCommandFields(c.Kind); !ok {
		return nil, ErrShape
	}
	m := map[string]any{"operation": c.Operation, "sequence": c.Sequence, "epoch": c.Epoch}
	switch c.Kind {
	case "sketch-begin":
		m["seg"] = c.Seg
		m["color"] = c.Color
	case "sketch-stroke-begin":
		m["sketch"] = c.Sketch
		m["id"] = c.ID
		m["node"] = c.Node
		m["box"] = c.Box
	case "sketch-points":
		m["sketch"] = c.Sketch
		m["id"] = c.ID
		m["chunk"] = c.Chunk
		m["pts"] = c.Pts
	case "sketch-stroke-end", "sketch-undo", "sketch-redo":
		m["sketch"] = c.Sketch
		m["id"] = c.ID
		if c.Kind == "sketch-stroke-end" && c.Truncated {
			m["truncated"] = true
		}
	case "sketch-label":
		m["sketch"] = c.Sketch
		m["label"] = c.Label
	case "sketch-status":
		if c.Sketch != "" {
			m["sketch"] = c.Sketch
		}
	case "sketch-source", "sketch-asset":
		m["sketch"] = c.Sketch
		m["source"] = c.Source
		m["offset"] = c.Offset
		m["length"] = c.Length
		if c.Kind == "sketch-asset" {
			m["asset"] = c.Asset
		}
	}
	for _, value := range m {
		if text, ok := value.(string); ok && !utf8.ValidString(text) {
			return nil, ErrShape
		}
	}
	raw, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	// Reparse to guarantee manually built command values satisfy the same union.
	if _, err := ParseSketchCommand(c.Kind, raw); err != nil {
		return nil, err
	}
	var normalized any
	if err := json.Unmarshal(raw, &normalized); err != nil {
		return nil, err
	}
	normalizeCommandZero(normalized)
	return json.Marshal(map[string]any{"kind": c.Kind, "command": normalized})
}
func normalizeCommandZero(v any) {
	switch x := v.(type) {
	case map[string]any:
		for k, v := range x {
			if f, ok := v.(float64); ok && f == 0 {
				x[k] = float64(0)
			} else {
				normalizeCommandZero(v)
			}
		}
	case []any:
		for i, v := range x {
			if f, ok := v.(float64); ok && f == 0 {
				x[i] = float64(0)
			} else {
				normalizeCommandZero(v)
			}
		}
	}
}
func SketchCommandDigest(c SketchCommand) (string, error) {
	raw, err := SketchCommandCanonical(c)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}

type SketchOutcome string

const (
	SketchAccepted    SketchOutcome = "accepted"
	SketchRefused     SketchOutcome = "refused"
	SketchUnknown     SketchOutcome = "unknown"
	SketchNotObserved SketchOutcome = "not-observed"
)

type SketchSourceDescriptor struct {
	Handle string             `json:"handle"`
	Time   int64              `json:"time"`
	Length int                `json:"length"`
	Doc    int                `json:"doc"`
	Tab    string             `json:"tab"`
	VP     recording.Viewport `json:"vp"`
}
type SketchStatus struct {
	Phase      string `json:"phase"`
	Reason     string `json:"reason"`
	Label      string `json:"label"`
	OpenStroke string `json:"open_stroke"`
	VisibleTop string `json:"visible_top"`
	RedoTop    string `json:"redo_top"`
}
type SketchReply struct {
	Operation    string                  `json:"operation"`
	Sketch       string                  `json:"sketch"`
	Sequence     uint64                  `json:"sequence"`
	NextSequence uint64                  `json:"next_sequence"`
	Outcome      SketchOutcome           `json:"outcome"`
	Code         string                  `json:"code"`
	Source       *SketchSourceDescriptor `json:"source,omitempty"`
	Status       *SketchStatus           `json:"status,omitempty"`
}

// ParseSketchReply is the closed reply boundary used by non-browser clients
// and fixtures. Source bytes travel separately; replies carry descriptors only.
func ParseSketchReply(body []byte) (SketchReply, error) {
	fail := func() (SketchReply, error) { return SketchReply{}, ErrShape }
	if len(body) > MaxSketchBody || strictjson.ValidateUnicode(body) != nil {
		return fail()
	}
	required := []string{"operation", "sketch", "sequence", "next_sequence", "outcome", "code"}
	known := map[string]bool{"source": true, "status": true}
	for _, k := range required {
		known[k] = true
	}
	owned, unknown, err := strictjson.SplitObject(body, known, nil)
	if err != nil || len(unknown) > 0 {
		return fail()
	}
	for _, k := range required {
		if _, ok := owned[k]; !ok {
			return fail()
		}
	}
	var out SketchReply
	if strictjson.Decode(body, &out) != nil {
		return fail()
	}
	if !sketchIdentifier(out.Operation) || out.Sketch != "" && !sketchIdentifier(out.Sketch) || !sketchSafe(out.Sequence) || !sketchSafe(out.NextSequence) {
		return fail()
	}
	switch out.Outcome {
	case SketchAccepted, SketchRefused, SketchUnknown, SketchNotObserved:
	default:
		return fail()
	}
	if len(out.Code) > 64 {
		return fail()
	}
	for _, c := range out.Code {
		if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-') {
			return fail()
		}
	}
	if out.Source != nil {
		if !sketchReplyObject(owned["source"], []string{"handle", "time", "length", "doc", "tab", "vp"}) {
			return fail()
		}
		var source map[string]json.RawMessage
		json.Unmarshal(owned["source"], &source)
		if !sketchReplyObject(source["vp"], []string{"w", "h", "dpr"}) {
			return fail()
		}
		v := out.Source
		if !sketchIdentifier(v.Handle) || v.Time < 0 || uint64(v.Time) > recording.MaxSketchInteger || v.Length <= 0 || v.Length > 16<<20 || v.Doc <= 0 || uint64(v.Doc) > recording.MaxSketchInteger || !sketchIdentifier(v.Tab) || v.VP.W <= 0 || uint64(v.VP.W) > recording.MaxSketchInteger || v.VP.H <= 0 || uint64(v.VP.H) > recording.MaxSketchInteger || !sketchNumber(v.VP.DPR) || v.VP.DPR <= 0 {
			return fail()
		}
	}
	if out.Status != nil {
		if !sketchReplyObject(owned["status"], []string{"phase", "reason", "label", "open_stroke", "visible_top", "redo_top"}) {
			return fail()
		}
		v := out.Status
		switch v.Phase {
		case "drawing", "complete", "interrupted", "cleared":
		default:
			return fail()
		}
		switch v.Reason {
		case "", recording.SketchInterruptExpiry, recording.SketchInterruptNavigation, recording.SketchInterruptHandoff, recording.SketchInterruptReset, recording.SketchInterruptEnd:
		default:
			return fail()
		}
		if v.Label != "" && (len(v.Label) > recording.SketchMaxLabelBytes || strings.TrimSpace(v.Label) == "" || !broadcast.PlainText(v.Label)) {
			return fail()
		}
		for _, id := range []string{v.OpenStroke, v.VisibleTop, v.RedoTop} {
			if id != "" && !sketchIdentifier(id) {
				return fail()
			}
		}
	}
	return out, nil
}
func sketchReplyObject(raw []byte, keys []string) bool {
	known := map[string]bool{}
	for _, k := range keys {
		known[k] = true
	}
	owned, unknown, err := strictjson.SplitObject(raw, known, nil)
	return err == nil && len(unknown) == 0 && len(owned) == len(keys)
}
