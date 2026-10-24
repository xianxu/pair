package couchcore

// startRollbackReceipt is request-local evidence returned by the existing
// revision/nonce-fenced store mutation. It grants no cleanup authority.
type startRollbackReceipt struct {
	Nonce  string
	Record ThreadRecord
}

func (c *Couch) rollbackTrackedStartObserved(thread ThreadRecord, nonce string, receipt *startRollbackReceipt) error {
	return c.rollbackTrackedStartFrom(c.Threads, thread, nonce, receipt)
}

// rollbackTrackedStartFrom also supports a published local claim whose
// namespace enrollment failed. The store that accepted it owns both the exact
// deletion and the positive absence needed before releasing its native claim.
func (c *Couch) rollbackTrackedStartFrom(store *ThreadStore, thread ThreadRecord, nonce string, receipt *startRollbackReceipt) error {
	rolled, err := store.deleteStartResult(thread.Address, thread.Revision, nonce)
	if err != nil {
		return err
	}
	if err := c.releaseClaimIfStoreThreadAbsent(store, thread.Address); err != nil {
		return err
	}
	if receipt != nil && rolled.Revision != 0 {
		*receipt = startRollbackReceipt{Nonce: nonce, Record: rolled}
	}
	return nil
}
