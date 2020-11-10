// +build tools

package tools

import (
	_ "github.com/github-release/github-release"
	_ "github.com/onsi/ginkgo/ginkgo"
	_ "k8s.io/release/cmd/release-notes"
	_ "sigs.k8s.io/controller-tools/cmd/controller-gen"
)
