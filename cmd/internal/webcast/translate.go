package webcast

import "encoding/json"

// The isolated world the in-page control runs in, and the binding it reports
// through. The page's own scripts see neither.
const (
	worldName   = "couchcast"
	bindingName = "__couchcast"
)

// Loop inputs translate produces besides TrackerInputs. Each names the
// session it arrived on; Chrome's loop maps sessions to targets.
type (
	pageAttached struct{ TargetID, SessionID string }
	// frameAttached: an out-of-process iframe's session, under Parent's.
	frameAttached struct{ TargetID, SessionID, Parent string }
	pageDetached  struct{ SessionID string }
	// contextCreated is the control's world in a page's top frame.
	contextCreated struct {
		SessionID string
		ContextID int64
	}
	contextDestroyed struct {
		SessionID string
		ContextID int64
	}
	contextsCleared struct{ SessionID string }
	// dialogOpened, dialogClosed: a page's JavaScript dialog (#467).
	dialogOpened struct{ SessionID string }
	dialogClosed struct{ SessionID string }
	// toggleRequested: the in-page Go live / Pause control was clicked.
	toggleRequested struct{ SessionID string }
	// helloRequested: a new document's control asks for the current state.
	helloRequested struct {
		SessionID string
		ContextID int64
	}
)

type targetInfo struct {
	TargetID string `json:"targetId"`
	Type     string `json:"type"`
	URL      string `json:"url"`
	Title    string `json:"title"`
	OpenerID string `json:"openerId"`
}

// translate maps one CDP event to the inputs it means. targetOf maps an
// attached session to its page's target id. Anything unrecognised, from a
// non-page target, or malformed yields nothing: an event we can't read
// changes no state.
func translate(sessionID, method string, params json.RawMessage, targetOf func(sessionID string) (string, bool)) []any {
	switch method {
	case "Target.targetCreated", "Target.targetInfoChanged":
		var p struct {
			TargetInfo targetInfo `json:"targetInfo"`
		}
		if json.Unmarshal(params, &p) != nil || p.TargetInfo.Type != "page" || p.TargetInfo.TargetID == "" {
			return nil
		}
		ti := p.TargetInfo
		if method == "Target.targetCreated" {
			return []any{TargetCreated{ID: ti.TargetID, URL: ti.URL, Title: ti.Title, Opener: ti.OpenerID}}
		}
		return []any{TargetChanged{ID: ti.TargetID, URL: ti.URL, Title: ti.Title}}
	case "Page.javascriptDialogOpening":
		return []any{dialogOpened{SessionID: sessionID}}
	case "Page.javascriptDialogClosed":
		return []any{dialogClosed{SessionID: sessionID}}
	case "Target.targetDestroyed":
		var p struct {
			TargetID string `json:"targetId"`
		}
		if json.Unmarshal(params, &p) != nil || p.TargetID == "" {
			return nil
		}
		return []any{TargetDestroyed{ID: p.TargetID}}
	case "Target.attachedToTarget":
		var p struct {
			SessionID  string     `json:"sessionId"`
			TargetInfo targetInfo `json:"targetInfo"`
		}
		if json.Unmarshal(params, &p) != nil || p.SessionID == "" {
			return nil
		}
		switch p.TargetInfo.Type {
		case "page":
			return []any{pageAttached{TargetID: p.TargetInfo.TargetID, SessionID: p.SessionID}}
		case "iframe":
			return []any{frameAttached{TargetID: p.TargetInfo.TargetID, SessionID: p.SessionID, Parent: sessionID}}
		}
		return nil
	case "Target.detachedFromTarget":
		var p struct {
			SessionID string `json:"sessionId"`
		}
		if json.Unmarshal(params, &p) != nil || p.SessionID == "" {
			return nil
		}
		return []any{pageDetached{SessionID: p.SessionID}}
	case "Runtime.executionContextCreated":
		var p struct {
			Context struct {
				ID      int64  `json:"id"`
				Name    string `json:"name"`
				AuxData struct {
					FrameID string `json:"frameId"`
				} `json:"auxData"`
			} `json:"context"`
		}
		if json.Unmarshal(params, &p) != nil || p.Context.Name != worldName {
			return nil
		}
		// A page's top frame has the target's id; subframes don't.
		if target, ok := targetOf(sessionID); !ok || p.Context.AuxData.FrameID != target {
			return nil
		}
		return []any{contextCreated{SessionID: sessionID, ContextID: p.Context.ID}}
	case "Runtime.executionContextDestroyed":
		var p struct {
			ID int64 `json:"executionContextId"`
		}
		if json.Unmarshal(params, &p) != nil {
			return nil
		}
		return []any{contextDestroyed{SessionID: sessionID, ContextID: p.ID}}
	case "Runtime.executionContextsCleared":
		return []any{contextsCleared{SessionID: sessionID}}
	case "Page.frameNavigated":
		// Cross-document navigations only; a same-document one (pushState)
		// keeps its document and its title.
		var p struct {
			Frame struct {
				ID       string `json:"id"`
				ParentID string `json:"parentId"`
			} `json:"frame"`
		}
		if json.Unmarshal(params, &p) != nil || p.Frame.ParentID != "" {
			return nil
		}
		if target, ok := targetOf(sessionID); ok && p.Frame.ID == target {
			return []any{DocumentChanged{ID: target}}
		}
		return nil
	case "Runtime.bindingCalled":
		return translateBinding(sessionID, params, targetOf)
	}
	return nil
}

func translateBinding(sessionID string, params json.RawMessage, targetOf func(string) (string, bool)) []any {
	var p struct {
		Name      string `json:"name"`
		Payload   string `json:"payload"`
		ContextID int64  `json:"executionContextId"`
	}
	if json.Unmarshal(params, &p) != nil || p.Name != bindingName {
		return nil
	}
	target, ok := targetOf(sessionID)
	if !ok {
		return nil
	}
	var m struct {
		T       string `json:"t"`
		Title   string `json:"title"`
		Visible bool   `json:"visible"`
		Focused bool   `json:"focused"`
	}
	if json.Unmarshal([]byte(p.Payload), &m) != nil {
		return nil
	}
	switch m.T {
	case "vis":
		return []any{PageReport{ID: target, Visible: m.Visible, Focused: m.Focused}}
	case "title":
		return []any{PageTitle{ID: target, Title: m.Title}}
	case "toggle":
		return []any{toggleRequested{SessionID: sessionID}}
	case "hello":
		return []any{helloRequested{SessionID: sessionID, ContextID: p.ContextID}}
	}
	return nil
}
