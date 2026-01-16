# Netplan Backend Support for kubernetes-nmstate

## Summary

Add support for using netplan as an alternative backend to nmstate for network configuration in kubernetes-nmstate. The backend can be selected via the NMState CR, enabling users to leverage netplan's D-Bus API for declarative network configuration on cluster nodes.

## Motivation

kubernetes-nmstate currently relies exclusively on nmstate/nmstatectl for network configuration, which in turn depends on NetworkManager. However, many environments use netplan as their primary network configuration tool, particularly on Ubuntu and Ubuntu-derived distributions. By adding netplan as an alternative backend, kubernetes-nmstate can:

1. Support environments where netplan is the preferred or required network configuration tool
2. Reduce dependency conflicts in environments that don't use NetworkManager
3. Enable future support for systemd-networkd as a renderer without requiring NetworkManager
4. Provide a more flexible architecture that can accommodate additional backends

### User Stories

- As a cluster administrator running Ubuntu nodes, I want to use kubernetes-nmstate with netplan instead of NetworkManager so I can maintain consistency with my existing network configuration tooling.
- As a platform engineer, I want to use systemd-networkd as the network renderer while still leveraging kubernetes-nmstate's declarative API and policy management.
- As a developer, I want the ability to experiment with different network configuration backends without modifying the core kubernetes-nmstate API.

### Goals

- Implement a pluggable backend architecture that abstracts network configuration operations
- Add netplan as an alternative backend using its D-Bus API
- Allow runtime backend selection via the NMState CR
- Maintain backward compatibility with existing nmstate backend
- Ensure both backends implement the same core operations: Show, Set, Commit, Rollback

### Non-Goals

- Automatic conversion between nmstate and netplan YAML formats (users must provide format appropriate to their backend)
- Support for all netplan features in the initial implementation
- Deprecation or removal of the nmstate backend
- Backend selection on a per-policy basis (selection is cluster-wide via NMState CR)

## Proposal

### User Roles

**Cluster administrator** is a user responsible for managing cluster node networking and installing operators.

### Workflow Description (Selecting Netplan Backend)

1. The cluster administrator deploys kubernetes-nmstate
2. The cluster administrator edits the NMState CR and sets `spec.backend: netplan`
3. The operator reads the backend field and configures handler DaemonSet pods with the `NMSTATE_BACKEND` environment variable
4. Handler pods initialize the netplan backend on startup
5. When a NodeNetworkConfigurationPolicy is created, the handler uses the netplan D-Bus API to apply network configuration
6. Network state is reported back through the same netplan backend

### Workflow Description (Switching Back to NMState Backend)

1. The cluster administrator edits the NMState CR and sets `spec.backend: nmstate` (or removes the backend field entirely)
2. The operator updates the handler DaemonSet environment variables
3. Handler pods restart and initialize the nmstate backend
4. Subsequent policies are applied via nmstatectl

### Architecture

#### Backend Abstraction Layer

A new `pkg/backend/` package provides the core abstraction:

```go
// Backend defines the interface for network configuration backends
type Backend interface {
    // Show returns the current network state
    Show() (string, error)

    // Set applies the desired network state with a timeout
    Set(desiredState nmstate.State, timeout time.Duration) (string, error)

    // Commit commits the pending network configuration changes
    Commit() (string, error)

    // Rollback rolls back pending network configuration changes
    Rollback() error

    // Name returns the backend name
    Name() string
}
```

Implementations:
- **NMStateBackend** (`pkg/backend/nmstate.go`): Wraps existing nmstatectl functionality
- **NetplanBackend** (`pkg/backend/netplan.go`): Uses netplan D-Bus API via pkg/netplanctl
- **Factory** (`pkg/backend/factory.go`): Creates appropriate backend based on configuration

#### Netplan D-Bus Client

A new `pkg/netplanctl/` package implements D-Bus communication with the netplan daemon:

**D-Bus Service Details:**
- Service Name: `io.netplan.Netplan`
- Object Path: `/io/netplan/Netplan`
- Interface: `io.netplan.Netplan`
- Config Object: `/io/netplan/Netplan/config/XXX` (dynamic, obtained via Config() method)

**Core Methods:**
- `Config()`: Returns a dynamic config object path for configuration operations
- `Set(config, origin)`: Sets network configuration (on config object)
- `Try(timeout)`: Applies configuration with automatic rollback after timeout (on config object)
- `Apply()`: Makes configuration persistent (on config object)
- `Cancel()`: Cancels pending Try operation (on config object)
- `Generate()`: Generates backend-specific configuration files
- `Info()`: Returns netplan daemon information
- `Status()`: Returns current network state (via netplan status command)

