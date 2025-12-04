# Testing Netplan Backend - Quick Reference

This is a quick reference guide for testing the netplan backend PoC.

## Prerequisites

- Local kubernetes cluster (kubevirtci or similar)
- buildah/podman or docker installed
- kubectl configured

## Quick Test Commands

### Test with default nmstate backend

```bash
make handler operator cluster-sync
```

### Test with netplan backend

```bash
make handler operator
BACKEND=netplan make cluster-sync
```

### Verify backend selection

```bash
# Check NMState CR backend field
kubectl get nmstate nmstate -o jsonpath='{.spec.backend}'

# Check environment variable in handler pods
kubectl get pods -n nmstate \
  -l component=kubernetes-nmstate-handler \
  -o jsonpath='{.items[0].spec.containers[0].env[?(@.name=="NMSTATE_BACKEND")].value}'
```

### Monitor handler logs

```bash
# Follow logs from all handler pods
kubectl logs -n nmstate -l component=kubernetes-nmstate-handler -f

# Look for backend initialization message
kubectl logs -n nmstate -l component=kubernetes-nmstate-handler \
  | grep -i "backend\|netplan"
```

### Switch backends

```bash
# Switch to netplan
BACKEND=netplan make cluster-sync

# Wait for handler pods to restart
kubectl wait --for=condition=Ready pod \
  -l component=kubernetes-nmstate-handler \
  -n nmstate --timeout=120s

# Switch back to nmstate
make cluster-sync
```

## Example Workflow

### 1. Initial deployment with nmstate

```bash
# Build
make handler operator

# Deploy with default backend (nmstate)
make cluster-sync

# Verify
kubectl get nmstate nmstate -o yaml | grep backend:
# Should be empty or "backend: nmstate"
```

### 2. Test with netplan backend

```bash
# Switch to netplan
BACKEND=netplan make cluster-sync

# Verify backend changed
kubectl get nmstate nmstate -o jsonpath='{.spec.backend}'
# Output: netplan

# Watch handler pods restart
kubectl get pods -n nmstate -w

# Check handler is using netplan
kubectl logs -n nmstate -l component=kubernetes-nmstate-handler \
  | grep "Initialized network configuration backend" | tail -1
```

### 3. Apply test policy

```bash
# Create a simple test policy
cat <<EOF | kubectl apply -f -
apiVersion: nmstate.io/v1beta1
kind: NodeNetworkConfigurationPolicy
metadata:
  name: test-netplan
spec:
  desiredState:
    interfaces:
      - name: eth1
        type: ethernet
        state: up
        ipv4:
          enabled: true
          dhcp: true
EOF

# Watch enactment status
kubectl get nnce -w
```

## Troubleshooting

### Backend not changing

```bash
# Check operator logs
kubectl logs -n nmstate -l component=kubernetes-nmstate-operator

# Manually patch NMState CR
kubectl patch nmstate nmstate \
  --patch '{"spec":{"backend":"netplan"}}' \
  --type=merge

# Force handler pod restart
kubectl delete pods -n nmstate -l component=kubernetes-nmstate-handler
```

### Netplan D-Bus errors

If you see D-Bus errors with netplan backend:

```bash
# SSH to a node and check netplan service
./cluster/ssh.sh node01 "systemctl status netplan-dbus || systemctl status netplan"

# Check if netplan D-Bus service is available
./cluster/ssh.sh node01 "busctl list | grep netplan"

# Check netplan is installed
./cluster/ssh.sh node01 "netplan --version"
```

### Handler pod not starting

```bash
# Describe handler pods
kubectl describe pods -n nmstate -l component=kubernetes-nmstate-handler

# Check events
kubectl get events -n nmstate --sort-by='.lastTimestamp'

# Check DaemonSet
kubectl describe daemonset -n nmstate nmstate-handler
```

## Environment Variables

The following environment variables control backend selection:

| Variable | Default | Description |
|----------|---------|-------------|
| `BACKEND` | `nmstate` | Backend to use when deploying via `make cluster-sync` |

Set in Makefile:
```makefile
export BACKEND ?= nmstate
```

Override when running make:
```bash
BACKEND=netplan make cluster-sync
```

Or export before make:
```bash
export BACKEND=netplan
make cluster-sync
```

## Files Added/Modified

### New Files
- `BACKEND_SELECTION.md` - Detailed backend selection documentation
- `BACKEND_TESTING.md` - This file
- `examples/deploy-with-netplan.sh` - Example deployment script
- `pkg/backend/*.go` - Backend interface and implementations
- `pkg/netplanctl/*.go` - Netplan D-Bus client

### Modified Files
- `Makefile` - Added `BACKEND` variable
- `cluster/sync.sh` - Added `patch_handler_backend()` function
- `api/v1/nmstate_types.go` - Added `Backend` field to NMStateSpec
- `controllers/operator/nmstate_controller.go` - Pass backend to handler
- `deploy/handler/operator.yaml` - Added `NMSTATE_BACKEND` env var

## See Also

- [NETPLAN_POC.md](NETPLAN_POC.md) - Complete PoC documentation
- [BACKEND_SELECTION.md](BACKEND_SELECTION.md) - Backend selection guide
- [pkg/netplanctl/README.md](pkg/netplanctl/README.md) - Netplan D-Bus client docs
