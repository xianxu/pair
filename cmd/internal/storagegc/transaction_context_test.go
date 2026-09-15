package storagegc

import (
	"context"
	"errors"
	"testing"
)

func TestCancelledTransactionInspectionDoesNotTraverse(t *testing.T) {
	for _, operation := range []string{"source", "quarantine", "pending-owner"} {
		t.Run(operation, func(t *testing.T) {
			c, item := transactionFixture(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			err := c.Coordinator.WithLock(ctx, func(held *Locked) error {
				tx, err := c.prepareCollection(held, item)
				if err != nil {
					return err
				}
				cancel()
				switch operation {
				case "source":
					return c.verifySourceTree(held, tx, tx.Entries[0])
				case "quarantine":
					return c.verifyTree(held, tx, true)
				default:
					return held.pendingOwnerTransaction(item.Owner)
				}
			})
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("inspection continued after cancellation: %v", err)
			}
		})
	}
}
