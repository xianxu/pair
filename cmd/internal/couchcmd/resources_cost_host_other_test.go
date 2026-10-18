//go:build !darwin

package couchcmd

// hostLoadAverage names the host state a cost measurement was taken in (#644).
// The darwin build reads vm.loadavg; elsewhere the report says it does not know,
// which changes no verdict — the cost gates are load-invariant by construction.
func hostLoadAverage() string { return "loadavg=unknown" }
