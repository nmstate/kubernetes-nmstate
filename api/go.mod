module github.com/nmstate/kubernetes-nmstate/api

go 1.16

require (
	github.com/google/go-cmp v0.5.6 // indirect
	github.com/nmstate/kubernetes-nmstate/pkg/names v0.0.0
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.15.0
	golang.org/x/net v0.0.0-20210805182204-aaa1db679c0d // indirect
	golang.org/x/sys v0.0.0-20211013075003-97ac67df715c // indirect
	google.golang.org/protobuf v1.27.1 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
	k8s.io/api v0.22.3
	k8s.io/apimachinery v0.22.3
	sigs.k8s.io/controller-runtime v0.10.3
	sigs.k8s.io/yaml v1.3.0
)

replace github.com/nmstate/kubernetes-nmstate/pkg/names => ../pkg/names
