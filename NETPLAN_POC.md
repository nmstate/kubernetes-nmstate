# Netplan Backend PoC for kubernetes-nmstate

## Overview

This Proof of Concept (PoC) implements support for using netplan as an alternative backend to nmstate for network configuration in kubernetes-nmstate. The backend can be selected via the NMState CR.

**Key Feature**: This implementation uses netplan's **D-Bus interface** for communication, similar to how the nmstate backend communicates with NetworkManager via D-Bus. This provides a more reliable and consistent integration compared to CLI-based approaches.

## Architecture

### Components Modified

1. **API (api/v1/nmstate_types.go)**
   - Added `Backend` field to `NMStateSpec` with support for "nmstate" (default) and "netplan"
   - Field is validated via kubebuilder enum validation

2. **Backend Interface (pkg/backend/)**
   - `backend.go`: Defines the `Backend` interface abstracting network configuration operations
   - `nmstate.go`: Implements the interface wrapping existing nmstatectl functionality
   - `netplan.go`: Implements the interface using netplan D-Bus API via pkg/netplanctl
   - `factory.go`: Factory function to create appropriate backend based on configuration

3. **Netplan D-Bus Client (pkg/netplanctl/)**
   - `netplanctl.go`: D-Bus client for netplan communication
   - Connects to `io.netplan.Netplan` D-Bus service on system bus
   - Implements methods: Get(), Set(), Try(), Apply(), Generate(), Cancel()
   - Provides helper functions matching nmstatectl interface: Show(), Set(), Commit(), Rollback()
   - Uses godbus/dbus/v5 for D-Bus communication

4. **Client Package (pkg/client/client.go)**
   - Modified to use the backend interface instead of directly calling nmstatectl
   - Added `InitBackend()` to initialize backend based on `NMSTATE_BACKEND` environment variable
   - Updated `ApplyDesiredState()` and `rollback()` to use the backend interface

5. **Handler (cmd/handler/main.go)**
   - Added backend initialization in `setupHandlerEnvironment()`

6. **Operator (controllers/operator/nmstate_controller.go)**
   - Modified `applyHandler()` to pass backend selection from NMState CR to handler DaemonSet

7. **DaemonSet Template (deploy/handler/operator.yaml)**
   - Added `NMSTATE_BACKEND` environment variable to pass backend selection to handler pods

## Usage

### Selecting the Backend

In your NMState CR, add the `backend` field:

```yaml
apiVersion: nmstate.io/v1
kind: NMState
metadata:
  name: nmstate
spec:
  backend: netplan  # or "nmstate" (default)
```

### Backend Operations

Both backends implement the following operations:

- **Show()**: Retrieve current network state
- **Set()**: Apply desired network state with timeout
- **Commit()**: Commit pending network changes
- **Rollback()**: Rollback pending network changes

### Netplan Backend Implementation Details

The netplan backend uses **D-Bus for all communication** with the netplan daemon, providing the same reliability and consistency as the nmstate/NetworkManager integration.

#### D-Bus Service Details
- **Service Name**: `io.netplan.Netplan`
- **Object Path**: `/io/netplan/Netplan`
- **Interface**: `io.netplan.Netplan`
- **Config Object**: `/io/netplan/Netplan/config` (io.netplan.Netplan.Config interface)

#### Operation Flow via D-Bus

1. **Show** (`pkg/netplanctl/netplanctl.go`):
   - Calls `io.netplan.Netplan.Config.Get()` via D-Bus
   - Returns current netplan configuration in YAML format

2. **Set**:
   - Calls `io.netplan.Netplan.Try(config, timeout)` via D-Bus
   - This is similar to nmstatectl's checkpoint mechanism
   - Netplan daemon automatically rolls back after timeout if not committed
   - No file-based backup needed - handled entirely by netplan daemon

3. **Commit**:
   - Calls `io.netplan.Netplan.Generate()` to generate backend configuration
   - Calls `io.netplan.Netplan.Apply()` to apply the configuration
   - Makes the pending configuration permanent

4. **Rollback**:
   - Calls `io.netplan.Netplan.Cancel()` via D-Bus
   - Cancels any pending Try operation
   - Netplan daemon handles restoration of previous state

#### Advantages of D-Bus Approach

