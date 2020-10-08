module github.com/nmstate/kubernetes-nmstate

go 1.13

require (
	github.com/evanphx/json-patch v4.9.0+incompatible
	github.com/github-release/github-release v0.8.1
	github.com/go-logr/logr v0.1.0
	github.com/go-openapi/spec v0.19.4
	github.com/gobwas/glob v0.2.3
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/nightlyone/lockfile v1.0.0
	github.com/onsi/ginkgo v1.12.1
	github.com/onsi/gomega v1.10.1
	github.com/openshift/cluster-network-operator v0.0.0-20200922032245-f47200e8dbc0
	github.com/operator-framework/operator-sdk v0.19.4
	github.com/phoracek/networkmanager-go v0.1.0
	github.com/pkg/errors v0.9.1
	github.com/qinqon/kube-admission-webhook v0.12.0
	github.com/spf13/pflag v1.0.5
	github.com/tidwall/gjson v1.6.1
	k8s.io/api v0.18.9
	k8s.io/apimachinery v0.18.9
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/klog v1.0.0
	k8s.io/kube-openapi v0.0.0-20200410145947-61e04a5be9a6
	k8s.io/kubectl v0.18.2
	k8s.io/release v0.4.0
	kubevirt.io/qe-tools v0.1.6
	sigs.k8s.io/controller-runtime v0.6.2
	sigs.k8s.io/controller-tools v0.3.0
	sigs.k8s.io/yaml v1.2.0
)

// Pinned to kubernetes-1.18.9
replace (
	k8s.io/api => k8s.io/api v0.18.9
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.18.9
	k8s.io/apimachinery => k8s.io/apimachinery v0.18.9
	k8s.io/apiserver => k8s.io/apiserver v0.18.9
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.18.9
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.18.9
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.18.9
	k8s.io/code-generator => k8s.io/code-generator v0.18.9
	k8s.io/component-base => k8s.io/component-base v0.18.9
	k8s.io/cri-api => k8s.io/cri-api v0.18.9
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.18.9
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.18.9
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.18.9
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.18.9
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.18.9
	k8s.io/kubectl => k8s.io/kubectl v0.18.9
	k8s.io/kubelet => k8s.io/kubelet v0.18.9
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.18.9
	k8s.io/metrics => k8s.io/metrics v0.18.9
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.18.9
)

replace (
	github.com/Azure/go-autorest => github.com/Azure/go-autorest v13.3.2+incompatible // Required by OLM
	github.com/mattn/go-sqlite3 => github.com/mattn/go-sqlite3 v1.10.0
	golang.org/x/text => golang.org/x/text v0.3.3
	k8s.io/client-go => k8s.io/client-go v0.18.2
	vbom.ml/util => github.com/fvbommel/util v0.0.0-20180919145318-efcd4e0f9787
)
