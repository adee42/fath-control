package fath

import "strings"

// FormatValue renders a datapoint's raw string value in a human-readable form
// based on the KNX datapoint type associated with its pairing ID (e.g. a
// DPT_SCALING value becomes "42%", a DPT_SWITCH value becomes "On"/"Off").
// Pairing IDs with no known metadata, or values that don't match the
// expected 1/0 encoding, are returned unchanged.
func FormatValue(pairingID uint16, raw string) string {
	meta, ok := pairingMetas[pairingID]
	if !ok {
		return raw
	}
	switch meta.DPT {
	case "DPT_SWITCH", "DPT_BOOL":
		return formatBit(raw, "On", "Off")
	case "DPT_SCALING":
		return raw + "%"
	case "DPT_VALUE_TEMP":
		return raw + " °C"
	case "DPT_STATE":
		return formatBit(raw, "Active", "Inactive")
	case "DPT_ALARM":
		return formatBit(raw, "Alarm", "No alarm")
	case "DPT_WINDOW_DOOR":
		return formatBit(raw, "Open", "Closed")
	case "DPT_UP_DOWN":
		return formatBit(raw, "Down", "Up")
	default:
		return raw
	}
}

func formatBit(raw, on, off string) string {
	switch raw {
	case "1":
		return on
	case "0":
		return off
	default:
		return raw
	}
}

// PairingName returns a human-readable name for a pairing ID, e.g.
// "Info Actual Dimming Value" for AL_INFO_ACTUAL_DIMMING_VALUE. Returns false
// if the pairing ID is not in the reference table.
func PairingName(pairingID uint16) (string, bool) {
	meta, ok := pairingMetas[pairingID]
	if !ok || meta.Name == "" {
		return "", false
	}
	name := strings.TrimPrefix(meta.Name, "AL_")
	parts := strings.Split(name, "_")
	for i, p := range parts {
		if p == "" || acronyms[p] {
			continue
		}
		low := strings.ToLower(p)
		parts[i] = strings.ToUpper(low[:1]) + low[1:]
	}
	return strings.Join(parts, " "), true
}
