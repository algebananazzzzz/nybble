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

// DefaultTarget fills in the Lark receive_id when the user configured none, so a
// blank target falls back to the supplied id — typically the user's own union_id,
// which makes the bot DM them. No-op when disabled, when an explicit target is
// already set, or when id is empty (e.g. identity couldn't be resolved).
func (d *Dispatcher) DefaultTarget(id string) {
	if !d.Enabled || id == "" {
		return
	}
	if l, ok := d.Lark.(Lark); ok && l.Target == "" {
		l.Target = id
		d.Lark = l
	}
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
