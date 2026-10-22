package webcast

import (
	"encoding/json"
	"github.com/xianxu/pair/cmd/internal/cdp"
	"github.com/xianxu/pair/cmd/internal/strictjson"
	"github.com/xianxu/pair/cmd/internal/webcast/board"
	"strconv"
)

type operatorSketchScope struct {
	scope SketchSceneScope
	life  chan struct{}
}

// Only Chrome's event loop creates or retires these witnesses. Context numbers
// and loader strings are descriptive; the original closeable lifetime fences ABA.
func (c *Chrome) operatorSketchScope(s *loopState, session string, id int64) *operatorSketchScope {
	if s.sketchScopes == nil {
		s.sketchScopes = map[operatorContextID]*operatorSketchScope{}
	}
	key := operatorContextID{session, id}
	if value := s.sketchScopes[key]; value != nil {
		return value
	}
	loader := s.sketchLoaders[session]
	if loader == "" {
		loader = "context-" + strconv.FormatInt(id, 10)
	}
	value := &operatorSketchScope{SketchSceneScope{ParentSession: session, ParentContext: id, Loader: loader}, make(chan struct{})}
	s.sketchScopes[key] = value
	return value
}
func (c *Chrome) retireOperatorSketchScopes(s *loopState, session string) {
	for key, scope := range s.sketchScopes {
		if session == "" || key.session == session {
			close(scope.life)
			delete(s.sketchScopes, key)
		}
	}
	c.refreshOperator()
}
func (c *Chrome) handleSketchBinding(s *loopState, event cdp.Event) bool {
	if event.Method != "Runtime.bindingCalled" || len(event.Params) > 6*board.MaxSketchBody+4096 {
		return false
	}
	var binding struct {
		Name    string `json:"name"`
		Context int64  `json:"executionContextId"`
		Payload string `json:"payload"`
	}
	if json.Unmarshal(event.Params, &binding) != nil || binding.Name != bindingName || len(binding.Payload) > board.MaxSketchBody+2048 {
		return false
	}
	var outer struct {
		T       string          `json:"t"`
		Request json.RawMessage `json:"request"`
	}
	if strictjson.Decode([]byte(binding.Payload), &outer) != nil || outer.T != "sketch" {
		return false
	}
	// Consume all recognized sketch attempts; a page's claimed context cannot
	// reach the recorder or earn a fresh witness later in the worker queue.
	if c.operatorSketch == nil || binding.Context <= 0 || s.contexts[event.SessionID] != binding.Context || s.sessions[event.SessionID] == "" {
		return true
	}
	envelope, err := parseOperatorSketchEnvelope(outer.Request)
	if err != nil {
		return true
	}
	scope := c.operatorSketchScope(s, event.SessionID, binding.Context)
	c.operatorSketch.Enqueue(operatorSketchRequest{Scope: scope.scope, Lifetime: scope.life, Envelope: envelope})
	return true
}
