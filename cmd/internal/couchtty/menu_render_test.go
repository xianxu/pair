package couchtty

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/ansi"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/textwidth"
)

func TestChooseMenuLayoutBoundsWideNarrowAndResize(t *testing.T) {
	state := NewMenuState(menuThreads(), menuAddress("couch-one"))
	state, _ = reduceKey(state, PanelKey{Kind: KeyTab})
	for _, tc := range []struct {
		name          string
		width, height int
		want          MenuLayoutKind
	}{
		{name: "wide", width: 120, height: 40, want: MenuLayoutWide},
		{name: "narrow minimum", width: 40, height: 10, want: MenuLayoutNarrow},
		{name: "too narrow", width: 39, height: 10, want: MenuLayoutResize},
		{name: "too short", width: 80, height: 9, want: MenuLayoutResize},
	} {
		t.Run(tc.name, func(t *testing.T) {
			layout := ChooseMenuLayout(state, tc.width, tc.height)
			if layout.Kind != tc.want {
				t.Fatalf("layout = %+v, want kind %v", layout, tc.want)
			}
			for _, rect := range layout.Frames {
				if rect.X < 0 || rect.Y < 0 || rect.Width < 1 || rect.Height < 1 || rect.X+rect.Width > tc.width || rect.Y+rect.Height > tc.height {
					t.Fatalf("layout rect escapes terminal %dx%d: %+v", tc.width, tc.height, rect)
				}
			}
		})
	}
}

func TestChooseMenuLayoutAnchorsWideChildrenToSelectedParentRows(t *testing.T) {
	state := NewMenuState(menuThreads(), menuAddress("couch-one"))
	state, _ = reduceKey(state, PanelKey{Kind: KeyDown})
	state, _ = reduceKey(state, PanelKey{Kind: KeyTab})
	layout := ChooseMenuLayout(state, 120, 40)
	if layout.Kind != MenuLayoutWide || len(layout.Frames) != 2 || layout.Frames[1].Y != 3 {
		t.Fatalf("wide child geometry = %+v, want child beside root row 3", layout)
	}

	state = NewMenuState(menuThreads(), menuAddress("couch-one"))
	state, _ = reduceKey(state, PanelKey{Kind: KeyTab})
	state, _ = reduceKey(state, PanelKey{Kind: KeyEnter})
	layout = ChooseMenuLayout(state, 160, 40)
	if layout.Kind != MenuLayoutWide || len(layout.Frames) != 3 || layout.Frames[1].Y != 2 || layout.Frames[2].Y != 4 {
		t.Fatalf("nested wide geometry = %+v, want child rows 2 then 4", layout)
	}
}

func TestChooseMenuLayoutPlacesNarrowChildBelowParentList(t *testing.T) {
	state := NewMenuState(menuThreads(), menuAddress("couch-one"))
	state, _ = reduceKey(state, PanelKey{Kind: KeyTab})
	layout := ChooseMenuLayout(state, 60, 20)
	if layout.Kind != MenuLayoutNarrow || len(layout.Frames) != 2 || layout.Frames[0].Height != 4 || layout.Frames[1].Y != 4 {
		t.Fatalf("narrow child geometry = %+v, want child below four-row root list", layout)
	}
}

func TestRenderMenuMinimumNarrowLayoutKeepsCurrentFrameOperable(t *testing.T) {
	state := NewMenuState(menuThreads(), menuAddress("couch-one"))
	state.Agents = []string{"claude"}
	state.RootAgent = "claude"
	state, _ = reduceKey(state, PanelKey{Kind: KeyTab})
	state, _ = reduceKey(state, PanelKey{Kind: KeyEnter})
	state, _ = reduceKey(state, PanelKey{Kind: KeyCtrlSpace})
	plain := string(ansi.Strip([]byte(RenderMenu(state, 40, 10, time.Time{}, false))))
	if !strings.Contains(plain, "path") || !strings.Contains(plain, "agent claude") {
		t.Fatalf("minimum narrow layout hid current start controls: %q", plain)
	}
}

func TestAgeBandBoundaries(t *testing.T) {
	now := time.Unix(10*24*60*60, 0).UTC()
	for _, tc := range []struct {
		age  time.Duration
		want AgeBand
	}{
		{age: 23*time.Hour + 59*time.Minute, want: AgeRecent},
		{age: 24 * time.Hour, want: AgeDays},
		{age: 7*24*time.Hour - time.Second, want: AgeDays},
		{age: 7 * 24 * time.Hour, want: AgeOld},
	} {
		if got := AgeBandFor(now, now.Add(-tc.age)); got != tc.want {
			t.Errorf("age %v band = %v, want %v", tc.age, got, tc.want)
		}
	}
}

