package tui

import (
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Fixed chrome heights inside the frame: status header and key-hint footer.
const (
	headerLines = 3 // brand+status, meta, rule
	footerLines = 2 // rule, hints
)

// frameSize is the border+padding the doc frame adds (horizontal, vertical).
func frameSize() (int, int) { return docStyle.GetFrameSize() }

// inner returns the content area inside the frame for a given terminal size.
func inner(width, height int) (int, int) {
	fw, fh := frameSize()
	w := width - fw
	h := height - fh
	if w < 20 {
		w = 20
	}
	if h < 8 {
		h = 8
	}
	return w, h
}

// bodySize returns the area a screen body may draw into: full inner width, and
// inner height minus the fixed header and footer.
func bodySize(width, height int) (int, int) {
	w, h := inner(width, height)
	bh := h - headerLines - footerLines
	if bh < 3 {
		bh = 3
	}
	return w, bh
}

// clampHeight cuts s to at most h lines so a body can never push past its box.
func clampHeight(s string, h int) string {
	if h < 1 {
		h = 1
	}
	lines := strings.Split(s, "\n")
	if len(lines) > h {
		lines = lines[:h]
	}
	return strings.Join(lines, "\n")
}

// fit clamps s to exactly w x h, top-left aligned: lines past h are dropped,
// lines wider than w are truncated, then it is padded to the full slot. This is
// what guarantees a body can never overflow its box.
func fit(s string, w, h int) string {
	lines := strings.Split(clampHeight(s, h), "\n")
	for i, ln := range lines {
		lines[i] = truncateLine(ln, w)
	}
	return lipgloss.Place(w, h, lipgloss.Left, lipgloss.Top, strings.Join(lines, "\n"))
}

// header renders the global status chrome: brand + live status badges, a dim
// meta line, and a rule — always exactly headerLines tall.
func header(w int, st State) string {
	brand := appTitleStyle.Render("nybble") + brandStyle.Render("  ·  canteen lunch autobooker")

	var status string
	switch {
	case st.Loading:
		status = metaStyle.Render(openDot + " checking session…")
	case st.LoggedIn:
		status = okStyle.Render(statusDot + " logged in")
	default:
		status = errStyle.Render(statusDot + " logged out")
	}

	top := spread(brand, status, w)

	meta := metaStyle.Render(strings.Join([]string{
		plural(st.FavCount, "favorite"),
		orDash(st.Building),
		"notify " + orDash(st.NotifyCh),
	}, "   "+dot+"   "))

	return lipgloss.JoinVertical(lipgloss.Left,
		truncateLine(top, w),
		truncateLine(meta, w),
		rule(w),
	)
}

// footer renders a rule and the screen's (already-styled) key hints — always
// footerLines tall. Children style their own hints so emphasis survives.
func footer(w int, hints string) string {
	return lipgloss.JoinVertical(lipgloss.Left,
		rule(w),
		truncateLine(hints, w),
	)
}

// frame composes header, body and footer into the bordered full-screen doc.
// body is fitted to exactly the body area, so it can never overflow the frame.
func frame(width, height int, st State, body, hints string) string {
	iw, _ := inner(width, height)
	bw, bh := bodySize(width, height)
	content := lipgloss.JoinVertical(lipgloss.Left,
		header(iw, st),
		fit(body, bw, bh),
		footer(iw, hints),
	)
	// content is already sized to the inner area (iw x ih); the border+padding
	// add exactly frameSize() back, hitting width x height. Setting Width/Height
	// here would re-wrap (they include padding), so we deliberately do not.
	return docStyle.Render(content)
}

// spread places left and right on one line w wide, right-aligned, with a gap.
func spread(left, right string, w int) string {
	lw, rw := lipgloss.Width(left), lipgloss.Width(right)
	gap := w - lw - rw
	if gap < 1 {
		return truncateLine(left+" "+right, w)
	}
	return left + strings.Repeat(" ", gap) + right
}

// truncateLine clamps a single rendered line to w columns, preserving styling by
// only trimming when it overflows the plain width.
func truncateLine(s string, w int) string {
	if lipgloss.Width(s) <= w {
		return s
	}
	return truncate(s, w)
}

func plural(n int, word string) string {
	if n == 1 {
		return "1 " + word
	}
	return strconv.Itoa(n) + " " + word + "s"
}

func orDash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}
