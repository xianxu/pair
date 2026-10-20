package couchcore

import (
	"github.com/xianxu/pair/cmd/internal/lifetimelease"
)

// threadLifecycleAddress is the lifecycle address a thread's tree runs under:
// its slot, by the physical starting path (pair#695). A team hub's thread is
// addressed by its team folder in pair#694.
func threadLifecycleAddress(thread ThreadRecord) (LifecycleAddress, error) {
	return NewSlotAddress(SlotIdentity{}, thread.StartingPath)
}

// lifetimeLease is the lease file of one thread's tree under its lifecycle
// address, in this Couch's namespace. It is per thread, not per address: a
// store from before one-thread-per-repository (pair#363) can hold sibling
// threads at one path, and teardown kills the lease's holders, so a shared
// lease would let retiring one thread kill its live sibling's pair.
func (c *Couch) lifetimeLease(address LifecycleAddress, thread ThreadAddress) (string, error) {
	return lifetimelease.Path(c.Namespace.Dir(), address.Key()+" "+thread.RepoScope+"/"+string(thread.Tag))
}

// lifetimeLeaseEnv prepares the thread's lifetime lease and names it for the
// tree Couch is about to launch; each long-lived pair role holds it.
func (c *Couch) lifetimeLeaseEnv(thread ThreadRecord) (string, error) {
	address, err := threadLifecycleAddress(thread)
	if err != nil {
		return "", err
	}
	path, err := c.lifetimeLease(address, thread.Address)
	if err != nil {
		return "", err
	}
	if err := lifetimelease.Prepare(path); err != nil {
		return "", err
	}
	return lifetimelease.Env + "=" + path, nil
}
