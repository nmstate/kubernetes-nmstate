module github.com/nmstate/kubernetes-nmstate

go 1.15

require (
	github.com/Masterminds/semver v1.5.0
	github.com/evanphx/json-patch v4.9.0+incompatible
	github.com/github-release/github-release v0.10.0
	github.com/go-logr/logr v0.3.0
	github.com/gobwas/glob v0.2.3
	github.com/gofrs/flock v0.8.0
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/onsi/ginkgo v1.15.0
	github.com/onsi/gomega v1.10.5
	github.com/openshift/cluster-network-operator v0.0.0-20200922032245-f47200e8dbc0
	github.com/operator-framework/operator-registry v1.15.3
	github.com/operator-framework/operator-sdk v1.4.2
	github.com/phoracek/networkmanager-go v0.1.0
	github.com/pkg/errors v0.9.1
	github.com/qinqon/kube-admission-webhook v0.15.0
	github.com/tidwall/gjson v1.8.0
	github.com/tidwall/sjson v1.1.7
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/api v0.20.2
	k8s.io/apimachinery v0.20.2
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/kubectl v0.20.1
	k8s.io/release v0.7.0
	kubevirt.io/qe-tools v0.1.7
	sigs.k8s.io/controller-runtime v0.8.2
	sigs.k8s.io/controller-tools v0.4.1
	sigs.k8s.io/yaml v1.2.0
)

replace (
	github.com/Azure/go-autorest => github.com/Azure/go-autorest v14.2.0+incompatible // Required by OLM
	// Using containerd 1.4.0+ resolves an issue with invalid error logging
	// from an init function in containerd. This replace can be removed when
	// one of our direct dependencies begins using containerd v1.4.0+
	github.com/containerd/containerd => github.com/containerd/containerd v1.4.3
	github.com/mattn/go-sqlite3 => github.com/mattn/go-sqlite3 v1.10.0
	golang.org/x/text => golang.org/x/text v0.3.3 // Required to fix CVE-2020-14040
	k8s.io/client-go => k8s.io/client-go v0.20.1
)

exclude github.com/spf13/viper v1.3.2 // Required to fix CVE-2018-1098