- **Consistency**: Same communication pattern as nmstate/NetworkManager
- **Reliability**: No race conditions with file-based approaches
- **Atomic Operations**: Configuration changes are atomic
- **Built-in Rollback**: Netplan's Try mechanism provides automatic rollback
- **No Root Filesystem Access**: No need to manage config files directly
- **Event Notifications**: Can subscribe to D-Bus signals for state changes (future enhancement)

## Limitations and Future Work

### Current Limitations

1. **State Format Conversion**: The PoC includes a placeholder `convertNMStateToNetplan()` function that needs full implementation. Currently, it doesn't actually convert nmstate YAML to netplan YAML format.

2. **Limited netplan Features**: The implementation covers basic operations but doesn't handle all netplan features:
   - Complex network topologies
   - VLANs, bridges, bonds
   - Advanced routing
   - DNS configuration

3. **State Reporting**: The `Show()` method returns netplan format, not nmstate format. For full compatibility, it should convert netplan state to nmstate format.

4. **Dependencies**: Nodes using the netplan backend must have:
   - netplan installed (version 0.103+ for full D-Bus API support)
   - netplan D-Bus service running (`netplan-dbus.service` or `systemd-networkd-wait-online.service` depending on distribution)
   - D-Bus system bus accessible from handler pod (already available via `/run/dbus/system_bus_socket` volume mount)

5. **Testing**: The netplan backend needs comprehensive testing including:
   - Unit tests
   - E2E tests
   - Different network configurations
   - Error handling scenarios

### Future Enhancements

1. **State Format Conversion**
   - Implement complete nmstate ↔ netplan format conversion
   - Support all network features (interfaces, routes, DNS, etc.)
   - Handle edge cases and validation

2. **Container Image**
   - Update handler container image to include netplan dependencies
   - Add netplan installation to Containerfile/Dockerfile

3. **Validation**
   - Add webhook validation for netplan-specific configuration
   - Validate netplan YAML syntax before applying

4. **Monitoring & Metrics**
   - Add backend-specific metrics
   - Track conversion errors and backend operations

5. **Documentation**
   - User guide for netplan backend
   - Migration guide from nmstate to netplan
   - Troubleshooting guide

6. **Feature Parity**
   - Ensure netplan backend supports all features available in nmstate backend
   - Handle NetworkManager-specific features when using netplan

## Testing the PoC

For detailed information about backend selection and testing, see [BACKEND_SELECTION.md](BACKEND_SELECTION.md).

### Prerequisites

1. Kubernetes cluster with nodes that have netplan installed
2. kubernetes-nmstate deployed

### Quick Start

1. **Generate CRDs** (after modifying the API):
   ```bash
   make gen-crds
   ```

2. **Build and Deploy with nmstate backend (default)**:
   ```bash
   make handler
   make operator
   make cluster-sync
   ```

3. **Build and Deploy with netplan backend**:
   ```bash
   make handler
   make operator
   BACKEND=netplan make cluster-sync
   ```

   Or set the variable before make:
   ```bash
   export BACKEND=netplan
   make cluster-sync
   ```

4. **Alternatively, create NMState CR manually with netplan backend**:
   ```yaml
   apiVersion: nmstate.io/v1
   kind: NMState
   metadata:
     name: nmstate
   spec:
     backend: netplan
   ```

5. **Apply a NodeNetworkConfigurationPolicy**:
   ```yaml
   apiVersion: nmstate.io/v1beta1
   kind: NodeNetworkConfigurationPolicy
   metadata:
     name: test-policy
   spec:
     desiredState:
       # Note: This needs to be in a format the netplan backend can handle
       # or implement conversion logic
   ```

6. **Monitor Handler Logs**:
   ```bash
   kubectl logs -n nmstate -l component=kubernetes-nmstate-handler
   ```

## D-Bus Integration Details

### Handler Pod D-Bus Access

The handler DaemonSet already has access to the D-Bus system bus via the volume mount:

```yaml
volumeMounts:
  - name: dbus-socket
    mountPath: /run/dbus/system_bus_socket
volumes:
  - name: dbus-socket
    hostPath:
      path: /run/dbus/system_bus_socket
      type: Socket
```

This means the netplan D-Bus client can communicate with the netplan daemon without any additional configuration.

### D-Bus Method Signatures

The netplan D-Bus API provides the following methods (from netplan source code):

```
io.netplan.Netplan:
  - Apply() -> ()
  - Generate() -> ()
  - Info() -> (map[string]variant)
  - Try(config: string, timeout: uint32) -> ()
  - Cancel() -> ()

io.netplan.Netplan.Config:
  - Get() -> (string)
  - Set(config: string, origin: string) -> ()
```

