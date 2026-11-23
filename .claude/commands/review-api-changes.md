---
description: Review API changes for compatibility and best practices
---

Review API changes in the kubernetes-nmstate repository for compatibility and best practices.

## Steps to Follow

1. **Identify Changed API Files**
   - Check `api/v1/*.go` (stable v1 API)
   - Check `api/v1beta1/*.go` (beta API)
   - Check `api/shared/*.go` (shared types)
   - Compare the PR with its target branch, which may be a release branch

2. **Check for Breaking Changes**

   Breaking changes (NOT allowed in v1):
   - Field removals or renames
   - Type changes for existing fields
   - Changed JSON/YAML tags
   - Removed enum values
   - Tightened validation rules

   Generally compatible changes:
   - New optional fields
   - New enum values
   - Relaxed validation
   - Documentation updates

   Apply the [in-place upgrade guidance](../../AGENTS.md#in-place-upgrades) to stored CRs.
   For example, if a new field is optional, trace how the new controller handles an old CR
   that omits it and whether the webhook still admits updates to that CR.

3. **Verify CRD Regeneration**
   - Run `make check-manifests` to verify CRDs are up to date
   - If fails, developer needs to run `make generate`

4. **Review Kubebuilder Markers**
   - Check `+kubebuilder:validation:*` markers
   - Check `+kubebuilder:printcolumn:*` for kubectl output
   - Verify `+optional` and `+required` markers
   - Check `+kubebuilder:subresource:status`

5. **Documentation Check**
   - All new/modified fields have godoc comments
   - Validation constraints documented
   - Default values documented
   - Examples provided where helpful

6. **Related Changes**
   - Controllers updated (`controllers/`)
   - Webhook validation updated (`pkg/webhook/`)
   - Status handling updated (`pkg/policyconditions/`, `pkg/enactmentstatus/`)

7. **Test Coverage**
   - Unit tests in `api/` package
   - E2E tests in `test/e2e/` for behavior changes
   - Example manifests in `docs/examples/`

## Provide Review Output

Structure your review as:

### Summary
[What was changed]

### Files Reviewed
[List of files]

### Breaking Changes
✅ None detected OR ❌ [List with severity]

### In-place Upgrade Impact
[Describe what happens to an existing CR and the evidence for that conclusion, or explain why it is unaffected.]

### CRD Generation Status
✅ Up to date OR ❌ Needs regeneration

### Issues Found
1. [Issue with file:line] - Severity: Critical/Major/Minor/Info

### Recommendations
[Actionable suggestions]

### Approval Status
✅ Approved / ⚠️ Approved with comments / ❌ Changes required

Remember: This is a declarative Kubernetes API for network configuration. API stability is critical for production clusters.
