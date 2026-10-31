package fath

import (
	"fmt"
	"strconv"
	"strings"
)

// ActuatorKind buckets an actuator function ID into the kind of value it
// accepts, so a human-supplied value (like "on" or "70%") can be routed to
// the right input datapoint.
type ActuatorKind int

const (
	KindOther ActuatorKind = iota
	KindSwitch
	KindDimmer
	KindBlind
)

// Well-known input pairing IDs, per
// https://developer.eu.mybuildings.abb.com/fah_local/reference/pairingids
const (
	pidSwitchOnOff       uint16 = 1  // AL_SWITCH_ON_OFF
	pidTimedStartStop    uint16 = 2  // AL_TIMED_START_STOP (staircase light)
	pidAbsoluteSetValue  uint16 = 17 // AL_ABSOLUTE_SET_VALUE_CONTROL (dimming %)
	pidMoveUpDown        uint16 = 32 // AL_MOVE_UP_DOWN
	pidStopStepUpDown    uint16 = 33 // AL_STOP_STEP_UP_DOWN
	pidDedicatedStop     uint16 = 34 // AL_DEDICATED_STOP
	pidSetPositionBlinds uint16 = 35 // AL_SET_ABSOLUTE_POSITION_BLINDS_PERCENTAGE
)

// ClassifyActuator determines what kind of value a function ID's actuator
// channel accepts.
func ClassifyActuator(id uint16) ActuatorKind {
	name, ok := functionIDNames[id]
	if !ok {
		return KindOther
	}
	if strings.Contains(name, "DIMMING_ACTUATOR") || name == "FID_DIMMER_GROUP" {
		return KindDimmer
	}
	if strings.Contains(name, "ACTUATOR") &&
		(strings.Contains(name, "BLIND") || strings.Contains(name, "SHUTTER") || strings.Contains(name, "AWNING") || strings.Contains(name, "ATTIC_WINDOW")) {
		return KindBlind
	}
	if name == "FID_BLIND_GROUP" || name == "FID_SHUTTER_GROUP" || name == "FID_SACE_BLIND_GROUP" {
		return KindBlind
	}
	if strings.Contains(name, "SWITCH_ACTUATOR") || strings.Contains(name, "STAIRCASE_LIGHT_ACTUATOR") || name == "FID_LIGHT_GROUP" {
		return KindSwitch
	}
	return KindOther
}

// SetCommand describes what to write to a channel's input datapoint.
type SetCommand struct {
	DatapointKey string
	PairingID    uint16
	Value        string
}

// ResolveSetCommand determines which input datapoint to write, and what
// value to send, for a human-supplied raw value against a given actuator
// channel. Accepted raw value formats depend on the actuator's kind:
// switches take on/off (also 1/0, true/false, yes/no); dimmers take a
// percentage (0-100, with or without a trailing "%") or on/off; blinds take
// up/down/stop or a percentage. Any other actuator kind falls back to
// writing the raw text verbatim, provided the channel has exactly one input
// datapoint.
//
// overridePairingID, if non-nil, bypasses kind-based auto-detection entirely
// and writes rawValue verbatim to the input datapoint with that pairing ID.
func ResolveSetCommand(id uint16, channel Channel, rawValue string, overridePairingID *uint16) (SetCommand, error) {
	if overridePairingID != nil {
		key, ok := findInput(channel, *overridePairingID)
		if !ok {
			return SetCommand{}, fmt.Errorf("channel has no input datapoint with pairing ID %d", *overridePairingID)
		}
		return SetCommand{DatapointKey: key, PairingID: *overridePairingID, Value: rawValue}, nil
	}

	switch ClassifyActuator(id) {
	case KindSwitch:
		return resolveSwitch(channel, rawValue)
	case KindDimmer:
		return resolveDimmer(channel, rawValue)
	case KindBlind:
		return resolveBlind(channel, rawValue)
	default:
		return resolveGeneric(channel, rawValue)
	}
}

