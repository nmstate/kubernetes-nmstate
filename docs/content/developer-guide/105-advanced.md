---
title: "Advanced Topics"
weight: 50
type: docs
---

This page covers advanced development topics including profiling and CI infrastructure.

## Profiling

Golang pprof profiler can be enabled for debugging and performance analysis.

### Enabling the Profiler

1. Enable profiler in `operator.yaml` by changing value of `ENABLE_PROFILER` to True
2. You can change profiler port by editing `PROFILER_PORT` - default is 6060
3. Deploy new code to cluster - example: `make cluster-sync`
4. Find nmstate-handler pod name - `kubectl get pods -n nmstate`
5. Create port forwarding to pod - example: `kubectl port-forward pod pod_name 6060:6060 -n nmstate`
6. Use `go tool pprof ...` to gather relevant metrics

### Profiling Examples

- Open in browser `http://localhost:6060/debug/pprof/`
- Download memory graph `go tool pprof -png http://localhost:6060/debug/pprof/heap > out.png`
- Open CLI for cpu 30s sample data `go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30`

More examples: https://golang.org/pkg/net/http/pprof/

## CI Infrastructure

The kubernetes-nmstate project uses the following CI infrastructure:

- [Prow](https://prow.apps.ovirt.org/) - Main CI system
- [Flakefinder](https://storage.googleapis.com/kubevirt-prow/reports/flakefinder/nmstate/kubernetes-nmstate/index.html) - Flaky test detection

## Code Structure Notes

- Controllers use controller-runtime reconciliation pattern
- Handler filters events to only its node using labels.SelectorFromSet
- NetworkManager compatibility: >= 1.22 for versions > 0.15.0
- The handler requires a file lock (`pkg/file/lock.go`) to prevent concurrent nmstatectl operations
- Profiling can be enabled via ENABLE_PROFILER env var (default port 6060)
