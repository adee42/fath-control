package fath

import "testing"

func TestFindActuatorChannelsExcludesSensors(t *testing.T) {
	cfg := testConfig()
	matches := FindActuatorChannels(cfg, "Light")
	if len(matches) != 3 {
		t.Fatalf("got %d matches, want 3 (sensor with the same channel name must be excluded)", len(matches))
	}
	for _, m := range matches {
		if m.DeviceName == "Temperature Sensor" {
			t.Errorf("sensor channel leaked into actuator matches: %+v", m)
		}
	}
}

func TestFindActuatorChannelsCaseInsensitive(t *testing.T) {
	cfg := testConfig()
	if got := len(FindActuatorChannels(cfg, "  LIGHT  ")); got != 3 {
		t.Errorf("case/whitespace-insensitive match: got %d, want 3", got)
	}
	if got := len(FindActuatorChannels(cfg, "Nonexistent")); got != 0 {
		t.Errorf("got %d matches for nonexistent name, want 0", got)
	}
}

func TestCascadingDisambiguation(t *testing.T) {
	cfg := testConfig()
	matches := FindActuatorChannels(cfg, "Light")
	if len(matches) != 3 {
		t.Fatalf("setup: got %d matches, want 3", len(matches))
	}

	byRoom := FilterByRoom(matches, "Kitchen")
	if len(byRoom) != 1 || byRoom[0].DeviceName != "Kitchen Switch" {
		t.Errorf("FilterByRoom(Kitchen) = %+v, want single Kitchen Switch match", byRoom)
	}

	// Ground Floor has two rooms with a "Light" channel (Living Room, Kitchen);
	// room alone isn't ambiguous here since each room only has one, but floor
	// alone still leaves two candidates.
	byFloor := FilterByFloor(matches, "Ground Floor")
	if len(byFloor) != 2 {
		t.Errorf("FilterByFloor(Ground Floor) = %d matches, want 2", len(byFloor))
	}

	byRoomThenFloor := FilterByFloor(FilterByRoom(matches, "Living Room"), "Ground Floor")
	if len(byRoomThenFloor) != 1 || byRoomThenFloor[0].DeviceName != "Living Room Switch" {
		t.Errorf("room+floor narrowing = %+v, want single Living Room Switch match", byRoomThenFloor)
	}
}

func testConfig() Configuration {
	return Configuration{
		"sysap1": {
			Floorplan: Floorplan{
				Floors: map[string]Floor{
					"01": {Name: "Ground Floor", Rooms: map[string]Room{
						"0001": {Name: "Living Room"},
						"0002": {Name: "Kitchen"},
					}},
					"02": {Name: "First Floor", Rooms: map[string]Room{
						"0001": {Name: "Bedroom"},
					}},
				},
			},
			Devices: map[string]Device{
				"dev1": {
					DisplayName: "Living Room Switch",
					Channels: map[string]Channel{
						"ch0000": {DisplayName: "Light", FunctionID: "0007", Floor: "01", Room: "0001"},
					},
				},
				"dev2": {
					DisplayName: "Kitchen Switch",
					Channels: map[string]Channel{
						"ch0000": {DisplayName: "Light", FunctionID: "0007", Floor: "01", Room: "0002"},
					},
				},
				"dev3": {
					DisplayName: "Bedroom Switch",
					Channels: map[string]Channel{
						"ch0000": {DisplayName: "Light", FunctionID: "0007", Floor: "02", Room: "0001"},
					},
				},
				"dev4": {
					DisplayName: "Temperature Sensor",
					Channels: map[string]Channel{
						"ch0000": {DisplayName: "Light", FunctionID: "0043", Floor: "01", Room: "0001"}, // sensor, same name "Light"
					},
				},
			},
		},
	}
}
