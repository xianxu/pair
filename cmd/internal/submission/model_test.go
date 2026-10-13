package submission

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestReadingGuideNamesBundleAndRecordingCommands(t *testing.T) {
	path := "/recordings/a space's recording"
	guide := ReadingGuide(path)
	for _, want := range []string{"submission.json", "crop-N.png", path, "couch --broadcast-web-query DIR notes|dom|events|link", "start with notes", "dom --at <t> --sel @N", "events --from <t-10000> --to <t>", "couch --skill"} {
		if !strings.Contains(guide, want) {
			t.Errorf("guide missing %q: %s", want, guide)
		}
	}
	data, err := json.Marshal(Submission{HowToRead: guide})
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded["how_to_read"] != guide {
		t.Fatal("guide missing from manifest", string(data))
	}
}

func TestBoxTextLimits(t *testing.T) {
	valid := BoxText{Region: [4]float64{0, 0, 20, 20}, VisibleText: []BoxTextEntry{{Selector: "#a", Text: "abc"}}}
	if err := ValidateBoxText(valid); err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func(*BoxText){
		func(b *BoxText) { b.VisibleText[0].Text = strings.Repeat("x", MaxBoxTextBytes) },
		func(b *BoxText) {
			b.AccessibleNames = []BoxTextEntry{{Kind: "title", Text: strings.Repeat("x", MaxBoxTextBytes)}}
		},
		func(b *BoxText) { b.AccessibleNames = make([]BoxTextEntry, MaxBoxTextEntries) },
		func(b *BoxText) { b.Region[2] = -1 },
		func(b *BoxText) { b.Unavailable = "page changed" },
	} {
		b := valid
		b.VisibleText = append([]BoxTextEntry(nil), valid.VisibleText...)
		mutate(&b)
		if ValidateBoxText(b) == nil {
			t.Fatalf("accepted out of bounds: %#v", b)
		}
	}
	if err := ValidateBoxText(BoxText{Unavailable: "page changed"}); err != nil {
		t.Fatal(err)
	}
}
