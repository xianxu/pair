// Package webcast is Couch's browser-control component for couch broadcast-web
// (#454): a dedicated Chrome, driven over the DevTools protocol, whose active
// tab the operator casts to viewers as a link.
//
// Browser is the seam: state, go-live, end. Couch's UI and the stream (later,
// the recorder) talk only to it; no CDP type crosses it. Chrome implements it
// over the debugging pipe; FakeBrowser is the stateful double.
//
// Pure: State, Tracker, translate, viewerState. Stateful shell: Chrome,
// FakeBrowser, Server, Cast.
package webcast
