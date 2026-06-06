package notify

import (
	"os"
	"testing"
)

// TestIntegrationLive exercises the real Lark notifier (no exec stub). Skipped unless
// CANTEEN_LIVE=1, so the normal suite stays hermetic. It probes the host and, when
// CANTEEN_LARK_TARGET is set, sends a real Lark message to that receive_id.
//
//	CANTEEN_LIVE=1 [CANTEEN_LARK_TARGET=ou_…] go test ./internal/notify -run IntegrationLive -v
func TestIntegrationLive(t *testing.T) {
	if os.Getenv("CANTEEN_LIVE") == "" {
		t.Skip("set CANTEEN_LIVE=1 to run live notifier checks")
	}

	st := ProbeLark()
	t.Logf("ProbeLark -> %+v", st)

	if tgt := os.Getenv("CANTEEN_LARK_TARGET"); tgt != "" {
		if err := (Lark{Target: tgt}).Send("Canteen test", "lark --as bot path ok"); err != nil {
			t.Errorf("Lark.Send: %v", err)
		} else {
			t.Logf("sent Lark message to %s", tgt)
		}
	}
}
