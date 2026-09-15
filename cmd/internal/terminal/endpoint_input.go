package terminal

// WriteInput admits trusted already-encoded input to the same FIFO as events
// and query replies. Like a buffered writer, success reports admission; Flush
// reports delivery. Compositor event routing uses Send instead.
func (e *Endpoint) WriteInput(p []byte) (int, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if err := e.check(); err != nil {
		return 0, err
	}
	if err := e.input.Enqueue(p); err != nil {
		return 0, err
	}
	return len(p), nil
}

// InputEnded reports loss of input capability independently of output disposal.
func (e *Endpoint) InputEnded() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.inputEnded || e.closed
}
