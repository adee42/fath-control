package fath

import "fmt"

// AmbiguousMatchError indicates a channel name matched more than one
// actuator and couldn't be narrowed down to a single one using the given
// room/floor. Candidates lists every remaining match. NeedsFlag says what
// would help narrow it further ("room" or "floor"), or "" if even name+room+
// floor together aren't enough (duplicate channel names within the same room
// and floor).
type AmbiguousMatchError struct {
	Name       string
	Candidates []ChannelMatch
	NeedsFlag  string
}

func (e *AmbiguousMatchError) Error() string {
	msg := fmt.Sprintf("channel name %q matches %d actuators", e.Name, len(e.Candidates))
	if e.NeedsFlag != "" {
		msg += fmt.Sprintf("; specify a %s to narrow it down", e.NeedsFlag)
	} else {
		msg += "; these can't be told apart by name, room and floor alone - rename one of the channels in the free@home app"
	}
	return msg
}

// ResolveActuator finds the single actuator channel matching name, narrowing
// with room and floor (both optional) if the name alone isn't unique among
// actuators. Room is tried before floor, mirroring how a person would
// naturally disambiguate: "which Light? ... the one in the Kitchen".
//
// Returns *AmbiguousMatchError if multiple channels still match after
// applying whatever of room/floor was given.
func ResolveActuator(cfg Configuration, name, room, floor string) (ChannelMatch, error) {
	matches := FindActuatorChannels(cfg, name)
	if len(matches) == 0 {
		return ChannelMatch{}, fmt.Errorf("no actuator channel named %q found", name)
	}

	if len(matches) > 1 {
		if room == "" {
			return ChannelMatch{}, &AmbiguousMatchError{Name: name, Candidates: matches, NeedsFlag: "room"}
		}
		matches = FilterByRoom(matches, room)
		if len(matches) == 0 {
			return ChannelMatch{}, fmt.Errorf("no actuator channel named %q found in room %q", name, room)
		}
	}

	if len(matches) > 1 {
		if floor == "" {
			return ChannelMatch{}, &AmbiguousMatchError{Name: name, Candidates: matches, NeedsFlag: "floor"}
		}
		matches = FilterByFloor(matches, floor)
		if len(matches) == 0 {
			return ChannelMatch{}, fmt.Errorf("no actuator channel named %q found in room %q on floor %q", name, room, floor)
		}
	}

	if len(matches) > 1 {
		return ChannelMatch{}, &AmbiguousMatchError{Name: name, Candidates: matches}
	}

	return matches[0], nil
}
