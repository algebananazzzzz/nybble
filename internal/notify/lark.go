package notify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// notifyTimeout caps a lark-cli subprocess so a stuck call (e.g. no network) can't
// hang the whole run.
const notifyTimeout = 10 * time.Second

// execCommand is the seam tests stub to avoid shelling out for real. Production
// code always uses exec.CommandContext.
var execCommand = func(ctx context.Context, name string, args ...string) *exec.Cmd {
	return exec.CommandContext(ctx, name, args...)
}

type Lark struct {
	Target string // receive_id: open_id (ou_…) for a DM, or chat_id (oc_…) for a group
}

// LarkStatus is the result of probing the host for a usable Lark setup.
type LarkStatus struct {
	Installed bool
	Authed    bool   // lark-cli present AND the bot identity can authenticate
	Reason    string // human hint shown when the channel is unavailable
}

// ProbeLark reports whether Lark notifications can be sent right now. Sends go out
// `--as bot` (the app's tenant token — no user login to expire), so the probe checks
// that exact identity: a no-side-effect `bot/v3/info` call that succeeds only when the
// app has valid bot credentials. Note this is distinct from `lark-cli auth status`,
// which reflects a separate *user* OAuth login that can't P2P-message via this API.
func ProbeLark() LarkStatus {
	if _, err := exec.LookPath("lark-cli"); err != nil {
		return LarkStatus{Reason: "lark-cli not installed (npm i -g @larksuite/cli)"}
	}
	ctx, cancel := context.WithTimeout(context.Background(), notifyTimeout)
	defer cancel()
	// bot/v3/info returns the app's own bot profile; it needs only a valid tenant
	// token and has no side effects. It exits non-zero on auth failure but still
	// prints its JSON verdict to stdout, so parse the output regardless.
	out, _ := execCommand(ctx, "lark-cli", "api", "GET", "/open-apis/bot/v3/info", "--as", "bot").Output()
	return parseBotInfo(out)
}

// parseBotInfo turns `bot/v3/info` stdout into a LarkStatus. The binary is known
// present by the time this runs, so Installed is always true here. The Lark API
// envelope reports success as code:0.
func parseBotInfo(out []byte) LarkStatus {
	st := LarkStatus{Installed: true}
	var res struct {
		Code int `json:"code"`
	}
	if err := json.Unmarshal(out, &res); err != nil {
		// No parseable envelope (e.g. lark-cli not configured with an app).
		st.Reason = "bot not configured (set lark-cli app credentials)"
		return st
	}
	if res.Code == 0 {
		st.Authed = true
		return st
	}
	st.Reason = "bot auth failed (re-authorize: `lark-cli auth login`)"
	return st
}

// apiArgs builds the lark-cli argv for sending one text message. Split out so the
// argument shape (identity, receive_id type) is unit-testable without a subprocess.
func (l Lark) apiArgs(title, message string) []string {
	idType := "open_id"
	switch {
	case strings.HasPrefix(l.Target, "oc_"):
		idType = "chat_id"
	case strings.HasPrefix(l.Target, "on_"):
		idType = "union_id" // the cross-app default: DM the user by their union_id
	}
	content, _ := json.Marshal(map[string]string{"text": title + ": " + message})
	params, _ := json.Marshal(map[string]string{"receive_id_type": idType})
	data, _ := json.Marshal(map[string]string{
		"receive_id": l.Target,
		"msg_type":   "text",
		"content":    string(content),
	})
	// --as bot: send from the app (tenant token, auto-refreshing — no user login to
	// expire). The user identity can't P2P-message via this endpoint (230027), so bot
	// is the correct identity for canteen alerts, matching the ProbeLark gate.
	return []string{
		"api", "POST", "/open-apis/im/v1/messages",
		"--params", string(params), "--data", string(data), "--as", "bot",
	}
}

// Send delivers a Lark text message via the system lark-cli (`@larksuite/cli`).
// content must be JSON per the IM API; lark-cli handles token refresh.
func (l Lark) Send(title, message string) error {
	if _, err := exec.LookPath("lark-cli"); err != nil {
		return fmt.Errorf("lark-cli not installed")
	}
	ctx, cancel := context.WithTimeout(context.Background(), notifyTimeout)
	defer cancel()
	out, err := execCommand(ctx, "lark-cli", l.apiArgs(title, message)...).Output()
	if err != nil {
		return err
	}
	// lark-cli returns code:0 and a message_id on success; verify delivery rather than
	// trust the exit code.
	if !bytes.Contains(out, []byte("message_id")) {
		return fmt.Errorf("lark send failed: %s", bytes.TrimSpace(out))
	}
	return nil
}
