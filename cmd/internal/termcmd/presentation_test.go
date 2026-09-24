package termcmd

import (
	"bytes"
	"context"
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"
	"time"

	uv "github.com/charmbracelet/ultraviolet"
	vt "github.com/charmbracelet/x/vt"
	"github.com/xianxu/pair/cmd/internal/hostty"
	"github.com/xianxu/pair/cmd/internal/ptychild"
	"github.com/xianxu/pair/cmd/internal/terminal"
	"github.com/xianxu/pair/cmd/internal/ttyio"
)

func presentationFixture(t *testing.T) (*terminalMux, *ttyio.Fake) {
	t.Helper()
	parent := ttyio.NewFake()
	m := newTerminalMux("sh", nil, parent, &fakeRuntime{})
	m.rows, m.cols = 5, 8
	t.Cleanup(m.closeAll)
	return m, parent
}
func addPresentationTab(t *testing.T, m *terminalMux, id int, seed string) *ptychild.Child {
	t.Helper()
	child := ptychild.NewFakeChild(nil)
	if err := child.Resize(m.childSizeLocked()); err != nil {
		t.Fatal(err)
	}
	child.Feed([]byte(seed))
	if err := m.admitTab(&terminalTab{id: id, name: "tab", child: child}); err != nil {
		t.Fatal(err)
	}
	child.SetSink(func(ctx context.Context, b ptychild.OutputBatch) error { return m.handleOutput(ctx, id, b) })
	return child
}
func flushPresentation(t *testing.T, m *terminalMux, c *ptychild.Child) {
	t.Helper()
	if err := c.FlushOutput(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := m.presenter.Flush(context.Background()); err != nil {
		t.Fatal(err)
	}
}
func TestPresentationHiddenStateAndUTF8StayIsolated(t *testing.T) {
	m, parent := presentationFixture(t)
	a := addPresentationTab(t, m, 1, "alpha")
	b := addPresentationTab(t, m, 2, "beta")
	before := len(parent.Bytes())
	a.Feed([]byte("\x1b[1;1Hhidden"))
	flushPresentation(t, m, a)
	if len(parent.Bytes()) != before {
		t.Fatal("hidden output painted parent")
	}
	m.previousTab()
	if m.presenter.View().Admitted != a.Endpoint().ID() {
		t.Fatal("selected endpoint mismatch")
	}
	a.Feed([]byte{0xe7})
	flushPresentation(t, m, a)
	m.beginRename()
	a.Feed([]byte{0x95, 0x8c})
	flushPresentation(t, m, a)
	f, err := a.Endpoint().Snapshot(time.Now())
	if err != nil {
		t.Fatal(err)
	}
	var text strings.Builder
	for _, c := range f.Cells {
		text.WriteString(c.Content)
	}
	if !strings.Contains(text.String(), "界") || strings.Contains(string(parent.Bytes()), "�") {
		t.Fatalf("split UTF-8 leaked through chrome:%q", text.String())
	}
	_ = b
}
func TestPresentationSelectionCompletesBeforeInput(t *testing.T) {
	m, parent := presentationFixture(t)
	a := addPresentationTab(t, m, 1, "a")
	b := addPresentationTab(t, m, 2, "b")
	block := make(chan struct{})
	parent.Enqueue(ttyio.WriteStep{Block: block})
	for {
		select {
		case <-parent.Started():
			continue
		default:
			goto drained
		}
	}
drained:
	switched := make(chan struct{})
	go func() { m.previousTab(); close(switched) }()
	<-parent.Started()
	sent := make(chan struct{})
	go func() { m.writeEvent(terminal.InputEvent{Event: uv.KeyPressEvent{Code: 'x', Text: "x"}}); close(sent) }()
	select {
	case <-sent:
		t.Fatal("input bypassed pending selection")
	case <-time.After(20 * time.Millisecond):
	}
	close(block)
	<-switched
	<-sent
	a.Endpoint().Flush(context.Background())
	b.Endpoint().Flush(context.Background())
	if got := string(bytes.Join(a.Writes(), nil)); got != "x" {
		t.Fatalf("new destination=%q", got)
	}
	if len(b.Writes()) != 0 {
		t.Fatal("old child received pending input")
	}
}
func TestPresentationFailedSwitchKeepsProductSelection(t *testing.T) {
	m, parent := presentationFixture(t)
	addPresentationTab(t, m, 1, "a")
	addPresentationTab(t, m, 2, "b")
	parent.Enqueue(ttyio.WriteStep{Limit: 3, Err: errors.New("parent failed")})
	m.previousTab()
	if m.active != 1 || m.presenter.View().Admitted != "" {
		t.Fatalf("failed landing committed product target:%d %+v", m.active, m.presenter.View())
	}
}

func renderedParent(t *testing.T, parent *ttyio.Fake, cols, rows int) string {
	t.Helper()
	e := vt.NewEmulator(cols, rows)
	defer e.Close()
	e.Write(parent.Bytes())
	return e.String()
}
func TestPresentationRenameAndRemovalRetainProductIdentity(t *testing.T) {
	m, parent := presentationFixture(t)
	m.cols = 40
	a := addPresentationTab(t, m, 1, "one")
	b := addPresentationTab(t, m, 2, "two")
	m.tabs[0].name = "one"
	m.tabs[1].name = "two"
	id, editor, err := m.beginRename()
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range "xyz" {
		editor, _ = editor.Apply(RenameEvent{Kind: RenameInsert, Rune: r})
		m.refreshRename(id, editor)
	}
	if len(m.rt.(*fakeRuntime).ops) != 0 {
		t.Fatal("rename keystroke invoked zellij")
	}
	m.removeTab(1)
	if m.activeTabLocked().id != 2 || m.rename.tabID != 2 {
		t.Fatal("background exit moved rename/selection")
	}
	if got := renderedParent(t, parent, 40, 5); !strings.Contains(got, "[rename: twoxyz│]") {
		t.Fatalf("rename strip stale:%q", got)
	}
	if err := m.finishRename(id, RenameOutcome{Kind: RenameOutcomeCommit, Name: editor.Text()}); err != nil {
		t.Fatal(err)
	}
	if b.Endpoint().ID() != m.presenter.View().Admitted || m.tabs[0].name != "twoxyz" {
		t.Fatal("rename changed endpoint identity")
	}
	_ = a
}
func TestPresentationRemovedRenameTargetCannotRenameSuccessor(t *testing.T) {
	m, _ := presentationFixture(t)
	m.cols = 40
	addPresentationTab(t, m, 1, "one")
	addPresentationTab(t, m, 2, "two")
	m.tabs[0].name = "one"
	m.tabs[1].name = "two"
	id, _, err := m.beginRename()
	if err != nil {
		t.Fatal(err)
	}
	m.removeTab(id)
	if err := m.finishRename(id, RenameOutcome{Kind: RenameOutcomeCommit, Name: "wrong"}); err != nil {
		t.Fatal(err)
	}
	if len(m.tabs) != 1 || m.tabs[0].name != "one" {
		t.Fatal("rename targeted replacement tab")
	}
}
func TestPresentationSwitchRestoresCellsModesAndNeverNudges(t *testing.T) {
	m, parent := presentationFixture(t)
	a := addPresentationTab(t, m, 1, "shell")
	b := addPresentationTab(t, m, 2, "\x1b[?1049h\x1b[?1002h\x1b[?1006happ")
	beforeA, beforeB := len(a.Resizes()), len(b.Resizes())
	for i := 0; i < 3; i++ {
		m.previousTab()
		if m.appMouseMode() || m.activeChildOwnsScreen() {
			t.Fatal("shell inherited app modes")
		}
		m.nextTab()
		if !m.appMouseMode() || !m.activeChildOwnsScreen() {
			t.Fatal("app modes lost")
		}
	}
	if len(a.Resizes()) != beforeA || len(b.Resizes()) != beforeB {
		t.Fatal("tab selection nudged PTY dimensions")
	}
	if text := renderedParent(t, parent, 8, 5); !strings.Contains(text, "app") {
		t.Fatalf("app frame missing:%q", text)
	}
	m.previousTab()
	if !strings.Contains(string(parent.Bytes()), "\x1b[?1002l") {
		t.Fatal("shell did not release parent mouse capture")
	}
}
func TestPresentationQueriesNeverReachParentAndRepliesStayAtOrigin(t *testing.T) {
	m, parent := presentationFixture(t)
	a := addPresentationTab(t, m, 1, "")
	addPresentationTab(t, m, 2, "visible")
	before := len(parent.Bytes())
	a.Feed([]byte("\x1b[6n\x1b[c"))
	flushPresentation(t, m, a)
	a.Endpoint().Flush(context.Background())
	if len(parent.Bytes()) != before {
		t.Fatal("hidden query reached parent")
	}
	if got := string(bytes.Join(a.Writes(), nil)); got != "\x1b[1;1R\x1b[?1;2c" {
		t.Fatalf("query replies:%q", got)
	}
}
func TestPresentationDiagnosticIsBoundedTypedChrome(t *testing.T) {
	m, parent := presentationFixture(t)
	m.cols = 40
	addPresentationTab(t, m, 1, "child")
	m.reportError(errors.New("bad\x1b[2J" + strings.Repeat("x", 1000)))
	if strings.Contains(m.notice, "\x1b") || len(m.notice) > diagnosticWidth+3 {
		t.Fatal("unbounded/control diagnostic")
	}
	text := renderedParent(t, parent, 40, 5)
	if !strings.Contains(text, "child") || !strings.Contains(text, "pair term:") {
		t.Fatalf("diagnostic destroyed child:%q", text)
	}
}

func TestPresentationMuxHasNoPhysicalWriterOrReplayAuthority(t *testing.T) {
	typ := reflect.TypeOf(terminalMux{})
	writer := reflect.TypeOf((*io.Writer)(nil)).Elem()
	contextWriter := reflect.TypeOf((*ttyio.Writer)(nil)).Elem()
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		if f.Type.Implements(writer) || f.Type.Implements(contextWriter) {
			t.Fatalf("mux owns physical writer %s", f.Name)
		}
	}
	if _, ok := typ.FieldByName("hostScan"); ok {
		t.Fatal("duplicate parent parser survives")
	}
}

