---
name: Bug Report
about: Report a bug to help us improve kubernetes-nmstate
title: ''
labels: kind/bug
assignees: ''
---

**What happened?**

<!-- A clear and concise description of the bug. -->

**What did you expect to happen?**

**How to reproduce it (as minimally and precisely as possible)**

1.
2.
3.

**Environment**

- `NodeNetworkState` on affected nodes:
  <!-- use: kubectl get nodenetworkstate <node_name> -o yaml -->
- Problematic `NodeNetworkConfigurationPolicy`:
  <!-- paste your NNCP YAML -->
- kubernetes-nmstate image:
  <!-- use: kubectl get pods --all-namespaces -l app=kubernetes-nmstate -o jsonpath='{.items[0].spec.containers[0].image}' -->
- NetworkManager version:
  <!-- use: nmcli --version -->
- Kubernetes version:
  <!-- use: kubectl version -->
- OS (e.g. from `/etc/os-release`):

**Additional context**

<!-- Logs, screenshots, or anything else that helps. -->