**Operation Flow:**

1. **Show()**: Calls `netplan status` to get current network state
2. **Set()**:
   - Calls `Config()` to get a dynamic config object
   - Extracts network section from desired state
   - Calls `Set()` on config object with network configuration
   - Calls `Try(timeout)` on config object for automatic rollback protection
3. **Commit()**:
   - Gets config object
   - Calls `Apply()` on config object to make changes persistent
4. **Rollback()**:
   - Gets config object
   - Calls `Cancel()` on config object to cancel pending Try operation

This design mirrors nmstatectl's checkpoint mechanism, providing automatic rollback if configuration is not committed within the timeout period.

#### Backend Selection Flow

```
NMState CR (spec.backend: "netplan")
    ↓
Operator (controllers/operator/nmstate_controller.go)
    ↓
Sets NMSTATE_BACKEND env var in handler DaemonSet
    ↓
Handler (cmd/handler/main.go)
    ↓
Initializes backend via pkg/backend.InitBackend()
    ↓
Policy Controllers use backend interface
    ↓
Network configuration applied via selected backend
```

#### Modified Components

1. **API (`api/v1/nmstate_types.go`)**
   - Added `Backend` field to `NMStateSpec`
   - Field is optional, defaults to "nmstate"
   - Validated via kubebuilder enum: `nmstate` or `netplan`

2. **Client Package (`pkg/client/client.go`)**
   - Modified to use backend interface instead of direct nmstatectl calls
   - Added `InitBackend()` to initialize backend based on environment variable
   - Updated `ApplyDesiredState()` and `rollback()` to use backend interface

3. **Handler (`cmd/handler/main.go`)**
   - Added backend initialization in `setupHandlerEnvironment()`

4. **Operator (`controllers/operator/nmstate_controller.go`)**
   - Modified `applyHandler()` to pass backend selection from NMState CR to handler DaemonSet via environment variable

5. **DaemonSet Template (`deploy/handler/operator.yaml`)**
   - Added `NMSTATE_BACKEND` environment variable

6. **Policy Generation (`pkg/nmpolicy/generate.go`)**
   - Made backend-aware to skip nmstatectl validation for netplan
   - Netplan backend passes through desired state without nmstatectl capture/validation

### API Extensions

The NMState CR gets a new optional `backend` field:

```yaml
apiVersion: nmstate.io/v1
kind: NMState
metadata:
  name: nmstate
spec:
  backend: netplan  # or "nmstate" (default)
```

No changes to NodeNetworkConfigurationPolicy API are required. However, when using the netplan backend, the `desiredState` should contain netplan-format YAML:

```yaml
apiVersion: nmstate.io/v1beta1
kind: NodeNetworkConfigurationPolicy
metadata:
  name: test-policy
spec:
  desiredState:
    network:
      version: 2
      renderer: NetworkManager  # or systemd-networkd
      ethernets:
        eth1:
          dhcp4: false
          addresses:
            - 10.10.10.10/24
```

### Implementation Details

#### D-Bus Communication

The handler DaemonSet already has access to the D-Bus system bus via volume mount:

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

This enables the netplan backend to communicate with the netplan daemon without additional configuration.

#### Automatic Rollback Mechanism

Both backends support automatic rollback:

| Aspect | nmstate Backend | netplan Backend |
|--------|----------------|-----------------|
| Checkpoint API | nmstatectl create-checkpoint | netplan Try() |
| Timeout | Specified in Set() | Specified in Try() |
| Commit | nmstatectl commit | netplan Apply() |
| Rollback | nmstatectl rollback / automatic timeout | netplan Cancel() / automatic timeout |
| Implementation | NetworkManager checkpoint via D-Bus | netplan Try() via D-Bus |

#### Backend Initialization

Backend selection happens at handler startup:

```go
// In cmd/handler/main.go
func setupHandlerEnvironment() error {
    // Initialize the selected backend
    if err := backend.InitBackend(); err != nil {
        return err
    }
    // ...
}
```

The backend is determined by the `NMSTATE_BACKEND` environment variable, which is set by the operator based on the NMState CR's `spec.backend` field.

#### Fallback Behavior

If backend initialization fails or an invalid backend is specified, the system falls back to the nmstate backend to maintain availability:

