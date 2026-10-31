package fath

import "testing"

func switchChannel() Channel {
	return Channel{
		FunctionID: "0007", // FID_SWITCH_ACTUATOR
		Inputs: map[string]Datapoint{
			"idp0000": {PairingID: "1", Value: "0"}, // AL_SWITCH_ON_OFF
		},
	}
}

func dimmerChannel() Channel {
	return Channel{
		FunctionID: "0012", // FID_DIMMING_ACTUATOR
		Inputs: map[string]Datapoint{
			"idp0000": {PairingID: "1", Value: "0"},  // AL_SWITCH_ON_OFF
			"idp0001": {PairingID: "17", Value: "0"}, // AL_ABSOLUTE_SET_VALUE_CONTROL
		},
	}
}

func blindChannel() Channel {
	return Channel{
		FunctionID: "0061", // FID_BLIND_ACTUATOR
		Inputs: map[string]Datapoint{
			"idp0000": {PairingID: "32", Value: "0"}, // AL_MOVE_UP_DOWN
			"idp0001": {PairingID: "34", Value: "0"}, // AL_DEDICATED_STOP
			"idp0002": {PairingID: "35", Value: "0"}, // AL_SET_ABSOLUTE_POSITION_BLINDS_PERCENTAGE
		},
	}
}

func TestResolveSetCommandSwitch(t *testing.T) {
	id, _ := ParseFunctionID("0007")
	ch := switchChannel()

	tests := []struct {
		value   string
		want    string
		wantErr bool
	}{
		{"on", "1", false},
		{"On", "1", false},
		{"true", "1", false},
		{"1", "1", false},
		{"off", "0", false},
		{"no", "0", false},
		{"maybe", "", true},
		{"50%", "", true},
	}
	for _, tt := range tests {
		cmd, err := ResolveSetCommand(id, ch, tt.value, nil)
		if tt.wantErr {
			if err == nil {
				t.Errorf("value %q: expected error, got %+v", tt.value, cmd)
			}
			continue
		}
		if err != nil {
			t.Errorf("value %q: unexpected error: %v", tt.value, err)
			continue
		}
		if cmd.DatapointKey != "idp0000" || cmd.PairingID != pidSwitchOnOff || cmd.Value != tt.want {
			t.Errorf("value %q: got %+v, want datapoint idp0000, pairing %d, value %q", tt.value, cmd, pidSwitchOnOff, tt.want)
		}
	}
}

func TestResolveSetCommandDimmer(t *testing.T) {
	id, _ := ParseFunctionID("0012")
	ch := dimmerChannel()

	cmd, err := ResolveSetCommand(id, ch, "70%", nil)
	if err != nil {
		t.Fatalf("70%%: unexpected error: %v", err)
	}
	if cmd.DatapointKey != "idp0001" || cmd.PairingID != pidAbsoluteSetValue || cmd.Value != "70" {
		t.Errorf("70%%: got %+v", cmd)
	}

	cmd, err = ResolveSetCommand(id, ch, "70", nil)
	if err != nil || cmd.Value != "70" {
		t.Errorf("70 (no %%): got %+v, err %v", cmd, err)
	}

	cmd, err = ResolveSetCommand(id, ch, "on", nil)
	if err != nil || cmd.DatapointKey != "idp0000" || cmd.PairingID != pidSwitchOnOff || cmd.Value != "1" {
		t.Errorf("on: got %+v, err %v", cmd, err)
	}

	if _, err := ResolveSetCommand(id, ch, "150%", nil); err == nil {
		t.Error("150%: expected error for out-of-range percentage")
	}
	if _, err := ResolveSetCommand(id, ch, "sideways", nil); err == nil {
		t.Error("sideways: expected error for unrecognized value")
	}
}

func TestResolveSetCommandBlind(t *testing.T) {
	id, _ := ParseFunctionID("0061")
	ch := blindChannel()

	cmd, err := ResolveSetCommand(id, ch, "up", nil)
	if err != nil || cmd.DatapointKey != "idp0000" || cmd.Value != "0" {
		t.Errorf("up: got %+v, err %v", cmd, err)
	}
	cmd, err = ResolveSetCommand(id, ch, "down", nil)
	if err != nil || cmd.DatapointKey != "idp0000" || cmd.Value != "1" {
		t.Errorf("down: got %+v, err %v", cmd, err)
	}
	cmd, err = ResolveSetCommand(id, ch, "stop", nil)
	if err != nil || cmd.DatapointKey != "idp0001" || cmd.PairingID != pidDedicatedStop || cmd.Value != "1" {
		t.Errorf("stop: got %+v, err %v", cmd, err)
	}
	cmd, err = ResolveSetCommand(id, ch, "42", nil)
	if err != nil || cmd.DatapointKey != "idp0002" || cmd.PairingID != pidSetPositionBlinds || cmd.Value != "42" {
		t.Errorf("42: got %+v, err %v", cmd, err)
	}
	if _, err := ResolveSetCommand(id, ch, "sideways", nil); err == nil {
		t.Error("sideways: expected error")
	}
}

func TestResolveSetCommandGenericFallback(t *testing.T) {
	id, _ := ParseFunctionID("00A6") // FID_WALLBOX - not switch/dimmer/blind
	single := Channel{
		FunctionID: "00A6",
		Inputs: map[string]Datapoint{
			"idp0000": {PairingID: "1201", Value: "0"}, // AL_SWITCH_CHARGING
		},
	}
	cmd, err := ResolveSetCommand(id, single, "start", nil)
	if err != nil || cmd.DatapointKey != "idp0000" || cmd.Value != "start" {
		t.Errorf("generic single-input passthrough: got %+v, err %v", cmd, err)
	}

	multi := Channel{
		FunctionID: "00A6",
		Inputs: map[string]Datapoint{
			"idp0000": {PairingID: "1201", Value: "0"},
			"idp0001": {PairingID: "1200", Value: "0"},
		},
	}
	if _, err := ResolveSetCommand(id, multi, "start", nil); err == nil {
		t.Error("generic multi-input: expected error asking to use -input")
	}

	none := Channel{FunctionID: "00A6"}
	if _, err := ResolveSetCommand(id, none, "start", nil); err == nil {
		t.Error("generic no-input: expected error")
	}
}

func TestResolveSetCommandOverride(t *testing.T) {
	id, _ := ParseFunctionID("0007")
	ch := switchChannel()
	pid := uint16(1)
	cmd, err := ResolveSetCommand(id, ch, "raw-value", &pid)
	if err != nil || cmd.DatapointKey != "idp0000" || cmd.Value != "raw-value" {
		t.Errorf("override: got %+v, err %v", cmd, err)
	}

	missing := uint16(999)
	if _, err := ResolveSetCommand(id, ch, "x", &missing); err == nil {
		t.Error("override with missing pairing id: expected error")
	}
}
