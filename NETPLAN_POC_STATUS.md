# Netplan Backend PoC - Status Report

## Overview
This is a Proof of Concept for adding netplan as an alternative backend to nmstate in kubernetes-nmstate. The backend is selectable via the NMState CR.

## Architecture

### Backend Abstraction Layer
Created a pluggable backend interface in `pkg/backend/`:
- `backend.go` - Interface definition with Show(), Set(), Commit(), Rollback() methods
- `nmstate.go` - Wrapper for existing nmstatectl backend
- `netplan.go` - New netplan backend implementation
- `factory.go` - Backend factory with type selection

### API Changes
- Added `Backend` field to `NMStateSpec` in `api/v1/nmstate_types.go`
- Field is optional, defaults to "nmstate"
- Validated enum: `nmstate` or `netplan`

### Netplan D-Bus Client
Implemented in `pkg/netplanctl/netplanctl.go`:
- Uses godbus/dbus/v5 for D-Bus communication
- Connects to system D-Bus and calls `io.netplan.Netplan` service
- Implements Config() workflow to get dynamic config object paths
- **Key Discovery**: netplan's Set() D-Bus method expects dot-notation key=value pairs (like `ethernets.eth0.dhcp4=true`), NOT full YAML documents
- **Solution**: File-based approach - write YAML to `/etc/netplan/90-kubernetes-nmstate.yaml`, then call Generate() and Apply()

### Backend Selection Flow
1. User sets `spec.backend: netplan` in NMState CR
2. Operator reads backend field and passes to handler via `NMSTATE_BACKEND` env var
3. Handler initializes appropriate backend in `pkg/client/client.go:InitBackend()`
4. Policy generation (`pkg/nmpolicy/generate.go`) skips nmstatectl validation for netplan
5. Network operations use selected backend

### Build System Integration
- Added `BACKEND` Makefile variable (default: nmstate)
- Usage: `BACKEND=netplan make cluster-sync`
- Script `cluster/sync.sh` patches NMState CR with selected backend

## Implementation Status

### ✅ Completed
1. Backend abstraction interface
2. API changes with backend field
3. Netplan D-Bus client with correct API workflow
4. Backend-aware policy generation (skips nmstatectl for netplan)
5. Deployment integration (Makefile + sync script)
6. Backend initialization in handler
7. File-based netplan configuration approach

### ⚠️ Known Limitations

#### 1. Host Filesystem Access
The handler pods need to write to `/etc/netplan/` on the host node. Current implementation uses `os.WriteFile("/etc/netplan/90-kubernetes-nmstate.yaml", ...)` which requires:
- hostPath volume mount of `/etc/netplan`
- OR privileged pod with host filesystem access
- OR nsenter-based approach to write to host namespace

**Solution for Production**: Add hostPath volume mount in handler DaemonSet:
```yaml
volumes:
- name: netplan-config
  hostPath:
    path: /etc/netplan
    type: DirectoryOrCreate
volumeMounts:
- name: netplan-config
  mountPath: /etc/netplan
```

#### 2. Netplan Statistics
The `nmstatectl.Statistic()` call at `nodenetworkconfigurationpolicy_controller.go:423` fails with netplan YAML format. This is non-critical (only logs error) but should be made backend-aware to skip for netplan.

#### 3. Netplan Try Mechanism
Netplan's `try` command has a different rollback mechanism than nmstatectl's checkpoint/rollback. Current implementation uses Apply() directly. A production version should investigate netplan's try workflow for proper automatic rollback.

## Testing

### Environment
- Cluster: kubevirtci k8s-1.32 with 3 nodes
- Netplan: 1.1.2 from EPEL (installed on nodes)
- Backend: netplan D-Bus service running on nodes

### Test Commands
```bash
# Build and deploy with netplan backend
make handler operator
BACKEND=netplan make cluster-sync

# Verify backend selection
kubectl get nmstate nmstate -o jsonpath='{.spec.backend}'

# Check handler env var
kubectl get pods -n nmstate -l component=kubernetes-nmstate-handler \
  -o jsonpath='{.items[0].spec.containers[0].env[?(@.name=="NMSTATE_BACKEND")].value}'

# Apply test policy with netplan format
cat <<EOF | kubectl apply -f -
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
EOF
```

## Files Modified

### New Files
- `pkg/backend/backend.go`
- `pkg/backend/nmstate.go`
- `pkg/backend/netplan.go`
- `pkg/backend/factory.go`
- `pkg/netplanctl/netplanctl.go`
- `BACKEND_TESTING.md`
- `BACKEND_SELECTION.md`
- `examples/deploy-with-netplan.sh`

### Modified Files
- `api/v1/nmstate_types.go` - Added Backend field
- `pkg/client/client.go` - Backend initialization
- `pkg/nmpolicy/generate.go` - Backend-aware policy generation
- `cmd/handler/main.go` - Backend setup call
- `Makefile` - BACKEND variable
- `cluster/sync.sh` - patch_handler_backend() function

## Next Steps for Production

1. **Add hostPath volume mount** for /etc/netplan in handler DaemonSet
2. **Make statistics call backend-aware** to avoid error logs
3. **Investigate netplan try workflow** for proper automatic rollback
4. **Add E2E tests** for netplan backend
5. **Document netplan YAML format** requirements for users
6. **Handle netplan installation** as prerequisite (or document requirement)

## Conclusion

The PoC successfully demonstrates:
- ✅ Clean backend abstraction architecture
- ✅ API-driven backend selection
- ✅ Correct netplan D-Bus API usage (file-based workflow)
- ✅ Backend-aware policy generation
- ✅ Deployment-time backend selection

The architecture is sound and extensible. The remaining work is operational (volume mounts, testing) rather than architectural.
