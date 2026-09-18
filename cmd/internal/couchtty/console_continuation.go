package couchtty

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/xianxu/pair/cmd/internal/checkpoint"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/orientation"
)

type ContinuationProvider func(context.Context, []couchcore.ThreadAddress) ([]couchcore.ContinuationStatus, error)

type continuationScanResult struct {
	addresses []couchcore.ThreadAddress
	statuses  []couchcore.ContinuationStatus
	err       error
}

// An accepted request outlives its source pane. queued lasts through delivery
// of the operation result, not merely through execution in operationQueue.
type continuationWatch struct {
	status  couchcore.ContinuationStatus
	queued  bool
	handled bool
}

func (c *Console) SetContinuationProvider(provider ContinuationProvider) {
	c.mu.Lock()
	c.continuationProvider = provider
	c.mu.Unlock()
}

func (c *Console) continuationAddresses() []couchcore.ThreadAddress {
	c.mu.Lock()
	defer c.mu.Unlock()
	seen := make(map[couchcore.ThreadAddress]bool)
	for _, p := range c.panes {
		seen[p.thread] = true
	}
	for address := range c.continuations {
		seen[address] = true
	}
	addresses := make([]couchcore.ThreadAddress, 0, len(seen))
	for address := range seen {
		addresses = append(addresses, address)
	}
	sort.Slice(addresses, func(i, j int) bool {
		if addresses[i].RepoScope != addresses[j].RepoScope {
			return addresses[i].RepoScope < addresses[j].RepoScope
		}
		return addresses[i].Tag < addresses[j].Tag
	})
	return addresses
}

func (c *Console) watchContinuations() {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		c.mu.Lock()
		provider := c.continuationProvider
		c.mu.Unlock()
		if provider != nil {
			addresses := c.continuationAddresses()
			if len(addresses) > 0 {
				statuses, err := provider(c.lifetime, addresses)
				select {
				case c.continuationResults <- continuationScanResult{addresses: addresses, statuses: statuses, err: err}:
				case <-c.stop:
					return
				}
			}
		}
		select {
		case <-ticker.C:
		case <-c.stop:
			return
		}
	}
}

// An orientation prompt has two producers -- a continuation's delivery and
// switch-agent's orientation watch -- sharing menu.Orientation by thread
// address. Each entry records its producer, and a prune removes an entry only
// when THAT producer's fact vanishes (#280, ARCH-ORDER). Without provenance the
// continuation scan, which visits every hosted thread, deleted switch-agent's
// still-useful Copy orientation prompt on every tick.
func continuationProducer(requestID string) string { return "continuation:" + requestID }
func switchAgentProducer(attempt string) string    { return "switch-agent:" + attempt }

// setOrientationLocked is the only writer of menu.Orientation. Callers hold c.mu.
func (c *Console) setOrientationLocked(address couchcore.ThreadAddress, request orientation.Request, producer string) {
	if c.menu.Orientation == nil {
		c.menu.Orientation = make(map[couchcore.ThreadAddress]orientation.Request)
	}
	if c.orientationFrom == nil {
		c.orientationFrom = make(map[couchcore.ThreadAddress]string)
	}
	c.menu.Orientation[address] = request
	c.orientationFrom[address] = producer
}

// dropOrientationLocked removes the address's prompt only if producer wrote it.
func (c *Console) dropOrientationLocked(address couchcore.ThreadAddress, producer string) {
	if producer == "" || c.orientationFrom[address] != producer {
		return
	}
	delete(c.menu.Orientation, address)
	delete(c.orientationFrom, address)
}

// reconcileContinuationOrientationLocked is the ONE place a continuation's
// prompt is pruned: it survives only while the address's current watch tracks
// THAT request and the request is not complete. It runs after every change to
// c.continuations, so dismissal, a vanished request, completion and replacement
// are one rule instead of four events -- this family's three findings were
// three events the enumerated prunes missed (#280). Callers hold c.mu.
func (c *Console) reconcileContinuationOrientationLocked() {
	for address, producer := range c.orientationFrom {
		if !strings.HasPrefix(producer, continuationProducer("")) {
			continue
		}
		watch, ok := c.continuations[address]
		if !ok || continuationProducer(watch.status.RequestID) != producer || watch.status.Phase == checkpoint.Complete {
			delete(c.menu.Orientation, address)
			delete(c.orientationFrom, address)
		}
	}
}

// supersedeOrientationLocked removes the address's prompt whoever wrote it: a
// new agent launch makes every earlier prompt for the thread obsolete.
func (c *Console) supersedeOrientationLocked(address couchcore.ThreadAddress) {
	delete(c.menu.Orientation, address)
	delete(c.orientationFrom, address)
}

func continuationOperation(status couchcore.ContinuationStatus, handled bool) string {
	switch status.Phase {
	case checkpoint.Pending:
		return "continue-thread"
	case checkpoint.Running:
		if handled {
			return "continuation-status"
		}
		return "continue-thread"
	default:
		return ""
	}
}

