package couchmessage

import "testing"

func TestEndpointRejectsRetiredDrainOperations(t *testing.T) {
	for _, op := range []string{"drain-observe", "drain-interrupt", "drain-interrupt-status", "drain-revoke", "drain-reserve", "drain-release"} {
		t.Run(op, func(t *testing.T) {
			if err := ValidateEndpointRequest(EndpointRequest{Op: op, Binding: brokerBinding("pair:1")}); err == nil {
				t.Fatalf("retired drain operation %q is still admitted", op)
			}
		})
	}
}
