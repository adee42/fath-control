package fath

import (
	"errors"
	"testing"
)

func TestResolveActuator(t *testing.T) {
	cfg := testConfig()

	m, err := ResolveActuator(cfg, "Light", "Kitchen", "")
	if err != nil {
		t.Fatalf("unique after room filter: unexpected error: %v", err)
	}
	if m.DeviceName != "Kitchen Switch" {
		t.Errorf("got device %q, want Kitchen Switch", m.DeviceName)
	}

	m, err = ResolveActuator(cfg, "Light", "Living Room", "Ground Floor")
	if err != nil || m.DeviceName != "Living Room Switch" {
		t.Errorf("room+floor narrowing: got %+v, err %v", m, err)
	}

	if _, err := ResolveActuator(cfg, "Nonexistent", "", ""); err == nil {
		t.Error("expected error for unknown channel name")
	}
}

func TestResolveActuatorAmbiguous(t *testing.T) {
	cfg := testConfig()

	_, err := ResolveActuator(cfg, "Light", "", "")
	var ambiguous *AmbiguousMatchError
	if !errors.As(err, &ambiguous) {
		t.Fatalf("expected *AmbiguousMatchError, got %v (%T)", err, err)
	}
	if ambiguous.NeedsFlag != "room" {
		t.Errorf("NeedsFlag = %q, want %q", ambiguous.NeedsFlag, "room")
	}
	if len(ambiguous.Candidates) != 3 {
		t.Errorf("got %d candidates, want 3", len(ambiguous.Candidates))
	}

	// Ground Floor alone still leaves two candidates (Living Room, Kitchen).
	_, err = ResolveActuator(cfg, "Light", "", "Ground Floor")
	if !errors.As(err, &ambiguous) {
		t.Fatalf("expected *AmbiguousMatchError after floor-only filter, got %v", err)
	}
	if ambiguous.NeedsFlag != "room" {
		t.Errorf("NeedsFlag = %q, want %q (floor alone isn't tried before room)", ambiguous.NeedsFlag, "room")
	}
}
