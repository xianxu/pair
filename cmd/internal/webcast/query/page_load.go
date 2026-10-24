package query

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

// PageLoadFacts is an immutable semantic projection, not recording wire state.
// Its zero value means unknown; the scalar fields keep source witnesses
// comparable without retaining mutable extension maps or pointer identities.
type PageLoadFacts struct {
	Target string `json:"target"`
	Loader string `json:"loader"`
	Root   string `json:"root"`
}

func PageLoadFactsOf(load *recording.PageLoad) PageLoadFacts {
	if load.Validate() != nil {
		return PageLoadFacts{}
	}
	return PageLoadFacts{Target: load.Target, Loader: load.Loader, Root: strconv.FormatInt(load.Root, 10)}
}

// CurrentSourceFacts describes this exact accepted scene boundary. Creation
// provenance remains separate when a sketch is finished on a later boundary.
func CurrentSourceFacts(state *recording.State, index int) SketchSourceFacts {
	facts := SketchSourceFacts{T: state.T, RecordIndex: index, Doc: state.DocSeg, Tab: state.Tab.ID, URL: state.URL, Title: state.Title, VP: state.VP, PageLoad: PageLoadFactsOf(state.CurrentPageLoad())}
	if state.Top != nil {
		facts.Scroll = state.Scrolls[state.Top.ID]
	}
	return facts
}

// SketchBeginSourceFacts uses the accepted BEGIN's source identity and viewport.
// Its load belongs to that creation segment, never a later reporting context.
func SketchBeginSourceFacts(state *recording.State, begin *recording.SketchBegin, index int) SketchSourceFacts {
	facts := CurrentSourceFacts(state, index)
	facts.T, facts.Doc, facts.Tab, facts.VP = begin.T, begin.Doc, begin.Tab, begin.VP
	facts.PageLoad = PageLoadFactsOf(state.PageLoadAt(begin.Doc))
	return facts
}

// UnmarshalJSON admits only a complete canonical wire identity when present.
// A missing optional field leaves the comparable zero value unchanged.
func (p *PageLoadFacts) UnmarshalJSON(raw []byte) error {
	var load recording.PageLoad
	if err := json.Unmarshal(raw, &load); err != nil {
		return err
	}
	*p = PageLoadFactsOf(&load)
	return nil
}

// Validate accepts unknown absence or a complete exact semantic identity.
func (p PageLoadFacts) Validate() error {
	if p == (PageLoadFacts{}) {
		return nil
	}
	root, err := strconv.ParseInt(p.Root, 10, 64)
	if err != nil || strconv.FormatInt(root, 10) != p.Root {
		return fmt.Errorf("invalid page load root")
	}
	return (&recording.PageLoad{Target: p.Target, Loader: p.Loader, Root: root}).Validate()
}
