package notify

import (
	"os"
	"testing"
)

// TestIntegrationLive exercises the real Lark notifier (no exec stub). Skipped unless
// NYBBLE_LIVE=1, so the normal suite stays hermetic. It probes the host and, when
// NYBBLE_LARK_TARGET is set, sends a real Lark message to that receive_id.
//
//	NYBBLE_LIVE=1 [NYBBLE_LARK_TARGET=ou_…] go test ./internal/notify -run IntegrationLive -v
func TestIntegrationLive(t *testing.T) {
	if os.Getenv("NYBBLE_LIVE") == "" {
		t.Skip("set NYBBLE_LIVE=1 to run live notifier checks")
	}

	st := ProbeLark()
	t.Logf("ProbeLark -> %+v", st)

	if tgt := os.Getenv("NYBBLE_LARK_TARGET"); tgt != "" {
		if err := (Lark{Target: tgt}).Send("Canteen test", "lark --as bot path ok"); err != nil {
			t.Errorf("Lark.Send: %v", err)
		} else {
			t.Logf("sent Lark message to %s", tgt)
		}
	}
}