// Run owns acceptance, focus and request ordering. The scanner only reads.
func (c *Console) acceptContinuationRequests(result continuationScanResult) {
	if result.err != nil {
		c.setNotice("Continuation status unavailable: " + result.err.Error())
	}
	c.mu.Lock()
	seen := make(map[couchcore.ThreadAddress]bool)
	for _, status := range result.statuses {
		seen[status.Address] = true
		watch := c.continuations[status.Address]
		if watch.status.RequestID != status.RequestID {
			watch = continuationWatch{status: status}
		} else {
			watch.status = status
		}
		c.continuations[status.Address] = watch
		if status.Phase == checkpoint.Complete {
			delete(c.continuations, status.Address)
			continue
		}
		operation := continuationOperation(status, watch.handled)
		if operation == "" || watch.queued || c.ops == nil || c.menu.InFlight.Attempt != 0 {
			continue
		}
		fn := c.ops
		origin := MenuOperationOrigin{Operation: operation, Address: status.Address, ContinuationID: status.RequestID, PreserveFocus: true}
		for id, p := range c.panes {
			if p.thread == status.Address && c.focus == FocusActor(id) {
				origin.PreserveFocus = false
			}
		}
		args := map[string]string{"repo-scope": status.Address.RepoScope, "tag": string(status.Address.Tag), "request-id": status.RequestID}
		if status.Attempt != "" {
			args["attempt"] = status.Attempt
		}
		if operation != "continuation-status" {
			delete(args, "attempt")
		}
		accepted, err := c.operationQueue.Enqueue(operationRequest{
			key:  "continuation\x00" + status.Address.RepoScope + "\x00" + string(status.Address.Tag) + "\x00" + status.RequestID,
			name: operation, origin: origin,
			run: func() (any, error) {
				return fn(couchcore.OperationCall{Name: operation, Args: args, Implicit: true, Context: c.lifetime})
			},
		})
		if err != nil || !accepted {
			continue
		}
		watch.queued, watch.handled = true, true
		c.continuations[status.Address] = watch
		if operation == "continue-thread" {
			for id, p := range c.panes {
				if p.thread == status.Address {
					c.expectedExits[id] = true
				}
			}
			if !origin.PreserveFocus {
				c.focus = FocusPanel()
				c.menu.ActiveAddress = status.Address
			}
		}
	}
	for _, address := range result.addresses {
		if result.err == nil && !seen[address] && !c.continuations[address].queued {
			// The record no longer holds a request -- completed elsewhere, or
			// DISMISSED (#280); reconcile below drops the prompt it produced.
			delete(c.continuations, address)
		}
	}
	c.reconcileContinuationOrientationLocked()
	c.mu.Unlock()
}

func (c *Console) finishContinuationOperation(completed operationCompletion, err error) {
	if completed.origin.ContinuationID == "" && (completed.name == "recover-thread" || completed.name == "recover-checkpoint") {
		if result, ok := completed.value.(couchcore.ContinuationResult); ok && result.Status.RequestID != "" && result.Status.Address == completed.origin.Address {
			completed.origin.ContinuationID = result.Status.RequestID
			c.mu.Lock()
			if current, exists := c.continuations[result.Status.Address]; exists && current.status.RequestID != result.Status.RequestID {
				c.mu.Unlock()
				return
			}
			c.continuations[result.Status.Address] = continuationWatch{status: result.Status, handled: !result.SourceReattached}
			c.reconcileContinuationOrientationLocked()
			c.mu.Unlock()
		}
	}
	if completed.origin.ContinuationID == "" {
		return
	}
	c.mu.Lock()
	address := completed.origin.Address
	watch, exists := c.continuations[address]
	if !exists || watch.status.RequestID != completed.origin.ContinuationID {
		c.mu.Unlock()
		return
	}
	watch.queued = false
	switch result := completed.value.(type) {
	case couchcore.ContinuationStatus:
		if result.RequestID == watch.status.RequestID && result.Address == address {
			watch.status = result
		}
	case couchcore.ContinuationResult:
		if result.Status.RequestID != watch.status.RequestID || result.Status.Address != address {
			break
		}
		watch.status = result.Status
		if result.SourceReattached {
			watch.handled = false
		}
		if result.Orientation != nil {
			c.setOrientationLocked(address, *result.Orientation, continuationProducer(watch.status.RequestID))
		}
	}
	c.continuations[address] = watch
	if watch.status.Phase == checkpoint.Complete {
		delete(c.continuations, address)
	}
	c.reconcileContinuationOrientationLocked()
	c.mu.Unlock()
	if err != nil {
		c.setNotice(fmt.Sprintf("Continuation %s: %v", address.Tag, err))
	}
	c.requestMenuRefresh()
}
