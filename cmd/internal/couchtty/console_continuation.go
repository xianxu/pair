package couchtty

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/xianxu/pair/cmd/internal/checkpoint"
	"github.com/xianxu/pair/cmd/internal/couchcore"
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
	status couchcore.ContinuationStatus
	queued bool
	// marked are the expected-exit marks THIS watch's queued operation added
	// (pair#205), so a refusal undoes its own marks and never one that a park,
	// detach or other operation set on the same thread's panes.
	marked []string
}

// exitMarks counts, per child handle, the operations that expect it to exit.
// An undo removes one owner's mark; the child's actual exit consumes them all.
type exitMarks map[string]int

func (m exitMarks) mark(id string) { m[id]++ }

func (m exitMarks) unmark(id string) {
	if m[id] <= 1 {
		delete(m, id)
		return
	}
	m[id]--
}

func (m exitMarks) has(id string) bool { return m[id] > 0 }

// consume reports whether any operation expected id to exit, and clears it:
// one exit satisfies every owner.
func (m exitMarks) consume(id string) bool {
	if m[id] == 0 {
		return false
	}
	delete(m, id)
	return true
}

// markThreadExitsLocked adds one expected-exit mark, owned by the caller, to
// every pane of address, and returns them so a refusal can undo exactly its
// own. Callers hold c.mu.
func (c *Console) markThreadExitsLocked(address couchcore.ThreadAddress) []string {
	var added []string
	for id, p := range c.panes {
		if p.thread == address {
			c.expectedExits.mark(id)
			added = append(added, id)
		}
	}
	return added
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

// continuationOperation is what a scanned request needs: Couch drives every
// unfinished request with continue-thread, including one a previous Couch left
// running (#522: there is no delivery to reconcile afterwards).
func continuationOperation(status couchcore.ContinuationStatus) string {
	switch status.Phase {
	case checkpoint.Pending, checkpoint.Running:
		return "continue-thread"
	default:
		return ""
	}
}

// Run owns acceptance, focus and request ordering. The scanner only reads.
func (c *Console) acceptContinuationRequests(result continuationScanResult) {
	paintWarning := false
	c.mu.Lock()
	if c.lifetime.Err() == nil && result.err != nil {
		text := boundedNoticeText("Continuation status unavailable: "+result.err.Error(), 4096)
		if text != c.continuationScanError {
			c.continuationScanError = text
			paintWarning = c.publishWarningLocked(Notice{Kind: "continuation-scan", Body: text})
		}
	} else if result.err == nil {
		c.continuationScanError = ""
	}
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
		operation := continuationOperation(status)
		_, uiBusy := menuRunningForAddress(c.menu, status.Address)
		if operation == "" || watch.queued || c.ops == nil || uiBusy {
			continue
		}
		fn := c.ops
		origin := MenuOperationOrigin{Operation: operation, Address: status.Address, ContinuationID: status.RequestID, NavigationGeneration: c.navigationGeneration, PreserveFocus: true}
		for id, p := range c.panes {
			if p.thread == status.Address && c.focus == FocusActor(id) {
				origin.PreserveFocus = false
			}
		}
		args := map[string]string{"repo-scope": status.Address.RepoScope, "tag": string(status.Address.Tag), "request-id": status.RequestID}
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
		watch.queued = true
		if operation == "continue-thread" {
			watch.marked = c.markThreadExitsLocked(status.Address)
		}
		c.continuations[status.Address] = watch
		if operation == "continue-thread" {
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
	c.mu.Unlock()
	if paintWarning {
		c.repaint()
	}
}

func (c *Console) finishContinuationOperation(completed operationCompletion, err error) {
	// A routed resume (pair#363) returns RecoverThread's or Continue's
	// ContinuationResult; a published request in it starts the watch, and a
	// request that didn't finish leaves its one line (#522).
	if missed := missedContinuation(completed.value); missed != nil && completed.origin.ContinuationID == "" {
		c.mu.Lock()
		if watch, ok := c.continuations[completed.origin.Address]; ok && !watch.queued {
			delete(c.continuations, completed.origin.Address)
		}
		label, live := c.threadRowLocked(completed.origin.Address)
		c.mu.Unlock()
		c.Warn(Notice{Kind: "continuation", Thread: completed.origin.Address, Label: label, Body: missed.Line(label, live)})
		return
	}
	if completed.origin.ContinuationID == "" && completed.name == "resume" {
		if result, ok := completed.value.(couchcore.ContinuationResult); ok && result.Status.RequestID != "" && result.Status.Address == completed.origin.Address {
			completed.origin.ContinuationID = result.Status.RequestID
			c.mu.Lock()
			if current, exists := c.continuations[result.Status.Address]; exists && current.status.RequestID != result.Status.RequestID {
				c.mu.Unlock()
				return
			}
			c.continuations[result.Status.Address] = continuationWatch{status: result.Status}
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
	busy := couchcore.IsThreadBusy(err)
	if busy {
		// Refused because another operation held the thread (pair#205): the
		// operation never ran. Undo exactly the exit marks it added -- not marks
		// another operation set on these panes -- or a later real exit would be
		// swallowed as expected.
		for _, id := range watch.marked {
			c.expectedExits.unmark(id)
		}
	}
	watch.marked = nil
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
	}
	c.continuations[address] = watch
	missed := missedContinuation(completed.value)
	if watch.status.Phase == checkpoint.Complete || missed != nil {
		delete(c.continuations, address)
	}
	label, live := c.threadRowLocked(address)
	c.mu.Unlock()
	if missed != nil {
		// The one visible line (#522): the request is gone and nothing waits
		// on it. The slot's row keeps it until the next start or compaction.
		c.Warn(Notice{Kind: "continuation", Thread: address, Label: label, Body: missed.Line(label, live)})
	} else if err != nil && !busy {
		// A busy refusal is retried on the next scan, silently.
		c.Warn(Notice{Kind: "continuation", Thread: address, Label: label, Body: fmt.Sprintf("Continuation %s: %v", address.Tag, couchcore.OperatorError(err))})
	}
	c.requestMenuRefresh()
}

// missedContinuation is the line a finished operation reports when its
// request didn't finish (#522), or nil.
func missedContinuation(value any) *checkpoint.Miss {
	if result, ok := value.(couchcore.ContinuationResult); ok {
		return result.Missed
	}
	return nil
}

// threadRowLocked is the switcher's label for a thread and whether it is live,
// from the inventory it last drew; the tag when the row isn't there yet.
// Callers hold c.mu.
func (c *Console) threadRowLocked(address couchcore.ThreadAddress) (string, bool) {
	if row, ok := menuThread(c.menu, address); ok {
		return row.Label(), row.State == couchcore.ThreadLive
	}
	return string(address.Tag), false
}
