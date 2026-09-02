package couchtty

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/ansi"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/textwidth"
)

func TestChooseMenuLayoutUsesOneLeftAnchoredSurfaceAtEverySupportedSize(t *testing.T) {
	state := NewMenuState(menuThreads(), menuAddress("couch-one"))
	state, _ = reduceKey(state, PanelKey{Kind: KeyTab})
	for _, tc := range []struct {
		name          string
		width, height int
		want          MenuLayoutKind
	}{
		{name: "wide", width: 120, height: 40, want: MenuLayoutSingle},
		{name: "minimum", width: 40, height: 10, want: MenuLayoutSingle},
		{name: "too narrow", width: 39, height: 10, want: MenuLayoutResize},
		{name: "too short", width: 80, height: 9, want: MenuLayoutResize},
	} {
		t.Run(tc.name, func(t *testing.T) {
			layout := ChooseMenuLayout(tc.width, tc.height)
			if layout.Kind != tc.want {
				t.Fatalf("layout = %+v, want kind %v", layout, tc.want)
			}
			for _, rect := range layout.Frames {
				if rect.X < 0 || rect.Y < 0 || rect.Width < 1 || rect.Height < 1 || rect.X+rect.Width > tc.width || rect.Y+rect.Height > tc.height {
					t.Fatalf("layout rect escapes terminal %dx%d: %+v", tc.width, tc.height, rect)
				}
			}
			if tc.want == MenuLayoutSingle && (len(layout.Frames) != 1 || layout.Frames[0].X != 0 || layout.Frames[0].Y != 0) {
				t.Fatalf("supported layout = %+v, want one surface at origin", layout)
			}
		})
	}
}

