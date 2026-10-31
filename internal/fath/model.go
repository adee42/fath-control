// Package fath implements a minimal client for the ABB/Busch-Jaeger free@home
// local REST API exposed by the System Access Point (SysAP) 2.0.
//
// API reference: https://developer.eu.mybuildings.abb.com/fah_local
package fath

import (
	"encoding/json"
	"fmt"
)

// Configuration is the top-level response of GET /fhapi/v1/api/rest/configuration.
// It is keyed by the SysAP UUID, since a response can in theory describe more
// than one access point.
type Configuration map[string]SysAP

// SysAP describes a single System Access Point and everything it knows about.
type SysAP struct {
	SysAPName string            `json:"sysapName"`
	Devices   map[string]Device `json:"devices"`
	Floorplan Floorplan         `json:"floorplan"`
}

// Floorplan resolves the floor/room IDs referenced by channels into names.
type Floorplan struct {
	Floors map[string]Floor `json:"floors"`
}

type Floor struct {
	Name  string          `json:"name"`
	Rooms map[string]Room `json:"rooms"`
}

type Room struct {
	Name string `json:"name"`
}

// Device is a single physical or virtual free@home device (identified by its
// serial number), grouping one or more channels.
type Device struct {
	DisplayName string             `json:"displayName"`
	Channels    map[string]Channel `json:"channels"`
}

// Channel represents one sensor/actor function exposed by a device, e.g. a
// single switch actuator or a single temperature sensor.
type Channel struct {
	DisplayName string               `json:"displayName"`
	FunctionID  string               `json:"functionID"`
	Floor       string               `json:"floor"`
	Room        string               `json:"room"`
	Inputs      map[string]Datapoint `json:"inputs"`
	Outputs     map[string]Datapoint `json:"outputs"`
}

// Datapoint is a single value inside a channel. Output datapoints ("odp...")
// reflect the current state of the channel; input datapoints ("idp...") are
// commands that can be sent to it. pairingID identifies the semantic meaning
// of the datapoint (see the free@home API docs for the pairing ID table).
type Datapoint struct {
	PairingID FlexString `json:"pairingID"`
	Value     string     `json:"value"`
}

// FlexString unmarshals a JSON string or number into a Go string, since some
// SysAP firmware versions encode pairingID as a JSON number and others as a
// numeric string.
type FlexString string

func (f *FlexString) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		*f = FlexString(s)
		return nil
	}
	var n json.Number
	if err := json.Unmarshal(data, &n); err == nil {
		*f = FlexString(n.String())
		return nil
	}
	return fmt.Errorf("pairingID is neither a string nor a number: %s", string(data))
}

func (f FlexString) String() string {
	return string(f)
}

// FloorRoomName resolves a channel's floor/room IDs to human-readable names
// using the SysAP's floorplan. Either return value falls back to the raw ID
// (or "-") when no matching name is found.
func (s SysAP) FloorRoomName(floorID, roomID string) (floorName, roomName string) {
	floorName, roomName = floorID, roomID
	if floorID == "" {
		floorName = "-"
	}
	if roomID == "" {
		roomName = "-"
	}
	floor, ok := s.Floorplan.Floors[floorID]
	if !ok {
		return floorName, roomName
	}
	if floor.Name != "" {
		floorName = floor.Name
	}
	if room, ok := floor.Rooms[roomID]; ok && room.Name != "" {
		roomName = room.Name
	}
	return floorName, roomName
}
