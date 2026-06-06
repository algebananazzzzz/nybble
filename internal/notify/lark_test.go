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
