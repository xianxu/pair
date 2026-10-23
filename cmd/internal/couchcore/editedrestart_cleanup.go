package couchcore

// startRollbackReceipt is request-local evidence returned by the existing
// revision/nonce-fenced store mutation. It grants no cleanup authority.
type startRollbackReceipt struct {
	Nonce  string
	Record ThreadRecord
}

func (c *Couch) rollbackTrackedStartObserved(thread ThreadRecord, nonce string, receipt *startRollbackReceipt) error {
	rolled, err := c.Threads.deleteStartResult(thread.Address, thread.Revision, nonce)
	if err != nil {
		return err
	}
	if err := c.releaseClaimIfThreadAbsent(thread.Address); err != nil {
		return err
	}
	if receipt != nil && rolled.Revision != 0 {
		*receipt = startRollbackReceipt{Nonce: nonce, Record: rolled}
	}
	return nil
}
