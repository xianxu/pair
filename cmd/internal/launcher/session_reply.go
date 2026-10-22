package launcher

import "github.com/xianxu/pair/cmd/internal/zellijquery"

func validateSessionListing(raw []byte, short bool) error {
	return zellijquery.ValidateSessionListing(raw, short)
}
func sessionListingStates(raw []byte) (map[string]bool, error) {
	return zellijquery.SessionListingStates(raw)
}
func sessionListingState(raw []byte, name string) (bool, bool, error) {
	return zellijquery.SessionListingState(raw, name)
}
func parseColoredSessionListing(raw []byte) (map[string]bool, error) {
	return zellijquery.ParseColoredSessionListing(raw)
}
