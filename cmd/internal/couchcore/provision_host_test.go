package couchcore

import "testing"

func TestProvisionHostDecisionTable(t *testing.T) {
	for _, tc := range []struct {
		o    HostObservation
		want HostAction
	}{
		{HostObservation{HostAbsent, SetupUnconfirmed}, CreateHost},
		{HostObservation{HostOwnedPartial, SetupUnconfirmed}, CompleteHost},
		{HostObservation{HostVerified, SetupUnconfirmed}, CompileHost},
		{HostObservation{HostVerified, SetupConfirmed}, ReuseHost},
		{HostObservation{HostConflict, SetupUnconfirmed}, RefuseHost},
		{HostObservation{HostAbsent, SetupConfirmed}, RefuseHost},
		{HostObservation{HostVerified, SetupConflict}, RefuseHost},
		{HostObservation{HostKind(255), HostSetup(255)}, RefuseHost},
	} {
		if got := NextHostAction(tc.o); got != tc.want {
			t.Errorf("%+v => %v want %v", tc.o, got, tc.want)
		}
	}
}
