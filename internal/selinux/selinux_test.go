package selinux

import "testing"

func TestRelabelModeOnlyOnEnabledHosts(t *testing.T) {
	for _, tc := range []struct {
		state   State
		enabled bool
		mode    string
	}{
		{Enforcing, true, "Z"},
		{Permissive, true, "Z"},
		{Disabled, false, ""},
		{Unavailable, false, ""},
		{State(""), false, ""},
	} {
		if got := tc.state.Enabled(); got != tc.enabled {
			t.Errorf("%q Enabled()=%v want %v", tc.state, got, tc.enabled)
		}
		if got := tc.state.RelabelMode(); got != tc.mode {
			t.Errorf("%q RelabelMode()=%q want %q", tc.state, got, tc.mode)
		}
	}
}

func TestDetectReturnsKnownState(t *testing.T) {
	switch Detect() {
	case Enforcing, Permissive, Disabled, Unavailable:
	default:
		t.Fatal("Detect returned an unknown state")
	}
}
