package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/algebananazzzzz/nybble/internal/config"
	"github.com/algebananazzzzz/nybble/internal/schedule"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
)

// scheduleOp is the minimum launchd/pmset operation a form submit implies, so the sudo
// pmset step only runs when the schedule actually changes.
type scheduleOp int

const (
	opNoop    scheduleOp = iota // nothing changed
	opInstall                   // off → on
	opRemove                    // on → off
	opReapply                   // on → on with new timing
)

// scheduleAction diffs the schedule's installed state before/after the form against
// whether the timing changed, and picks the minimal operation.
func scheduleAction(wasOn, nowOn, timingChanged bool) scheduleOp {
	switch {
	case !wasOn && !nowOn:
		return opNoop
	case !wasOn && nowOn:
		return opInstall
	case wasOn && !nowOn:
		return opRemove
	case timingChanged:
		return opReapply
	default:
		return opNoop
	}
}

// maxLeadMin caps the heads-up lead. It's generous (the user picks freely) but bounded
// so the derived wake time can't wrap past midnight and misfire the weekday.
const maxLeadMin = 180

func validateLead(v string) error {
	n, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil {
		return fmt.Errorf("enter a number of minutes")
	}
	if n < 1 || n > maxLeadMin {
		return fmt.Errorf("1–%d minutes", maxLeadMin)
	}
	return nil
}

type schedulePhase int

const (
	schedForm schedulePhase = iota
	schedApplying
	schedDone
)

// scheduleScreen drives the Schedule page: a huh form for the timing + an Enable toggle,
// then a single submit that installs/removes the launchd job and runs the sudo pmset
// wake through tea.ExecProcess (which surfaces macOS's own password prompt).
type scheduleScreen struct {
	phase      schedulePhase
	form       *huh.Form
	cfg        config.Config
	wasOn      bool   // schedule installed when the page opened
	enable     bool   // form-bound Enable toggle
	hour       string // form-bound open hour
	lead       string // form-bound notify lead
	prevTiming string // timing snapshot to detect a change on submit
	op         scheduleOp
	width      int
	height     int
	err        error
	note       string
}

// wakeDoneMsg carries the result of the suspended sudo pmset command.
type wakeDoneMsg struct{ err error }

func newSchedule() (*scheduleScreen, tea.Cmd) {
	dir, _ := config.ConfigDir()
	cfg, err := config.Load(filepath.Join(dir, "config.json"))
	if err != nil {
		cfg = config.Default()
	}
	cfg.Schedule.Weekday = normalizeRunDay(cfg.Schedule.Weekday)
	if cfg.Schedule.LeadMin <= 0 {
		cfg.Schedule.LeadMin = config.DefaultLeadMin
	}
	if len(cfg.BookDays) == 0 {
		cfg.BookDays = append([]string(nil), config.Weekdays...)
	}
	s := &scheduleScreen{
		cfg:   cfg,
		wasOn: schedule.Installed(),
		hour:  strconv.Itoa(cfg.Schedule.Hour),
		lead:  strconv.Itoa(cfg.Schedule.LeadMin),
	}
	s.enable = s.wasOn
	s.prevTiming = timingKey(cfg)
	s.buildForm()
	return s, s.form.Init()
}

// timingKey is everything that, if changed while the schedule is on, requires the job to
// be re-installed (and the wake re-scheduled).
func timingKey(cfg config.Config) string {
	return fmt.Sprintf("%s|%d|%d|%d|%s",
		cfg.Schedule.Weekday, cfg.Schedule.Hour, cfg.Schedule.Minute, cfg.Schedule.LeadMin, cfg.Schedule.TZ)
}

func (s *scheduleScreen) buildForm() {
	s.form = huh.NewForm(huh.NewGroup(
		huh.NewSelect[string]().Title("Run day").
			Description("weekday the booker fires — when next week's menu opens").
			Options(daySelectOptions()...).
			Value(&s.cfg.Schedule.Weekday),
		huh.NewInput().Title("Open hour (0-23)").
			Description("hour the booking window opens, local time").
			Value(&s.hour).Validate(validateHour),
		huh.NewMultiSelect[string]().Title("Book on days").
			Description("which weekdays to grab lunch for (space toggles)").
			Options(dayMultiOptions(s.cfg.BookDays)...).
			Value(&s.cfg.BookDays).
			Validate(validateBookDays),
		huh.NewInput().Title("Notify me (min before)").
			Description("heads-up before the run; the Mac wakes a touch earlier").
			Value(&s.lead).Validate(validateLead),
		huh.NewConfirm().Title("Enable schedule").
			Description("install the weekly job + wake (asks for your password once)").
			Affirmative("On").Negative("Off").
			Value(&s.enable),
	)).WithShowHelp(false).WithShowErrors(true).WithTheme(huhTheme())
	if s.width > 0 {
		s.form = s.form.WithWidth(s.width).WithHeight(s.height)
	}
}

