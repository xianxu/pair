package couchtty

import (
	"reflect"
	"strings"
	"testing"
)

// Drive the real input owner in both focus contexts. A populated paging state
// makes the old direct jump observable instead of coincidentally staying put.
func TestCtrlReturnTogglesAttentionWithoutNavigation(t *testing.T) {
	for _, key := range []string{"\x1b[13;5u", "\x1b[13;5:1u", "\x1b[13;5:2u"} {
		for _, panel := range []bool{false, true} {
			name := strings.ReplaceAll(key, "\x1b", "ESC")
			if panel {
				name += "/switcher"
			} else {
				name += "/actor"
			}
			t.Run(name, func(t *testing.T) {
				f, threads := pagingFixture(t)
				f.con.switchTo("c2", true, arrivalOrdinary)
				f.con.switchTo("c1", true, arrivalOrdinary)
				page(f, threads[2], "newest page")
				f.con.ObserveAttention(hubNotify(threads[2], "check this"))
				if panel {
					_, _ = f.stdin.Write([]byte("\x00"))
					waitFor(t, "switcher", func() bool { return f.con.menuSnapshot().CurrentFrame().SelectedAddress == threads[2] })
				}
				before := landingOf(f, threads)
				menu := f.con.menuSnapshot().CurrentFrame().SelectedAddress
				_, _ = f.stdin.Write([]byte(key))
				waitFor(t, "attention toggle or old jump", func() bool { return f.attentionOpen() || activeOf(f) != "c1" })
				if !f.attentionOpen() {
					t.Fatal("Ctrl+Return navigated instead of opening attention")
				}
				if got := landingOf(f, threads); !reflect.DeepEqual(got, before) {
					t.Fatalf("opening changed navigation/acknowledgement: got %+v want %+v", got, before)
				}
				_, _ = f.stdin.Write([]byte(key))
				waitFor(t, "second press closes", func() bool { return !f.attentionOpen() })
				if got := landingOf(f, threads); !reflect.DeepEqual(got, before) {
					t.Fatalf("closing changed navigation: %+v", got)
				}
				if got := f.con.menuSnapshot().CurrentFrame().SelectedAddress; got != menu {
					t.Fatalf("switcher selection changed: %v -> %v", menu, got)
				}
				if f.attentionLen() != 1 {
					t.Fatal("toggle consumed the attention item")
				}
				_, _ = f.stdin.Write([]byte(ctrlAltA))
				waitFor(t, "existing shortcut opens", f.attentionOpen)
				_, _ = f.stdin.Write([]byte(key))
				waitFor(t, "Ctrl+Return closes the same overlay", func() bool { return !f.attentionOpen() })
				// An ordinary switcher Enter still selects its paging row.
				if panel {
					_, _ = f.stdin.Write([]byte("\r"))
					waitFor(t, "ordinary switcher Return", func() bool { return activeOf(f) == "c3" })
				} else {
					for _, previous := range []string{"\x08", "\x1b[127;5u"} {
						_, _ = f.stdin.Write([]byte(previous))
						waitFor(t, "previous still returns", func() bool { return activeOf(f) == "c2" })
						f.con.switchTo("c1", true, arrivalOrdinary)
					}
				}
			})
		}
	}
}

func TestCtrlReturnEmptyAttentionUsesBellNotice(t *testing.T) {
	f, _ := pagingFixture(t)
	_, _ = f.stdin.Write([]byte("\x1b[13;5u"))
	waitFor(t, "notice", func() bool { return noticeOf(f) != "" })
	if got := noticeOf(f); !strings.Contains(got, "Nothing is waiting on you") {
		t.Fatalf("notice = %q", got)
	}
	if activeOf(f) != "c1" || f.attentionOpen() {
		t.Fatal("empty attention changed the active thread or opened")
	}
}

func TestCtrlReturnItemSelectionPreservesPreviousBothEncodings(t *testing.T) {
	for _, previous := range []string{"\x08", "\x1b[127;5u"} {
		t.Run(strings.ReplaceAll(previous, "\x1b", "ESC"), func(t *testing.T) {
			f, threads := pagingFixture(t)
			f.con.ObserveAttention(hubNotify(threads[2], "select me"))
			_, _ = f.stdin.Write([]byte("\x1b[13;5u"))
			waitFor(t, "attention", f.attentionOpen)
			_, _ = f.stdin.Write([]byte("\r"))
			waitFor(t, "item selected", func() bool { return activeOf(f) == "c3" })
			_, _ = f.stdin.Write([]byte(previous))
			waitFor(t, "previous goes home", func() bool { return activeOf(f) == "c1" })
		})
	}
}
