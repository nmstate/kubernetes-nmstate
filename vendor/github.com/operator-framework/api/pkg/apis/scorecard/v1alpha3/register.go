package v1alpha3

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	// GroupVersion is the group and version of this package. Used for parsing purposes only.
	GroupVersion = schema.GroupVersion{Group: "scorecard.operatorframework.io", Version: "v1alpha3"}
)
