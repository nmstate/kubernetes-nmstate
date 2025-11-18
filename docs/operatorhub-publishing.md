# Publishing to OperatorHub

This document describes how to publish kubernetes-nmstate to OperatorHub using GitHub Actions.

## Overview

The kubernetes-nmstate operator can be published to two OperatorHub repositories:

1. **community-operators** - For upstream Kubernetes clusters (https://operatorhub.io)
2. **community-operators-prod** - For OpenShift/OKD clusters (Red Hat Certified)

We provide two GitHub Actions workflows for publishing:

- **Manual Publishing**: `publish-operatorhub.yml` - Triggered manually for testing or specific versions
- **Automatic Publishing**: `publish-operatorhub-on-release.yml` - Triggered automatically when a release is published

## Prerequisites

### Required Secrets

The workflows require a GitHub secret named `OPERATORHUB_TOKEN` with permissions to:

1. Fork and create PRs in the k8s-operatorhub/community-operators repository
2. Fork and create PRs in the redhat-openshift-ecosystem/community-operators-prod repository

To set up the token:

1. Create a GitHub Personal Access Token (PAT) with `repo` and `workflow` scopes
2. Add it to your repository secrets as `OPERATORHUB_TOKEN`:
   - Go to Settings > Secrets and variables > Actions
   - Click "New repository secret"
   - Name: `OPERATORHUB_TOKEN`
   - Value: Your PAT

### Required Images

Before publishing, ensure that the operator and handler images are built and pushed to the registry:

```bash
# Build and push handler image
IMAGE_REGISTRY=quay.io IMAGE_REPO=nmstate HANDLER_IMAGE_TAG=v0.88.0 make push-handler

# Build and push operator image
IMAGE_REGISTRY=quay.io IMAGE_REPO=nmstate OPERATOR_IMAGE_TAG=v0.88.0 make push-operator
```

Images should be publicly accessible at:
- `quay.io/nmstate/kubernetes-nmstate-handler:v<VERSION>`
- `quay.io/nmstate/kubernetes-nmstate-operator:v<VERSION>`

## Automatic Publishing (Recommended)

The automatic workflow triggers when a GitHub release is published.

### Process

1. **Create and publish a release** on GitHub:
   ```bash
   # Tag the release
   git tag v0.88.0
   git push origin v0.88.0
   ```

2. **Publish the release** through the GitHub UI or using the `gh` CLI:
   ```bash
   gh release create v0.88.0 --title "v0.88.0" --notes "Release notes here"
   ```

3. **The workflow automatically**:
   - Extracts the version from the release tag
   - Generates the OLM bundle
   - Creates a PR to `community-operators` (for all releases)
   - Creates a PR to `community-operators-prod` (for stable releases only, not pre-releases)

### Workflow Behavior

- **All releases**: Published to `community-operators` (operatorhub.io)
- **Stable releases only**: Published to `community-operators-prod` (OpenShift)
  - Pre-releases (e.g., `v0.88.0-rc1`) are NOT published to production

### Manual Trigger

You can also manually trigger the automatic workflow:

```bash
gh workflow run publish-operatorhub-on-release.yml -f tag=v0.88.0
```

## Manual Publishing

Use the manual workflow for testing or when you need fine-grained control.

### Usage

1. **Navigate to Actions** tab in GitHub
2. **Select** "Publish to OperatorHub" workflow
3. **Click** "Run workflow"
4. **Fill in the parameters**:
   - `version`: Operator version (e.g., `0.88.0` - without the 'v' prefix)
   - `operator_hub_repo`: Choose the target repository
     - `community-operators` - For operatorhub.io
     - `community-operators-prod` - For OpenShift
   - `create_pr`: Enable/disable PR creation (disable for testing)

### Using GitHub CLI

```bash
# Publish to community-operators
gh workflow run publish-operatorhub.yml \
  -f version=0.88.0 \
  -f operator_hub_repo=community-operators \
  -f create_pr=true

# Publish to community-operators-prod (OpenShift)
gh workflow run publish-operatorhub.yml \
  -f version=0.88.0 \
  -f operator_hub_repo=community-operators-prod \
  -f create_pr=true

# Test bundle generation without creating a PR
gh workflow run publish-operatorhub.yml \
  -f version=0.88.0 \
  -f operator_hub_repo=community-operators \
  -f create_pr=false
```

## Workflow Details

### What the workflows do

1. **Checkout** the kubernetes-nmstate repository at the specified tag/version
2. **Generate** the OLM bundle using `make bundle`
3. **Validate** the bundle structure and contents
4. **Checkout** the target OperatorHub repository
5. **Create** a new directory for the operator version
6. **Copy** bundle manifests, metadata, and Dockerfile
7. **Commit** and push the changes to a new branch
8. **Create** a Pull Request to the OperatorHub repository

### Bundle Structure

The generated bundle follows the Operator Framework format:

```
operators/kubernetes-nmstate-operator/<version>/
├── manifests/
│   ├── kubernetes-nmstate-operator.clusterserviceversion.yaml
│   ├── nmstate.io_nmstates.yaml
│   ├── nmstate.io_nodenetworkconfigurationenactments.yaml
│   ├── nmstate.io_nodenetworkconfigurationpolicies.yaml
│   └── nmstate.io_nodenetworkstates.yaml
├── metadata/
│   └── annotations.yaml
└── bundle.Dockerfile
```

### Image References

The workflows ensure that:
- Handler image: `quay.io/nmstate/kubernetes-nmstate-handler:v<VERSION>`
- Operator image: `quay.io/nmstate/kubernetes-nmstate-operator:v<VERSION>`
- Pull policy: `IfNotPresent` (production setting)

## Post-Publishing

After the workflow creates a PR:

1. **Monitor** the PR for CI checks in the OperatorHub repository
2. **Address** any issues flagged by OperatorHub CI
3. **Wait** for review and approval from OperatorHub maintainers
4. **Merge** will be handled by OperatorHub maintainers

Common CI checks include:
- Bundle validation
- Image accessibility checks
- Operator scorecard tests
- OpenShift compatibility (for community-operators-prod)

## Troubleshooting

### Bundle generation fails

```bash
# Locally test bundle generation
make bundle VERSION=0.88.0 \
  HANDLER_IMAGE_TAG=v0.88.0 \
  OPERATOR_IMAGE_TAG=v0.88.0 \
  HANDLER_PULL_POLICY=IfNotPresent \
  OPERATOR_PULL_POLICY=IfNotPresent

# Validate the bundle
make bundle VERSION=0.88.0
```

### Images not found

Ensure images are built and pushed:

```bash
# Check if images exist
podman pull quay.io/nmstate/kubernetes-nmstate-handler:v0.88.0
podman pull quay.io/nmstate/kubernetes-nmstate-operator:v0.88.0
```

### PR creation fails

- Verify `OPERATORHUB_TOKEN` has correct permissions
- Check if a PR for this version already exists
- Ensure the bot user has access to fork repositories

### Version format errors

The workflows expect semantic versioning:
- ✅ Correct: `0.88.0`, `1.0.0`, `0.88.1`
- ❌ Incorrect: `v0.88.0`, `0.88`, `latest`

For release tags, use the `v` prefix (e.g., `v0.88.0`), but for the workflow `version` input, omit it.

## Testing Locally

To test bundle generation locally before running the workflow:

```bash
# Set variables
export VERSION=0.88.0
export HANDLER_IMAGE_TAG=v0.88.0
export OPERATOR_IMAGE_TAG=v0.88.0

# Generate bundle
make bundle VERSION=$VERSION \
  HANDLER_IMAGE_TAG=$HANDLER_IMAGE_TAG \
  OPERATOR_IMAGE_TAG=$OPERATOR_IMAGE_TAG \
  HANDLER_PULL_POLICY=IfNotPresent \
  OPERATOR_PULL_POLICY=IfNotPresent

# Inspect generated files
ls -la bundle/
cat bundle/manifests/kubernetes-nmstate-operator.clusterserviceversion.yaml
```

## References

- [OperatorHub.io](https://operatorhub.io)
- [Operator Framework Documentation](https://sdk.operatorframework.io/)
- [community-operators Repository](https://github.com/k8s-operatorhub/community-operators)
- [community-operators-prod Repository](https://github.com/redhat-openshift-ecosystem/community-operators-prod)
- [Operator Lifecycle Manager (OLM)](https://olm.operatorframework.io/)
