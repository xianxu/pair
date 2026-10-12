package broadcast

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// EventStream writes one server-sent event stream: shared by the screen
// broadcast and the web cast (#454).
type EventStream struct {
	w    http.ResponseWriter
	rc   *http.ResponseController
	ping time.Duration
}

// OpenEventStream starts an event stream on w and sends its first comment.
// ping is the keepalive interval the caller pings at; a write that can't
// finish within two of them fails, so a viewer that stops reading can't hold
// its goroutine (and viewer slot) forever. Nil if the first write failed.
func OpenEventStream(w http.ResponseWriter, ping time.Duration) *EventStream {
	w.Header().Set("Content-Type", "text/event-stream")
	w.WriteHeader(http.StatusOK)
	e := &EventStream{w: w, rc: http.NewResponseController(w), ping: ping}
	if !e.Comment("connected") {
		return nil
	}
	return e
}

func (e *EventStream) write(chunk string) bool {
	_ = e.rc.SetWriteDeadline(time.Now().Add(2 * e.ping))
	if _, err := fmt.Fprint(e.w, chunk); err != nil {
		return false
	}
	return e.rc.Flush() == nil
}

// Event sends v as JSON under name; false once the viewer is gone.
func (e *EventStream) Event(name string, v any) bool {
	data, _ := json.Marshal(v)
	return e.write("event: " + name + "\ndata: " + string(data) + "\n\n")
}

// Comment sends a comment line (": text"), as a ping does.
func (e *EventStream) Comment(text string) bool { return e.write(": " + text + "\n\n") }
