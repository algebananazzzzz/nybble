package notify

import (
	"slices"
	"strings"
	"testing"
)

func TestParseBotInfo(t *testing.T) {
	cases := []struct {
		name       string
		out        string
		wantAuthed bool
	}{
		{"ok", `{"bot":{"app_name":"x"},"code":0,"msg":"ok"}`, true},
		{"ok with notice wrapper", `{"_notice":{"update":{"latest":"1.0.48"}},"bot":{},"code":0}`, true},
		{"auth failed", `{"code":99991663,"msg":"app ticket invalid"}`, false},
		{"not configured / empty", ``, false},
		{"garbage", `Error: no app configured`, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			st := parseBotInfo([]byte(c.out))
			if !st.Installed {
				t.Errorf("Installed = false, want true (binary known present)")
			}
			if st.Authed != c.wantAuthed {
				t.Errorf("Authed = %v, want %v", st.Authed, c.wantAuthed)
			}
			if !st.Authed && st.Reason == "" {
				t.Errorf("unavailable status must carry a Reason hint")
			}
		})
	}
}

func TestLarkApiArgsIdentityAndTarget(t *testing.T) {
	dm := Lark{Target: "ou_abc"}.apiArgs("Canteen", "booked")
	if !slices.Contains(dm, "--as") || dm[slices.Index(dm, "--as")+1] != "bot" {
		t.Errorf("expected --as bot, got %v", dm)
	}
	if !sliceHasJSONField(dm, "receive_id_type", "open_id") {
		t.Errorf("ou_ target should use open_id, got %v", dm)
	}

	group := Lark{Target: "oc_xyz"}.apiArgs("Canteen", "booked")
	if !sliceHasJSONField(group, "receive_id_type", "chat_id") {
		t.Errorf("oc_ target should use chat_id, got %v", group)
	}

	// A union_id (on_…) is the cross-app default the bot DMs you with.
	dmUnion := Lark{Target: "on_def"}.apiArgs("Canteen", "booked")
	if !sliceHasJSONField(dmUnion, "receive_id_type", "union_id") {
		t.Errorf("on_ target should use union_id, got %v", dmUnion)
	}
}

func TestDispatcherDefaultTarget(t *testing.T) {
	// Blank target → fall back to the supplied id (typically the user's union_id).
	d := Dispatcher{Enabled: true, Lark: Lark{Target: ""}}
	d.DefaultTarget("on_me")
	if got := d.Lark.(Lark).Target; got != "on_me" {
		t.Errorf("blank target should default to on_me, got %q", got)
	}

	// An explicit target is never overridden.
	d2 := Dispatcher{Enabled: true, Lark: Lark{Target: "oc_group"}}
	d2.DefaultTarget("on_me")
	if got := d2.Lark.(Lark).Target; got != "oc_group" {
		t.Errorf("explicit target must win, got %q", got)
	}

	// Disabled, or no id to fall back to → no-op (no panic on nil Lark).
	d3 := Dispatcher{Enabled: false}
	d3.DefaultTarget("on_me")
	d4 := Dispatcher{Enabled: true, Lark: Lark{Target: ""}}
	d4.DefaultTarget("")
	if got := d4.Lark.(Lark).Target; got != "" {
		t.Errorf("empty id should leave target blank, got %q", got)
	}
}

// sliceHasJSONField reports whether any arg contains "key":"val" (the params/data
// args are JSON strings).
func sliceHasJSONField(args []string, key, val string) bool {
	needle := `"` + key + `":"` + val + `"`
	for _, a := range args {
		if strings.Contains(a, needle) {
			return true
		}
	}
	return false
}
