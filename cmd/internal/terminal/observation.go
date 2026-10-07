package terminal

// Observation describes ordered parser input or an endpoint lifecycle change.
// Data is borrowed for the duration of the callback and must not be modified.
type Observation struct {
	Kind       string
	EndpointID string
	Geometry   Geometry
	Data       []byte
}

// Observer runs under the endpoint mutex. It must never re-enter the endpoint
// or perform blocking work. An asynchronous consumer must copy Data before
// returning and enqueue it without waiting for file IO.
type Observer func(Observation)

// observe is called while holding mu, including initial publication.
func (e *Endpoint) observe(kind string, data []byte) {
	if e.observer != nil {
		e.observer(Observation{Kind: kind, EndpointID: e.id, Geometry: e.geometry, Data: data})
	}
}
