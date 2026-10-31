package fath

import "strings"

// ChannelMatch identifies one located actuator channel together with enough
// context (device, floor, room) to disambiguate it from others sharing the
// same channel display name.
type ChannelMatch struct {
	SysAPID     string
	DeviceID    string
	ChannelID   string
	DeviceName  string
	ChannelName string
	FloorName   string
	RoomName    string
	FunctionID  uint16
	Channel     Channel
}

// FindActuatorChannels returns every actuator channel whose display name
// matches name (case-insensitive, whitespace-trimmed). Sensor channels are
// never returned, since only actuators can be set.
func FindActuatorChannels(cfg Configuration, name string) []ChannelMatch {
	target := strings.ToLower(strings.TrimSpace(name))

	var out []ChannelMatch
	for sysapID, sysap := range cfg {
		for deviceID, device := range sysap.Devices {
			deviceName := device.DisplayName
			if deviceName == "" {
				deviceName = deviceID
			}
			for channelID, channel := range device.Channels {
				fid, ok := ParseFunctionID(channel.FunctionID)
				if !ok || !IsActuatorFunction(fid) {
					continue
				}
				channelName := channel.DisplayName
				if channelName == "" {
					channelName = channelID
				}
				if strings.ToLower(channelName) != target {
					continue
				}
				floorName, roomName := sysap.FloorRoomName(channel.Floor, channel.Room)
				out = append(out, ChannelMatch{
					SysAPID:     sysapID,
					DeviceID:    deviceID,
					ChannelID:   channelID,
					DeviceName:  deviceName,
					ChannelName: channelName,
					FloorName:   floorName,
					RoomName:    roomName,
					FunctionID:  fid,
					Channel:     channel,
				})
			}
		}
	}
	return out
}

// FilterByRoom returns the subset of matches whose room name equals room
// (case-insensitive, whitespace-trimmed).
func FilterByRoom(matches []ChannelMatch, room string) []ChannelMatch {
	target := strings.ToLower(strings.TrimSpace(room))
	var out []ChannelMatch
	for _, m := range matches {
		if strings.ToLower(m.RoomName) == target {
			out = append(out, m)
		}
	}
	return out
}

// FilterByFloor returns the subset of matches whose floor name equals floor
// (case-insensitive, whitespace-trimmed).
func FilterByFloor(matches []ChannelMatch, floor string) []ChannelMatch {
	target := strings.ToLower(strings.TrimSpace(floor))
	var out []ChannelMatch
	for _, m := range matches {
		if strings.ToLower(m.FloorName) == target {
			out = append(out, m)
		}
	}
	return out
}
