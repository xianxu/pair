package recorder

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"strings"
)

// ExcludeSketchFrame is an actor barrier. Call it before installing any private
// source. Roots and ancestry belong to live CDP frames, not recording history.
func (r *Recorder) ExcludeSketchFrame(ctx context.Context, frameID, session string) error {
	if frameID == "" || session == "" {
		return errors.New("recorder: missing private frame scope")
	}
	return r.acknowledged(ctx, func() error {
		if r.sketchFrames == nil {
			r.sketchFrames = map[string]string{}
		}
		if _, ok := r.sketchFrames[frameID]; !ok {
			r.sketchFrames[frameID] = session
		}
		if r.frameIDs[session] == frameID {
			if r.sketchSessions == nil {
				r.sketchSessions = map[string]bool{}
			}
			r.sketchSessions[session] = true
		}
		if r.frameParents == nil {
			r.frameParents = map[string]string{}
		}
		if _, ok := r.frameParents[frameID]; !ok {
			r.frameParents[frameID] = ""
		}
		for id, h := range r.headers {
			if r.excludedFrame(h.FrameID) {
				delete(r.headers, id)
			}
		}
		for id, q := range r.requests {
			if r.excludedFrame(q.frame) {
				delete(r.requests, id)
			}
		}
		return nil
	})
}

// AdmitRecordingFrame orders setup before any observer injection. A parent
// frame tree refresh resolves OOPIF attachment delivered before frameAttached.
func (r *Recorder) AdmitRecordingFrame(ctx context.Context, frameID, session, parent string) (bool, error) {
	allowed := false
	err := r.acknowledged(ctx, func() error {
		if r.excludedSession(parent) {
			return nil
		}
		if _, known := r.frameParents[frameID]; !known && parent != "" {
			var tree struct {
				FrameTree frameTree `json:"frameTree"`
			}
			if _, err := r.call(parent, "Page.getFrameTree", nil, &tree); err != nil {
				return err
			}
			r.learnFrameTree(tree.FrameTree, "")
			if _, known := r.frameParents[frameID]; !known && len(r.sketchFrames) > 0 {
				return errors.New("recorder: unresolved frame ancestry during private scene")
			}
		}
		allowed = !r.excludedFrame(frameID) && !r.excludedSession(parent)
		return nil
	})
	return allowed, err
}

type frameTree struct {
	Frame struct {
		ID     string `json:"id"`
		Parent string `json:"parentId"`
	} `json:"frame"`
	Children []frameTree `json:"childFrames"`
}

func (r *Recorder) learnFrameTree(t frameTree, parent string) {
	if t.Frame.ID == "" {
		return
	}
	if t.Frame.Parent != "" {
		parent = t.Frame.Parent
	}
	if r.frameParents == nil {
		r.frameParents = map[string]string{}
	}
	r.frameParents[t.Frame.ID] = parent
	for _, c := range t.Children {
		r.learnFrameTree(c, t.Frame.ID)
	}
}
func (r *Recorder) excludedFrame(frame string) bool {
	for n := 0; frame != "" && n < 128; n++ {
		if _, ok := r.sketchFrames[frame]; ok {
			return true
		}
		frame = r.frameParents[frame]
	}
	return false
}
func (r *Recorder) excludedSession(session string) bool {
	return r.sketchSessions[session] || r.excludedFrame(r.frameIDs[session])
}
func (r *Recorder) forgetFrame(frame string) {
	if frame == "" {
		return
	}
	remove := map[string]bool{frame: true}
	for changed := true; changed; {
		changed = false
		for f, p := range r.frameParents {
			if remove[p] && !remove[f] {
				remove[f] = true
				changed = true
			}
		}
	}
	for url, owner := range r.sketchAssetURLs {
		if remove[owner] {
			delete(r.sketchAssetURLs, url)
		}
	}
	for session, f := range r.frameIDs {
		if remove[f] && r.excludedSession(session) {
			if r.sketchSessions == nil {
				r.sketchSessions = map[string]bool{}
			}
			r.sketchSessions[session] = true
		}
	}
	for f := range remove {
		delete(r.frameParents, f)
		delete(r.sketchFrames, f)
	}
	for _, ctxs := range r.contexts {
		for id, f := range ctxs {
			if remove[f] {
				delete(ctxs, id)
			}
		}
	}
}

