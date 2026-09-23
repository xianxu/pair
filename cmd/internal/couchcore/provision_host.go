package couchcore

import (
	"fmt"
	"strings"
)

type HostKind uint8

const (
	HostConflict HostKind = iota
	HostAbsent
	HostOwnedPartial
	HostVerified
)

type HostSetup uint8

const (
	SetupUnconfirmed HostSetup = iota
	SetupConfirmed
	SetupConflict
)

type HostObservation struct {
	Kind  HostKind
	Setup HostSetup
}
type HostAction uint8

const (
	RefuseHost HostAction = iota
	CreateHost
	CompleteHost
	CompileHost
	ReuseHost
)

// NextHostAction derives the next safe step from observed evidence. It is not a
// stored phase: every invocation rebuilds its observation from Git and files.
func NextHostAction(o HostObservation) HostAction {
	if o.Setup == SetupConflict {
		return RefuseHost
	}
	switch o.Kind {
	case HostAbsent:
		if o.Setup == SetupUnconfirmed {
			return CreateHost
		}
	case HostOwnedPartial:
		if o.Setup == SetupUnconfirmed {
			return CompleteHost
		}
	case HostVerified:
		if o.Setup == SetupConfirmed {
			return ReuseHost
		}
		if o.Setup == SetupUnconfirmed {
			return CompileHost
		}
	}
	return RefuseHost
}

// ParseFetchBaseline takes the immutable new OID from fetch --verbose --porcelain
// stdout. Reading a remote-tracking ref afterward would race another fetch.
func ParseFetchBaseline(raw []byte, ref string) (string, error) {
	if len(raw) > 1<<20 {
		return "", fmt.Errorf("fetch output too large")
	}
	lines := strings.Split(strings.TrimRight(string(raw), "\n"), "\n")
	if len(lines) != 1 || len(lines[0]) < 2 {
		return "", fmt.Errorf("expected exactly one fetch result")
	}
	line := lines[0]
	if !strings.ContainsRune(" =*+", rune(line[0])) || line[1] != ' ' {
		return "", fmt.Errorf("fetch did not confirm a successful ref update")
	}
	fields := strings.Fields(line[1:])
	if len(fields) != 3 || fields[2] != ref {
		return "", fmt.Errorf("unexpected fetch result ref")
	}
	old := fields[0]
	zero := strings.Repeat("0", len(old))
	if !(validWorkspaceOID(old) || (len(old) == 40 || len(old) == 64) && old == zero) || !validWorkspaceOID(fields[1]) || len(old) != len(fields[1]) {
		return "", fmt.Errorf("invalid fetch object ID")
	}
	return fields[1], nil
}