func TestPresentationResizeCrossesSingleRowBoundary(t *testing.T) {
	m, _ := presentationFixture(t)
	child := addPresentationTab(t, m, 1, "hello")
	host := hostty.NewFakeHost(ptychild.Size{Rows: 1, Cols: 8})
	for _, rows := range []uint16{1, 5, 1, 5} {
		host.SetSize(ptychild.Size{Rows: rows, Cols: 8})
		m.inheritSize(host)
		if m.failure != nil {
			t.Fatal(m.failure)
		}
		frame, err := child.Endpoint().Snapshot(time.Now())
		if err != nil {
			t.Fatal(err)
		}
		want := int(rows)
		if want > 1 {
			want--
		}
		if frame.Geometry.Rows != want {
			t.Fatalf("host rows %d: child rows %d, want %d", rows, frame.Geometry.Rows, want)
		}
		if m.rows != rows {
			t.Fatalf("product geometry did not commit: %d", m.rows)
		}
	}
}

func TestPresentationRealChildFinalOutputAndEnvironment(t *testing.T) {
	m, parent := presentationFixture(t)
	m.cols = 80
	m.shellName = "/bin/sh"
	m.shellArgs = []string{"-c", `printf '%s:%s' "$TERM" "$TERMINFO"`}
	m.shellEnv = []string{"TERM=pair-vt-256color", "TERMINFO=/owned/profile"}
	if err := m.newTab(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-m.done:
	case <-time.After(5 * time.Second):
		t.Fatal("child exit publication did not drain")
	}
	m.mu.Lock()
	failure := m.failure
	m.mu.Unlock()
	if failure != nil {
		t.Fatal(failure)
	}
	got := renderedParent(t, parent, 80, 5)
	if !strings.Contains(got, "pair-vt-256color:/owned/profile") {
		t.Fatalf("final child frame lost: %q", got)
	}
	m.closeAll()
	if m.closeErr != nil {
		t.Fatal(m.closeErr)
	}
}

