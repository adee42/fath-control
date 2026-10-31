package fath

import "testing"

func TestParseFunctionID(t *testing.T) {
	cases := map[string]uint16{"0007": 0x0007, "0x0012": 0x0012, "1820": 0x1820}
	for in, want := range cases {
		got, ok := ParseFunctionID(in)
		if !ok || got != want {
			t.Errorf("ParseFunctionID(%q) = %#x, %v; want %#x, true", in, got, ok, want)
		}
	}
	if _, ok := ParseFunctionID(""); ok {
		t.Error("ParseFunctionID(\"\") should fail")
	}
}

func TestFunctionNameAndActuator(t *testing.T) {
	tests := []struct {
		id       uint16
		wantName string
		actuator bool
	}{
		{0x0007, "Switch Actuator", true},
		{0x0012, "Dimming Actuator", true},
		{0x0061, "Blind Actuator", true},
		{0x0009, "Shutter Actuator", true},
		{0x0000, "Switch Sensor", false},
		{0x0043, "Temperature Sensor", false},
		{0x0011, "Movement Detector", false},
		{0x00A6, "Wallbox", true},
		{0x4000, "Light Group", true},
	}
	for _, tt := range tests {
		name, ok := FunctionName(tt.id)
		if !ok || name != tt.wantName {
			t.Errorf("FunctionName(%#04x) = %q, %v; want %q", tt.id, name, ok, tt.wantName)
		}
		if got := IsActuatorFunction(tt.id); got != tt.actuator {
			t.Errorf("IsActuatorFunction(%#04x) = %v; want %v", tt.id, got, tt.actuator)
		}
	}
}

func TestIsSensorFunction(t *testing.T) {
	tests := []struct {
		id     uint16
		sensor bool
	}{
		{0x0000, true},  // FID_SWITCH_SENSOR
		{0x0043, true},  // FID_TEMPERATURE_SENSOR
		{0x0011, true},  // FID_MOVEMENT_DETECTOR (no "SENSOR" in name)
		{0x00C8, true},  // FID_WATER_METER
		{0x0083, true},  // FID_THERMOSTAT (whitelisted)
		{0x0007, false}, // FID_SWITCH_ACTUATOR
		{0x0012, false}, // FID_DIMMING_ACTUATOR
		{0x00A6, false}, // FID_WALLBOX (actuator)
	}
	for _, tt := range tests {
		if got := IsSensorFunction(tt.id); got != tt.sensor {
			t.Errorf("IsSensorFunction(%#04x) = %v; want %v", tt.id, got, tt.sensor)
		}
	}
}
