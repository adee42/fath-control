// Command fath-control connects to a local free@home System Access Point
// (SysAP) and lists all sensors/actors (devices and their channels) together
// with their current datapoint values.
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/adee42/fath-control/internal/fath"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	var (
		host          = flag.String("host", envOr("FATH_HOST", ""), "SysAP host/IP, e.g. 192.168.1.50 (env FATH_HOST)")
		user          = flag.String("user", envOr("FATH_USER", ""), "local API username (env FATH_USER)")
		password      = flag.String("password", envOr("FATH_PASSWORD", ""), "local API password (env FATH_PASSWORD); prompted for if omitted")
		scheme        = flag.String("scheme", "https", `URL scheme to use ("https" or "http")`)
		apiBase       = flag.String("api-base", "/fhapi/v1/api/rest", "REST API path prefix on the SysAP")
		insecureTLS   = flag.Bool("insecure", true, "skip TLS certificate verification (SysAPs use a self-signed certificate by default)")
		timeout       = flag.Duration("timeout", 10*time.Second, "HTTP request timeout")
		showInputs    = flag.Bool("show-inputs", false, "also list input (command) datapoints, not just outputs/current state")
		rawJSON       = flag.Bool("raw", false, "print the raw configuration JSON returned by the SysAP instead of a table")
		asJSON        = flag.Bool("json", false, "print the parsed sensor/actor list as JSON instead of a table")
		actuatorsOnly = flag.Bool("actuators-only", false, "list only actuators (switches, dimmers, blinds, ...) one row per channel, with a descriptive function name and humanized current state")
		sensorsOnly   = flag.Bool("sensors-only", false, "list only sensors (temperature, movement, window/door, ...) one row per channel, with a descriptive function name and humanized current value")
		setChannel    = flag.String("set", "", "set a value on the actuator channel with this display name, then exit (requires -value)")
		setValue      = flag.String("value", "", "value to write with -set: on/off for switches, up/down/stop or a percentage for blinds, a percentage (e.g. 70 or 70%) for dimmers, or raw text for other actuator types")
		setRoom       = flag.String("room", "", "room name to disambiguate -set when the channel name alone isn't unique")
		setFloor      = flag.String("floor", "", "floor name to disambiguate -set when name+room aren't unique either")
		setInput      = flag.Int("input", -1, "advanced: input datapoint pairing ID to write to directly with -set, bypassing type auto-detection")
	)
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s -host <ip> -user <user> [-password <pass>] [flags]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Lists all sensors/actors known to a free@home SysAP's local API, with their current state.\n")
		fmt.Fprintf(os.Stderr, "Pass -set <channel name> -value <value> to control an actuator instead of listing.\n\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if *host == "" {
		return fmt.Errorf("-host (or FATH_HOST) is required")
	}
	if *user == "" {
		return fmt.Errorf("-user (or FATH_USER) is required")
	}
	if *actuatorsOnly && *sensorsOnly {
		return fmt.Errorf("-actuators-only and -sensors-only are mutually exclusive")
	}
	if *setChannel != "" && *setValue == "" {
		return fmt.Errorf("-value is required together with -set")
	}
	if *setChannel == "" && *setValue != "" {
		return fmt.Errorf("-value is only meaningful together with -set")
	}
	if *password == "" {
		var err error
		*password, err = promptPassword()
		if err != nil {
			return err
		}
	}

	client := fath.NewClient(fath.Options{
		Host:               *host,
		Scheme:             *scheme,
		APIBase:            *apiBase,
		User:               *user,
		Password:           *password,
		InsecureSkipVerify: *insecureTLS,
		Timeout:            *timeout,
	})

	ctx, cancel := context.WithTimeout(context.Background(), *timeout+2*time.Second)
	defer cancel()

	if *rawJSON {
		body, err := client.GetRawConfiguration(ctx)
		if err != nil {
			return err
		}
		var pretty map[string]any
		if err := json.Unmarshal(body, &pretty); err != nil {
			os.Stdout.Write(body)
			return nil
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(pretty)
	}

	cfg, err := client.GetConfiguration(ctx)
	if err != nil {
		return err
	}

	if *setChannel != "" {
		return runSet(ctx, client, cfg, *setChannel, *setRoom, *setFloor, *setValue, *setInput)
	}

	if *actuatorsOnly || *sensorsOnly {
		label, filter := "actuator", fath.IsActuatorFunction
		if *sensorsOnly {
			label, filter = "sensor", fath.IsSensorFunction
		}
		rows := fath.CollectChannelRows(cfg, filter)
		if *asJSON {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(rows)
		}
		printChannelTable(rows)
		fmt.Printf("\n%d %s(s).\n", len(rows), label)
		return nil
	}

	rows := collectRows(cfg, *showInputs)

	if *asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(rows)
	}

	printTable(rows)
	fmt.Printf("\n%d datapoint(s) across %d channel(s).\n", len(rows), countChannels(cfg))
	return nil
}

