package couchcore

import (
	"errors"

	"github.com/xianxu/pair/cmd/internal/launcher"
)

// ThreadArtifactClaim is retained when ThreadStore accepts the same address and
// released only when its subsequent no-replace record claim fails.
type ThreadArtifactClaim interface {
	Release() error
}

// ThreadArtifactClaimer atomically serializes a prospective composite address
// with every current Pair artifact/session producer.
type ThreadArtifactClaimer interface {
	Claim(ThreadAddress) (ThreadArtifactClaim, error)
	Release(ThreadAddress) error
	Registration(ThreadAddress) (RegistrationEvidence, error)
}

type noopThreadArtifactClaim struct{}

func (noopThreadArtifactClaim) Release() error { return nil }

type NoThreadArtifactCollisions struct{}

func (NoThreadArtifactCollisions) Claim(ThreadAddress) (ThreadArtifactClaim, error) {
	return noopThreadArtifactClaim{}, nil
}
func (NoThreadArtifactCollisions) Release(ThreadAddress) error { return nil }
func (NoThreadArtifactCollisions) Registration(ThreadAddress) (RegistrationEvidence, error) {
	return RegistrationEstablished, nil
}

type ScopedThreadArtifactCollisionChecker struct{ GlobalDataDir string }

func NewScopedThreadArtifactCollisionChecker(globalDataDir string) ScopedThreadArtifactCollisionChecker {
	return ScopedThreadArtifactCollisionChecker{GlobalDataDir: globalDataDir}
}

func (c ScopedThreadArtifactCollisionChecker) Claim(address ThreadAddress) (ThreadArtifactClaim, error) {
	if err := validateThreadAddress(address); err != nil {
		return nil, err
	}
	if c.GlobalDataDir == "" {
		return nil, errors.New("artifact claimer has no Pair data directory")
	}
	claim, err := launcher.ClaimNewThreadAddress(c.GlobalDataDir,
		launcher.RepoScope{Key: address.RepoScope}, string(address.Tag))
	if err != nil {
		return nil, err
	}
	return claim, nil
}

func (c ScopedThreadArtifactCollisionChecker) Release(address ThreadAddress) error {
	paths := launcher.NewScopedPaths(c.GlobalDataDir,
		launcher.RepoScope{Key: address.RepoScope}, string(address.Tag))
	return launcher.ReleaseThreadAddressClaim(paths.ThreadClaim())
}

func (c ScopedThreadArtifactCollisionChecker) Registration(address ThreadAddress) (RegistrationEvidence, error) {
	if err := validateThreadAddress(address); err != nil {
		return RegistrationUnknown, err
	}
	established, err := launcher.ThreadAddressEstablished(c.GlobalDataDir,
		launcher.RepoScope{Key: address.RepoScope}, string(address.Tag))
	if err != nil {
		return RegistrationUnknown, err
	}
	if established {
		return RegistrationEstablished, nil
	}
	return RegistrationAbsent, nil
}
