package fath

import (
	"sort"
	"strconv"
)

// primaryStatePriority ranks pairing IDs by how well they represent "the"
// current state of a channel, most representative first. A channel commonly
// exposes several output datapoints beyond its main reading — e.g. a switch
// actuator also reports a forced-operation flag (AL_INFO_FORCE), a
// group-mirrored on/off state (AL_SYSAP_INFO_ON_OFF), and an error/overload
// flag (AL_INFO_ERROR) alongside its actual on/off state. This list picks the
// one value a user actually cares about instead of showing all of them.
var primaryStatePriority = []uint16{
	289,  // AL_CURRENT_ABSOLUTE_POSITION_BLINDS_PERCENTAGE - blind/shutter position
	272,  // AL_INFO_ACTUAL_DIMMING_VALUE - dimming level
	256,  // AL_INFO_ON_OFF - switch/light state
	304,  // AL_MEASURED_TEMPERATURE
	337,  // AL_HUMIDITY
	279,  // AL_INFO_RGB
	280,  // AL_INFO_COLOR_TEMPERATURE
	283,  // AL_INFO_HSV
	305,  // AL_INFO_VALUE_HEATING - heating valve position
	306,  // AL_INFO_VALUE_COOLING - cooling valve position
	354,  // AL_INFO_ABSOLUTE_FAN_SPEED
	1123, // AL_INFO_ACTUAL_VOLUME
	360,  // AL_INFO_LOCKED_SENSOR
	53,   // AL_WINDOW_DOOR - open/closed
	1205, // AL_INFO_CHARGING - wallbox
}

// ChannelState summarizes a channel's current state as a short human string
// (e.g. "On", "42%", "18.5 °C") by picking its single most representative
// output datapoint (per primaryStatePriority) and formatting it according to
// its pairing ID's datapoint type. Output datapoints reflect the channel's
// actual state, as opposed to input datapoints which are commands. Works for
// both actuators and sensors. Returns "-" if the channel has no outputs.
func ChannelState(channel Channel) string {
	if len(channel.Outputs) == 0 {
		return "-"
	}

	type entry struct {
		id    uint16
		value string
	}
	var entries []entry
	for _, dp := range channel.Outputs {
		id, err := parsePairingID(dp.PairingID.String())
		if err != nil {
			continue
		}
		entries = append(entries, entry{id, dp.Value})
	}
	if len(entries) == 0 {
		// No output had a parseable (numeric, 16-bit) pairing ID - fall back
		// to the raw value of a deterministically chosen output rather than
		// showing nothing.
		keys := make([]string, 0, len(channel.Outputs))
		for k := range channel.Outputs {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		return channel.Outputs[keys[0]].Value
	}

	for _, pid := range primaryStatePriority {
		for _, e := range entries {
			if e.id == pid {
				return FormatValue(pid, e.value)
			}
		}
	}

	// No known "primary" datapoint matched (e.g. an unmapped or unusual
	// channel) - fall back to the numerically lowest pairing ID, since
	// free@home tends to assign the main state the lowest pairing ID on a
	// channel (auxiliary info like force/error/group-mirroring uses higher
	// ones).
	sort.Slice(entries, func(i, j int) bool { return entries[i].id < entries[j].id })
	return FormatValue(entries[0].id, entries[0].value)
}

func parsePairingID(s string) (uint16, error) {
	n, err := strconv.ParseUint(s, 10, 16)
	return uint16(n), err
}
