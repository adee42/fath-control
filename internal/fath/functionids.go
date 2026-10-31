package fath

import (
	"strconv"
	"strings"
)

// actuatorWhitelist lists function ID constant names that denote an actuator
// (something with a controllable/observable device state) even though their
// name doesn't contain "ACTUATOR". Curated from the free@home function ID
// reference: https://developer.eu.mybuildings.abb.com/fah_local/reference/functionids
var actuatorWhitelist = map[string]bool{
	"FID_DISHWASHER":                true,
	"FID_LAUNDRY":                   true,
	"FID_DRYER":                     true,
	"FID_OVEN":                      true,
	"FID_FRIDGE":                    true,
	"FID_FREEZER":                   true,
	"FID_FRIDGE_FREEZER":            true,
	"FID_HOOD":                      true,
	"FID_COFFEE_MACHINE":            true,
	"FID_HOB":                       true,
	"FID_VACUUM_CLEANER":            true,
	"FID_WALLBOX":                   true,
	"FID_MEDIA_PLAYER":              true,
	"FID_MEDIA_PLAYER_TYPE1":        true,
	"FID_UNDERFLOOR_HEATING":        true,
	"FID_FAN_COIL":                  true,
	"FID_SPLIT_UNIT":                true,
	"FID_SPLIT_UNIT_EXTENSION_UNIT": true,
	"FID_LIGHT_GROUP":               true,
	"FID_BLIND_GROUP":               true,
	"FID_DIMMER_GROUP":              true,
	"FID_RGB_W_GROUP":               true,
	"FID_RGB_GROUP":                 true,
	"FID_RGBW_GROUP":                true,
	"FID_SACE_BLIND_GROUP":          true,
	"FID_SHUTTER_GROUP":             true,
	"FID_DOOR_LOCK_CONTROL":         true,
	"FID_TWO_LEVEL_CONTROLLER":      true,
	"FID_EXTERNAL_SIREN":            true,
}

// sensorWhitelist lists function ID constant names that denote a sensor
// (a device reporting a measured or detected value) even though their name
// contains neither "SENSOR", "DETECTOR" nor "METER". Curated from the
// free@home function ID reference: https://developer.eu.mybuildings.abb.com/fah_local/reference/functionids
var sensorWhitelist = map[string]bool{
	"FID_THERMOSTAT":           true,
	"FID_THERMOSTAT_SLAVE":     true,
	"FID_CO_2":                 true,
	"FID_VOC":                  true,
	"FID_FLOOD_DETECTION":      true,
	"FID_DOMUS_WINDOW_CONTACT": true,
	"FID_CARD_READER":          true,
	"FID_INVERTER_INFO":        true,
	"FID_BATTERY_INFO":         true,
	"FID_INVERTER_BATTERY":     true,
	"FID_ENERGY_INVERTER":      true,
	"FID_ENERGY_BATTERY":       true,
}

// ParseFunctionID parses a channel's functionID hex string (e.g. "0007" or
// "0x0007") into its numeric form.
func ParseFunctionID(s string) (uint16, bool) {
	s = strings.TrimPrefix(strings.TrimPrefix(s, "0x"), "0X")
	if s == "" {
		return 0, false
	}
	n, err := strconv.ParseUint(s, 16, 16)
	if err != nil {
		return 0, false
	}
	return uint16(n), true
}

// FunctionName returns a human-readable name for a function ID, e.g.
// "Switch Actuator" for FID_SWITCH_ACTUATOR. Returns false if the function ID
// is not in the reference table.
func FunctionName(id uint16) (string, bool) {
	name, ok := functionIDNames[id]
	if !ok {
		return "", false
	}
	return humanizeConstName(name), true
}

// IsActuatorFunction reports whether a function ID represents an actuator
// (a device with a controllable/observable state), as opposed to a sensor,
// group icon, or other non-actuator entry.
func IsActuatorFunction(id uint16) bool {
	name, ok := functionIDNames[id]
	if !ok {
		return false
	}
	if strings.Contains(name, "SENSOR") {
		return false
	}
	if strings.Contains(name, "ACTUATOR") {
		return true
	}
	return actuatorWhitelist[name]
}

// IsSensorFunction reports whether a function ID represents a sensor (a
// device reporting a measured or detected value), as opposed to an actuator,
// group icon, or other non-sensor entry.
func IsSensorFunction(id uint16) bool {
	name, ok := functionIDNames[id]
	if !ok {
		return false
	}
	if strings.Contains(name, "ACTUATOR") {
		return false
	}
	if strings.Contains(name, "SENSOR") || strings.Contains(name, "DETECTOR") || strings.Contains(name, "METER") {
		return true
	}
	return sensorWhitelist[name]
}

var acronyms = map[string]bool{
	"RGB": true, "RGBW": true, "HVAC": true, "IR": true, "IP": true,
	"PIR": true, "DES": true, "CT": true, "API": true, "RTC": true,
	"AC": true, "PB": true, "1X1": true,
}

func humanizeConstName(constName string) string {
	name := strings.TrimPrefix(constName, "FID_")
	name = strings.TrimPrefix(name, "DEPRECATED_")
	parts := strings.Split(name, "_")
	for i, p := range parts {
		if p == "" {
			continue
		}
		if acronyms[p] {
			continue
		}
		low := strings.ToLower(p)
		parts[i] = strings.ToUpper(low[:1]) + low[1:]
	}
	return strings.Join(parts, " ")
}
