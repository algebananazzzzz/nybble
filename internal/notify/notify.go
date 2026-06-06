package notify

type Notifier interface {
	Send(title, message string) error
}

// Dispatcher delivers run notifications. macOS desktop banners were dropped — a
// plain CLI's osascript notification posts as "Script Editor" and is silently
// suppressed when that app's notifications are off, with no dependency-free fix —
// so Lark is the only channel. Enabled mirrors config Notify.Channel == "lark".
type Dispatcher struct {
	Enabled bool
	Lark    Notifier
}

func (d Dispatcher) Notify(title, message string) error {
	return d.NotifyDetailed(title, message, message)
}

// NotifyDetailed sends the full detail body to Lark. The short body (a compact
// desktop summary, kept for callers) is unused now that macOS is gone; Lark has
// room for everything.
func (d Dispatcher) NotifyDetailed(title, short, detail string) error {
	if !d.Enabled || d.Lark == nil {
		return nil
	}
	return d.Lark.Send(title, detail)
}