func TestRenderMenuKeepsSelectedRowVisibleAndBounded(t *testing.T) {
	threads := make([]couchcore.ActionableThreadSummary, 100)
	for i := range threads {
		threads[i] = couchcore.ActionableThreadSummary{
			Address:      menuAddress(fmt.Sprintf("couch-%03d", i)),
			WorkingPath:  fmt.Sprintf("/repo/thread-%03d", i),
			Name:         fmt.Sprintf("thread-%03d", i),
			State:        couchcore.ThreadParked,
			LastActiveAt: time.Unix(int64(i+1), 0).UTC(),
		}
	}
	state := NewMenuState(threads, couchcore.ThreadAddress{})
	state.Frames[0].SelectedAddress = threads[99].Address
	got := RenderMenu(state, 40, 10, time.Unix(200000, 0).UTC(), true)
	if !strings.Contains(got, "thread-099") || !strings.Contains(got, "▸") {
		t.Fatalf("selected row is outside viewport: %q", got)
	}
	assertRenderedBounds(t, got, 40, 10)
}

func TestRenderMenuWideChildStaysInsideTerminal(t *testing.T) {
	threads := menuThreads()
	threads[0].Name = strings.Repeat("wide", 40)
	state := NewMenuState(threads, menuAddress("couch-one"))
	state, _ = reduceKey(state, PanelKey{Kind: KeyTab})
	got := RenderMenu(state, 120, 40, time.Unix(200000, 0).UTC(), true)
	if !strings.Contains(got, "actions · wide") || !strings.Contains(got, "park") {
		t.Fatalf("wide render omitted child frame: %q", got)
	}
	assertRenderedBounds(t, got, 120, 40)
}

func TestRenderMenuUsesOperationPresentationLabels(t *testing.T) {
	state := NewMenuState(menuThreads(), menuAddress("couch-one"))
	state, _ = reduceKey(state, PanelKey{Kind: KeyTab})
	plain := string(ansi.Strip([]byte(RenderMenu(state, 120, 40, time.Time{}, false))))
	if !strings.Contains(plain, "rename") || strings.Contains(plain, "\nname") || strings.Contains(plain, "\r\n  name") {
		t.Fatalf("action menu leaked operation identifier instead of label: %q", plain)
	}
}

func TestRenderMenuSanitizesClipsAndKeepsStateText(t *testing.T) {
	now := time.Unix(10*24*60*60, 0).UTC()
	threads := []couchcore.ActionableThreadSummary{
		{Address: menuAddress("live"), Name: "live界\x1b[2J", WorkingPath: "/very/long/界界界/path", State: couchcore.ThreadLive},
		{Address: menuAddress("parked"), Name: "parked", WorkingPath: "/repo", State: couchcore.ThreadParked, LastActiveAt: now.Add(-2 * 24 * time.Hour)},
	}
	got := RenderMenu(NewMenuState(threads, threads[0].Address), 40, 10, now, false)
	if strings.Contains(got, "\x1b[2J") || !strings.Contains(string(ansi.Strip([]byte(got))), "live") || !strings.Contains(got, "2d ago") {
		t.Fatalf("rendered unsafe or missing state text: %q", got)
	}
	if strings.Contains(strings.Split(got, "\r\n")[2], "ago") {
		t.Fatalf("live row rendered historical age: %q", got)
	}
	assertRenderedBounds(t, got, 40, 10)
}

func TestRenderMenuBelowMinimumOnlyRequestsResize(t *testing.T) {
	got := RenderMenu(NewMenuState(menuThreads(), menuAddress("couch-one")), 39, 9, time.Time{}, true)
	plain := string(ansi.Strip([]byte(got)))
	if plain != "resize terminal to at least 40x10" {
		t.Fatalf("small render = %q", got)
	}
}

func TestClipMenuLineUsesTerminalColumns(t *testing.T) {
	if got := clipMenuLine("ab界cd", 4); got != "ab界" {
		t.Fatalf("clip = %q, want %q", got, "ab界")
	}
}

func assertRenderedBounds(t *testing.T, rendered string, width, height int) {
	t.Helper()
	lines := strings.Split(rendered, "\r\n")
	if len(lines) > height {
		t.Fatalf("rendered %d lines into height %d: %q", len(lines), height, rendered)
	}
	for i, line := range lines {
		if got := textwidth.Width(string(ansi.Strip([]byte(line)))); got > width {
			t.Fatalf("line %d is %d columns into width %d: %q", i, got, width, line)
		}
	}
}
