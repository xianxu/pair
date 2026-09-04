package couchcore

import "testing"

func TestThreadLabelPrefersNameThenDirectory(t *testing.T) {
	tag := ThreadTag("couch-0123456789abcdef")
	for _, tc := range []struct{ name, path, want string }{
		{"compiler", "/Users/x/workspace/brain", "compiler"},
		{"", "/Users/x/workspace/brain", "brain"},
		{"", "/Users/x/workspace/kbench/competition/arc-agi-3", "arc-agi-3"},
		{"", "", string(tag)},
		{"", "/", string(tag)},
	} {
		if got := threadLabel(tc.name, tc.path, tag); got != tc.want {
			t.Errorf("threadLabel(%q, %q) = %q, want %q", tc.name, tc.path, got, tc.want)
		}
	}
}

// Six rows reading `brain` is readable and useless. The operator's store has
// exactly that shape, so a colliding label has to stay actionable.
func TestDisambiguateLabelsOnlyQualifiesCollisions(t *testing.T) {
	rows := []LabelRow{
		{Address: ThreadAddress{RepoScope: "s", Tag: "couch-000000000000aaaa"}, Label: "brain"},
		{Address: ThreadAddress{RepoScope: "s", Tag: "couch-000000000000bbbb"}, Label: "brain"},
		{Address: ThreadAddress{RepoScope: "s", Tag: "couch-000000000000cccc"}, Label: "pair"},
	}
	got := DisambiguateLabels(rows)

	if got[rows[2].Address] != "pair" {
		t.Fatalf("unique label was qualified: %q", got[rows[2].Address])
	}
	if got[rows[0].Address] == got[rows[1].Address] {
		t.Fatalf("colliding labels stayed identical: %q", got[rows[0].Address])
	}
	for _, row := range rows[:2] {
		if want := "brain·"; len(got[row.Address]) <= len(want) || got[row.Address][:len(want)] != want {
			t.Fatalf("qualified label = %q, want it to keep the name", got[row.Address])
		}
	}
}
