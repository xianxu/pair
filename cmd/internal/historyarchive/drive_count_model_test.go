package historyarchive

import (
	"reflect"
	"strconv"
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"
)

func TestParseDriveMaxFiles(t *testing.T) {
	for _, tc := range []struct {
		raw  string
		want int
		bad  bool
	}{
		{"", 20000, false}, {"1", 1, false}, {"+42", 42, false}, {"0007", 7, false},
		{"0", 0, true}, {"-1", 0, true}, {" 42", 0, true}, {"42\n", 0, true}, {"1.0", 0, true},
		{"20000files", 0, true}, {"99999999999999999999999999999999", 0, true},
	} {
		t.Run(strconv.Quote(tc.raw), func(t *testing.T) {
			got, err := ParseDriveMaxFiles(tc.raw)
			if (err != nil) != tc.bad || !tc.bad && got != tc.want {
				t.Fatalf("ParseDriveMaxFiles(%q)=(%d,%v), want %d bad=%v", tc.raw, got, err, tc.want, tc.bad)
			}
		})
	}
}

func TestObserveDriveInventoryCountsObjectsWithoutMutation(t *testing.T) {
	for seed := 0; seed < 50; seed++ {
		inventory := map[string][]RemoteEntry{}
		wantFiles, wantEntries := 0, 0
		for key := 0; key < 7; key++ {
			for n := 0; n < (seed+key)%11; n++ {
				entry := RemoteEntry{Name: "same-unrelated-name", ID: strconv.Itoa(n), IsDir: (n+key)%3 == 0, Hashes: map[string]string{"md5": "hash"}}
				inventory[strconv.Itoa(key)] = append(inventory[strconv.Itoa(key)], entry)
				wantEntries++
				if !entry.IsDir {
					wantFiles++
				}
			}
		}
		before := cloneDriveCountInventory(inventory)
		destination := DriveDestination{Remote: "backup", Folder: "configured/root"}
		got, entries := observeDriveInventory(destination, "root-id", "before", inventory)
		want := DriveInventoryObservation{Destination: "backup:configured/root", FolderID: "root-id", Phase: "before", Files: wantFiles}
		if got != want || entries != wantEntries {
			t.Fatalf("seed %d: got %+v/%d want %+v/%d", seed, got, entries, want, wantEntries)
		}
		if !reflect.DeepEqual(inventory, before) {
			t.Fatal("observation mutated inventory")
		}
	}
	got, total := observeDriveInventory(DriveDestination{"r", "root"}, "id", "after", nil)
	if got.Files != 0 || total != 0 || got.Phase != "after" {
		t.Fatalf("empty inventory: %+v/%d", got, total)
	}
}

func cloneDriveCountInventory(in map[string][]RemoteEntry) map[string][]RemoteEntry {
	out := make(map[string][]RemoteEntry, len(in))
	for k, entries := range in {
		out[k] = append([]RemoteEntry(nil), entries...)
		for i, e := range entries {
			if e.Hashes != nil {
				out[k][i].Hashes = map[string]string{}
				for h, v := range e.Hashes {
					out[k][i].Hashes[h] = v
				}
			}
		}
	}
	return out
}

func TestDriveInventoryObservationLinesAndThreshold(t *testing.T) {
	var unknown *DriveInventoryObservation
	if unknown.Line() != "Drive file count unavailable" || unknown.Warning(1) != "" {
		t.Fatalf("unknown: %q %q", unknown.Line(), unknown.Warning(1))
	}
	for files := 0; files < 120; files++ {
		for limit := -1; limit < 120; limit++ {
			o := &DriveInventoryObservation{Destination: "remote:root", FolderID: "identity", Phase: "after", Files: files}
			line := o.Line()
			for _, part := range []string{strconv.Itoa(files), o.Destination, o.Phase} {
				if !strings.Contains(line, part) {
					t.Fatalf("line %q missing %q", line, part)
				}
			}
			warning := o.Warning(limit)
			if (warning != "") != (limit > 0 && files > limit) {
				t.Fatalf("threshold files=%d limit=%d warning=%q", files, limit, warning)
			}
			if warning != "" {
				for _, part := range []string{strconv.Itoa(files), strconv.Itoa(limit), o.Destination, "Ask the hub to inspect this folder and its archive layout policy."} {
					if strings.Count(warning, part) < 1 {
						t.Fatalf("warning %q missing %q", warning, part)
					}
				}
			}
		}
	}
}

func TestDriveInventoryObservationTextIsBoundedAndSafe(t *testing.T) {
	o := &DriveInventoryObservation{Destination: strings.Repeat("界\n\x1b\u202e\u200d", 2000), FolderID: strings.Repeat("x", 2000), Phase: strings.Repeat("after\r", 2000), Files: 20001}
	for _, line := range []string{o.Line(), o.Warning(20000)} {
		if !utf8.ValidString(line) || utf8.RuneCountInString(line) > 600 {
			t.Fatalf("unbounded/invalid output: %d runes", utf8.RuneCountInString(line))
		}
		for _, r := range line {
			if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) {
				t.Fatalf("unsafe output rune %U", r)
			}
		}
	}
	if strings.Count(o.Warning(20000), "Ask the hub to inspect this folder and its archive layout policy.") != 1 {
		t.Fatal("missing single recovery action")
	}
}