```go
func InitBackend() error {
    backendType := os.Getenv("NMSTATE_BACKEND")
    if backendType == "" {
        backendType = BackendNMState
    }

    var err error
    configBackend, err = NewBackend(backendType)
    if err != nil {
        log.Error(err, "Failed to initialize backend, falling back to nmstate")
        configBackend = NewNMStateBackend()
    }

    log.Info("Initialized network configuration backend", "backend", configBackend.Name())
    return nil
}
```

### Dependencies

#### Go Module Dependencies

The netplan backend requires the `godbus/dbus/v5` library for D-Bus communication:

```
require (
    github.com/godbus/dbus/v5 v5.1.0
)
```

This dependency is already commonly used in Kubernetes projects.

#### Node Dependencies

**For nmstate backend (no change):**
- NetworkManager >= 1.22
- nmstatectl
- D-Bus system bus

**For netplan backend (new requirements):**
- netplan >= 0.103 (for full D-Bus API support)
- netplan D-Bus service running (`systemctl status netplan-dbus`)
- D-Bus system bus (already required)

### Build System Integration

The Makefile supports backend selection for development clusters:

```bash
# Deploy with default nmstate backend
make cluster-sync

# Deploy with netplan backend
BACKEND=netplan make cluster-sync
```

The `cluster/sync.sh` script patches the NMState CR with the selected backend:

```bash
function patch_handler_backend() {
    if [ -n "${BACKEND}" ] && [ "${BACKEND}" != "nmstate" ]; then
        echo "Patching NMState CR to use backend: ${BACKEND}"
        kubectl patch nmstate nmstate \
          --patch "{\"spec\": {\"backend\": \"${BACKEND}\"}}" \
          --type=merge
    fi
}
```

### Limitations and Future Work

#### Current Limitations

1. **No Format Conversion**: The implementation does not convert between nmstate and netplan YAML formats. Users must provide configuration in the appropriate format for their selected backend.

2. **Statistics Call**: The `nmstatectl.Statistic()` call logs errors with netplan format (non-critical, only affects logging).

3. **State Format**: The `Show()` method returns netplan-format state when using netplan backend, not nmstate format.

4. **Testing**: Limited E2E test coverage for netplan backend scenarios.

#### Future Enhancements

1. **State Format Conversion**
   - Implement bidirectional conversion between nmstate and netplan formats
   - Allow users to use nmstate format regardless of backend
   - Support all network features (interfaces, routes, DNS, bonds, bridges, VLANs, etc.)

2. **Validation Webhook**
   - Add backend-aware validation
   - Validate netplan YAML syntax before applying
   - Provide helpful error messages for format mismatches

3. **Enhanced Testing**
   - Add E2E tests for netplan backend
   - Test various network configurations (bonds, bridges, VLANs, routes)
   - Test rollback scenarios
   - Test backend switching

4. **Monitoring & Metrics**
   - Add backend-specific metrics
   - Track conversion errors
   - Monitor backend operation success/failure rates

5. **Documentation**
   - User guide for netplan backend
   - Migration guide from nmstate to netplan
   - Troubleshooting guide
   - Format conversion reference

6. **Additional Backends**
   - systemd-networkd (direct, without netplan)
   - iproute2-only backend
   - Other network configuration tools

### Alternatives Considered

#### Alternative 1: Automatic Format Detection and Conversion

**Approach:** Automatically detect whether the input is nmstate or netplan format and convert as needed.

**Pros:**
- Better user experience - users don't need to know backend details
- Easier migration between backends

**Cons:**
- Complex conversion logic required
- Potential for conversion errors or unsupported features
- Performance overhead for parsing and conversion
- Maintenance burden for keeping conversion logic in sync

**Decision:** Rejected for initial implementation. Format conversion can be added as a future enhancement once the basic architecture is proven.

#### Alternative 2: Per-Policy Backend Selection

**Approach:** Allow backend selection per NodeNetworkConfigurationPolicy instead of cluster-wide.

**Pros:**
- More flexible - different policies could use different backends
- Easier gradual migration

**Cons:**
- More complex implementation
- Potential for conflicting configurations
- Harder to reason about system state
- Increased testing complexity

**Decision:** Rejected. Cluster-wide backend selection is simpler and sufficient for the use cases identified. Per-policy selection can be considered in the future if needed.

#### Alternative 3: CLI-Based Netplan Integration

**Approach:** Use `netplan` CLI commands instead of D-Bus API.

**Pros:**
- Simpler implementation (no D-Bus client needed)
- No additional Go dependencies

**Cons:**
- Less reliable than D-Bus (parsing CLI output)
- No atomic operations
- No built-in rollback mechanism
- File-based race conditions
- Inconsistent with nmstate/NetworkManager architecture

