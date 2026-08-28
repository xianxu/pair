package sessioninventory

// Inventory is one complete, stably ordered snapshot of native forests and
// their Pair bindings, ambiguities, and diagnostics.
// pair:155-concept pure new M2
type Inventory struct {
	Forests     []Forest     `json:"forests"`
	Bindings    []Binding    `json:"bindings"`
	Ambiguities []Ambiguity  `json:"ambiguities"`
	Diagnostics []Diagnostic `json:"diagnostics"`
}
