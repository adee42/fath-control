package fath

import "sort"

// ChannelRow is one channel matching a function filter (actuator or sensor),
// with a descriptive function name and a humanized summary of its current
// state/value (e.g. "On", "42%", "On, 42%", "18.5 °C").
type ChannelRow struct {
	Floor      string `json:"floor"`
	Room       string `json:"room"`
	Device     string `json:"device"`
	DeviceID   string `json:"deviceId"`
	Channel    string `json:"channel"`
	ChannelID  string `json:"channelId"`
	FunctionID string `json:"functionId"`
	Function   string `json:"function"`
	State      string `json:"state"`
}

// CollectChannelRows returns one ChannelRow per channel whose function ID
// passes include (typically IsActuatorFunction or IsSensorFunction), sorted
// by floor, then room, then device, then channel.
func CollectChannelRows(cfg Configuration, include func(uint16) bool) []ChannelRow {
	var rows []ChannelRow
	for _, sysap := range cfg {
		for deviceID, device := range sysap.Devices {
			deviceName := device.DisplayName
			if deviceName == "" {
				deviceName = deviceID
			}
			for channelID, channel := range device.Channels {
				fid, ok := ParseFunctionID(channel.FunctionID)
				if !ok || !include(fid) {
					continue
				}
				channelName := channel.DisplayName
				if channelName == "" {
					channelName = channelID
				}
				floorName, roomName := sysap.FloorRoomName(channel.Floor, channel.Room)
				functionName, _ := FunctionName(fid)

				rows = append(rows, ChannelRow{
					Floor:      floorName,
					Room:       roomName,
					Device:     deviceName,
					DeviceID:   deviceID,
					Channel:    channelName,
					ChannelID:  channelID,
					FunctionID: channel.FunctionID,
					Function:   functionName,
					State:      ChannelState(channel),
				})
			}
		}
	}

	sort.Slice(rows, func(i, j int) bool {
		a, b := rows[i], rows[j]
		switch {
		case a.Floor != b.Floor:
			return a.Floor < b.Floor
		case a.Room != b.Room:
			return a.Room < b.Room
		case a.Device != b.Device:
			return a.Device < b.Device
		default:
			return a.Channel < b.Channel
		}
	})
	return rows
}