func resolveSwitch(channel Channel, rawValue string) (SetCommand, error) {
	v, ok := parseOnOff(rawValue)
	if !ok {
		return SetCommand{}, fmt.Errorf("this actuator accepts on/off values (on, off, 1, 0, true, false), got %q", rawValue)
	}
	if key, ok := findInput(channel, pidSwitchOnOff); ok {
		return SetCommand{key, pidSwitchOnOff, v}, nil
	}
	if key, ok := findInput(channel, pidTimedStartStop); ok {
		return SetCommand{key, pidTimedStartStop, v}, nil
	}
	return SetCommand{}, fmt.Errorf("no on/off input datapoint found on this channel")
}

func resolveDimmer(channel Channel, rawValue string) (SetCommand, error) {
	if v, ok := parsePercent(rawValue); ok {
		if key, ok := findInput(channel, pidAbsoluteSetValue); ok {
			return SetCommand{key, pidAbsoluteSetValue, v}, nil
		}
		return SetCommand{}, fmt.Errorf("no dimming-value input datapoint found on this channel")
	}
	if v, ok := parseOnOff(rawValue); ok {
		if key, ok := findInput(channel, pidSwitchOnOff); ok {
			return SetCommand{key, pidSwitchOnOff, v}, nil
		}
	}
	return SetCommand{}, fmt.Errorf(`this actuator accepts a percentage (0-100, e.g. "70" or "70%%") or on/off, got %q`, rawValue)
}

func resolveBlind(channel Channel, rawValue string) (SetCommand, error) {
	switch strings.ToLower(strings.TrimSpace(rawValue)) {
	case "up":
		if key, ok := findInput(channel, pidMoveUpDown); ok {
			return SetCommand{key, pidMoveUpDown, "0"}, nil
		}
		return SetCommand{}, fmt.Errorf("no up/down input datapoint found on this channel")
	case "down":
		if key, ok := findInput(channel, pidMoveUpDown); ok {
			return SetCommand{key, pidMoveUpDown, "1"}, nil
		}
		return SetCommand{}, fmt.Errorf("no up/down input datapoint found on this channel")
	case "stop":
		if key, ok := findInput(channel, pidDedicatedStop); ok {
			return SetCommand{key, pidDedicatedStop, "1"}, nil
		}
		if key, ok := findInput(channel, pidStopStepUpDown); ok {
			return SetCommand{key, pidStopStepUpDown, "1"}, nil
		}
		return SetCommand{}, fmt.Errorf("no stop input datapoint found on this channel")
	}
	if v, ok := parsePercent(rawValue); ok {
		if key, ok := findInput(channel, pidSetPositionBlinds); ok {
			return SetCommand{key, pidSetPositionBlinds, v}, nil
		}
		return SetCommand{}, fmt.Errorf("no position input datapoint found on this channel")
	}
	return SetCommand{}, fmt.Errorf("this actuator accepts up, down, stop, or a percentage (0-100), got %q", rawValue)
}

func resolveGeneric(channel Channel, rawValue string) (SetCommand, error) {
	switch len(channel.Inputs) {
	case 0:
		return SetCommand{}, fmt.Errorf("this channel has no input (command) datapoints")
	case 1:
		for k, dp := range channel.Inputs {
			id, _ := parsePairingID(dp.PairingID.String())
			return SetCommand{k, id, rawValue}, nil
		}
	}
	return SetCommand{}, fmt.Errorf("this actuator has %d input datapoints; use -input <pairing-id> to pick one", len(channel.Inputs))
}

func findInput(channel Channel, pairingID uint16) (key string, ok bool) {
	for k, dp := range channel.Inputs {
		id, err := parsePairingID(dp.PairingID.String())
		if err == nil && id == pairingID {
			return k, true
		}
	}
	return "", false
}

func parseOnOff(raw string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "on", "1", "true", "yes":
		return "1", true
	case "off", "0", "false", "no":
		return "0", true
	}
	return "", false
}

func parsePercent(raw string) (string, bool) {
	s := strings.TrimSuffix(strings.TrimSpace(raw), "%")
	n, err := strconv.Atoi(s)
	if err != nil || n < 0 || n > 100 {
		return "", false
	}
	return strconv.Itoa(n), true
}
