//go:build !darwin

package workbenchperftest

const ledgerCloneAvailable = false

func cloneLedgerFile(_, _ string) error { return errLedgerCloneUnsupported }
