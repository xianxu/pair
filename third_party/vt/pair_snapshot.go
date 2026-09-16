package vt

// Cursor returns the active screen's authoritative cursor value. Snapshot
// consumers must read state directly: callbacks are effects, not a second model
// of cursor state, and restore/reset/screen switches can replace the whole value.
func (e *Emulator) Cursor() Cursor { return e.scr.Cursor() }