type presentationSessionHost struct {
	*hostty.FakeHost
	readDone chan struct{}
}

func (h *presentationSessionHost) Read(p []byte) (int, error) {
	return h.ReadContext(context.Background(), p)
}
func (h *presentationSessionHost) ReadContext(ctx context.Context, p []byte) (int, error) {
	defer close(h.readDone)
	<-ctx.Done()
	return 0, ctx.Err()
}

type presentationSessionRuntime struct{ fakeRuntime }

func (*presentationSessionRuntime) ShellCommand() (string, []string) {
	return "/bin/sh", []string{"-c", "printf final-session-frame"}
}
func TestPresentationSessionJoinsInputBeforeHostCleanup(t *testing.T) {
	host := &presentationSessionHost{FakeHost: hostty.NewFakeHost(ptychild.Size{Rows: 5, Cols: 80}), readDone: make(chan struct{})}
	done := make(chan error, 1)
	go func() { done <- runShellOnHost(host, &presentationSessionRuntime{}) }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("session failed to join")
	}
	select {
	case <-host.readDone:
	default:
		t.Fatal("input reader outlived session")
	}
	if !host.Closed() || host.RawDepth() != 0 {
		t.Fatalf("host ownership leaked closed=%v raw=%d", host.Closed(), host.RawDepth())
	}
	e := vt.NewEmulator(80, 5)
	defer e.Close()
	e.Write([]byte(host.Written()))
	if !strings.Contains(e.String(), "final-session-frame") {
		t.Fatalf("exit lost final output: %q", e.String())
	}
}

