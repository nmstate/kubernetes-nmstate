**What happened**:

**What you expected to happen**:

**How to reproduce it (as minimally and precisely as possible)**:

**Anything else we need to know?**:

**Environment**:

- `NodeNetworkState` on affected nodes (use `kubectl get nodenetworkstate <node_name> -o yaml`):
- Problematic `NodeNetworkConfigurationPolicy`:
- kubernetes-nmstate image (use `kubectl get pods --all-namespaces -l app=kubernetes-nmstate -o jsonpath='{.items[0].spec.containers[0].image}'`):
- NetworkManager version (use `nmcli --version`)
- Kubernetes version (use `kubectl version`):
- OS (e.g. from /etc/os-release):
- Others:
