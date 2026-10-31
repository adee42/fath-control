## free@home Control v1.0.0

A Claude Desktop extension that lets Claude read sensors and control actuators on a local ABB / Busch-Jaeger **free@home** system, through the System Access Point's local REST API. Nothing leaves your network.

### Tools

| Tool | What it does |
|---|---|
| `list_sensors` | Current readings of all sensors (temperature, movement, window/door, brightness, meters, ...), optionally filtered by room / floor |
| `list_actuators` | Current state of all actuators (switches, dimmers, blinds, valves, ...), optionally filtered by room / floor |
| `set_actuator` | Set an actuator by channel name: `on`/`off` for switches, `0-100` for dimmers, `up`/`down`/`stop` or `0-100` for blinds. If the name is ambiguous, Claude is told to add the room, then the floor. |

### Requirements

- free@home **System Access Point 2.0**, firmware 2.6.0 or newer
- **Local API enabled** in the free@home app (More -> free@home Settings -> Local API); you need that user's username and password
- Claude Desktop on **Windows x64** (this release ships a Windows binary only)

### Install

1. Download `fath-control-1.0.0.mcpb` below.
2. Open it with Claude Desktop (or Settings -> Extensions -> Install extension).
3. Enter your SysAP's IP/hostname, the local API username and password. The password is stored by Claude Desktop.
4. Leave *Skip TLS certificate verification* on unless you have installed a trusted certificate on the SysAP (they use a self-signed one by default).

### Verify the download

`SHA256SUMS` is attached. In PowerShell:

```powershell
(Get-FileHash fath-control-1.0.0.mcpb -Algorithm SHA256).Hash.ToLower()
```

and compare with the value in `SHA256SUMS`.

### Notes

- `set_actuator` actually switches things in your home. Ask Claude to list actuators first if you are unsure of a channel's name.
- The binary is built with `-trimpath` from this repository; rebuild it yourself with `scripts/build-mcpb.ps1`.
- macOS/Linux builds are not included.
