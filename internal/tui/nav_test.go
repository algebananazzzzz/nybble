package tui

import "testing"

func TestNavGatesAuthScreensWhenLoggedOut(t *testing.T) {
	m := New()
	m.state = State{LoggedIn: false}

	for _, to := range []screen{scrFavorites, scrSettings, scrSchedule} {
		got, _ := m.navigate(to)
		if got.(Model).screen != scrReauth {
			t.Errorf("logged out: nav to %d should redirect to reauth, got %d", to, got.(Model).screen)
		}
	}

	// Reauth and dashboard are always reachable.
	if got, _ := m.navigate(scrReauth); got.(Model).screen != scrReauth {
		t.Error("reauth must be reachable when logged out")
	}
	if got, _ := m.navigate(scrDashboard); got.(Model).screen != scrDashboard {
		t.Error("dashboard must be reachable when logged out")
	}
}

func TestNavAllowsAuthScreensWhenLoggedIn(t *testing.T) {
	m := New()
	m.state = State{LoggedIn: true}

	for _, to := range []screen{scrFavorites, scrSettings, scrSchedule} {
		got, _ := m.navigate(to)
		if got.(Model).screen != to {
			t.Errorf("logged in: nav to %d blocked, got %d", to, got.(Model).screen)
		}
	}
}
