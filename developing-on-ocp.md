# Developing on and for OpenShift Container Platform

This document gives some hints for developing and testing kubernetes-nmstate on the OpenShift Container Platform (OCP).

## Running on a local dev-scripts cluster

To test your changes you can either deploy a cluster via cluster-bot or use the [dev-scripts](https://github.com/openshift-metal3/dev-scripts) to spin up a cluster on your hardware. In the following we are focusing on the dev-scripts approach.

Please note: The MIRROR_IMAGES option in dev-scripts may cause issues pulling the operator images from some sources. When in doubt, do not use image mirroring for clusters where you intend to install kubernetes-nmstate.

### General cluster config

To specify your CNI plugin, you have set the `NETWORK_TYPE` env var in your dev-scripts `config_$USER.sh` file. Valid values are `OVNKubernetes` and `OpenShiftSDN`. E.g.:

```bash
export NETWORK_TYPE="OVNKubernetes"
```

To specify the IP stack, use the `IP_STACK` env var in the config file. Valid values are `v4`, `v6` and `v4v6`, where the later two are only supported on OVN. E.g.:

```bash
export IP_STACK=v4v6
```

### Additional required NICs for E2E tests

If you're planning to run the e2e tests you need to have some additional NICs configured to not mess up with your hosts primary NIC. Add the following to your dev-scripts `config_$USER.sh` file:

```bash
export EXTRA_NETWORK_NAMES="nmstate1 nmstate2"
export NMSTATE1_NETWORK_SUBNET_V4='192.168.221.0/24'
export NMSTATE1_NETWORK_SUBNET_V6='fd2e:6f44:5dd8:ca56::/120'
export NMSTATE2_NETWORK_SUBNET_V4='192.168.222.0/24'
export NMSTATE2_NETWORK_SUBNET_V6='fd2e:6f44:5dd8:cc56::/120'
```

### Credentials for dev-scripts clusters

The kubeconfig of your dev-scripts cluster can be found in `<dev-scripts-folder>/ocp/ostest/auth/kubeconfig`.

The password for the kubeadmin to login to your [cluster console](https://console-openshift-console.apps.ostest.test.metalkube.org/) can be found in `<dev-scripts-folder>/ocp/ostest/auth/kubeadmin-password`.

## Deploying the Operator

There are multiple ways to deploy the operator the a cluster. Since the user probably installs the operator later via the marketplace, we provided some helpers to build and deploy the operator as it is done via the marketplace.

To build and install the operator, run the following command (Before that, make sure that no kubernetes-nmstate operator is installed already):

```bash
$ IMAGE_REPO=<your quay.io username> KUBECONFIG=<path to your kubeconfig> make ocp-build-and-deploy-bundle
```

This builds the operator, handler, bundle and index images and pushes them to your quay.io account. Make sure, that the created repositories are public available otherwise the images can't be pulled.

The following other parameters are available as well:

|Parameter|Default Value|Description|
|-|-|-|
|`IMAGE_REGISTRY`|`quay.io`|The registry where the images for the operator, handler, bundle and index image will be stored and loaded from|
|`IMAGE_REPO`|`openshift`|The username for the image registry|
|`CHANNEL`|the latest version it can find in the `manifests/ folder`|The channel for the bundle|
|`VERSION`|`${CHANNEL}.0`|The version / tag for the images. Keep in mind to use different versions for each build in case you haven't set `imagePullPolicy: Always`. To get around this, you could include the timestamp into the tag (e.g. `VERSION=1.2.3-$(date +%Y%m%d%H%M%S)`)|
|`NAMESPACE`|`openshift-nmstate`|The namespace where to deploy the components to|
|`HANDLER_IMAGE_NAME`|`origin-kubernetes-nmstate-handler`|Image name for the handler|
|`HANDLER_IMAGE_TAG`|`$VERSION`|Tag for the handler image|
|`HANDLER_NAMESPACE`|`$NAMESPACE`|Namespace for the handler|
|`OPERATOR_IMAGE_NAME`|`origin-kubernetes-nmstate-operator`|Image name for the operator|
|`OPERATOR_IMAGE_TAG`|`$VERSION`|Tag for the operator image|
|`OPERATOR_NAMESPACE`|`$NAMESPACE`|Namespace for the operator|
|`BUNDLE_VERSION`|`$VERSION`|Version / Tag for the bundle image|
|`INDEX_VERSION`|`$VERSION`|Version / Tag for the index image|
|`SKIP_IMAGE_BUILD`|`false`|Skip the image build and install the operators version directly|
|`INSTALL_OPERATOR_VIA_UI`|`false`|If you want to install the built operator via the [UI](https://console-openshift-console.apps.ostest.test.metalkube.org/operatorhub) and only build the images and create the `CatalogSource` so it can be found in marketplace, set this parameter to `true`.|

To uninstall the operator again, run the following command:
```bash
$ KUBECONFIG=<path to your kubeconfig> make ocp-uninstall-bundle
```

Just in case, the official documentation for the deploy is [located here](https://docs.openshift.com/container-platform/4.13/networking/k8s_nmstate/k8s-nmstate-about-the-k8s-nmstate-operator.html) but do not mix the deploy methods.

Finally, there [is a script](https://github.com/openshift/kubernetes-nmstate/blob/07faf0dbb8ebcb76174e12efba1515c816c36d20/hack/ocp-install-nightly-art-operators.sh) that install the latest operator but requires VPN.

For debugging there are commands like:
```
oc get sub -n openshift-nmstate
oc get csv -n openshift-nmstate
```

Official doc for troubleshooting [is here](https://docs.openshift.com/container-platform/4.13/support/troubleshooting/troubleshooting-operator-issues.html)

## Running the E2E Tests locally

The E2E tests run automated in CI. If you want to run them locally anyhow, you can run them with the following commands:

```bash
# Handler e2e tests
$ IMAGE_BUILDER=podman IMAGE_REPO=<your quay.io user name> KUBECONFIG=<path to your kubeconfig> make test-e2e-handler-ocp

# Operator e2e tests
$ IMAGE_BUILDER=podman IMAGE_REPO=<your quay.io user name> KUBECONFIG=<path to your kubeconfig> make test-e2e-operator-ocp
```

Keep the following in mind when running the e2e tests:

* The tests deploy the operator by themselves. There is no need to deploy it on your own beforehand.
* The tests deploy the operator not via the bundle. Instead they apply the required manifest files on their own.
* On OCP some machine configs on the nodes needs to be applied to configure the interfaces. This is done by the script too.
* Some tests need to be skipped on OCP. These tests can be found in the `SKIPPED_TESTS` var in [hack/ocp-e2e-tests-handler.sh](hack/ocp-e2e-tests-handler.sh).

## Rebasing to upstreams

To stay up to date with the changes in [upstreams](https://github.com/nmstate/kubernetes-nmstate) kubernetes-nmstate, the downstreams repo needs to be rebased on the latest changes / fixes regularly. To rebase, run the following script and follow the instructions:

```bash
$ ./hack/ocp-rebase.sh
```

Make sure the script took every `UPSTREAM: <carry>` commit. Especially those commits between the last merge commit and the last `UPSTREAM: <carry>` commits from the rebase before that can't be catched by the script. This most likely happens, when a rebase PR was opened and in the meantime some other carry commits got merged in. So check the git logs that you took all the carry commits.

Since 4.12 we are rebasing directly on the upstream main branch. This means the downstream master can always be rebased on upstream main branch. For older releases the fixes needs to be cherry-picked manually and backported. This means this script can't be used to rebase other branches (besides master) except an older release depends on an upstream release version. The following table shows which downstream release depends on which upstream version as of today (Sept. 29, 2022):

|Downstream release|Corresponding upstream version tag|
|-|-|
|>=4.12|_none_. As said, since 4.12 we are rebasing on u/s main directly|
|4.11|v0.71.0|
|4.10|v0.64.14|
|4.9|v0.52.4|
|4.8|v0.47.11|

## Update bundle manifest

After the manifest files of the operator where adjusted (e.g. by updating a RBAC role, adding a label to the deployment or rebasing), it is required to update the bundle manifest files to theses changes will be reflected in the `ClusterServiceVersion` (in file `manifests/<release version>/manifests/kubernetes-nmstate-operator.clusterserviceversion.yaml`) for the OCP release. To do so, run the following command:

```bash
$ make ocp-update-bundle-manifests
```

## Add manifests for a new release

With each new planned release (e.g. 4.12, 4.13), replace every occurrence of the old release in the `manifests` folder with the new release. E.g.:

```bash
$ find manifests -type f -exec sed -i 's/4.12/4.13/g' {} +
```

## Making Changes in CI

The jobs which run in CI are defined in the [openshift/release](https://github.com/openshift/release) repository in the [openshift-kubernetes-nmstate-master.yaml](https://github.com/openshift/release/blob/master/ci-operator/config/openshift/kubernetes-nmstate/openshift-kubernetes-nmstate-master.yaml) file. For each release a new file will be created by the aos-art-bot and defines the steps for a certain release (e.g. openshift-kubernetes-nmstate-release-4.11.yaml). 

Check out the [CI Operator docs](https://docs.ci.openshift.org/docs/architecture/ci-operator/) about how to configure CI jobs.
