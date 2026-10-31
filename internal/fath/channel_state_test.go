package fath

import "testing"

func TestChannelState(t *testing.T) {
	tests := []struct {
		name    string
		outputs map[string]Datapoint
		want    string
	}{
		{
			name:    "no outputs",
			outputs: nil,
			want:    "-",
		},
		{
			name: "switch on",
			outputs: map[string]Datapoint{
				"odp0000": {PairingID: "256", Value: "1"}, // AL_INFO_ON_OFF
			},
			want: "On",
		},
		{
			name: "switch off",
			outputs: map[string]Datapoint{
				"odp0000": {PairingID: "256", Value: "0"},
			},
			want: "Off",
		},
		{
			name: "dimmer prefers dimming value over plain on/off",
			outputs: map[string]Datapoint{
				"odp0000": {PairingID: "256", Value: "1"},  // AL_INFO_ON_OFF
				"odp0001": {PairingID: "272", Value: "65"}, // AL_INFO_ACTUAL_DIMMING_VALUE
			},
			want: "65%",
		},
		{
			name: "switch actuator ignores force/group-mirror/error auxiliary datapoints",
			outputs: map[string]Datapoint{
				"odp0000": {PairingID: "256", Value: "1"}, // AL_INFO_ON_OFF
				"odp0001": {PairingID: "257", Value: "0"}, // AL_INFO_FORCE
				"odp0002": {PairingID: "261", Value: "1"}, // AL_SYSAP_INFO_ON_OFF
				"odp0003": {PairingID: "262", Value: "0"}, // AL_SYSAP_INFO_FORCE
				"odp0004": {PairingID: "273", Value: "0"}, // AL_INFO_ERROR
			},
			want: "On",
		},
		{
			name: "no known primary datapoint falls back to lowest pairing id",
			outputs: map[string]Datapoint{
				"odp0000": {PairingID: "257", Value: "0"}, // AL_INFO_FORCE
				"odp0001": {PairingID: "273", Value: "0"}, // AL_INFO_ERROR
			},
			want: "0",
		},
		{
			name: "blind position",
			outputs: map[string]Datapoint{
				"odp0000": {PairingID: "289", Value: "42"}, // AL_CURRENT_ABSOLUTE_POSITION_BLINDS_PERCENTAGE
			},
			want: "42%",
		},
		{
			name: "temperature sensor",
			outputs: map[string]Datapoint{
				"odp0000": {PairingID: "304", Value: "18.5"}, // AL_MEASURED_TEMPERATURE
			},
			want: "18.5 °C",
		},
		{
			name: "window/door sensor",
			outputs: map[string]Datapoint{
				"odp0000": {PairingID: "53", Value: "1"}, // AL_WINDOW_DOOR
			},
			want: "Open",
		},
		{
			name: "unknown pairing id falls back to raw value",
			outputs: map[string]Datapoint{
				"odp0000": {PairingID: "9999", Value: "raw"},
			},
			want: "raw",
		},
		{
			name: "malformed pairing id falls back to raw value of first output",
			outputs: map[string]Datapoint{
				"odp0000": {PairingID: "not-a-number", Value: "raw"},
			},
			want: "raw",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ch := Channel{Outputs: tt.outputs}
			if got := ChannelState(ch); got != tt.want {
				t.Errorf("ChannelState() = %q, want %q", got, tt.want)
			}
		})
	}
}
