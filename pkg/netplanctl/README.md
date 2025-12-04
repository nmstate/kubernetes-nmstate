# netplanctl - Netplan D-Bus Client

This package provides a D-Bus client for interacting with the netplan daemon, similar to how nmstatectl wraps the nmstate library.

## Overview

The netplanctl package communicates with netplan via D-Bus, providing a reliable and consistent interface for network configuration management. This approach mirrors the architecture of the nmstate backend, which uses D-Bus to communicate with NetworkManager.

## D-Bus Architecture

### Service Details

- **Service Name**: `io.netplan.Netplan`
- **Main Object Path**: `/io/netplan/Netplan`
- **Config Object Path**: `/io/netplan/Netplan/config`

### Available Methods

#### Main Interface (`io.netplan.Netplan`)

```
Apply() -> ()
  - Applies the current netplan configuration to the system
  - Generates backend-specific config and activates it

Generate() -> ()
  - Generates backend-specific configuration from netplan YAML
  - Does not apply the configuration

Info() -> (map[string]variant)
  - Returns information about the netplan daemon
  - Useful for version checking and feature detection

Try(config: string, timeout: uint32) -> ()
  - Applies configuration with automatic rollback
  - If not confirmed within timeout seconds, reverts to previous state
  - This is similar to nmstatectl's checkpoint mechanism

Cancel() -> ()
  - Cancels a pending Try operation
  - Immediately reverts to the previous configuration state
```

#### Config Interface (`io.netplan.Netplan.Config`)

```
Get() -> (string)
  - Retrieves the current netplan configuration as YAML

Set(config: string, origin: string) -> ()
  - Sets a new netplan configuration
  - origin parameter identifies the source of the configuration
```

## Usage

### Basic Operations

```go
import "github.com/nmstate/kubernetes-nmstate/pkg/netplanctl"

// Get current configuration
config, err := netplanctl.Show()

// Apply configuration with automatic rollback
desiredState := nmstate.State{Raw: []byte(netplanYAML)}
output, err := netplanctl.Set(desiredState, 120*time.Second)

// Commit the pending configuration
output, err := netplanctl.Commit()

// Rollback/cancel pending configuration
err := netplanctl.Rollback()
```

### Using the Client Directly

```go
client, err := netplanctl.NewNetplanClient()
if err != nil {
    return err
}
defer client.Close()

// Get current config
config, err := client.Get()

// Try configuration with 60 second timeout
err = client.Try(netplanConfig, 60)

// Generate backend configuration
err = client.Generate()

// Apply configuration
err = client.Apply()

// Cancel pending changes
err = client.Cancel()
```

## Transaction Flow

The netplan backend uses a transaction-based flow similar to nmstatectl:

1. **Set (Try)**: Apply configuration with automatic rollback timer
   ```
   Try(config, timeout) via D-Bus
   → netplan daemon applies config
   → Sets timeout for automatic rollback
   ```

2. **Commit**: Make changes permanent
   ```
   Generate() via D-Bus → Generate backend config
   Apply() via D-Bus → Apply to system
   → Clears rollback timer
   ```

3. **Rollback**: Revert to previous state
   ```
   Cancel() via D-Bus
   → netplan daemon reverts changes
   → Previous configuration restored
   ```

## Comparison with nmstatectl

| Feature | nmstatectl | netplanctl |
|---------|-----------|------------|
| Communication | CLI wrapper around libnmstate | D-Bus client |
| Backend | NetworkManager (via D-Bus) | netplan daemon (via D-Bus) |
| Checkpoint | `nmstatectl apply --no-commit` | `Try()` D-Bus method |
| Commit | `nmstatectl commit` | `Generate()` + `Apply()` |
| Rollback | `nmstatectl rollback` | `Cancel()` D-Bus method |
| State Query | `nmstatectl show` | `Get()` D-Bus method |

## Requirements

### System Requirements

- netplan version 0.103 or later (for full D-Bus API support)
- netplan D-Bus service running:
  - `io.netplan.Netplan` service active on system bus
  - Usually provided by `netplan-dbus.service` or integrated with systemd-networkd

### Go Dependencies

```go
require github.com/godbus/dbus/v5 v5.1.0
```

## Error Handling

D-Bus errors are wrapped with context:

```go
output, err := netplanctl.Set(desiredState, timeout)
if err != nil {
    // Error will include D-Bus method name and details
    // Example: "failed to call netplan D-Bus Try method: ..."
}
```

## Debugging

Enable debug mode for verbose output:

```go
netplanctl.SetDebugMode(true)
```

## Testing

When testing, you can:

1. Use a mock D-Bus connection
2. Test against a real netplan daemon in a container/VM
3. Mock the `NetplanClient` interface for unit tests

Example mock:

```go
type MockNetplanClient struct {
    GetFunc func() (string, error)
    TryFunc func(config string, timeout uint32) error
    // ...
}
```

## Security Considerations

- **D-Bus Permissions**: The netplan D-Bus service typically requires root privileges
- **System Bus**: Communication occurs over the system D-Bus, subject to D-Bus policy
- **Privileged Container**: The handler pod needs access to `/run/dbus/system_bus_socket`

## Future Enhancements

1. **Signal Support**: Subscribe to netplan D-Bus signals for state change notifications
2. **Async Operations**: Support asynchronous configuration application
3. **Validation**: Pre-validate netplan YAML before sending to daemon
4. **Format Conversion**: Complete nmstate ↔ netplan format conversion
5. **Feature Detection**: Query netplan version and capabilities via Info() method
