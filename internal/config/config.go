package config

import (
	"encoding/json"
	"os"
)

type NamedCode struct {
	Code string `json:"code"`
	Name string `json:"name"`
}
type Pickup struct {
	Code int    `json:"code"`
	Name string `json:"name"`
}

// DefaultLeadMin is how many minutes before the open time the job fires (and the
// "starts in N min" heads-up sends) when a config doesn't specify its own lead.
const DefaultLeadMin = 5

type Schedule struct {
	Weekday string `json:"weekday"`
	Hour    int    `json:"hour"`
	Minute  int    `json:"minute"`
	TZ      string `json:"tz"`
	LeadMin int    `json:"leadMin"` // minutes before open to fire + notify
}

// Lead returns the effective heads-up lead in minutes, falling back to DefaultLeadMin
// for a legacy config.json that predates the field (unmarshals LeadMin to 0).
func (s Schedule) Lead() int {
	if s.LeadMin <= 0 {
		return DefaultLeadMin
	}
	return s.LeadMin
}

type Notify struct {
	Channel    string `json:"channel"` // lark | off
	LarkTarget string `json:"larkTarget"`
}

// LarkOn reports whether Lark notifications are enabled, tolerating the legacy
// "both" value written before macOS notifications were removed (so an existing
// config keeps notifying without needing a settings re-save).
func (n Notify) LarkOn() bool { return n.Channel == "lark" || n.Channel == "both" }

type Config struct {
	Building NamedCode `json:"building"`
	Pickup   Pickup    `json:"pickup"`
	MealType string    `json:"mealType"`
	BookDays []string  `json:"bookDays"` // weekday codes to book: mon tue wed thu fri
	Schedule Schedule  `json:"schedule"`
	Notify   Notify    `json:"notify"`
}

// Weekdays is the full Mon–Fri set, the default booking selection.
var Weekdays = []string{"mon", "tue", "wed", "thu", "fri"}

// BookSet returns the configured booking weekdays as a lookup set. An empty
// config (e.g. a pre-BookDays config.json) falls back to the full Mon–Fri week so
// an upgrade never silently stops booking.
func (c Config) BookSet() map[string]bool {
	days := c.BookDays
	if len(days) == 0 {
		days = Weekdays
	}
	set := make(map[string]bool, len(days))
	for _, d := range days {
		set[d] = true
	}
	return set
}

// Default is the starting config for a fresh install. Building and pickup are
// left empty — they are deployment-specific and set by the user via the TUI,
// then persisted to the gitignored config.json. TZ defaults to UTC.
func Default() Config {
	return Config{
		Building: NamedCode{},
		Pickup:   Pickup{},
		MealType: "lunch",
		BookDays: append([]string(nil), Weekdays...),
		Schedule: Schedule{Weekday: "thu", Hour: 10, Minute: 0, TZ: "UTC", LeadMin: DefaultLeadMin},
		Notify:   Notify{Channel: "off"},
	}
}

func Load(path string) (Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	var c Config
	err = json.Unmarshal(raw, &c)
	return c, err
}

func Save(path string, c Config) error {
	raw, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o644)
}

// Favorites is an ordered list of dish-name patterns, highest priority first.
type Favorites []string

func LoadFavorites(path string) (Favorites, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var f Favorites
	err = json.Unmarshal(raw, &f)
	return f, err
}
func SaveFavorites(path string, f Favorites) error {
	raw, _ := json.MarshalIndent(f, "", "  ")
	return os.WriteFile(path, raw, 0o644)
}
