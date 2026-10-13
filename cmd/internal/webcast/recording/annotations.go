package recording

// OperatorID attributes cast-wide controls to their authoritative operator.
// HTTP identities never confer the ability to produce these records.
const OperatorID = "operator"

// AnnotationClear removes current annotations at this point in the timeline.
// It is valid during a pause; previous replay moments remain unchanged.
type AnnotationClear struct {
	Header
	Who Author `json:"who"`
}

func (*AnnotationClear) Kind() string { return "annotation-clear" }

func (r *AnnotationClear) check() error {
	return need(r.Who.ID == OperatorID, "operator attribution")
}