func TestRenderMenuUsesSingleSurfaceBreadcrumbs(t *testing.T) {
	root := NewMenuState(menuThreads(), menuAddress("couch-one"))
	actions, _ := reduceKey(root, PanelKey{Kind: KeyTab})
	park, _ := reduceKey(actions, PanelKey{Kind: KeyEnter})
	rename := cloneMenuState(actions)
	rename.Frames[len(rename.Frames)-1].SelectedItem = "name"
	rename, _ = reduceKey(rename, PanelKey{Kind: KeyEnter})
	describe := cloneMenuState(actions)
	describe.Frames[len(describe.Frames)-1].SelectedItem = "describe"
	describe, _ = reduceKey(describe, PanelKey{Kind: KeyEnter})
	leave, _ := ReduceMenu(root, MenuEvent{Kind: MenuEventParkHotkey, Operation: "leave", Address: menuAddress("couch-one")})
	startFromConfirmation, _ := reduceKey(park, PanelKey{Kind: KeyCtrlSpace})

	for _, tc := range []struct {
		name       string
		state      MenuState
		breadcrumb string
		absent     []string
	}{
		{name: "root", state: root, breadcrumb: "threads"},
		{name: "actions", state: actions, breadcrumb: "threads › compiler › actions", absent: []string{"/repo/one", "review"}},
		{name: "park", state: park, breadcrumb: "threads › compiler › park", absent: []string{"/repo/one", "rename"}},
		{name: "rename", state: rename, breadcrumb: "threads › compiler › rename", absent: []string{"/repo/one", "park"}},
		{name: "describe", state: describe, breadcrumb: "threads › compiler › describe", absent: []string{"/repo/one", "park"}},
		// Leave is a GLOBAL frame since #170: it names couch, not a thread, so
		// its breadcrumb no longer borrows an actor's label -- and "compiler"
		// is now asserted ABSENT, because borrowing one is the bug.
		{name: "leave", state: leave, breadcrumb: "threads › leave couch", absent: []string{"actions", "/repo/one", "compiler"}},
		{name: "global start", state: startFromConfirmation, breadcrumb: "start thread", absent: []string{"threads", "compiler", "park"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, size := range [][2]int{{120, 40}, {40, 10}} {
				plain := string(ansi.Strip([]byte(RenderMenu(tc.state, size[0], size[1], time.Time{}, false))))
				if first := strings.Split(plain, "\r\n")[0]; first != tc.breadcrumb {
					t.Fatalf("%dx%d breadcrumb = %q, want %q; render=%q", size[0], size[1], first, tc.breadcrumb, plain)
				}
				for _, absent := range tc.absent {
					if strings.Contains(plain, absent) {
						t.Fatalf("%dx%d retained inactive %q: %q", size[0], size[1], absent, plain)
					}
				}
				assertRenderedBounds(t, plain, size[0], size[1])
			}
		})
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

func TestRenderMenuSingleSurfaceStaysInsideTerminalWithLongBreadcrumb(t *testing.T) {
	threads := menuThreads()
	threads[0].Name = strings.Repeat("wide", 40)
	state := NewMenuState(threads, menuAddress("couch-one"))
	state, _ = reduceKey(state, PanelKey{Kind: KeyTab})
	got := RenderMenu(state, 120, 40, time.Unix(200000, 0).UTC(), true)
	plain := string(ansi.Strip([]byte(got)))
	if !strings.HasPrefix(plain, "threads › wide") || !strings.Contains(plain, "park") || strings.Contains(plain, "/repo/one") {
		t.Fatalf("single-surface render retained or omitted the wrong content: %q", got)
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

func TestRenderMenuUsesSubduedBreadcrumbWhenColorIsAvailable(t *testing.T) {
	state := NewMenuState(menuThreads(), menuAddress("couch-one"))
	state, _ = reduceKey(state, PanelKey{Kind: KeyTab})
	got := RenderMenu(state, 120, 40, time.Time{}, true)
	want := "\x1b[38;5;245mthreads › compiler › actions\x1b[0m"
	if first := strings.Split(got, "\r\n")[0]; first != want {
		t.Fatalf("colored breadcrumb = %q, want %q", first, want)
	}
	plain := RenderMenu(state, 120, 40, time.Time{}, false)
	if first := strings.Split(plain, "\r\n")[0]; first != "threads › compiler › actions" {
		t.Fatalf("plain breadcrumb = %q", first)
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

func TestRenderMenuProtectsStateAgeAndBellSuffixAtMinimumWidth(t *testing.T) {
	now := time.Unix(10*24*60*60, 0).UTC()
	long := strings.Repeat("very-long-label-and-path-", 4)
	threads := []couchcore.ActionableThreadSummary{
		{Address: menuAddress("live"), Name: long, WorkingPath: "/" + long, State: couchcore.ThreadLive},
		{Address: menuAddress("parked"), Name: long, WorkingPath: "/" + long, State: couchcore.ThreadParked, LastActiveAt: now.Add(-2 * 24 * time.Hour)},
	}
	state := NewMenuState(threads, threads[0].Address)
	state.Attention = map[couchcore.ThreadAddress][]AttentionMessage{
		threads[0].Address: {{Sequence: 1, Text: "ready"}}, threads[1].Address: {{Sequence: 2, Text: "approval"}},
	}
	plain := string(ansi.Strip([]byte(RenderMenu(state, 40, 10, now, false))))
	lines := strings.Split(plain, "\r\n")
	if len(lines) < 6 || !strings.HasSuffix(lines[2], "live") || strings.TrimSpace(lines[3]) != "ready" || !strings.HasSuffix(lines[4], "parked · 2d ago") || strings.TrimSpace(lines[5]) != "approval" {
		t.Fatalf("minimum-width semantic suffixes were clipped: %q", plain)
	}
	assertRenderedBounds(t, plain, 40, 10)
}

func TestRenderMenuAttentionChildrenAreIndentedAndDisplayOnly(t *testing.T) {
	threads := menuThreads()
	state := NewMenuState(threads, threads[0].Address)
	state.Attention = map[couchcore.ThreadAddress][]AttentionMessage{
		threads[0].Address: {{Sequence: 1, Text: "review ready"}, {Sequence: 2, Text: "tests need approval"}},
	}
	plain := string(ansi.Strip([]byte(RenderMenu(state, 80, 12, time.Time{}, false))))
	if !strings.Contains(plain, "\r\n    review ready\r\n    tests need approval") {
		t.Fatalf("notification children are not vertical and indented: %q", plain)
	}
	next, _ := reduceKey(state, PanelKey{Kind: KeyDown})
	if next.CurrentFrame().SelectedAddress != threads[1].Address {
		t.Fatalf("navigation selected a message child: %+v", next.CurrentFrame())
	}
}

func TestRenderMenuBelowMinimumOnlyRequestsResize(t *testing.T) {
	got := RenderMenu(NewMenuState(menuThreads(), menuAddress("couch-one")), 39, 9, time.Time{}, true)
	plain := string(ansi.Strip([]byte(got)))
	if plain != "resize terminal to at least 40x10" {
		t.Fatalf("small render = %q", got)
	}
}

func TestRenderMenuAnimatesProgressWithStableOneCellFrame(t *testing.T) {
	state := NewMenuState(menuThreads(), menuAddress("couch-one"))
	state.Notice = MenuNotice{Level: MenuNoticeProgress, Text: "resolving", Owner: MenuProgressOwner{PreviewGeneration: 7}}
	for phase, frame := range []string{"◐", "◓", "◑", "◒"} {
		state.SpinnerPhase = uint8(phase)
		body := string(ansi.Strip([]byte(RenderMenuView(state, 80, 23, time.Unix(100, 0), false).Body)))
		if !strings.Contains(body, frame+" resolving…") {
			t.Fatalf("phase %d body = %q, want %q", phase, body, frame+" resolving…")
		}
	}
}

func TestRenderMenuPlacesTypedBannerBelowEveryBreadcrumb(t *testing.T) {
	root := NewMenuState(menuThreads(), menuAddress("couch-one"))
	actions, _ := reduceKey(root, PanelKey{Kind: KeyTab})
	start, _ := reduceKey(actions, PanelKey{Kind: KeyCtrlSpace})

	errorState, _ := ReduceMenu(actions, MenuEvent{Kind: MenuEventInventory, Error: "store unavailable"})
	progressState, _ := reduceKey(start, PanelKey{Kind: KeyEnter})
	for _, tc := range []struct {
		name       string
		state      MenuState
		breadcrumb string
		banner     string
		control    string
	}{
		{name: "nested error", state: errorState, breadcrumb: "threads › compiler › actions", banner: "error: thread inventory unavailable: store unavailable", control: "park"},
		{name: "start progress", state: progressState, breadcrumb: "start thread", banner: "resolving", control: "▸ path"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			plain := string(ansi.Strip([]byte(RenderMenu(tc.state, 120, 40, time.Time{}, false))))
			lines := strings.Split(plain, "\r\n")
			if len(lines) < 4 || lines[0] != tc.breadcrumb || !strings.Contains(lines[1], tc.banner) || lines[2] != "" || !strings.Contains(lines[3], tc.control) {
				t.Fatalf("banner/control placement = %#v", lines)
			}
		})
	}
}

func TestRenderMenuRetainsNoticeWhileBelowMinimum(t *testing.T) {
	state := NewMenuState(menuThreads(), menuAddress("couch-one"))
	state, _ = ReduceMenu(state, MenuEvent{Kind: MenuEventInventory, Error: "offline"})
	if got := RenderMenu(state, 39, 9, time.Time{}, false); got != "resize terminal to at least 40x10" {
		t.Fatalf("below-minimum render = %q", got)
	}
	if got := RenderMenu(state, 40, 10, time.Time{}, false); !strings.Contains(got, "\r\nerror: thread inventory unavailable: off\r\n") {
		t.Fatalf("clipped notice did not reappear after resize: %q", got)
	}
}

func TestRenderMenuCursorIntentUsesFinalFieldCells(t *testing.T) {
	root := NewMenuState(menuThreads(), menuAddress("couch-one"))
	root.Frames[0].Filter = "界"
	actions, _ := reduceKey(NewMenuState(menuThreads(), menuAddress("couch-one")), PanelKey{Kind: KeyTab})
	actions.Frames[len(actions.Frames)-1].Filter = "pa"
	start, _ := reduceKey(NewMenuState(menuThreads(), menuAddress("couch-one")), PanelKey{Kind: KeyCtrlSpace})
	startUnicode := cloneMenuState(start)
	startUnicode.Frames[len(startUnicode.Frames)-1].Path = "e\u0301界"
	startBanner := cloneMenuState(start)
	startBanner.Notice = errorMenuNotice("bad path")
	startAgent := cloneMenuState(start)
	startAgent.Frames[len(startAgent.Frames)-1].FormField = MenuFieldAgent
	rename := NewMenuState(menuThreads(), menuAddress("couch-one"))
	appendMenuFrame(&rename, MenuFrame{Kind: MenuFrameText, Thread: menuAddress("couch-one"), Action: "name"})

	for _, tc := range []struct {
		name          string
		state         MenuState
		width, height int
		want          *MenuCursorIntent
	}{
		{name: "root wide glyph filter", state: root, width: 120, height: 40, want: &MenuCursorIntent{Row: 4, Col: 11}},
		{name: "action filter", state: actions, width: 120, height: 40, want: &MenuCursorIntent{Row: 4, Col: 11}},
		{name: "empty rename", state: rename, width: 120, height: 40, want: &MenuCursorIntent{Row: 3, Col: 3}},
		{name: "empty path", state: start, width: 120, height: 40, want: &MenuCursorIntent{Row: 3, Col: 9}},
		{name: "combining and wide path", state: startUnicode, width: 120, height: 40, want: &MenuCursorIntent{Row: 3, Col: 12}},
		{name: "banner shifts path", state: startBanner, width: 120, height: 40, want: &MenuCursorIntent{Row: 4, Col: 9}},
		{name: "agent focus hides", state: startAgent, width: 120, height: 40},
		{name: "resize hides", state: start, width: 39, height: 9},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := RenderMenuView(tc.state, tc.width, tc.height, time.Time{}, false).Cursor
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("cursor = %+v, want %+v", got, tc.want)
			}
		})
	}

	clipped := cloneMenuState(start)
	clipped.Frames[len(clipped.Frames)-1].Path = strings.Repeat("界", 40)
	if got := RenderMenuView(clipped, 40, 10, time.Time{}, false).Cursor; got == nil || got.Row != 3 || got.Col != 40 {
		t.Fatalf("clipped cursor = %+v, want row 3 column 40", got)
	}
}

func TestRenderMenuStartCompletionBoundsSelectedViewportAndControls(t *testing.T) {
	state := NewMenuState(menuThreads(), menuAddress("couch-one"))
	state, _ = reduceKey(state, PanelKey{Kind: KeyCtrlSpace})
	frame := &state.Frames[len(state.Frames)-1]
	frame.Path = "src/"
	frame.CompletionCandidates = []string{
		"src/d00/", "src/d01/", "src/d02/", "src/d03/", "src/d04/", "src/d05/",
		"src/d06/", "src/d07/", "src/d08/", "src/d09/", "src/\x1b[31md10/", "src/d11/",
	}
	frame.CompletionSelected = 10
	frame.CompletionTruncated = true

	view := RenderMenuView(state, 40, 10, time.Time{}, false)
	plain := string(ansi.Strip([]byte(view.Body)))
	lines := strings.Split(plain, "\r\n")
	if len(lines) != 10 || !strings.Contains(plain, "path  src/") || !strings.Contains(plain, "agent ") || !strings.Contains(plain, "more matching directories") {
		t.Fatalf("bounded completion controls = %q", plain)
	}
	if !strings.Contains(plain, "src/d10/") || strings.Contains(plain, "\x1b") || strings.Contains(plain, "src/d00/") {
		t.Fatalf("selected sanitized viewport = %q", plain)
	}
	if view.Cursor == nil || view.Cursor.Row != 3 {
		t.Fatalf("path cursor = %+v, want row 3", view.Cursor)
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
