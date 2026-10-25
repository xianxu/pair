package couchtty

import (
	"github.com/xianxu/pair/cmd/internal/broadcast"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/webcast"
	"github.com/xianxu/pair/cmd/internal/webcast/capability"
)

// castControlIntent keeps copy-only clicks independent from capability recovery.
func castControlIntent(right bool, p capability.Presentation) string {
	if right {
		return "copy"
	}
	if p.Action == "enable" {
		return "enable-copy"
	}
	return p.Action
}
func capabilitySGR(on bool) string {
	if on {
		return broadcast.PointerSGR
	}
	return dimSGR
}
func capabilityNotice(s webcast.Capabilities) string {
	if s.Broken() {
		reason := s.Annotation.Problem
		if reason == "" {
			reason = s.Drive.Problem
		}
		return "Cast broken; Ctrl+click Chrome to end cast: " + reason
	}
	for _, v := range []struct {
		k          webcast.CapabilityKind
		icon, what string
		s          capability.Snapshot
	}{{webcast.CapabilityAnnotation, "👆", "clear marks", s.Annotation}, {webcast.CapabilityDrive, "👽", "release control", s.Drive}} {
		switch s.Project(v.k).Action {
		case "retry":
			return "Click " + v.icon + " to retry; couldn't " + v.what + ": " + v.s.Problem
		case "wait":
			return v.icon + " turning off; wait…"
		}
	}
	return ""
}

func (c *Console) onWebCapabilityClick(kind webcast.CapabilityKind, right bool) <-chan struct{} {
	c.mu.Lock()
	cast, running := c.wcast.cast, c.wcast.phase == webcastRunning
	c.mu.Unlock()
	if !running || cast == nil {
		return nil
	}
	link := webcastPointerLink
	if kind == webcast.CapabilityDrive {
		link = webcastDriveLink
	}
	caps := cast.Capabilities()
	if kind == webcast.CapabilityAnnotation && !caps.AnnotationSupported || kind == webcast.CapabilityDrive && !caps.DriveSupported {
		c.copyWebcastLinkOf(link)
		return nil
	}
	intent := castControlIntent(right, caps.Project(kind))
	if intent == "copy" {
		c.copyWebcastLinkOf(link)
		return nil
	}
	var action capability.EventKind
	switch intent {
	case "enable-copy":
		action = capability.Enable
	case "disable":
		action = capability.Disable
	case "retry":
		action = capability.Retry
	default:
		c.setNotice(capabilityNotice(caps))
		return nil
	}
	expected := caps.Annotation.CapabilityState
	if kind == webcast.CapabilityDrive {
		expected = caps.Drive.CapabilityState
	}
	expected, transitionErr := capability.NextCapability(expected, capability.Event{Kind: action})
	if transitionErr != nil {
		c.setNotice(transitionErr.Error())
		return nil
	}
	result, err := cast.EnqueueCapability(kind, action)
	if err != nil {
		c.setNotice(err.Error())
		return nil
	}
	done := make(chan struct{})
	c.GoTracked(func() {
		defer close(done)
		err := <-result
		_ = c.runTerminalCommand(c.lifetime, func() error {
			c.mu.Lock()
			current := c.wcast.cast == cast && c.wcast.phase == webcastRunning
			c.mu.Unlock()
			if !current {
				return nil
			}
			latest := cast.Capabilities()
			state := latest.Annotation.CapabilityState
			if kind == webcast.CapabilityDrive {
				state = latest.Drive.CapabilityState
			}
			if state.Generation != expected.Generation || state.Attempt != expected.Attempt {
				return nil
			}
			if err != nil {
				notice := capabilityNotice(cast.Capabilities())
				if notice == "" {
					notice = err.Error()
				}
				if expectedControlFailure(err) {
					c.setNotice(notice)
				} else {
					c.Warn(Notice{Class: couchcore.NoticeOperatorCapabilityFailed, Origin: couchcore.NoticeOperatorOrigin, Kind: "warning", Body: notice})
				}
				return nil
			}
			if intent == "enable-copy" {
				if !latest.Project(kind).Enabled {
					return nil
				}
				c.copyWebcastLinkOf(link)
			} else {
				c.setNotice(link.name + " capability off.")
			}
			c.repaint()
			return nil
		})
	})
	return done
}
