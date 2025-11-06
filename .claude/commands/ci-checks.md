---
description: Run all CI checks except e2e tests (formatting, linting, unit tests, etc.)
---

Run all CI pre-push checks to catch issues before they hit CI.

## What This Does

This command runs the `make pre-push` target which executes all CI pre-merge checks except end-to-end tests.

The `pre-push` target includes:
1. **Code Generation Verification** (`make check-gen`) - Verifies CRDs, RBAC, and client code are up to date
2. **Static Code Checks** (`make check`) - Runs lint, vet, whitespace-check, gofmt-check, promlint-check
3. **Unit Tests** (`make test/unit`) - Runs all unit tests with race detection
4. **API Unit Tests** (`make test/unit/api`) - Runs API-specific unit tests
5. **Bundle Verification** (`make check-bundle`) - Verifies OLM bundle is valid
6. **OCP Bundle Verification** (`make check-ocp-bundle`) - Verifies OpenShift bundle (if target exists)

The target automatically:
- Unsets IMAGE_REPO to match CI environment
- Rolls back bundle timestamp-only changes
- Provides clear pass/fail summary

## Execution

Simply run:
```bash
make pre-push
```

## Output Interpretation

- If all checks pass: ✅ All pre-push checks passed!
- If checks fail: Review the error output and run suggested fix commands
- Common fixes:
  - `make generate` - if code generation is out of date
  - `make format` - if formatting issues exist
  - Fix failing unit tests individually

## Notes

- This does NOT run e2e tests (test-e2e, test-e2e-handler, test-e2e-operator)
- The target is also automatically triggered by the pre-push git hook
- All logic is centralized in the Makefile for consistency