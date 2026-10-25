package couchcore

import "github.com/xianxu/pair/cmd/internal/launcher"

// SelectedListingEvidence accompanies a decision; it never grants authority.
type SelectedListingEvidence struct {
	Address       ThreadAddress
	Name          string
	Claims        int
	SelectedState string
	Projection    SessionState
	Listing       launcher.ListingEvidence
}