func TestPresentationHiddenPhysicalEffectsStaySuppressedAfterSelection(t *testing.T) {
	m, parent := presentationFixture(t)
	a := addPresentationTab(t, m, 1, "a")
	addPresentationTab(t, m, 2, "b")
	before := len(parent.Bytes())
	a.Feed([]byte("\a\x1b]2;HIDDEN-TITLE\a\x1b]52;c;SGVsbG8=\a"))
	flushPresentation(t, m, a)
	if len(parent.Bytes()) != before {
		t.Fatalf("hidden physical effects escaped: %q", parent.Bytes()[before:])
	}
	m.previousTab()
	if bytes.Contains(parent.Bytes(), []byte("HIDDEN-TITLE")) || bytes.Contains(parent.Bytes(), []byte("SGVsbG8=")) {
		t.Fatal("suppressed effects replayed on select")
	}
}

func TestPresentationCloseActiveRetiresBeforeEndpointDisposal(t *testing.T) {
	m, _ := presentationFixture(t)
	a := addPresentationTab(t, m, 1, "survivor")
	b := addPresentationTab(t, m, 2, "\x1b[?1002;1006hclosing")
	m.writeEvent(terminal.InputEvent{Event: uv.MouseClickEvent{X: 1, Y: 1, Button: uv.MouseLeft}})
	b.Feed([]byte("\x1b[1;1Hpending"))
	if err := b.FlushOutput(context.Background()); err != nil {
		t.Fatal(err)
	}
	m.closeActive()
	if m.failure != nil {
		t.Fatal(m.failure)
	}
	if m.presenter.View().Admitted != a.Endpoint().ID() || m.activeTabLocked().child != a {
		t.Fatal("closed tab was not replaced before disposal")
	}
	if !b.Done() {
		t.Fatal("retired child was not disposed")
	}
	if got := string(bytes.Join(b.Writes(), nil)); got != "\x1b[<0;2;2M\x1b[<0;2;2m" {
		t.Fatalf("drag cancellation=%q", got)
	}
	if err := m.presenter.Flush(context.Background()); err != nil {
		t.Fatalf("closed endpoint remained dirty: %v", err)
	}
	m.writeEvent(terminal.InputEvent{Event: uv.MouseReleaseEvent{X: 2, Y: 1, Button: uv.MouseLeft}})
	m.writeEvent(terminal.InputEvent{Event: uv.KeyPressEvent{Code: 'x', Text: "x"}})
	if err := a.Endpoint().Flush(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := string(bytes.Join(a.Writes(), nil)); got != "x" {
		t.Fatalf("survivor input=%q", got)
	}
	m.closeActive()
	if a.Done() || m.presenter.View().Admitted != a.Endpoint().ID() {
		t.Fatal("close last tab changed existing product policy")
	}
	m.nextTab()
	if m.failure != nil {
		t.Fatalf("subsequent switch failed: %v", m.failure)
	}
}

// #311: the strip is clickable in a plain shell tab, so the parent asks for
// mouse reports itself -- couch's any-motion policy -- whatever the child holds.
func TestPresentationParentRequestsMouseForAPlainChild(t *testing.T) {
	m, parent := presentationFixture(t)
	addPresentationTab(t, m, 1, "")
	if err := m.presenter.Flush(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(parent.Bytes()), "\x1b[?1003h") {
		t.Fatalf("parent did not request mouse reports for a plain child: %q", parent.Bytes())
	}
}

func TestPresentationClickingStripChipSelectsThatTab(t *testing.T) {
	m, _ := presentationFixture(t)
	m.rows, m.cols = 5, 40
	a := addPresentationTab(t, m, 1, "")
	addPresentationTab(t, m, 2, "")
	c := addPresentationTab(t, m, 3, "")
	strip := 4 // bottom row; body "tab tab [tab]"

	if m.clickStrip(1, 2) {
		t.Fatal("a click on the child's rows was consumed by the strip")
	}
	if !m.clickStrip(3, strip) || m.active != 2 {
		t.Fatalf("separator click: active=%d, want unchanged 2", m.active)
	}
	if !m.clickStrip(10, strip) || m.active != 2 || m.rename != nil {
		t.Fatalf("active-chip click: active=%d rename=%v, want harmless", m.active, m.rename)
	}
	if !m.clickStrip(30, strip) || m.active != 2 {
		t.Fatalf("empty-space click: active=%d, want unchanged 2", m.active)
	}
	if !m.clickStrip(1, strip) || m.active != 0 {
		t.Fatalf("first-chip click: active=%d, want 0", m.active)
	}
	if m.presenter.View().Admitted != a.Endpoint().ID() {
		t.Fatal("presenter did not select the clicked tab's endpoint")
	}
	if !m.clickStrip(11, strip) || m.presenter.View().Admitted != c.Endpoint().ID() {
		t.Fatal("clicking the third chip did not select it")
	}
}
