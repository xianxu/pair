// Package terminalcapture records explicitly enabled terminal diagnostics.
// It reads no environment variables and collects no input by itself.
package terminalcapture

import "time"

// Record is the version-one JSONL wire schema. Data contains exact bytes and
// encoding/json represents it as base64. Accepted is deliberately not omitted:
// a host write accepting zero bytes is evidence, not a missing measurement.
// Recorder owns the stamp fields; callers supply only observation fields.
type Record struct {
	Version    int       `json:"version"`
	Seq        uint64    `json:"seq"`
	Time       time.Time `json:"time"`
	ElapsedNS  int64     `json:"elapsed_ns"`
	Kind       string    `json:"kind"`
	EndpointID string    `json:"endpoint_id,omitempty"`
	Scope      string    `json:"scope,omitempty"`
	Tag        string    `json:"tag,omitempty"`
	Actor      string    `json:"actor,omitempty"`
	Cols       int       `json:"cols,omitempty"`
	Rows       int       `json:"rows,omitempty"`
	Data       []byte    `json:"data,omitempty"`
	Requested  int       `json:"requested"`
	Accepted   int       `json:"accepted"`
	Error      string    `json:"error,omitempty"`
	DurationNS int64     `json:"duration_ns,omitempty"`
	Status     string    `json:"status,omitempty"`
	PID        int       `json:"pid,omitempty"`
	Build      string    `json:"build,omitempty"`
}

// retainedBytes bounds caller-controlled retained memory as well as record
// overhead; JSON's base64 expansion is separately bounded by the disk cap.
func (r Record) retainedBytes() int {
	return 384 + len(r.Data) + len(r.Kind) + len(r.EndpointID) + len(r.Scope) + len(r.Tag) + len(r.Actor) + len(r.Error) + len(r.Status) + len(r.Build)
}
