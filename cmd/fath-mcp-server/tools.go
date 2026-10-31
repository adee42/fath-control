package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/adee42/fath-control/internal/fath"
)

var toolDefs = []toolDef{
	{
		Name: "list_sensors",
		Description: "List free@home sensors (temperature, movement, window/door, brightness, humidity, " +
			"energy/water/gas meters, ...) and their current readings. Optionally filter by room and/or floor.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"room":  map[string]any{"type": "string", "description": "Only return sensors in this room (case-insensitive)."},
				"floor": map[string]any{"type": "string", "description": "Only return sensors on this floor (case-insensitive)."},
			},
		},
	},
	{
		Name: "list_actuators",
		Description: "List free@home actuators (switches, lights, dimmers, blinds/shutters/awnings, heating/cooling " +
			"valves, ...) and their current state. Optionally filter by room and/or floor.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"room":  map[string]any{"type": "string", "description": "Only return actuators in this room (case-insensitive)."},
				"floor": map[string]any{"type": "string", "description": "Only return actuators on this floor (case-insensitive)."},
			},
		},
	},
	{
		Name: "set_actuator",
		Description: "Set a value on a free@home actuator, identified by its channel display name (e.g. \"Living Room Light\"), " +
			"as shown by list_actuators. If the name isn't unique among actuators, pass room to disambiguate; if name+room still " +
			"aren't unique, also pass floor. Accepted values depend on the actuator's type: switches/lights take on/off; dimmers " +
			"take a percentage (0-100) or on/off; blinds/shutters/awnings take up, down, stop, or a percentage (0-100, meaning " +
			"closed amount). Other actuator types accept raw text if the channel has exactly one input datapoint.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"channel": map[string]any{"type": "string", "description": "Channel display name to control, as shown by list_actuators."},
				"value":   map[string]any{"type": "string", "description": `Value to set, e.g. "on", "off", "70", "70%", "up", "down", or "stop".`},
				"room":    map[string]any{"type": "string", "description": "Room name, to disambiguate if channel alone isn't unique."},
				"floor":   map[string]any{"type": "string", "description": "Floor name, to disambiguate if channel+room aren't unique."},
				"input_pairing_id": map[string]any{
					"type":        "integer",
					"description": "Advanced: a specific input datapoint pairing ID to write to directly, bypassing automatic type detection.",
				},
			},
			"required": []string{"channel", "value"},
		},
	},
}

func (s *server) callTool(ctx context.Context, p toolsCallParams) toolsCallResult {
	switch p.Name {
	case "list_sensors":
		return s.listChannels(ctx, p.Arguments, fath.IsSensorFunction, "sensor")
	case "list_actuators":
		return s.listChannels(ctx, p.Arguments, fath.IsActuatorFunction, "actuator")
	case "set_actuator":
		return s.setActuator(ctx, p.Arguments)
	default:
		return errorResult(fmt.Sprintf("unknown tool %q", p.Name))
	}
}

type listArgs struct {
	Room  string `json:"room"`
	Floor string `json:"floor"`
}

func (s *server) listChannels(ctx context.Context, rawArgs json.RawMessage, include func(uint16) bool, label string) toolsCallResult {
	var args listArgs
	if len(rawArgs) > 0 {
		if err := json.Unmarshal(rawArgs, &args); err != nil {
			return errorResult("invalid arguments: " + err.Error())
		}
	}

	cfg, err := s.client.GetConfiguration(ctx)
	if err != nil {
		return errorResult(fmt.Sprintf("could not reach the free@home SysAP: %v", err))
	}

	rows := fath.CollectChannelRows(cfg, include)
	if args.Room != "" {
		rows = filterRows(rows, func(r fath.ChannelRow) bool { return strings.EqualFold(r.Room, args.Room) })
	}
	if args.Floor != "" {
		rows = filterRows(rows, func(r fath.ChannelRow) bool { return strings.EqualFold(r.Floor, args.Floor) })
	}

	if len(rows) == 0 {
		return textResult(fmt.Sprintf("No %ss found matching the given filters.", label))
	}

	b, err := json.MarshalIndent(rows, "", "  ")
	if err != nil {
		return errorResult("failed to encode result: " + err.Error())
	}
	return textResult(string(b))
}

func filterRows(rows []fath.ChannelRow, keep func(fath.ChannelRow) bool) []fath.ChannelRow {
	var out []fath.ChannelRow
	for _, r := range rows {
		if keep(r) {
			out = append(out, r)
		}
	}
	return out
}

type setActuatorArgs struct {
	Channel        string `json:"channel"`
	Value          string `json:"value"`
	Room           string `json:"room"`
	Floor          string `json:"floor"`
	InputPairingID *int   `json:"input_pairing_id"`
}

func (s *server) setActuator(ctx context.Context, rawArgs json.RawMessage) toolsCallResult {
	var args setActuatorArgs
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		return errorResult("invalid arguments: " + err.Error())
	}
	if args.Channel == "" || args.Value == "" {
		return errorResult(`"channel" and "value" are required`)
	}

	cfg, err := s.client.GetConfiguration(ctx)
	if err != nil {
		return errorResult(fmt.Sprintf("could not reach the free@home SysAP: %v", err))
	}

	m, err := fath.ResolveActuator(cfg, args.Channel, args.Room, args.Floor)
	if err != nil {
		var ambiguous *fath.AmbiguousMatchError
		if errors.As(err, &ambiguous) {
			return errorResult(formatAmbiguous(ambiguous))
		}
		return errorResult(err.Error())
	}

	var overridePtr *uint16
	if args.InputPairingID != nil {
		v := uint16(*args.InputPairingID)
		overridePtr = &v
	}

	cmd, err := fath.ResolveSetCommand(m.FunctionID, m.Channel, args.Value, overridePtr)
	if err != nil {
		functionName, _ := fath.FunctionName(m.FunctionID)
		return errorResult(fmt.Sprintf("cannot set %q (%s / %s, %s): %v", m.ChannelName, m.FloorName, m.RoomName, orDash(functionName), err))
	}

	if err := s.client.SetDatapoint(ctx, m.SysAPID, m.DeviceID, m.ChannelID, cmd.DatapointKey, cmd.Value); err != nil {
		return errorResult(fmt.Sprintf("failed to write to the SysAP: %v", err))
	}

	functionName, _ := fath.FunctionName(m.FunctionID)
	pairingName, _ := fath.PairingName(cmd.PairingID)
	return textResult(fmt.Sprintf("Set %q (%s / %s, %s) to %s (%s).",
		m.ChannelName, m.FloorName, m.RoomName, orDash(functionName), cmd.Value, orDash(pairingName)))
}

func formatAmbiguous(e *fath.AmbiguousMatchError) string {
	var b strings.Builder
	fmt.Fprintf(&b, "channel name %q matches %d actuators:\n", e.Name, len(e.Candidates))
	for _, m := range e.Candidates {
		fmt.Fprintf(&b, "- %s / %s / %s (device %q)\n", m.FloorName, m.RoomName, m.ChannelName, m.DeviceName)
	}
	if e.NeedsFlag != "" {
		fmt.Fprintf(&b, "Call set_actuator again with %q set to disambiguate.", e.NeedsFlag)
	} else {
		b.WriteString("These can't be told apart by name, room and floor alone; rename one of the channels in the free@home app.")
	}
	return b.String()
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
