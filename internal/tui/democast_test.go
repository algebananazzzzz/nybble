package tui

// Demo-cast generator. This is the "protocol" that produces docs/demo.cast (and,
// via `agg`, docs/demo.gif): it drives the real screen models with mock data and
// captures each rendered frame into an asciinema v2 cast.
//
// It runs ONLY when NYBBLE_GEN_DEMO=1 so a normal `go test ./...` skips it. The
// frames are synthetic full redraws (clear + home + View) rather than a live PTY
// recording, because the interesting screens (Favorites, Schedule) are auth-gated
// and can't be reached with a mock session — here we just set State.LoggedIn and
// render them directly. No real building, vendor, or org name appears: every value
// is the mock data written below.
//
//	NYBBLE_GEN_DEMO=1 go test ./internal/tui -run TestGenerateDemoCast
//	make demo   # renders the cast to docs/demo.gif

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/algebananazzzzz/nybble/internal/config"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

const (
	demoW = 100
	demoH = 30
)

// demoState is the header snapshot every frame is rendered against — logged in so
// the gated screens read coherently, with purely fictional mock data.
var demoState = State{
	LoggedIn: true,
	FavCount: 10,
	Building: "Example Tower",
	NotifyCh: "lark",
}

// castEvent is one asciinema "o" (output) event: [time, "o", data].
type castEvent struct {
	t    float64
	data string
}

func TestGenerateDemoCast(t *testing.T) {
	if os.Getenv("NYBBLE_GEN_DEMO") != "1" {
		t.Skip("set NYBBLE_GEN_DEMO=1 to regenerate docs/demo.cast")
	}

	// Force full 256-color output; a test has no TTY, so termenv would otherwise
	// strip the ANSI styling the gif relies on.
	lipgloss.SetColorProfile(termenv.ANSI256)

	// Stamp the About screen's version. main() (which GoReleaser ldflags-stamps)
	// never runs under `go test`, so Version would otherwise show "dev"; derive the
	// latest tag from git instead.
	Version = gitVersion()

	cfgDir := writeMockConfig(t)
	t.Setenv("XDG_CONFIG_HOME", cfgDir)
	t.Setenv("NYBBLE_API_BASE", "https://canteen.example.com/api/order")
	t.Setenv("NYBBLE_LOGIN_URL", "https://canteen.example.com/sso/login")

	bw, bh := bodySize(demoW, demoH)

	var ev []castEvent
	at := 0.0
	add := func(dt float64, body, footer string) {
		at += dt
		ev = append(ev, castEvent{at, screenData(body, footer)})
	}

	// 1) Dashboard — walk the cursor down through the concern panels.
	d := newDashboard()
	add(0.0, d.View(bw, bh), d.Footer()) // Booking · Favorites & menu
	d = key(d, tea.KeyDown)
	add(1.0, d.View(bw, bh), d.Footer()) // Booking · Schedule
	d = key(d, tea.KeyDown)
	d = key(d, tea.KeyDown)
	add(1.0, d.View(bw, bh), d.Footer()) // Settings · Re-authenticate

	// 2) Favorites & menu — the vendor fallback ranking, then the dish ranking.
	f := newFav(bw, bh)
	add(1.3, f.View(bw, bh), f.Footer()) // Vendors
	f = key(f, tea.KeyTab)
	add(1.2, f.View(bw, bh), f.Footer()) // Dishes, top of list
	f = key(f, tea.KeyDown)
	add(0.7, f.View(bw, bh), f.Footer())
	f = key(f, tea.KeyDown)
	add(0.6, f.View(bw, bh), f.Footer())
	f = key(f, tea.KeyDown)
	add(0.6, f.View(bw, bh), f.Footer())

	// 3) Schedule — the timing form (run day, hour, book days, lead, enable).
	s := newSched(bw, bh)
	add(1.3, s.View(bw, bh), s.Footer())
	s = key(s, tea.KeyDown) // move off Run day onto the hour field
	add(1.0, s.View(bw, bh), s.Footer())

	// 4) About — endpoints, live status and on-disk paths.
	a := newAbout(demoState)
	add(1.3, a.View(bw, bh), a.Footer())

	// 5) Back home.
	home := newDashboard()
	add(1.4, home.View(bw, bh), home.Footer())
	add(1.5, home.View(bw, bh), home.Footer()) // hold the last frame

	writeCast(t, ev)
}

