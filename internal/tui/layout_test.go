package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

// The frame must always fill exactly width x height with no line overflowing,
// even when the body content is far larger than the box. This is the regression
// guard for the old overflow/border-break bug.
func TestFrameFitsTerminalNoOverflow(t *testing.T) {
	st := State{LoggedIn: true, FavCount: 12, Building: "Example Tower", NotifyCh: "lark"}
	huge := strings.Repeat("a very long body line that should be clamped to the box width\n", 200)

	for _, sz := range [][2]int{{80, 24}, {100, 40}, {120, 30}, {40, 14}} {
		w, h := sz[0], sz[1]
		out := frame(w, h, st, huge, footStyle.Render("esc back"))
		lines := strings.Split(out, "\n")
		if len(lines) != h {
			t.Fatalf("%dx%d: got %d lines, want %d", w, h, len(lines), h)
		}
		for i, ln := range lines {
			if lipgloss.Width(ln) > w {
				t.Fatalf("%dx%d: line %d width %d > %d", w, h, i, lipgloss.Width(ln), w)
			}
		}
	}
}

func TestBodySizeLeavesRoomForChrome(t *testing.T) {
	_, ih := inner(80, 24)
	bw, bh := bodySize(80, 24)
	if bh != ih-headerLines-footerLines {
		t.Fatalf("body height %d, want %d", bh, ih-headerLines-footerLines)
	}
	if bw <= 0 || bh <= 0 {
		t.Fatalf("non-positive body size %dx%d", bw, bh)
	}
}

func TestClampHeightCaps(t *testing.T) {
	if got := clampHeight("a\nb\nc\nd", 2); got != "a\nb" {
		t.Fatalf("clampHeight = %q, want %q", got, "a\nb")
	}
}
