---
description: Run all CI checks except e2e tests (formatting, linting, unit tests, etc.)
---

Run all CI pre-merge checks except end-to-end tests to catch issues before they hit CI.

## Checks to Run

Execute the following checks in order and report results:

1. **Code Generation Verification**
   ```bash
   make check-gen
   ```
   Verifies that generated CRDs, RBAC, and client code are up to date.
   If this fails, developer needs to run `make generate`.

2. **Static Code Checks**
   ```bash
   make check
   ```
   Runs: lint, vet, whitespace-check, gofmt-check, promlint-check.
   If this fails, developer can run `make format` to auto-fix formatting issues.

3. **Unit Tests**
   ```bash
   make test/unit
   ```
   Runs all unit tests with race detection.

4. **API Unit Tests**
   ```bash
   make test/unit/api
   ```
   Runs API-specific unit tests.

5. **Bundle Verification** (if applicable)
   ```bash
   make check-bundle
   ```
   Verifies OLM bundle is valid.

6. **OCP Bundle Verification** (if applicable)
   ```bash
   make check-ocp-bundle
   ```
   Verifies OpenShift bundle is up to date.

## Execution Instructions

**Environment Setup:**
- Before running ANY checks, temporarily unset IMAGE_REPO to ensure bundle manifests use default image references:
  ```bash
  unset IMAGE_REPO
  ```
- This prevents local IMAGE_REPO settings from causing false positives in bundle verification
- The variable is only unset for the duration of the checks, not permanently

**Execution:**
- Run each check sequentially
- Continue even if individual checks fail to get complete picture
- Track which checks pass/fail
- For failures, provide the error output and suggest fixes
- Provide a summary at the end with:
  - ✅ Passed checks
  - ❌ Failed checks
  - 📝 Suggested actions to fix failures

**Bundle Change Filtering & Cleanup:**
- After all checks complete, check if `bundle/manifests/kubernetes-nmstate-operator.clusterserviceversion.yaml` was modified
- If the ONLY change is the `createdAt:` timestamp line:
  - Automatically rollback the file using: `git checkout bundle/manifests/kubernetes-nmstate-operator.clusterserviceversion.yaml`
  - Report in summary that the timestamp-only change was automatically rolled back
- If other changes exist beyond just the timestamp, keep the changes and report them in the summary
- To verify, use: `git diff bundle/manifests/kubernetes-nmstate-operator.clusterserviceversion.yaml`

## Output Format

Structure the output as:

### CI Checks Summary

#### ✅ Passed
- [List of passed checks]

#### ❌ Failed
- [Check name]: [Brief error description]

#### 📝 Fixes Required

For each failed check:
1. **[Check Name]**
   - Error: [Key error message]
   - Fix: [Command or action to resolve]
   - Details: [Additional context if needed]

### Next Steps

[Instructions for developer to fix issues]

## Notes

- This does NOT run e2e tests (test-e2e, test-e2e-handler, test-e2e-operator)
- Some bundle checks may not apply if you haven't modified bundle files
- The `check-ocp-bundle` target may not exist - if the make command fails with "No rule to make target", skip this check and mark it as N/A
- If checks fail, fix them before committing to avoid CI failures
- IMAGE_REPO is temporarily unset during execution to match CI environment