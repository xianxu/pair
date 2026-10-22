package webcast

import (
	"github.com/xianxu/pair/cmd/internal/webcast/capability"
	"github.com/xianxu/pair/cmd/internal/webcast/drive"
)

// wireCapability publishes availability without exposing internal errors or
// granting authority. The server still rechecks every mutation's admission.
type wireCapability struct {
	Supported  bool   `json:"supported"`
	Permitted  bool   `json:"permitted"`
	Allowed    bool   `json:"allowed"`
	State      string `json:"state"`
	Generation uint64 `json:"generation"`
	Attempt    uint64 `json:"attempt"`
}

type wireCapabilities struct {
	Annotation wireCapability         `json:"annotation"`
	Drive      wireCapability         `json:"drive"`
	Drawing    drive.DrawingAuthority `json:"drawing"`
}

func capabilityWire(snapshot capability.Snapshot, supported, permitted bool) wireCapability {
	p := capability.Project(snapshot.CapabilityState)
	return wireCapability{Supported: supported, Permitted: permitted, Allowed: supported && permitted && p.Enabled,
		State: p.State, Generation: snapshot.Generation, Attempt: snapshot.Attempt}
}

func (s *Server) viewerCapabilities(link castLink) wireCapabilities {
	var a, d capability.Snapshot
	if s.board != nil {
		a = s.board.board.AnnotationCapability()
	}
	if s.drive != nil {
		d = s.drive.ctrl.DriveCapability()
	}
	out := wireCapabilities{Annotation: capabilityWire(a, s.board != nil, link.annotates()),
		Drive: capabilityWire(d, s.drive != nil, link.drives())}
	if s.drive != nil {
		out.Drawing = s.drive.ctrl.DrawingAuthority()
		out.Drawing.Ready = out.Drawing.Ready && out.Annotation.Allowed && link.drives()
	}
	if (Capabilities{Annotation: a, Drive: d}).Broken() {
		out.Annotation.Allowed, out.Drive.Allowed = false, false
		out.Annotation.State, out.Drive.State = "broken", "broken"
		out.Drawing.Ready = false
	}
	return out
}
