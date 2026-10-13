package webcast

import "github.com/xianxu/pair/cmd/internal/broadcast"

// castLink is which of a cast's links a request came under (pair#506), like
// the terminal cast's: each grants what the one before it does, and more.
// The terminal's pointer link is a phase switched on and off; these are
// fixed for the cast's life, so the web cast keeps its own model.
type castLink uint8

const (
	linkNone    castLink = iota
	linkView             // watch
	linkPointer          // watch, point, notes
	linkDrive            // watch, point, notes, ask to drive (#467)
)

// annotates: the link may draw strokes, leave notes and resolve them.
func (l castLink) annotates() bool { return l >= linkPointer }

// drives: the link offers a seat that can ask to drive the cast tab.
func (l castLink) drives() bool { return l == linkDrive }

// WithPointer serves the pointer link under token (pair#506). Without it the
// cast has only its view-only link and, when driven, its drive link.
func (s *Server) WithPointer(token string) *Server {
	s.pointerToken = token
	return s
}

// linkOf classifies a request's first path segment against the cast's
// tokens, each compared in constant time; a link that wasn't made has an
// empty token, which matches nothing.
func (s *Server) linkOf(segment string) castLink {
	switch {
	case s.drive != nil && broadcast.OwnsToken(segment, s.drive.token):
		return linkDrive
	case broadcast.OwnsToken(segment, s.pointerToken):
		return linkPointer
	case broadcast.OwnsToken(segment, s.token):
		return linkView
	}
	return linkNone
}