### Comparison with nmstate/NetworkManager

Both backends use D-Bus for communication:

| Aspect | nmstate Backend | netplan Backend |
|--------|----------------|-----------------|
| D-Bus Service | `org.freedesktop.NetworkManager` | `io.netplan.Netplan` |
| Configuration Tool | nmstatectl (wraps libnmstate) | netplanctl (D-Bus client) |
| Checkpoint/Try | nmstatectl checkpoint API | netplan Try() method |
| Apply | nmstatectl commit | netplan Generate() + Apply() |
| Rollback | nmstatectl rollback | netplan Cancel() |
| Backend Renderer | NetworkManager | NetworkManager or systemd-networkd |

## Files Modified/Added

### Added Files
- `pkg/backend/backend.go` - Backend interface definition
- `pkg/backend/nmstate.go` - NMState backend implementation
- `pkg/backend/netplan.go` - Netplan backend implementation (D-Bus based)
- `pkg/backend/factory.go` - Backend factory
- `pkg/netplanctl/netplanctl.go` - Netplan D-Bus client implementation
- `examples/nmstate-netplan-backend.yaml` - Example NMState CR
- `NETPLAN_POC.md` - This documentation

### Modified Files
- `api/v1/nmstate_types.go` - Added Backend field to NMStateSpec
- `pkg/client/client.go` - Use backend interface instead of direct nmstatectl calls
- `cmd/handler/main.go` - Initialize backend on handler startup
- `controllers/operator/nmstate_controller.go` - Pass backend config to handler
- `deploy/handler/operator.yaml` - Add NMSTATE_BACKEND env var to handler DaemonSet

## Implementation Notes

### Design Decisions

1. **D-Bus Communication**: The netplan backend uses D-Bus for all interactions with the netplan daemon, matching the architecture of the nmstate backend's communication with NetworkManager. This provides:
   - Consistent integration pattern across backends
   - Better reliability than CLI-based approaches
   - Atomic operations and built-in transaction support
   - No file system race conditions

2. **Environment Variable Approach**: The backend selection is passed via environment variable (`NMSTATE_BACKEND`) from the operator to handler pods. This is simpler than having handlers query the NMState CR directly.

3. **Backend Interface**: Using an interface allows easy addition of more backends in the future (e.g., systemd-networkd, iproute2-only, etc.).

4. **Fallback Behavior**: If backend initialization fails or an invalid backend is specified, the system falls back to the nmstate backend to maintain availability.

5. **Netplan Try Mechanism**: The implementation uses netplan's built-in `Try()` method which provides automatic rollback after timeout, similar to nmstatectl's checkpoint mechanism. No manual file-based backup is needed.

### Security Considerations

1. **Privileged Access**: The handler runs with `privileged: true` and can modify host network configuration via D-Bus
2. **D-Bus System Bus**: Communication occurs over the system D-Bus, which requires appropriate permissions:
   - The netplan D-Bus service runs as root
   - Handler pod has access to system bus socket via volume mount
   - D-Bus policy files control which processes can call netplan methods
3. **No Direct File Access**: Unlike file-based approaches, the D-Bus method doesn't require the handler to directly modify `/etc/netplan/` files, reducing potential security issues
4. **Atomic Operations**: D-Bus calls are atomic, reducing the window for race conditions or partial state changes

## Dependencies

### Go Module Dependencies

The netplan backend requires the `godbus/dbus/v5` library for D-Bus communication:

```bash
go get github.com/godbus/dbus/v5
```

This should be added to `go.mod`:
```
require (
    ...
    github.com/godbus/dbus/v5 v5.1.0
    ...
)
```

## Contributing

To extend this PoC:

1. Implement `convertNMStateToNetplan()` function in `pkg/netplanctl/netplanctl.go`
2. Add unit tests for the backend implementations:
   - Mock D-Bus interfaces for testing
   - Test format conversion logic
   - Test error handling
3. Add E2E tests for netplan backend:
   - Test with real netplan daemon
   - Test rollback scenarios
   - Test various network configurations
4. Update handler container image to include:
   - netplan package
   - godbus/dbus library (bundled in Go binary)
5. Add validation for netplan-specific configuration
6. Add D-Bus signal handling for netplan state change notifications

## Questions and Feedback

For questions or feedback about this PoC, please open an issue describing:
- Use case for netplan backend
- Required features
- Platform/environment details
