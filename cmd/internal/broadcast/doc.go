// Package broadcast streams the composed Couch screen, view-only, to remote
// viewers (#395). The Presenter's tap supplies frames exactly as painted to the
// operator; this package withholds any frame on which the operator's screen
// did not show the LIVE indicator, replaces private frames with a placeholder,
// and renders one shared stream with terminal.Render.
//
// Pure: Stream, ViewerFrame, IndicatorShown. Stateful shell: Hub, Server,
// Session, tunnels.
package broadcast
