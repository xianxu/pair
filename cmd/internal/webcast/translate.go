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
	pageDetached struct{ SessionID string }
	// contextCreated is the control's world in a page's top frame.
	contextCreated   struct{ SessionID string; ContextID int64 }
	contextDestroyed struct{ SessionID string; ContextID int64 }
	contextsCleared  struct{ SessionID string }
	// toggleRequested: the in-page Go live / Pause control was clicked.
	toggleRequested struct{ SessionID string }
	// helloRequested: a new document's control asks for the current state.
	helloRequested struct{ SessionID string; ContextID int64 }
)

type targetInfo struct {
	TargetID string `json:"targetId"`
	Type     string `json:"type"`
	URL      string `json:"url"`
	Title    string `json:"title"`
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
			return []any{TargetCreated{ID: ti.TargetID, URL: ti.URL, Title: ti.Title}}
		}
		return []any{TargetChanged{ID: ti.TargetID, URL: ti.URL, Title: ti.Title}}
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
		if json.Unmarshal(params, &p) != nil || p.TargetInfo.Type != "page" || p.SessionID == "" {
			return nil
		}
		return []any{pageAttached{TargetID: p.TargetInfo.TargetID, SessionID: p.SessionID}}
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
		Visible bool   `json:"visible"`
		Focused bool   `json:"focused"`
	}
	if json.Unmarshal([]byte(p.Payload), &m) != nil {
		return nil
	}
	switch m.T {
	case "vis":
		return []any{PageReport{ID: target, Visible: m.Visible, Focused: m.Focused}}
	case "toggle":
		return []any{toggleRequested{SessionID: sessionID}}
	case "hello":
		return []any{helloRequested{SessionID: sessionID, ContextID: p.ContextID}}
	}
	return nil
}
