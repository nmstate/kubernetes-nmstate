# Design

The system is implemented as an k8s operator using the
[operator-sdk](https://github.com/operator-framework/operator-sdk) but is
deployed as a DaemonSet instead of Deployment with
[filtering](https://sdk.operatorframework.io/docs/building-operators/golang/references/event-filtering/)
only events for the DaemonSet pod node.


# NetworkManager compatibility

kubernetes-nmstate is connecting to NetworkManager running on a host. That
implies following dependency requirements:

| kubernetes-nmstate version | NetworkManager version |
| ---                        | ---                    |
| main, `> 0.15.0`         | `>= 1.22`              |
| `<= 0.15.0`                | `>= 1.20`              |


# Development

## Local Kubernetes cluster

See [local virtualized cluster guide](docs/deployment/local-cluster.md) to learn
how to deploy a cluster that can be used to try kubernetes-nmstate, run e2e
tests and perform debugging.

## External ocp cluster using custom container registry and repo

Set the current env vars `KUBEVIRT_PROVIDER=external` and `KUBECONFIG` pointing
to the k8s cluster config.

After that you can follow "Local Kubernetes cluster" but using
`DEV_IMAGE_REGISTRY` and `IMAGE_REPO` to specify where the dev
containers are being pushed

For example quay.io/foo/ is being used as the dev place for containers and
want to deploy some changes at external cluster, following steps would be
enough:

```bash
doker login -u foo quay.io
make DEV_IMAGE_REGISTRY=quay.io IMAGE_REPO=foo cluster-sync
```

## Building

```shell
# If pkg/apis/ has been changed, run generator to update client code
make gen-k8s

# Build handler operator (both its binary and docker image)
make handler
```

### Building on an Apple Silicon Mac
Building on Apple Silicon Macs is only supported with podman. To build amd64 with podman developers are required to set up a `podman machine`.
This machine will need to be configured specifically to run amd64 containers. To achieve that, simply run the following commands to set up a machine:

```shell
# Initialize the machine with your preferred specs
podman machine init --cpus=8 --disk-size=20 --memory 8192
podman machine start

# Once the machine is ready and started up ssh into it.
podman machine ssh
sudo -i

# Install qemu-user-static (if not installed already)
rpm-ostree install qemu-user-static
systemctl reboot
```

## Testing

```shell
# Static checking
make check

# Run unit tests
make test/unit

# Run e2e tests, you need a running k8s/openshift cluster with with kubernets-nmstate running.
make test/e2e

# Run tests matching the regex "NodeSelector"
make test/e2e E2E_TEST_ARGS='-ginkgo.focus=NodeSelector'

# Conversely, exclude tests that match the regex "Simple\ OVS*"
make test/e2e E2E_TEST_ARGS='--ginkgo.skip="Simple\ OVS*"'
```

## Publishing containers

```shell
# Push nmstate-handler container to remote registry
make push-handler
```

It is possible to adjust the built container images with the following
environment variables.

```shell
IMAGE_REGISTRY # quay.io
IMAGE_REPO # nmstate

HANDLER_IMAGE_NAME # kubernetes-nmstate-handler
HANDLER_IMAGE_TAG # latest
```

## Manifests

The operator `operator.yaml` manifest from the `deploy` folder is a template to
be able to replace the with correct docker image to use.

Everytime cluster-sync is called it will regenerate the operator yaml with
correct kubernets-nmstate-handler image and apply it.


# Open a Pull Request

kubernetes-nmstate generally follows the [standard github pull request
process](https://gist.github.com/Chaser324/ce0505fbed06b947d962), but there is a
layer of additional specific differences:

The first difference you'll see is that a bot will begin applying structured
labels to your PR.

The bot may also make some helpful suggestions for commands to run in your PR to
facilitate review. These `/command` options can be entered in comments to
trigger auto-labeling and notifications. Refer to its command reference
documentation.

Common new contributor PR issues are:

- Missing DCO sign-off:\
  Developers Certificate of Origin (DCO) Sign-off is a requirement for getting
  patches into the project (see [Developers Certificate of
  Origin](https://developercertificate.org/)). You can "sign" this certificate
  by including a line in the git commit of "Signed-off-by: Legal Name
  <email-address>". If you forgot to add the sign-off, you can also amend your
  commit with the sign-off: `git commit --amend -s`.

## Code Review

To make it easier for your PR to receive reviews, consider the reviewers will
need you to:

- Follow the project [coding
  conventions](https://github.com/golang/go/wiki/CodeReviewComments).
- Write [good commit messages](https://chris.beams.io/posts/git-commit/).
- Break large changes into a logical series of smaller patches which
  individually make easily understandable changes, and in aggregate solve a
  broader issue.

## Best Practices

- Write clear and meaningful git commit messages.
- If the PR will *completely* fix a specific issue, include `fixes #123` in the
  PR body (where `123` is the specific issue number the PR will fix. This will
  automatically close the issue when the PR is merged.
- Make sure you don't include `@mentions` or `fixes` keywords in your git commit
  messages. These should be included in the PR body instead.
- Make sure you include a clear and detailed PR description explaining the
  reasons for the changes, and ensuring there is sufficient information for the
  reviewer to understand your PR.

# Releasing

To cut a release, push images to quay and publish it on GitHub
the command `make release` do all this automatically, the version  is at
`version/version.go` and the description at `version/description`.

So the step would be:
 - Prepare a release calling `make prepare-(patch|minor|major)`
 - Edit version/description to set a description and order commits
 - Create a PR to review it
 - Merge it to main
 - Wait for Prow to do the release.

The environment variable `GITHUB_TOKEN` is needed to publish at GitHub and it must
point to a token generated by github to access projects.

# Profiling

There is a possibility to enable golang pprof profiler.
 - Enable profiler in `operator.yaml` by changing value of 'ENABLE_PROFILER' to True
 - You can change profiler port by editing 'PROFILER_PORT' - default is 6060
 - Deploy new code to cluster - example:  `make cluster-sync`
 - Find nmstate-handler pod name - `kubectl get pods -n nmstate`
 - Create port forwarding to pod - example: `kubectl port-forward pod pod_name 6060:6060 -n nmstate`
 - Use `go tool pprof ...` to gather relevant metrics.
   Examples:
    - open in browser `http://localhost:6060/debug/pprof/`
    - download memory graph `go tool pprof -png http://localhost:6060/debug/pprof/heap > out.png`
    - open cli for cpu 30s sample data `go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30`

   More examples can be found here: https://golang.org/pkg/net/http/pprof/

# CI infraestructure

- [prow](https://prow.apps.ovirt.org/)
- [flakefinder](https://storage.googleapis.com/kubevirt-prow/reports/flakefinder/nmstate/kubernetes-nmstate/index.html)

# FAQ

## The NodeNetworkState does not show the state correctly at ubuntu 18.04 nodes.

In Ubuntu 18.04 they introduced netplan for the network configuration. So to enable NetworkManager you need to
follow these steps:

```yaml
# 1.- - edit /etc/netplan with:
network:
  version: 2
  renderer: NetworkManager
```

```bash
# 2.- apply the changes
netplan generate
netplan apply
```

References:

- https://netplan.io/
- https://askubuntu.com/questions/1031956/network-manager-not-working-when-installing-ubuntu-desktop-on-a-ubuntu-18-04-ser