// frameLifecycle runs even when paused and before recording filters.
func (r *Recorder) frameLifecycle(e Event) {
	var p struct {
		FrameID string `json:"frameId"`
		Parent  string `json:"parentFrameId"`
		Reason  string `json:"reason"`
		Frame   struct {
			ID     string `json:"id"`
			Parent string `json:"parentId"`
		} `json:"frame"`
	}
	switch e.Method {
	case "Page.frameAttached", "Page.frameNavigated":
		if json.Unmarshal(e.Params, &p) != nil {
			return
		}
		f, parent := p.FrameID, p.Parent
		if e.Method == "Page.frameNavigated" {
			f, parent = p.Frame.ID, p.Frame.Parent
		}
		if f != "" {
			if e.Method == "Page.frameNavigated" && parent == "" && f == r.frameIDs[e.Session] {
				var nav struct {
					Frame struct {
						Loader string `json:"loaderId"`
					} `json:"frame"`
				}
				_ = json.Unmarshal(e.Params, &nav)
				if old := r.loader[e.Session]; old != "" && nav.Frame.Loader != "" && old != nav.Frame.Loader {
					for root, owner := range r.sketchFrames {
						if owner == e.Session {
							r.forgetFrame(root)
						}
					}
				}
			}
			if r.frameParents == nil {
				r.frameParents = map[string]string{}
			}
			if parent != "" || r.frameParents[f] == "" {
				r.frameParents[f] = parent
			}
		}
	case "Page.frameDetached":
		if json.Unmarshal(e.Params, &p) == nil && p.Reason != "swap" {
			r.forgetFrame(p.FrameID)
		}
	}
}

// ExcludeSketchAssetURL registers an opaque private blob before the child
// scene may render it. The URL is never persisted or sent to a viewer.
func (r *Recorder) ExcludeSketchAssetURL(ctx context.Context, frameID, url string) error {
	return r.acknowledged(ctx, func() error {
		if !r.excludedFrame(frameID) || !strings.HasPrefix(url, "blob:") {
			return errors.New("recorder: invalid private asset scope")
		}
		if r.sketchAssetURLs == nil {
			r.sketchAssetURLs = map[string]string{}
		}
		r.sketchAssetURLs[url] = frameID
		return nil
	})
}

var cssAssetURL = regexp.MustCompile(`(?i)url\(\s*(?:"([^"\r\n]*)"|'([^'\r\n]*)'|([^\s)'"\r\n]*))\s*\)`)

func (r *Recorder) excludedAssetURL(src string) bool {
	if _, ok := r.sketchAssetURLs[strings.TrimSpace(src)]; ok {
		return true
	}
	for _, m := range cssAssetURL.FindAllStringSubmatch(src, -1) {
		for _, url := range m[1:] {
			if _, ok := r.sketchAssetURLs[url]; ok {
				return true
			}
		}
	}
	return false
}

// Mark registered frame owners even when their host has not yet reached the
// mirror. The mirror's existing hidden-subtree logic also suppresses later DOM
// mutations without treating them as missing-node desynchronization.
func (r *Recorder) hideSketchNodes(n *cnode) {
	if n == nil {
		return
	}
	if r.excludedFrame(n.FrameID) {
		n.Attributes = append(n.Attributes, controlAttr, "")
	}
	for _, c := range n.Children {
		r.hideSketchNodes(c)
	}
	for _, c := range n.ShadowRoots {
		r.hideSketchNodes(c)
	}
	r.hideSketchNodes(n.ContentDocument)
	r.hideSketchNodes(n.TemplateContent)
}
func (r *Recorder) hideSketchEvent(e Event) Event {
	if e.Method != "DOM.setChildNodes" && e.Method != "DOM.childNodeInserted" {
		return e
	}
	var p map[string]json.RawMessage
	if json.Unmarshal(e.Params, &p) != nil {
		return e
	}
	if b := p["node"]; b != nil {
		var n cnode
		if json.Unmarshal(b, &n) == nil {
			r.hideSketchNodes(&n)
			p["node"], _ = json.Marshal(n)
		}
	}
	if b := p["nodes"]; b != nil {
		var ns []*cnode
		if json.Unmarshal(b, &ns) == nil {
			for _, n := range ns {
				r.hideSketchNodes(n)
			}
			p["nodes"], _ = json.Marshal(ns)
		}
	}
	e.Params, _ = json.Marshal(p)
	return e
}
