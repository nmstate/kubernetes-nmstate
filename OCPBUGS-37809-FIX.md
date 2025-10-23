# Fix for OCPBUGS-37809: NodeNetworkConfigurationPolicy not evaluated after ungraceful reboot

## Summary

This fix addresses the issue where NodeNetworkConfigurationPolicy (NNCP) resources lose their STATUS field after an ungraceful cluster reboot. The problem occurs because the handler relies entirely on Kubernetes watch events, which are not fired for existing resources when a pod restarts.

## Root Cause

The NNCP controller in the handler DaemonSet only reconciles policies when:
1. A new NNCP is created
2. An existing NNCP is deleted
3. An existing NNCP is updated **with a generation change**
4. Node labels change

When the handler pod restarts after an ungraceful reboot:
- Existing NNCPs don't trigger Create events (they already exist)
- No Update events fire (the policies haven't changed)
- The controller never reconciles these policies
- Without reconciliation, NNCEs (NodeNetworkConfigurationEnactment) are not created/updated
- Without NNCEs, the NNCP status remains empty

## Solution

This fix implements two complementary mechanisms:

### 1. Startup Reconciliation (Quick Fix)
On handler startup, after a short delay to allow cache synchronization, the controller:
- Lists all NNCPs in the cluster
- Filters for policies matching this node's selectors
- Enqueues them for reconciliation via generic events

This ensures all applicable policies are evaluated immediately after restart, regardless of whether events were lost.

### 2. Periodic Reconciliation (Long-term Fix)
The controller now triggers reconciliation of all matching NNCPs every 10 minutes. This provides continuous resilience against:
- Missed events
- Cache inconsistencies
- Race conditions during startup
- Any other transient failures

## Implementation Details

### Modified Files

**`controllers/handler/nodenetworkconfigurationpolicy_controller.go`**

#### Changes:
1. Added new fields to `NodeNetworkConfigurationPolicyReconciler`:
   - `eventChannel`: Channel for triggering reconciliation via generic events

2. Added timing constants:
   - `periodicReconcileInterval = 10 * time.Minute`: How often to trigger periodic reconciliation
   - `startupReconcileDelay = 5 * time.Second`: Delay before startup reconciliation

3. Modified `SetupWithManager()`:
   - Creates event channel for generic events
   - Adds watch on the event channel
   - Registers `policyReconciliationTrigger` as a manager Runnable

4. Added `policyReconciliationTrigger` type:
   - Implements `manager.Runnable` interface
   - Runs two goroutines:
     - Startup reconciliation: Triggers once after delay
     - Periodic reconciliation: Triggers every 10 minutes
   - Uses `enqueuePoliciesForNode()` to send events to channel

### Key Features

1. **Non-blocking**: Uses goroutines and channels to avoid blocking the main controller
2. **Selector-aware**: Only enqueues policies matching the current node
3. **Context-aware**: Respects context cancellation for graceful shutdown
4. **Buffered channel**: 100-element buffer prevents blocking on bursts
5. **Graceful degradation**: Logs errors but continues operation if enqueue fails

## Testing

### Unit Tests

**Test Coverage Added:**
- 10 new comprehensive unit tests for the reconciliation trigger functionality
- All tests use Ginkgo/Gomega framework
- Tests cover both success and failure scenarios
- Race condition free (passes with `-race` flag)

**New Test Suites:**
1. **enqueuePoliciesForNode** (5 tests)
   - ✅ Enqueues policies matching node selector
   - ✅ Handles empty policy list gracefully
   - ✅ Enqueues all policies when no selector specified
   - ✅ Stops enqueuing when context cancelled
   - ✅ Handles full event channel gracefully

2. **startupReconciliation** (2 tests)
   - ✅ Triggers reconciliation after delay
   - ✅ Respects context cancellation

3. **periodicReconciliation** (2 tests)
   - ✅ Triggers reconciliation periodically
   - ✅ Stops when context cancelled

4. **Start (integration)** (1 test)
   - ✅ Starts both startup and periodic reconciliation

**Test Results:**
```
Ran 29 of 29 Specs in 3.940 seconds
SUCCESS! -- 29 Passed | 0 Failed | 0 Pending | 0 Skipped

Full suite: 14 suites, 200+ total specs - all passing
```

### Manual Testing
To test this fix:

1. Deploy NNCPs to a cluster:
```bash
kubectl apply -f <nncp-manifest>.yaml
```

2. Verify policies are applied:
```bash
kubectl get nncp
# Should show STATUS: Available, REASON: SuccessfullyConfigured
```

3. Ungracefully kill handler pod:
```bash
kubectl delete pod -n nmstate <handler-pod> --grace-period=0 --force
```

4. Wait for new pod and verify status returns:
```bash
kubectl get nncp -w
# Status should reappear within ~5-10 seconds
```

### Expected Behavior

**Before Fix:**
- After ungraceful reboot, some NNCPs show empty STATUS
- STATUS remains empty until policy is manually updated or node labels change

**After Fix:**
- After ungraceful reboot, all NNCPs are re-evaluated within 5 seconds
- STATUS is populated correctly
- Periodic reconciliation ensures status stays current

## Configuration

The reconciliation timing can be adjusted via the constants:

```go
// How long to wait before startup reconciliation (allows cache to sync)
startupReconcileDelay = 5 * time.Second

// How often to trigger periodic reconciliation
periodicReconcileInterval = 10 * time.Minute
```

These are currently hardcoded but could be made configurable via environment variables if needed.

## Performance Impact

### Minimal Impact:
- **Startup**: Single burst of reconciliation requests after 5-second delay
- **Periodic**: One reconciliation per matching NNCP every 10 minutes
- **Memory**: ~100 KB for buffered channel (100 events * ~1KB each)
- **CPU**: Negligible (only runs selector matching and event queueing)

### Optimization:
- Only policies matching the node selector are enqueued
- Reconciliation uses existing code paths (no duplication)
- Events are processed through standard controller queue (rate-limited)

## Compatibility

- **Kubernetes**: Compatible with all supported versions (uses standard controller-runtime)
- **Backwards compatible**: No API changes, no breaking changes
- **Deployment**: Drop-in replacement, no configuration changes needed

## Related Issues

- **OCPBUGS-37809**: NodeNetworkConfigurationPolicy not evaluated after cluster's ungraceful reboot
- **OCPBUGS-37666**: Related issue (linked in OCPBUGS-37809)
- **KNIECO-11440**: Related issue (linked in OCPBUGS-37809)

## Future Enhancements

Potential improvements for future iterations:

1. **Configurable timing**: Environment variables for reconciliation intervals
2. **Metrics**: Expose metrics for startup/periodic reconciliation counts
3. **Alerting**: Alert if policies remain without status for extended period
4. **Health checks**: Add health check that verifies all matching NNCPs have status

## Logging

The fix adds structured logging to aid debugging:

```
INFO Starting policy reconciliation trigger
INFO Waiting before startup reconciliation delay=5s
INFO Starting startup reconciliation of all NNCPs matching this node
INFO Reconciliation trigger completed trigger=startup enqueued=5 skipped=2 total=7
INFO Started periodic reconciliation interval=10m0s
INFO Triggering periodic reconciliation of NNCPs
INFO Reconciliation trigger completed trigger=periodic enqueued=5 skipped=2 total=7
```

## DCO Sign-off

All commits include proper DCO sign-off as required by the project.
