package notify

import "testing"

type spy struct{ called int }

func (s *spy) Send(title, msg string) error { s.called++; return nil }

func TestDispatchSendsToLarkWhenEnabled(t *testing.T) {
	l := &spy{}
	d := Dispatcher{Enabled: true, Lark: l}
	_ = d.Notify("t", "m")
	if l.called != 1 {
		t.Fatalf("lark not called: %d", l.called)
	}
}

func TestDispatchSkipsWhenDisabled(t *testing.T) {
	l := &spy{}
	d := Dispatcher{Enabled: false, Lark: l}
	_ = d.Notify("t", "m")
	if l.called != 0 {
		t.Fatalf("disabled dispatcher should not send: %d", l.called)
	}
}

type capture struct{ msg string }

func (c *capture) Send(_, msg string) error { c.msg = msg; return nil }

func TestNotifyDetailedSendsDetailToLark(t *testing.T) {
	l := &capture{}
	d := Dispatcher{Enabled: true, Lark: l}
	_ = d.NotifyDetailed("Canteen", "short status", "short status\n- a\n- b")
	if l.msg != "short status\n- a\n- b" {
		t.Errorf("lark got %q, want full detail", l.msg)
	}
}
