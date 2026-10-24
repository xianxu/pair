package recording

import (
	"encoding/json"
	"errors"
	"strconv"
	"unicode/utf8"

	"github.com/xianxu/pair/cmd/internal/strictjson"
)

// PageLoad names a browser document by trusted CDP facts, independently of
// activation segments and recording NodeIDs. Root is a decimal string on wire.
type PageLoad struct {
	Target  string `json:"target"`
	Loader  string `json:"loader"`
	Root    int64  `json:"-"`
	unknown map[string]json.RawMessage
}

func (p *PageLoad) Validate() error {
	if p == nil || p.Target == "" || p.Loader == "" || !utf8.ValidString(p.Target) || !utf8.ValidString(p.Loader) || p.Root <= 0 {
		return errors.New("invalid page load identity")
	}
	return nil
}

// SamePageLoad compares owned identity only. Unknown never matches unknown.
func SamePageLoad(a, b *PageLoad) bool {
	return a != nil && b != nil && a.Validate() == nil && b.Validate() == nil && a.Target == b.Target && a.Loader == b.Loader && a.Root == b.Root
}

var pageLoadFields = map[string]bool{"target": true, "loader": true, "root": true}

func (p *PageLoad) UnmarshalJSON(raw []byte) error {
	owned, unknown, err := strictjson.SplitObjectPreserveNumbers(raw, pageLoadFields, nil)
	if err != nil {
		return err
	}
	for _, key := range []string{"target", "loader", "root"} {
		if _, ok := owned[key]; !ok {
			return errors.New("missing page load " + key)
		}
		if err := strictjson.ValidateUnicode(owned[key]); err != nil {
			return err
		}
	}
	if err := checkSketchOwnedKeyUnicode(raw, pageLoadFields); err != nil {
		return err
	}
	var target, loader, root string
	if err = json.Unmarshal(owned["target"], &target); err != nil {
		return err
	}
	if err = json.Unmarshal(owned["loader"], &loader); err != nil {
		return err
	}
	if err = json.Unmarshal(owned["root"], &root); err != nil {
		return err
	}
	if root == "" || root[0] < '1' || root[0] > '9' {
		return errors.New("invalid page load root")
	}
	for _, c := range root {
		if c < '0' || c > '9' {
			return errors.New("invalid page load root")
		}
	}
	n, err := strconv.ParseInt(root, 10, 64)
	if err != nil {
		return err
	}
	if len(unknown) == 0 {
		unknown = nil
	}
	next := PageLoad{Target: target, Loader: loader, Root: n, unknown: unknown}
	if err = next.Validate(); err != nil {
		return err
	}
	*p = next
	return nil
}
func (p PageLoad) MarshalJSON() ([]byte, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}
	owned := map[string]json.RawMessage{}
	for k, v := range map[string]string{"target": p.Target, "loader": p.Loader, "root": strconv.FormatInt(p.Root, 10)} {
		owned[k], _ = json.Marshal(v)
	}
	return strictjson.EncodeObjectPreserveNumbers(owned, p.unknown, pageLoadFields)
}
func clonePageLoad(p *PageLoad) *PageLoad {
	if p == nil {
		return nil
	}
	out := *p
	out.unknown = strictjson.CloneRaw(p.unknown)
	return &out
}

// PageLoadAt returns an owned copy; callers cannot mutate accepted membership.
func (s *State) PageLoadAt(seg int) *PageLoad { return clonePageLoad(s.pageLoads[seg]) }
func (s *State) CurrentPageLoad() *PageLoad   { return s.PageLoadAt(s.Seg) }

// OnPageLoad tests creation membership without copying preserved extensions.
func (s *State) OnPageLoad(seg int, load *PageLoad) bool {
	return SamePageLoad(s.pageLoads[seg], load)
}
