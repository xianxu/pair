package titlepoller

import "testing"

func TestParseTitleOwner(t *testing.T) {
	for _, pair := range [][2]string{{"0", "1.2"}, {"-1", "1.2"}, {"42", ""}, {"42", "bad"}, {"42", "1.2 extra"}, {"99999999999999", "1.2"}} {
		if _, err := ParseTitleOwner(pair[0], pair[1]); err == nil {
			t.Errorf("accepted invalid owner %q", pair)
		}
	}
	owner, err := ParseTitleOwner("42", "123456.789")
	if err != nil || owner.PID != 42 || owner.Start != "123456.789" {
		t.Fatalf("owner=%+v err=%v", owner, err)
	}
}
func FuzzParseTitleOwner(f *testing.F) {
	f.Add("42", "123.4")
	f.Add("-1", "")
	f.Fuzz(func(t *testing.T, pid, start string) {
		owner, err := ParseTitleOwner(pid, start)
		if err == nil && (owner.PID <= 0 || owner.Start == "") {
			t.Fatal("invalid owner escaped parser")
		}
	})
}
