package notes

import "testing"

func TestFakeModelMeetsTheModelContract(t *testing.T) {
	RunModelContract(t, func() Model { return FakeModel{} })
}
