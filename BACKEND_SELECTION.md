# Backend Selection for kubernetes-nmstate

This document explains how to select the network configuration backend when deploying kubernetes-nmstate to a development cluster.

## Available Backends

- **nmstate** (default): Uses nmstatectl and NetworkManager for network configuration
- **netplan**: Uses netplan's D-Bus API for network configuration (PoC)

## Using the BACKEND Variable

The `BACKEND` Makefile variable allows you to select which backend to use when deploying to a local development cluster.

### Default Behavior (nmstate)

```bash
make cluster-sync
```

This deploys kubernetes-nmstate with the default nmstate backend.

### Using Netplan Backend

To deploy with the netplan backend, set the `BACKEND` variable:

```bash
BACKEND=netplan make cluster-sync
```

Or export it before running make:

```bash
export BACKEND=netplan
make cluster-sync
```

## How It Works

1. The `BACKEND` variable is defined in the Makefile (defaults to `nmstate`)
2. It's exported so it's available to shell scripts
3. The `cluster/sync.sh` script patches the NMState CR with the selected backend
4. The operator configures handler pods with the appropriate `NMSTATE_BACKEND` environment variable
5. Handlers initialize the selected backend at startup

## Example Workflow

### Testing with nmstate (default)

```bash
# Build images
make handler operator

# Deploy to cluster
make cluster-sync

# Verify deployment
kubectl get nmstate nmstate -o jsonpath='{.spec.backend}'
# Output: (empty - defaults to nmstate)

# Check handler pods are using nmstate
kubectl get pods -n nmstate -l component=kubernetes-nmstate-handler -o jsonpath='{.items[0].spec.containers[0].env[?(@.name=="NMSTATE_BACKEND")].value}'
# Output: nmstate
```

### Testing with netplan

```bash
# Build images
make handler operator

# Deploy with netplan backend
BACKEND=netplan make cluster-sync

# Verify deployment
kubectl get nmstate nmstate -o jsonpath='{.spec.backend}'
# Output: netplan

# Check handler pods are using netplan
kubectl get pods -n nmstate -l component=kubernetes-nmstate-handler -o jsonpath='{.items[0].spec.containers[0].env[?(@.name=="NMSTATE_BACKEND")].value}'
# Output: netplan

# Monitor handler logs for netplan D-Bus interactions
kubectl logs -n nmstate -l component=kubernetes-nmstate-handler -f
```

### Switching Backends

To switch from one backend to another:

```bash
# Switch to netplan
BACKEND=netplan make cluster-sync

# The operator will update the NMState CR
# Handler pods will be restarted with the new backend

# Switch back to nmstate
BACKEND=nmstate make cluster-sync
# Or simply:
make cluster-sync
```

## Implementation Details

### Makefile

```makefile
# Backend selection for network configuration (nmstate or netplan)
export BACKEND ?= nmstate
```

### cluster/sync.sh

```bash
function patch_handler_backend() {
    if [ -n "${BACKEND}" ] && [ "${BACKEND}" != "nmstate" ]; then
        echo "Patching NMState CR to use backend: ${BACKEND}"
        $kubectl patch -f $nmstate_cr_manifest --patch "{\"spec\": {\"backend\": \"${BACKEND}\"}}" --type=merge
    fi
}
```

This function:
- Only patches if `BACKEND` is set and not the default "nmstate"
- Uses `kubectl patch` to update the NMState CR's `spec.backend` field
- The operator watches for changes and updates handler pods accordingly

## Requirements by Backend

### nmstate Backend

**Requirements:**
- NetworkManager >= 1.22 running on nodes
- nmstatectl CLI available
- D-Bus system bus access

**Already provided by default handler image:**
- nmstate package
- NetworkManager dependencies

### netplan Backend (PoC)

**Requirements:**
- netplan >= 0.103 installed on nodes
- netplan D-Bus service running (`io.netplan.Netplan`)
- D-Bus system bus access (already available)
- godbus/dbus/v5 Go library (needs to be added to go.mod)

**Additional setup needed:**
- Handler container image needs netplan package
- Format conversion between nmstate and netplan YAML

## Troubleshooting

### Backend not applied

If the backend doesn't change:

```bash
# Check NMState CR
kubectl get nmstate nmstate -o yaml

# Check if operator is running
kubectl get pods -n nmstate -l component=kubernetes-nmstate-operator

# Check operator logs
kubectl logs -n nmstate -l component=kubernetes-nmstate-operator
```

### Handler pods not updated

```bash
# Restart handler pods to pick up new configuration
kubectl delete pods -n nmstate -l component=kubernetes-nmstate-handler

# Watch pods restart
kubectl get pods -n nmstate -w
```

### Netplan backend errors

```bash
# Check if netplan D-Bus service is running on nodes
./cluster/ssh.sh node01 "systemctl status netplan-dbus || systemctl status netplan"

# Check D-Bus service availability
./cluster/ssh.sh node01 "busctl list | grep netplan"

# Check handler logs for D-Bus errors
kubectl logs -n nmstate -l component=kubernetes-nmstate-handler | grep -i dbus
```

## Production Considerations

**Note:** The `BACKEND` Makefile variable is for **development and testing only**. In production:

1. Set the backend in the NMState CR directly:
   ```yaml
   apiVersion: nmstate.io/v1
   kind: NMState
   metadata:
     name: nmstate
   spec:
     backend: netplan  # or nmstate
   ```

2. Ensure nodes have the required backend software installed
3. Test thoroughly before deploying to production
4. Monitor handler logs for backend-specific errors
5. Have a rollback plan to switch backends if needed
