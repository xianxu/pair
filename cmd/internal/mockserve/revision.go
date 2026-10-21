package mockserve

import "strconv"

// Observation is the current selected version and its lifetime-qualified revision.
type Observation struct {
	Version  string `json:"version"`
	Revision string `json:"revision"`
	Error    string `json:"error"`
}

// RevisionState is pure transition state. Its caller owns synchronization.
type RevisionState struct {
	initialized bool
	version     string
	digest      string
	failure     string
	failed      bool
	sequence    uint64
}

// Observe advances once for each distinct healthy or failed observation.
func (s *RevisionState) Observe(version, digest string, err error) bool {
	failure := ""
	if err != nil {
		failure = err.Error()
		digest = ""
	}
	if s.initialized && s.version == version && s.digest == digest && s.failure == failure && s.failed == (err != nil) {
		return false
	}
	s.initialized = true
	s.version = version
	s.digest = digest
	s.failure = failure
	s.failed = err != nil
	s.sequence++
	return true
}

// Snapshot returns a copied wire value qualified by this server's lifetime ID.
func (s *RevisionState) Snapshot(lifetime string) Observation {
	failure := s.failure
	if s.failed && failure == "" {
		failure = "Mock observation failed."
	}
	return Observation{Version: s.version, Revision: lifetime + ":" + strconv.FormatUint(s.sequence, 10), Error: failure}
}
