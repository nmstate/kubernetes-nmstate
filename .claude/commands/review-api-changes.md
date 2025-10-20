---
description: Review API changes for compatibility and best practices
---

Review API changes in the kubernetes-nmstate repository for compatibility and best practices.

## Steps to Follow

1. **Identify Changed API Files**
   - Check `api/v1/*.go` (stable v1 API)
   - Check `api/v1beta1/*.go` (beta API)
   - Check `api/shared/*.go` (shared types)
   - Use `git diff main` to see what changed (main is the default branch)

2. **Check for Breaking Changes**

   Breaking changes (NOT allowed in v1):
   - Field removals or renames
   - Type changes for existing fields
   - Changed JSON/YAML tags
   - Removed enum values
   - Tightened validation rules

   Safe changes:
   - New optional fields
   - New enum values
   - Relaxed validation
   - Documentation updates

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

### CRD Generation Status
✅ Up to date OR ❌ Needs regeneration

### Issues Found
1. [Issue with file:line] - Severity: Critical/Major/Minor/Info

### Recommendations
[Actionable suggestions]

### Approval Status
✅ Approved / ⚠️ Approved with comments / ❌ Changes required

Remember: This is a declarative Kubernetes API for network configuration. API stability is critical for production clusters.