**Decision:** Rejected. D-Bus approach provides better reliability and consistency with existing nmstate backend design.

### Testing

#### Unit Tests

- Backend interface implementations
- Backend factory
- Netplan D-Bus client (with mocked D-Bus)
- Backend initialization logic

#### E2E Tests

The following test scenarios should be covered:

1. **Backend Selection**
   - Deploy with nmstate backend (default)
   - Deploy with netplan backend
   - Switch between backends

2. **Network Configuration**
   - Apply simple interface configuration
   - Apply static IP configuration
   - Apply DHCP configuration
   - Apply complex configurations (bonds, bridges, VLANs)

3. **Rollback Scenarios**
   - Configuration that breaks connectivity triggers automatic rollback
   - Manual rollback via Cancel
   - Timeout-based rollback

4. **Error Handling**
   - Invalid configuration
   - D-Bus service unavailable
   - Backend initialization failure

### Security Considerations

1. **Privileged Access**: Handler runs with `privileged: true` and can modify host network configuration via D-Bus (no change from current behavior)

2. **D-Bus System Bus**: Communication occurs over system D-Bus
   - Netplan D-Bus service runs as root
   - Handler pod has access to system bus socket via volume mount
   - D-Bus policy files control method call permissions

3. **No Direct File Access**: Unlike file-based approaches, the D-Bus method doesn't require the handler to directly modify `/etc/netplan/` files, reducing potential security issues

4. **Atomic Operations**: D-Bus calls are atomic, reducing the window for race conditions or partial state changes

5. **RBAC**: No changes to RBAC permissions required - handler already has necessary cluster permissions

### Risks and Mitigations

| Risk | Impact | Likelihood | Mitigation |
|------|--------|------------|-----------|
| Netplan D-Bus API changes in future versions | High | Low | Pin to minimum version (0.103), add version detection |
| Format conversion errors | Medium | High (if implemented) | Extensive testing, validation, clear error messages |
| Backend switching causes downtime | Medium | Medium | Document switching procedure, recommend during maintenance windows |
| Netplan not available on nodes | High | Medium | Clear documentation of requirements, validation at startup |
| D-Bus communication failures | High | Low | Retry logic, clear error messages, fallback to nmstate |

## Implementation Plan

### Phase 1: Core Architecture (Completed in PoC)
- [x] Backend interface definition
- [x] NMState backend wrapper
- [x] Netplan backend implementation
- [x] Backend factory
- [x] Netplan D-Bus client
- [x] API changes (Backend field in NMState CR)
- [x] Operator integration
- [x] Handler integration
- [x] Build system integration

### Phase 2: Testing and Validation
- [ ] Unit tests for backend implementations
- [ ] Unit tests for netplan D-Bus client
- [ ] E2E tests for netplan backend
- [ ] E2E tests for backend switching
- [ ] Integration tests with real netplan daemon

### Phase 3: Documentation and Polish
- [ ] User documentation for backend selection
- [ ] API documentation updates
- [ ] Troubleshooting guide
- [ ] Make statistics call backend-aware
- [ ] Add validation for backend configuration

### Phase 4: Future Enhancements (Optional)
- [ ] Format conversion (nmstate ↔ netplan)
- [ ] Enhanced validation webhook
- [ ] Backend-specific metrics
- [ ] Additional backends (systemd-networkd, etc.)

## Graduation Criteria

### Alpha (Current State)
- Basic backend architecture implemented
- Netplan backend functional with D-Bus API
- Manual testing demonstrates successful network configuration
- Documentation exists for developers

### Beta
- Unit test coverage > 80% for backend package
- E2E tests for netplan backend scenarios
- User-facing documentation complete
- Successfully tested on multiple environments (Ubuntu, Fedora with netplan)
- Backend switching tested and documented

### GA
- Production deployments using netplan backend
- Format conversion implemented (if needed based on user feedback)
- Comprehensive metrics and monitoring
- All known bugs resolved
- Performance validated in large-scale deployments

## References

- [Netplan D-Bus API Documentation](https://netplan.readthedocs.io/en/stable/dbus-api/)
- [Netplan YAML Format Reference](https://netplan.readthedocs.io/en/stable/netplan-yaml/)
- [nmstate Documentation](https://nmstate.io/)
- [kubernetes-nmstate GitHub Repository](https://github.com/nmstate/kubernetes-nmstate)
- [D-Bus Specification](https://dbus.freedesktop.org/doc/dbus-specification.html)
