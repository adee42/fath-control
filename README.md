# fath-control

Read sensors and control actuators on an ABB / Busch-Jaeger **free@home** smart home system through the System Access Point's local REST API. Everything stays on your local network.

This repository contains two programs that share one Go library:

| Program | What it is |
|---|---|
| **Claude Desktop extension** (`fath-mcp-server`) | An [MCP](https://modelcontextprotocol.io) server packaged as a `.mcpb` extension, so Claude can list your sensors and actuators and switch things for you. |
| **`fath-control` command-line tool** | Lists sensors/actuators with their current state and sets actuator values from a terminal. |

## Download

Get the latest extension from the **[Releases page](https://github.com/adee42/fath-control/releases/latest)** ([all releases](https://github.com/adee42/fath-control/releases)). Each release includes a `.mcpb` file and a `SHA256SUMS` file for verifying the download.

## Requirements

- free@home **System Access Point 2.0**, firmware 2.6.0 or newer
- **Local API enabled** in the free@home app: *More -> free@home Settings -> Local API*. Note the username shown there and the password you set.
- For the extension: Claude Desktop on **Windows x64**, **macOS (Apple Silicon)** or **Linux x64**. The macOS build has not been run and the macOS/Linux installs through Claude Desktop are untested (see the release notes). Intel Macs and ARM Linux are not covered.
- For the CLI: Go (see `go.mod`) to build from source; it is pure Go and runs wherever Go does

## Claude Desktop extension

### Install

1. Download `fath-control-<version>.mcpb` from the [latest release](https://github.com/adee42/fath-control/releases/latest).
2. Open it with Claude Desktop (or *Settings -> Extensions -> Install extension*).
3. Enter your SysAP's IP/hostname, the local API username and password. Claude Desktop stores the password.
4. Leave *Skip TLS certificate verification* on unless you installed a trusted certificate on the SysAP; they use a self-signed one by default.

### Tools Claude gets

| Tool | What it does |
|---|---|
| `list_sensors` | Current readings of all sensors (temperature, movement, window/door, brightness, energy/water/gas meters, ...). Optional `room` / `floor` filters. |
| `list_actuators` | Current state of all actuators (switches, lights, dimmers, blinds/shutters, heating/cooling valves, ...). Optional `room` / `floor` filters. |
| `set_actuator` | Sets an actuator by its channel name. Optional `room` / `floor` to disambiguate; optional `input_pairing_id` for advanced use. |

Example requests: *"Which windows are open?"*, *"What's the temperature in the bedroom?"*, *"Set the living room blind to 40%"*, *"Turn off the kitchen light."*

> `set_actuator` really switches things in your home. If you're unsure of a channel's name, have Claude list the actuators first.

### Accepted values

| Actuator type | Values |
|---|---|
| Switch / light / staircase light | `on`, `off` (also `1`/`0`, `true`/`false`, `yes`/`no`) |
| Dimmer | `0`-`100` (with or without `%`), or `on` / `off` |
| Blind / shutter / awning / attic window | `up`, `down`, `stop`, or `0`-`100` for the position |
| Anything else | Raw text, if the channel has exactly one input datapoint (otherwise pass `input_pairing_id`) |

### Which channel gets controlled?

The channel name is the primary identifier. If several actuators share that name, the **room** is used to narrow it down; if name and room still match more than one, the **floor** is used. If it is still ambiguous, you get the list of candidates and are asked to be more specific. Matching is case-insensitive, and only actuators are ever matched (sensors can't be set).

## Command-line tool

Build and run:

```bash
go build -o fath-control .
./fath-control -host 192.168.1.50 -user <local-api-user> -password '<password>'
```

The connection can also come from the environment: `FATH_HOST`, `FATH_USER`, `FATH_PASSWORD`. If no password is given you are prompted for it.

```bash
# every datapoint of every channel (raw view; add -show-inputs for command datapoints)
fath-control -host 192.168.1.50 -user <user>

# actuators only, one row per channel, with a readable function name and state
fath-control -host 192.168.1.50 -user <user> -actuators-only

# sensors only
fath-control -host 192.168.1.50 -user <user> -sensors-only

# set a value (room / floor only needed if the name isn't unique)
fath-control -host 192.168.1.50 -user <user> -set "Reading Lamp" -value 70%
fath-control -host 192.168.1.50 -user <user> -set "Light" -room Kitchen -value on
fath-control -host 192.168.1.50 -user <user> -set "Blind" -value stop
```

Example output of `-actuators-only`:

```
FLOOR         ROOM         DEVICE            CHANNEL        FUNCTION          STATE
Ground Floor  Living Room  Dimming Actuator  Reading Lamp   Dimming Actuator  70%
Ground Floor  Living Room  Switch Actuator   Ceiling Light  Switch Actuator   On
```

| Flag | Meaning |
|---|---|
| `-host`, `-user`, `-password` | SysAP address and local API credentials |
| `-actuators-only` / `-sensors-only` | One row per actuator / sensor channel with function name and current state |
| `-set <channel>` `-value <v>` | Write a value to an actuator |
| `-room`, `-floor` | Disambiguate `-set` when the channel name isn't unique |
| `-input <pairingID>` | Advanced: write to a specific input datapoint, bypassing type detection |
| `-json` | JSON output for scripting |
| `-raw` | Print the SysAP's unmodified configuration JSON (useful for debugging) |
| `-show-inputs` | Also list input (command) datapoints in the default view |
| `-scheme`, `-insecure`, `-timeout`, `-api-base` | Connection options (defaults: `https`, skip certificate check, 10s, `/fhapi/v1/api/rest`) |

Run `fath-control -h` for the full list.

## How it works

The SysAP exposes its configuration at `/fhapi/v1/api/rest/configuration` as devices -> channels -> input/output datapoints. A channel's **function ID** says what it is (switch actuator, temperature sensor, ...) and each datapoint's **pairing ID** says what it means (on/off, dimming value, blind position, ...). This project maps those IDs, using the tables in ABB's [local API reference](https://developer.eu.mybuildings.abb.com/fah_local), to readable names, to values like `On` / `70%` / `18.5 °C`, and to the right input datapoint when you set a value.

A single channel usually reports several output datapoints (for example a switch also reports forced state, group state and error flags). The tools show the one that represents the channel's actual state; use `-raw` or the default CLI view to see all of them.

The pairing ID table covers the commonly used IDs; anything unmapped is shown as its raw value.

## Building from source

```bash
go test ./...
go build ./...
```

To build the extension bundle (works on any OS with Go; it cross-compiles the server for Windows x64, macOS arm64 and Linux x64):

```bash
go run ./scripts/mcpb      # writes bin/fath-control-<version>.mcpb and bin/SHA256SUMS
```

The build is reproducible: the same source and Go version give a byte-identical bundle.

## Disclaimer

This is an independent project and is not affiliated with or endorsed by ABB or Busch-Jaeger. *free@home* is a trademark of its owner. Use at your own risk; the tool sends real commands to your installation.

## License

[MIT](LICENSE)
