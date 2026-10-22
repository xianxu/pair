package couchcore

import (
	"context"
	"fmt"
	"sync"

	"github.com/xianxu/pair/cmd/internal/launcher"
)

// AcceptedConfiguration is one launch's explicit configuration acceptance.
// Its fields are private so ordinary result construction cannot invent intent.
// Copies of the result share one completion; there is no durable retry intent.
type AcceptedConfiguration struct {
	repoIdentity string
	profile      LaunchProfile
	once         sync.Once
	mu           sync.Mutex
	warning      string
}

func acceptedConfigurationForStart(thread ThreadRecord, in trackedThreadLaunch) *AcceptedConfiguration {
	if in.Warm || in.Recovery {
		return nil
	}
	incarnation, err := exactStartIncarnation(&thread, in.Nonce)
	if err != nil || !incarnation.Start.RememberConfiguration || incarnation.Start.RecoveryIntent != nil || incarnation.Start.LaunchProfile == nil {
		return nil
	}
	return &AcceptedConfiguration{repoIdentity: incarnation.RepoIdentity, profile: cloneLaunchProfile(*incarnation.Start.LaunchProfile)}
}

func (a *AcceptedConfiguration) publicationWarning() string {
	if a == nil {
		return ""
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.warning
}

// PublishAdoptedConfiguration is called by the console attachment owner only
// after its last fallible adoption step. A storage failure preserves the live
// child and is reported through every copy of the original operation result.
func (c *Couch) PublishAdoptedConfiguration(started StartResult) {
	started, _ = started.Started()
	accepted := started.Configuration
	if accepted == nil {
		return
	}
	accepted.once.Do(func() {
		publish := func() error {
			if err := launcher.ValidateFreshAgentArgs(accepted.profile.Agent, accepted.profile.Argv); err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(context.Background(), repoIdentityTimeout)
			defer cancel()
			_, sources, err := c.previewRepositoryPreferences(ctx)
			if err != nil {
				return err
			}
			return c.Threads.rememberRepoConfigurationContext(ctx, accepted.repoIdentity, accepted.profile, sources)
		}
		if err := publish(); err != nil {
			accepted.mu.Lock()
			accepted.warning = fmt.Sprintf("started, but startup parameters were not remembered: %v", err)
			accepted.mu.Unlock()
		}
	})
}
