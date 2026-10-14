// Package orientation describes the outgoing session's evidence a fresh agent
// session, or the operator, can be pointed at (`couch --peek --previous`).
// It owns no filesystem, process, or terminal IO.
package orientation

import "github.com/xianxu/pair/cmd/internal/artifactpath"

// OrientationContext is what Couch could resolve about a thread's outgoing
// session; an unavailable source is named in Unavailable, never guessed.
type OrientationContext struct {
	Owner                                        *artifactpath.StorageOwner
	Tag, WorkingPath, SourceAgent, SourceSession string
	PairLog, ScrollbackRaw, ScrollbackEvents     string
	Renderer                                     string
	NativeTranscripts, Unavailable               []string
}
