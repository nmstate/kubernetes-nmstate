// +build tools

package tools

import (
	_ "github.com/github-release/github-release"
	_ "github.com/onsi/ginkgo/ginkgo"
	_ "github.com/operator-framework/operator-sdk/cmd/operator-sdk"
	_ "k8s.io/kube-openapi/cmd/openapi-gen"
	_ "k8s.io/release/cmd/release-notes"
	_ "sigs.k8s.io/controller-tools/cmd/controller-gen"
)
