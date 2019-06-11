package helper

import (
	"os"

	"k8s.io/apimachinery/pkg/apis/meta/v1"
)

func IsForThisPod(meta v1.Object) bool {
	createdNodeName := meta.GetName()
	podNodeName := os.Getenv("NODE_NAME")
	// Only reconcile is it's for this pod
	return createdNodeName == podNodeName
}
