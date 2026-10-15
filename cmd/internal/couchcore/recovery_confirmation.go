package couchcore

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"time"
)

const recoveryConfirmationTTL = 2 * time.Minute
const maxRecoveryConfirmations = 256

type recoveryConfirmerKey struct{}

// WithRecoveryConfirmer binds local operator input or authenticated hub input
// to one supervisor-local identity. It is never decoded from request payloads.
func WithRecoveryConfirmer(ctx context.Context, identity string) context.Context {
	return context.WithValue(ctx, recoveryConfirmerKey{}, identity)
}
func recoveryConfirmer(ctx context.Context) string {
	if identity, ok := ctx.Value(recoveryConfirmerKey{}).(string); ok && identity != "" {
		return identity
	}
	return "local-operator"
}

type RecoveryConfirmationOffer struct {
	Token     string                `json:"token"`
	Target    string                `json:"target"`
	ExpiresAt time.Time             `json:"expires_at"`
	Stops     []RecoveryStopProcess `json:"stops"`
	Command   string                `json:"command"`
	NativeID  string                `json:"native_id,omitempty"`
}

type RecoveryConfirmationRequired struct{ Offer RecoveryConfirmationOffer }

func (e *RecoveryConfirmationRequired) Error() string {
	return fmt.Sprintf("Confirm stopping %v; %s (expires %s)", e.Offer.Stops, e.Offer.Command, e.Offer.ExpiresAt.Local().Format("15:04:05 MST"))
}

type recoveryConfirmationWitness struct {
	Address   ThreadAddress
	Revision  uint64
	Selection RecoverySelection
	Digest    string
	NativeID  string
}
type recoveryConfirmationState struct {
	Witness   recoveryConfirmationWitness
	Actor     string
	ExpiresAt time.Time
	Consumed  bool
}

// AdvanceRecoveryConfirmation is the pure acceptance gate. The memory owner
// publishes its result under one mutex; a lookup validates without publishing.
func AdvanceRecoveryConfirmation(state recoveryConfirmationState, actor string, witness recoveryConfirmationWitness, now time.Time) (recoveryConfirmationState, error) {
	if state.Consumed || !now.Before(state.ExpiresAt) {
		return state, errors.New("recovery confirmation expired or already used; refresh the recovery choice")
	}
	if state.Actor != actor || state.Witness != witness {
		return state, errors.New("recovery choice or owner changed; refresh the recovery choice and confirm the named stop set")
	}
	state.Consumed = true
	return state, nil
}

func (c *Couch) issueRecoveryConfirmation(ctx context.Context, witness recoveryConfirmationWitness, stops []RecoveryStopProcess) (RecoveryConfirmationOffer, error) {
	if err := ctx.Err(); err != nil {
		return RecoveryConfirmationOffer{}, err
	}
	if witness.Selection.Reference == "" || witness.Revision == 0 || witness.Digest == "" {
		return RecoveryConfirmationOffer{}, errors.New("recovery confirmation has incomplete source authority")
	}
	entropy := c.Entropy
	if entropy == nil {
		entropy = rand.Reader
	}
	var b [24]byte
	if _, err := io.ReadFull(entropy, b[:]); err != nil {
		return RecoveryConfirmationOffer{}, err
	}
	token := hex.EncodeToString(b[:])
	now := c.Clock.Now()
	state := recoveryConfirmationState{Witness: witness, Actor: recoveryConfirmer(ctx), ExpiresAt: now.Add(recoveryConfirmationTTL)}
	c.recoveryMu.Lock()
	defer c.recoveryMu.Unlock()
	if c.recoveryConfirmations == nil {
		c.recoveryConfirmations = make(map[string]recoveryConfirmationState)
	}
	for key, old := range c.recoveryConfirmations {
		if old.Consumed || !now.Before(old.ExpiresAt) {
			delete(c.recoveryConfirmations, key)
		}
	}
	if _, exists := c.recoveryConfirmations[token]; exists {
		return RecoveryConfirmationOffer{}, errors.New("recovery confirmation entropy collision")
	}
	if len(c.recoveryConfirmations) >= maxRecoveryConfirmations {
		oldestKey := ""
		var oldest time.Time
		for key, old := range c.recoveryConfirmations {
			if oldestKey == "" || old.ExpiresAt.Before(oldest) {
				oldestKey, oldest = key, old.ExpiresAt
			}
		}
		delete(c.recoveryConfirmations, oldestKey)
	}
	c.recoveryConfirmations[token] = state
	return RecoveryConfirmationOffer{Token: token, Target: witness.Selection.Reference, ExpiresAt: state.ExpiresAt, Stops: append([]RecoveryStopProcess(nil), stops...), Command: "couch --recovery-confirmation " + token, NativeID: witness.NativeID}, nil
}

func (c *Couch) recoveryConfirmation(ctx context.Context, token string) (recoveryConfirmationState, error) {
	if err := ctx.Err(); err != nil {
		return recoveryConfirmationState{}, err
	}
	c.recoveryMu.Lock()
	defer c.recoveryMu.Unlock()
	state, ok := c.recoveryConfirmations[token]
	if !ok {
		return state, errors.New("recovery confirmation unavailable; refresh the recovery choice")
	}
	if _, err := AdvanceRecoveryConfirmation(state, recoveryConfirmer(ctx), state.Witness, c.Clock.Now()); err != nil {
		return state, err
	}
	return state, nil
}

func (c *Couch) RecoveryConfirmationReference(ctx context.Context, token string) (string, error) {
	state, err := c.recoveryConfirmation(ctx, token)
	return state.Witness.Selection.Reference, err
}

func (c *Couch) consumeRecoveryConfirmation(ctx context.Context, token string, witness recoveryConfirmationWitness) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	c.recoveryMu.Lock()
	defer c.recoveryMu.Unlock()
	state, ok := c.recoveryConfirmations[token]
	if !ok {
		return errors.New("recovery confirmation unavailable; refresh the recovery choice")
	}
	next, err := AdvanceRecoveryConfirmation(state, recoveryConfirmer(ctx), witness, c.Clock.Now())
	if err != nil {
		return err
	}
	c.recoveryConfirmations[token] = next
	return nil
}