func (s *scheduleScreen) Update(msg tea.Msg) (screenModel, tea.Cmd) {
	switch msg := msg.(type) {
	case bodySizeMsg:
		s.width = msg.w
		s.height = msg.h - 1
		if s.height < 1 {
			s.height = 1
		}
		if s.form != nil {
			s.form = s.form.WithWidth(s.width).WithHeight(s.height)
		}
		return s, nil
	case wakeDoneMsg:
		s.phase = schedDone
		if msg.err != nil {
			// The job install/remove already succeeded; only the wake step failed.
			switch s.op {
			case opRemove:
				s.note = "schedule disabled — but pmset cancel failed; run it manually"
			default:
				s.note = "enabled — but wake wasn't scheduled (sudo cancelled?); re-apply to retry"
			}
		}
		return s, nil
	}

	switch s.phase {
	case schedForm:
		return s.updateForm(msg)
	case schedDone:
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

func (s *scheduleScreen) updateForm(msg tea.Msg) (screenModel, tea.Cmd) {
	if k, ok := msg.(tea.KeyMsg); ok && k.String() == "esc" {
		return s, nav(scrDashboard) // cancel without saving
	}
	fm, cmd := s.form.Update(msg)
	if f, ok := fm.(*huh.Form); ok {
		s.form = f
	}
	switch s.form.State {
	case huh.StateCompleted:
		return s, s.apply()
	case huh.StateAborted:
		return s, nav(scrDashboard)
	}
	return s, cmd
}

// apply persists the form, then performs the minimal launchd/pmset change. The wake
// (sudo) runs via tea.ExecProcess; everything else is in-process. Returns the Cmd that
// drives the suspended sudo prompt, or nil when the work is already finished.
func (s *scheduleScreen) apply() tea.Cmd {
	s.applyHour()
	s.applyLead()
	if err := s.save(); err != nil {
		s.err, s.phase = err, schedDone
		return nil
	}
	bin, _ := os.Executable()
	// A stale plist (the binary moved or was renamed since install) counts as a timing
	// change: launchd would silently fail to spawn it, so force a reinstall.
	binMoved := s.wasOn && bin != "" && schedule.InstalledBin() != bin
	s.op = scheduleAction(s.wasOn, s.enable, timingKey(s.cfg) != s.prevTiming || binMoved)

	switch s.op {
	case opNoop:
		s.note, s.phase = "no change", schedDone
		return nil
	case opRemove:
		if err := schedule.RemoveJob(); err != nil {
			s.err, s.phase = err, schedDone
			return nil
		}
		s.phase = schedApplying
		return tea.ExecProcess(schedule.WakeCancelCmd(), func(e error) tea.Msg { return wakeDoneMsg{e} })
	default: // opInstall / opReapply
		wd, hh, mm, err := schedule.LocalFire(s.cfg.Schedule.Weekday, s.cfg.Schedule.Hour, s.cfg.Schedule.Minute, s.cfg.Schedule.TZ, time.Now())
		if err != nil {
			s.err, s.phase = err, schedDone
			return nil
		}
		if err := schedule.InstallJob(bin, wd, hh, mm, s.cfg.Schedule.Lead()); err != nil {
			s.err, s.phase = err, schedDone
			return nil
		}
		s.phase = schedApplying
		return tea.ExecProcess(schedule.WakeCmd(wd, hh, mm, s.cfg.Schedule.Lead()), func(e error) tea.Msg { return wakeDoneMsg{e} })
	}
}

func (s *scheduleScreen) applyHour() {
	if h, err := strconv.Atoi(strings.TrimSpace(s.hour)); err == nil {
		s.cfg.Schedule.Hour = h
	}
}

func (s *scheduleScreen) applyLead() {
	if n, err := strconv.Atoi(strings.TrimSpace(s.lead)); err == nil {
		s.cfg.Schedule.LeadMin = n
	}
}

func (s *scheduleScreen) save() error {
	dir, _ := config.ConfigDir()
	return config.Save(filepath.Join(dir, "config.json"), s.cfg)
}

func (s *scheduleScreen) statusLine() string {
	if !s.wasOn {
		return metaStyle.Render("Currently: off")
	}
	return metaStyle.Render(fmt.Sprintf("Currently: on · %s %02d:00 · notify %d min before",
		dayLabel(s.cfg.Schedule.Weekday), s.cfg.Schedule.Hour, s.cfg.Schedule.Lead()))
}

func (s *scheduleScreen) View(w, h int) string {
	switch s.phase {
	case schedApplying:
		return titleStyle.Render("Schedule") + "\n\n" + textStyle.Render("scheduling wake…")
	case schedDone:
		var status string
		switch {
		case s.err != nil:
			status = errNote("failed: " + truncate(s.err.Error(), w-12))
		case s.note != "":
			status = okNote("saved") + "  " + textStyle.Render(s.note)
		case s.op == opRemove:
			status = okNote("schedule disabled")
		default:
			status = okNote("schedule enabled") + "  " + textStyle.Render("the Mac will wake and book for you")
		}
		return titleStyle.Render("Schedule") + "\n\n" + status + "\n\n" + metaStyle.Render("press enter to return")
	default: // schedForm
		return titleStyle.Render("Schedule") + "\n" + s.statusLine() + "\n" + s.form.View()
	}
}

func (s *scheduleScreen) Footer() string {
	switch s.phase {
	case schedDone:
		return footStyle.Render("enter return")
	case schedApplying:
		return footStyle.Render("working…")
	default:
		return footStyle.Render("↑/↓ move   space toggle   enter next   esc cancel")
	}
}
