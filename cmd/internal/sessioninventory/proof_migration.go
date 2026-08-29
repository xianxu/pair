package sessioninventory

import (
	"errors"
	"sync"

	"github.com/xianxu/pair/cmd/internal/sessionledger"
)

// ProofMigrationKey is the complete authority boundary for one background
// proof upgrade. A worker may validate this root only; it may not widen to an
// agent inventory scan.
type ProofMigrationKey struct {
	ScopeKey string
	Tag      string
	Agent    Agent
	NativeID string
}

type ProofMigrationResult struct {
	Proof *sessionledger.AuthorizationProof
	Err   error
}

type ProofMigrationWork func() (*sessionledger.AuthorizationProof, error)

type proofMigrationCall struct {
	waiters []chan ProofMigrationResult
}

// ProofMigrator coalesces concurrent background upgrades for one named owner
// and root. Ordinary lookups remain fail-closed while the work is in flight.
// pair:156-concept integration new final
type ProofMigrator struct {
	mu       sync.Mutex
	inFlight map[ProofMigrationKey]*proofMigrationCall
}

func (m *ProofMigrator) Request(key ProofMigrationKey, work ProofMigrationWork) <-chan ProofMigrationResult {
	result := make(chan ProofMigrationResult, 1)
	if key.ScopeKey == "" || key.Tag == "" || !validAgent(key.Agent) || key.NativeID == "" || work == nil {
		result <- ProofMigrationResult{Err: errors.New("invalid proof migration request")}
		close(result)
		return result
	}
	m.mu.Lock()
	if m.inFlight == nil {
		m.inFlight = make(map[ProofMigrationKey]*proofMigrationCall)
	}
	if call := m.inFlight[key]; call != nil {
		call.waiters = append(call.waiters, result)
		m.mu.Unlock()
		return result
	}
	m.inFlight[key] = &proofMigrationCall{waiters: []chan ProofMigrationResult{result}}
	m.mu.Unlock()

	go m.run(key, work)
	return result
}

func (m *ProofMigrator) run(key ProofMigrationKey, work ProofMigrationWork) {
	proof, err := work()
	if err == nil {
		if proof == nil || proof.RootNativeID != key.NativeID {
			err = errors.New("proof migration widened or omitted the named root")
		} else if validationErr := sessionledger.ValidateAuthorizationProof(*proof, key.NativeID); validationErr != nil {
			err = validationErr
		}
	}
	if err != nil {
		proof = nil
	}
	m.mu.Lock()
	call := m.inFlight[key]
	delete(m.inFlight, key)
	m.mu.Unlock()
	for _, waiter := range call.waiters {
		waiter <- ProofMigrationResult{Proof: cloneMigrationProof(proof), Err: err}
		close(waiter)
	}
}

func cloneMigrationProof(proof *sessionledger.AuthorizationProof) *sessionledger.AuthorizationProof {
	if proof == nil {
		return nil
	}
	cloned := *proof
	cloned.ScannerState = append([]byte(nil), proof.ScannerState...)
	cloned.Artifacts = append([]sessionledger.ArtifactProof(nil), proof.Artifacts...)
	return &cloned
}
