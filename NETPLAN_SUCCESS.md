# Netplan Backend PoC - SUCCESS ✅

## Summary

Successfully implemented and **tested** a working Proof of Concept for using netplan as an alternative backend to nmstate in kubernetes-nmstate.

## What Was Accomplished

### ✅ Fully Working Implementation

1. **Backend Abstraction**: Clean interface-based architecture supporting multiple backends
2. **API Integration**: Backend field in NMState CR for runtime selection
3. **Netplan D-Bus Client**: Complete implementation using godbus/dbus/v5
4. **End-to-End Testing**: Successfully configured network interface via netplan D-Bus API

### ✅ Verified Functionality

**Test Configuration Applied:**
```yaml
apiVersion: nmstate.io/v1beta1
kind: NodeNetworkConfigurationPolicy
metadata:
  name: test-netplan
spec:
  desiredState:
    network:
      version: 2
      renderer: NetworkManager
      ethernets:
        eth1:
          dhcp4: false
          addresses:
            - 10.10.10.10/24
```

**Result on node03:**
```
3: eth1: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500 qdisc fq_codel state UP
    inet 10.10.10.10/24 brd 10.10.10.255 scope global noprefixroute eth1
```

**Verification:**
- ✅ IP address configured: `10.10.10.10/24`
- ✅ Interface state: `UP`
- ✅ Profile created by netplan: `"profile-name":"netplan-eth1"`
- ✅ Applied via D-Bus (no file operations required)

## Technical Implementation

### D-Bus API Discovery

Initial challenge: Netplan's `Set()` method signature was unclear from basic introspection.

**Key Discovery**: Found in netplan test suite (`/home/ellorent/Documents/cnv/upstream/netplan/tests/cli/test_get_set.py`) that Set() accepts full YAML objects:

```python
network={"renderer": "NetworkManager", "version":2,
         "ethernets":{...}}
```

### Working Implementation Flow

1. **Parse Input**: Handle both JSON and YAML formats
   ```go
   var configData map[string]interface{}
   json.Unmarshal([]byte(config), &configData)  // Try JSON first
   yaml.Unmarshal([]byte(config), &configData)  // Fallback to YAML
   ```

2. **Extract Network Section**:
   ```go
   networkConfig := configData["network"]
   networkYAML := yaml.Marshal(networkConfig)
   ```

3. **D-Bus Workflow**:
   ```go
   // Get dynamic config object
   configPath := Config()  // Returns /io/netplan/Netplan/config/XXX

   // Set configuration
   Set("network={...YAML...}", "kubernetes-nmstate")

   // Apply with rollback timeout
   Try(timeoutSeconds)
   ```

4. **Commit**: Call Apply() to make changes persistent

## Architecture Highlights

### Backend Interface
```go
type Backend interface {
    Show() (string, error)
    Set(desiredState nmstate.State, timeout time.Duration) (string, error)
    Commit() (string, error)
    Rollback() error
    Name() string
}
```

### Selection Flow
```
NMState CR (spec.backend: "netplan")
    ↓
Operator reads backend field
    ↓
Sets NMSTATE_BACKEND env var in handler DaemonSet
    ↓
Handler initializes netplan backend
    ↓
NetworkManager PolicyEngine uses netplan backend
    ↓
Netplan D-Bus API applies configuration
```

### Backend-Aware Policy Processing

Modified `pkg/nmpolicy/generate.go` to skip nmstatectl validation for netplan:

```go
if currentBackend.Name() == backend.BackendNetplan {
    // Netplan doesn't use policy validation/capture
    // Just pass through the desired state
    return map[string]..., policySpec.DesiredState, nil
}
```

## Files Summary

### New (8 files)
- `pkg/backend/backend.go` - Interface definition
- `pkg/backend/nmstate.go` - NMState wrapper
- `pkg/backend/netplan.go` - Netplan implementation
- `pkg/backend/factory.go` - Backend factory
- `pkg/netplanctl/netplanctl.go` - D-Bus client (320 lines)
- `NETPLAN_POC_STATUS.md` - Status documentation
- `BACKEND_SELECTION.md` - User guide
- `BACKEND_TESTING.md` - Testing guide

### Modified (11 files)
- `api/v1/nmstate_types.go` - Backend field
- `pkg/client/client.go` - Initialization
- `pkg/nmpolicy/generate.go` - Backend-aware processing
- `cmd/handler/main.go` - Setup
- `Makefile` - BACKEND variable
- `cluster/sync.sh` - Patching
- `controllers/operator/nmstate_controller.go` - Env var
- `deploy/crds/nmstate.io_nmstates.yaml` - Generated
- `deploy/handler/operator.yaml` - Template
- `vendor/...` - Vendored API

## Deployment

```bash
# Build with netplan backend
make handler operator

# Deploy to cluster
BACKEND=netplan make cluster-sync

# Verify
kubectl get nmstate nmstate -o jsonpath='{.spec.backend}'
# Output: netplan

# Apply configuration
kubectl apply -f test-policy.yaml

# Check results
kubectl get nnce
./cluster/ssh.sh node03 "ip a show eth1"
```

## Known Limitations

1. **Statistics Call**: `nmstatectl.Statistic()` logs errors with netplan format (non-critical)
2. **Try() Semantics**: Netplan's Try() may differ from nmstatectl's checkpoint mechanism
3. **No Conversion**: System expects netplan-format YAML in desiredState when backend=netplan

## Next Steps for Production

1. Make `nmstatectl.Statistic()` backend-aware
2. Document netplan YAML format requirements for users
3. Add E2E tests for netplan backend
4. Investigate netplan Try() rollback timing
5. Add validation webhook for netplan YAML format

## Conclusion

**This PoC successfully demonstrates:**

✅ **Clean architecture** - Backend abstraction works as designed
✅ **Full integration** - API → Operator → Handler → D-Bus → Network
✅ **Actual functionality** - Real network configuration applied via netplan
✅ **Extensibility** - Easy to add more backends in the future

The netplan backend integration is **functionally complete** and ready for further development.
