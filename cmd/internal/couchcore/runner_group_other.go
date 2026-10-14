//go:build !darwin

package couchcore

// Other platforms retain the conservative EPERM interpretation: permission
// denied does not establish absence. No additional process probe is needed.
func processGroupMembers(int) (bool, error) { return true, nil }