func TestValidateDriveInventoryObservation(t *testing.T) {
	if err := validateDriveInventoryObservation(nil); err != nil {
		t.Fatal(err)
	}
	valid := DriveInventoryObservation{Destination: "remote:configured/root", FolderID: "root-id", Phase: "before", Files: 0}
	for _, count := range []int{0, 20000, maxFlatInventoryEntries + 1, int(^uint(0) >> 1)} {
		o := valid
		o.Files = count
		if err := validateDriveInventoryObservation(&o); err != nil {
			t.Fatalf("valid count %d: %v", count, err)
		}
	}
	after := valid
	after.Phase = "after"
	if err := validateDriveInventoryObservation(&after); err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func(*DriveInventoryObservation){
		func(o *DriveInventoryObservation) { o.Destination = "" }, func(o *DriveInventoryObservation) { o.Destination = ":folder" }, func(o *DriveInventoryObservation) { o.Destination = "remote:" }, func(o *DriveInventoryObservation) { o.Destination = "remote:../escape" }, func(o *DriveInventoryObservation) { o.Destination = "remote:root:extra" },
		func(o *DriveInventoryObservation) { o.FolderID = "" }, func(o *DriveInventoryObservation) { o.FolderID = "bad\nidentity" }, func(o *DriveInventoryObservation) { o.Phase = "during" }, func(o *DriveInventoryObservation) { o.Files = -1 },
	} {
		o := valid
		mutate(&o)
		if err := validateDriveInventoryObservation(&o); err == nil {
			t.Fatalf("accepted invalid observation %+v", o)
		}
	}
}

func FuzzParseDriveMaxFiles(f *testing.F) {
	for _, seed := range []string{"", "+1", "20000", "0", "-1", " 2", "9999999999999999999999"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, raw string) {
		got, err := ParseDriveMaxFiles(raw)
		if raw == "" {
			if err != nil || got != DefaultDriveMaxFiles {
				t.Fatalf("default %d %v", got, err)
			}
			return
		}
		want, parseErr := strconv.Atoi(raw)
		if parseErr != nil || want <= 0 {
			if err == nil {
				t.Fatalf("accepted %q", raw)
			}
			return
		}
		if err != nil || got != want {
			t.Fatalf("positive %q: %d %v", raw, got, err)
		}
	})
}

func FuzzDriveInventoryWarningSafety(f *testing.F) {
	f.Add("remote:root", "before", 20001, 20000)
	f.Add("\x1b\u202e\n", "\u200d", 3, 2)
	f.Fuzz(func(t *testing.T, destination, phase string, files, limit int) {
		o := &DriveInventoryObservation{Destination: destination, Phase: phase, Files: files}
		warning := o.Warning(limit)
		if (warning != "") != (limit > 0 && files > limit) {
			t.Fatal("threshold law")
		}
		for _, line := range []string{o.Line(), warning} {
			if !utf8.ValidString(line) || utf8.RuneCountInString(line) > 600 {
				t.Fatal("text bound")
			}
			for _, r := range line {
				if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) {
					t.Fatalf("unsafe %U", r)
				}
			}
		}
	})
}

func TestLatestDriveInventories(t *testing.T) {
	root := DriveInventoryObservation{Destination: "remote:root", FolderID: "root-id", Phase: "before", Files: 2}
	after := root
	after.Phase = "after"
	after.Files = 4
	leaf := DriveInventoryObservation{Destination: "remote:root/2026/10", FolderID: "month-id", Phase: "before", Files: 0}
	for name, tc := range map[string]struct {
		report *DriveReport
		want   []DriveInventoryObservation
	}{
		"nil": {nil, nil}, "unknown": {&DriveReport{}, nil},
		"legacy":              {&DriveReport{Inventory: &root}, []DriveInventoryObservation{root}},
		"legacy and additive": {&DriveReport{Inventory: &after, Inventories: []DriveInventoryObservation{root, leaf, after, leaf}}, []DriveInventoryObservation{after, leaf}},
		"after before":        {&DriveReport{Inventories: []DriveInventoryObservation{after, root, leaf}}, []DriveInventoryObservation{after, leaf}},
	} {
		t.Run(name, func(t *testing.T) {
			got := LatestDriveInventories(tc.report)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("got=%+v want=%+v", got, tc.want)
			}
			if len(got) > 0 {
				got[0].Files = -1
				if reflect.DeepEqual(LatestDriveInventories(tc.report), got) {
					t.Fatal("projection mutated report")
				}
			}
		})
	}
}
