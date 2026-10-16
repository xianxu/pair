package historyarchive

// PublicationObservation contains evidence obtained by the IO coordinator.
// Matches means both the encoded label and ciphertext match the expected manifest.
// Durable is set only after successful readback AND file/directory sync barriers.
// pair:m5-concept pure
type PublicationObservation struct{ BundlePresent, LabelPresent, Matches, Durable bool }

// pair:m5-concept pure
type PublicationDecision string

const (
	PublicationAbsent           PublicationDecision = "absent"
	PublicationVerifiedExisting PublicationDecision = "verified-existing"
	PublicationConflict         PublicationDecision = "conflict"
)

func DecidePublication(o PublicationObservation) PublicationDecision {
	if !o.BundlePresent && !o.LabelPresent {
		return PublicationAbsent
	}
	if o.BundlePresent && o.LabelPresent && o.Matches && o.Durable {
		return PublicationVerifiedExisting
	}
	return PublicationConflict
}
