package tui

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/algebananazzzzz/bytecanteen/internal/config"
	"github.com/algebananazzzzz/bytecanteen/internal/notify"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
)

// settingsPhase drives the Settings flow as a small state machine:
//
//	probing – async-checking whether Lark can be used (gates the Notify channel)
//	form    – the huh form (mode/day/hour/notify[/lark target])
//	done    – saved (or save failed)
type settingsPhase int

const (
	phaseProbing settingsPhase = iota
	phaseForm
	phaseDone
)

// settings embeds a huh.Form as a sub-model. It NEVER calls form.Run() (which
// would start a second Bubble Tea program); it drives form.Update/View itself.
type settings struct {
	phase settingsPhase
	form  *huh.Form
	cfg   config.Config
	hour  string
	lark  notify.LarkStatus // result of the async probe; gates the Lark channel

	width, height int
	saveErr       error
}

// larkProbeMsg carries the async probe result through the Bubble Tea loop.
type larkProbeMsg notify.LarkStatus

func newSettings() (*settings, tea.Cmd) {
	dir, _ := config.ConfigDir()
	cfg, err := config.Load(filepath.Join(dir, "config.json"))
	if err != nil {
		cfg = config.Default()
	}
	cfg.Notify.Channel = migrateChannel(cfg.Notify.Channel)
	cfg.Schedule.Weekday = normalizeRunDay(cfg.Schedule.Weekday)
	if len(cfg.BookDays) == 0 {
		cfg.BookDays = append([]string(nil), config.Weekdays...)
	}
	s := &settings{cfg: cfg, hour: strconv.Itoa(cfg.Schedule.Hour), phase: phaseProbing}
	return s, probeLarkCmd()
}

// normalizeRunDay keeps the run weekday within the Mon–Fri options the form offers,
// defaulting an out-of-range/legacy value to Thursday so the bound Select is valid.
func normalizeRunDay(code string) string {
	for _, d := range config.Weekdays {
		if d == code {
			return code
		}
	}
	return "thu"
}

func dayLabel(code string) string {
	return map[string]string{
		"mon": "Monday", "tue": "Tuesday", "wed": "Wednesday",
		"thu": "Thursday", "fri": "Friday",
	}[code]
}

func daySelectOptions() []huh.Option[string] {
	opts := make([]huh.Option[string], 0, len(config.Weekdays))
	for _, d := range config.Weekdays {
		opts = append(opts, huh.NewOption(dayLabel(d), d))
	}
	return opts
}

func dayMultiOptions(selected []string) []huh.Option[string] {
	set := map[string]bool{}
	for _, d := range selected {
		set[d] = true
	}
	opts := make([]huh.Option[string], 0, len(config.Weekdays))
	for _, d := range config.Weekdays {
		opts = append(opts, huh.NewOption(dayLabel(d), d).Selected(set[d]))
	}
	return opts
}

func validateBookDays(v []string) error {
	if len(v) == 0 {
		return fmt.Errorf("pick at least one day")
	}
	return nil
}

// migrateChannel folds the retired macOS channels into the Lark/Off model: an old
// "both"/"lark" config keeps Lark; "macos" (or anything unknown) becomes Off.
func migrateChannel(ch string) string {
	if (config.Notify{Channel: ch}).LarkOn() {
		return "lark"
	}
	return "off"
}

func probeLarkCmd() tea.Cmd {
	return func() tea.Msg { return larkProbeMsg(notify.ProbeLark()) }
}

// notifyOptions lists the selectable channels. Lark appears only when a usable
// lark-cli bot is detected — "the setting is available only once detected".
func notifyOptions(larkAuthed bool) []huh.Option[string] {
	if larkAuthed {
		return []huh.Option[string]{
			huh.NewOption("Lark", "lark"),
			huh.NewOption("Off", "off"),
		}
	}
	return []huh.Option[string]{huh.NewOption("Off", "off")}
}