// runSet resolves channelName (optionally narrowed by room/floor) to a single
// actuator channel and writes value to it.
func runSet(ctx context.Context, client *fath.Client, cfg fath.Configuration, channelName, room, floor, value string, inputOverride int) error {
	m, err := fath.ResolveActuator(cfg, channelName, room, floor)
	if err != nil {
		var ambiguous *fath.AmbiguousMatchError
		if errors.As(err, &ambiguous) {
			return formatAmbiguousError(ambiguous)
		}
		return err
	}

	var overridePtr *uint16
	if inputOverride >= 0 {
		v := uint16(inputOverride)
		overridePtr = &v
	}

	cmd, err := fath.ResolveSetCommand(m.FunctionID, m.Channel, value, overridePtr)
	if err != nil {
		return fmt.Errorf("cannot set %q (%s / %s, device %q): %w", m.ChannelName, m.FloorName, m.RoomName, m.DeviceName, err)
	}

	if err := client.SetDatapoint(ctx, m.SysAPID, m.DeviceID, m.ChannelID, cmd.DatapointKey, cmd.Value); err != nil {
		return err
	}

	functionName, _ := fath.FunctionName(m.FunctionID)
	pairingName, _ := fath.PairingName(cmd.PairingID)
	fmt.Printf("Set %q (%s / %s, %s) %s [%s] to %s\n",
		m.ChannelName, m.FloorName, m.RoomName, orDash(functionName), m.ChannelID+"/"+cmd.DatapointKey, orDash(pairingName), cmd.Value)
	return nil
}

// formatAmbiguousError renders an *fath.AmbiguousMatchError as a CLI-friendly
// message listing every candidate channel, so the user can see exactly what
// to disambiguate.
func formatAmbiguousError(e *fath.AmbiguousMatchError) error {
	var b strings.Builder
	fmt.Fprintf(&b, "channel name %q matches %d actuators:\n", e.Name, len(e.Candidates))
	for _, m := range e.Candidates {
		fmt.Fprintf(&b, "  - %s / %s / %s (device %q)\n", m.FloorName, m.RoomName, m.ChannelName, m.DeviceName)
	}
	if e.NeedsFlag != "" {
		fmt.Fprintf(&b, "specify -%s to narrow it down", e.NeedsFlag)
	} else {
		b.WriteString("these can't be told apart by name, room and floor alone; rename one of the channels in the free@home app")
	}
	return fmt.Errorf("%s", b.String())
}

func printChannelTable(rows []fath.ChannelRow) {
	w := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
	fmt.Fprintln(w, "FLOOR\tROOM\tDEVICE\tCHANNEL\tFUNCTION\tSTATE")
	for _, r := range rows {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			r.Floor, r.Room, r.Device, r.Channel, orDash(r.Function), r.State)
	}
	w.Flush()
}

// Row is one printable datapoint reading, flattened out of the
// SysAP/device/channel hierarchy for easy display or JSON export.
type Row struct {
	Floor      string `json:"floor"`
	Room       string `json:"room"`
	Device     string `json:"device"`
	DeviceID   string `json:"deviceId"`
	Channel    string `json:"channel"`
	ChannelID  string `json:"channelId"`
	FunctionID string `json:"functionId"`
	Direction  string `json:"direction"` // "output" (state) or "input" (command)
	Datapoint  string `json:"datapoint"`
	PairingID  string `json:"pairingId"`
	Value      string `json:"value"`
}

func collectRows(cfg fath.Configuration, showInputs bool) []Row {
	var rows []Row
	for _, sysap := range cfg {
		for deviceID, device := range sysap.Devices {
			deviceName := device.DisplayName
			if deviceName == "" {
				deviceName = deviceID
			}
			for channelID, channel := range device.Channels {
				channelName := channel.DisplayName
				if channelName == "" {
					channelName = channelID
				}
				floorName, roomName := sysap.FloorRoomName(channel.Floor, channel.Room)

				base := Row{
					Floor:      floorName,
					Room:       roomName,
					Device:     deviceName,
					DeviceID:   deviceID,
					Channel:    channelName,
					ChannelID:  channelID,
					FunctionID: channel.FunctionID,
				}

				if len(channel.Outputs) == 0 && len(channel.Inputs) == 0 {
					rows = append(rows, base)
					continue
				}

				for dpID, dp := range channel.Outputs {
					r := base
					r.Direction = "output"
					r.Datapoint = dpID
					r.PairingID = dp.PairingID.String()
					r.Value = dp.Value
					rows = append(rows, r)
				}
				if showInputs {
					for dpID, dp := range channel.Inputs {
						r := base
						r.Direction = "input"
						r.Datapoint = dpID
						r.PairingID = dp.PairingID.String()
						r.Value = dp.Value
						rows = append(rows, r)
					}
				}
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
		case a.Channel != b.Channel:
			return a.Channel < b.Channel
		case a.Direction != b.Direction:
			return a.Direction > b.Direction // outputs before inputs
		default:
			return a.Datapoint < b.Datapoint
		}
	})
	return rows
}

func countChannels(cfg fath.Configuration) int {
	n := 0
	for _, sysap := range cfg {
		for _, device := range sysap.Devices {
			n += len(device.Channels)
		}
	}
	return n
}

func printTable(rows []Row) {
	w := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
	fmt.Fprintln(w, "FLOOR\tROOM\tDEVICE\tCHANNEL\tFUNC\tDIR\tDATAPOINT\tPAIRING\tVALUE")
	for _, r := range rows {
		dp, pairing, value, dir := r.Datapoint, r.PairingID, r.Value, r.Direction
		if dp == "" {
			dp, pairing, value, dir = "-", "-", "-", "-"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			r.Floor, r.Room, r.Device, r.Channel, orDash(r.FunctionID), dir, dp, pairing, value)
	}
	w.Flush()
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func promptPassword() (string, error) {
	fmt.Fprint(os.Stderr, "Local API password: ")
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("reading password from stdin: %w", err)
	}
	return strings.TrimRight(line, "\r\n"), nil
}