// screenData wraps a body+footer in the full frame and turns it into the bytes an
// asciinema "o" event carries: clear screen, home cursor, then the frame with CRLF
// line endings so each row starts at column 0 in the replayed terminal.
func screenData(body, footer string) string {
	out := frame(demoW, demoH, demoState, body, footer)
	out = strings.ReplaceAll(out, "\n", "\r\n")
	return "\x1b[2J\x1b[H" + out
}

// key feeds one keypress to a screen model and returns it, dropping the Cmd — the
// demo never needs the async side effects (nav, network), only the new view.
func key[T screenModel](m T, k tea.KeyType) T {
	next, _ := m.Update(tea.KeyMsg{Type: k})
	return next.(T)
}

func newFav(bw, bh int) *favModel {
	f := newFavModel()
	next, _ := f.Update(bodySizeMsg{bw, bh})
	return next.(*favModel)
}

func newSched(bw, bh int) *scheduleScreen {
	s, _ := newSchedule()
	next, _ := s.Update(bodySizeMsg{bw, bh})
	return next.(*scheduleScreen)
}

// writeMockConfig lays down a throwaway config dir of fictional data and returns
// the XDG_CONFIG_HOME root that points at it.
func writeMockConfig(t *testing.T) string {
	t.Helper()
	// A fixed, tidy path so the About screen's "Config" line reads cleanly in the
	// gif (a t.TempDir() would print an ugly /var/folders/… hash). Throwaway.
	root := "/tmp/nybble-demo"
	dir := filepath.Join(root, "nybble")
	if err := os.RemoveAll(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(root) })
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}

	cfg := config.Config{
		Building: config.NamedCode{Code: "EXT", Name: "Example Tower"},
		Pickup:   config.Pickup{Code: 1, Name: "Level 3 Pantry"},
		MealType: "lunch",
		BookDays: config.Weekdays,
		Schedule: config.Schedule{Weekday: "thu", Hour: 10, Minute: 0, TZ: "Asia/Singapore", LeadMin: 5},
		Notify:   config.Notify{Channel: "lark", LarkTarget: "you@example.com"},
	}
	if err := config.Save(filepath.Join(dir, "config.json"), cfg); err != nil {
		t.Fatal(err)
	}

	dishes := config.Favorites{
		"Teriyaki Chicken Bowl",
		"Green Curry with Rice",
		"Caesar Salad",
		"Margherita Pizza",
		"Beef Pho",
		"Mushroom Risotto",
		"Falafel Wrap",
		"Mapo Tofu",
		"Pad Thai Noodles",
		"Grilled Salmon Plate",
	}
	vendors := config.Favorites{
		"Wok Express",
		"Green Bowl",
		"Pasta Bar",
		"Taco Stand",
		"Sushi Counter",
	}
	must(t, config.SaveFavorites(filepath.Join(dir, "favorites.json"), dishes))
	must(t, config.SaveFavorites(filepath.Join(dir, "catalog.json"), dishes))
	must(t, config.SaveFavorites(filepath.Join(dir, "vendors.json"), vendors))

	return root
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

// writeCast serializes the events as an asciinema v2 cast at docs/demo.cast,
// resolved relative to the repo root (two dirs up from internal/tui).
func writeCast(t *testing.T, ev []castEvent) {
	t.Helper()
	var b strings.Builder
	b.WriteString(`{"version": 2, "width": 100, "height": 30, "timestamp": 0, "env": {"TERM": "xterm-256color"}}` + "\n")
	for _, e := range ev {
		data, err := json.Marshal(e.data)
		if err != nil {
			t.Fatal(err)
		}
		// json.Marshal escapes the data string; assemble the [t, "o", data] line.
		b.WriteString("[" + trimFloat(e.t) + `, "o", ` + string(data) + "]\n")
	}

	out := filepath.Join("..", "..", "docs", "demo.cast")
	if err := os.WriteFile(out, []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Logf("wrote %d frames to %s", len(ev), out)
}

// trimFloat formats a timestamp with one decimal place (matching the original cast).
func trimFloat(f float64) string {
	return strconv.FormatFloat(f, 'f', 1, 64)
}

// gitVersion returns the latest release tag (e.g. "v0.1.1") so the demo's About
// screen matches what a released build stamps, falling back to "dev".
func gitVersion() string {
	out, err := exec.Command("git", "describe", "--tags", "--abbrev=0").Output()
	if err != nil {
		return "dev"
	}
	if v := strings.TrimSpace(string(out)); v != "" {
		return v
	}
	return "dev"
}