// buildForm constructs the huh form for the current probe result. Called once the
// probe lands.
func (s *settings) buildForm() {
	// A bound Select value must be one of its options. If Lark is unavailable, the
	// only option is Off, so force the channel there.
	if !s.lark.Authed {
		s.cfg.Notify.Channel = "off"
	}

	notifySel := huh.NewSelect[string]().Title("Notify").
		Options(notifyOptions(s.lark.Authed)...).
		Value(&s.cfg.Notify.Channel)
	if !s.lark.Authed {
		notifySel = notifySel.Description("Lark unavailable: " + s.lark.Reason)
	}

	base := huh.NewGroup(
		huh.NewSelect[string]().Title("Run day").
			Description("weekday the booker fires — when next week's menu opens").
			Options(daySelectOptions()...).
			Value(&s.cfg.Schedule.Weekday),
		huh.NewMultiSelect[string]().Title("Book on days").
			Description("which weekdays to grab lunch for (space toggles)").
			Options(dayMultiOptions(s.cfg.BookDays)...).
			Value(&s.cfg.BookDays).
			Validate(validateBookDays),
		huh.NewInput().Title("Open hour (0-23)").
			Description("hour the booking window opens, local time").
			Value(&s.hour).Validate(validateHour),
		notifySel,
	)

	// huh hides at the group level, not the field level. A hidden trailing group makes
	// the form submit straight after the base group (form.go), so the Lark target is
	// asked for only when the Lark channel is chosen.
	larkGroup := huh.NewGroup(
		huh.NewInput().Title("Lark target").
			Description("receive_id: ou_… (DM) or oc_… (group)").
			Value(&s.cfg.Notify.LarkTarget).
			Validate(validateLarkTarget),
	).WithHideFunc(func() bool {
		return !s.lark.Authed || s.cfg.Notify.Channel != "lark"
	})

	// Help is suppressed: the screen footer carries the key hints, so huh's own help
	// line (and its "/ filter" noise on the short day lists) would just be clutter.
	s.form = huh.NewForm(base, larkGroup).
		WithShowHelp(false).WithShowErrors(true).WithTheme(huhTheme())

	if s.width > 0 {
		s.form = s.form.WithWidth(s.width).WithHeight(s.height)
	}
}

func validateHour(v string) error {
	h, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil || h < 0 || h > 23 {
		return fmt.Errorf("enter a number 0–23")
	}
	return nil
}

func validateLarkTarget(v string) error {
	v = strings.TrimSpace(v)
	if v == "" {
		return fmt.Errorf("enter a receive_id (ou_… or oc_…)")
	}
	if !strings.HasPrefix(v, "ou_") && !strings.HasPrefix(v, "oc_") {
		return fmt.Errorf("must start with ou_ (DM) or oc_ (group)")
	}
	return nil
}

func (s *settings) Update(msg tea.Msg) (screenModel, tea.Cmd) {
	switch msg := msg.(type) {
	case bodySizeMsg:
		s.width = msg.w
		s.height = msg.h - 1 // leave a line for the screen title
		if s.height < 1 {
			s.height = 1
		}
		if s.form != nil {
			s.form = s.form.WithWidth(s.width).WithHeight(s.height)
		}
		return s, nil
	case larkProbeMsg:
		s.lark = notify.LarkStatus(msg)
		s.buildForm()
		s.phase = phaseForm
		return s, s.form.Init()
	}

	switch s.phase {
	case phaseProbing:
		if k, ok := msg.(tea.KeyMsg); ok && k.String() == "esc" {
			return s, nav(scrDashboard)
		}
		return s, nil
	case phaseForm:
		return s.updateForm(msg)
	case phaseDone:
		if k, ok := msg.(tea.KeyMsg); ok {
			switch k.String() {
			case "esc", "q", "enter":
				return s, nav(scrDashboard)
			}
		}
		return s, nil
	}
	return s, nil
}

func (s *settings) updateForm(msg tea.Msg) (screenModel, tea.Cmd) {
	if k, ok := msg.(tea.KeyMsg); ok && k.String() == "esc" {
		return s, nav(scrDashboard) // cancel without saving
	}
	fm, cmd := s.form.Update(msg)
	if f, ok := fm.(*huh.Form); ok {
		s.form = f
	}
	switch s.form.State {
	case huh.StateCompleted:
		s.applyHour()
		s.saveErr = s.save()
		s.phase = phaseDone
		return s, nil
	case huh.StateAborted:
		return s, nav(scrDashboard)
	}
	return s, cmd
}

func (s *settings) applyHour() {
	if h, err := strconv.Atoi(strings.TrimSpace(s.hour)); err == nil {
		s.cfg.Schedule.Hour = h
	}
}

func (s *settings) save() error {
	dir, _ := config.ConfigDir()
	return config.Save(filepath.Join(dir, "config.json"), s.cfg)
}

func (s *settings) View(w, h int) string {
	switch s.phase {
	case phaseProbing:
		return titleStyle.Render("Settings") + "\n\n" +
			textStyle.Render("checking Lark availability…")
	case phaseDone:
		status := okNote("saved") + textStyle.Render("  config.json updated")
		if s.saveErr != nil {
			status = errNote("save failed: " + truncate(s.saveErr.Error(), w-16))
		}
		return titleStyle.Render("Settings") + "\n\n" + status + "\n\n" +
			metaStyle.Render("press enter to return")
	default: // phaseForm
		return titleStyle.Render("Settings") + "\n" + s.form.View()
	}
}

func (s *settings) Footer() string {
	switch s.phase {
	case phaseProbing:
		return footStyle.Render("esc cancel")
	case phaseDone:
		return footStyle.Render("enter return")
	default:
		return footStyle.Render("↑/↓ move   space toggle   enter next   esc cancel")
	}
}
