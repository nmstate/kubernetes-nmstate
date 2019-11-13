package helper

import (
	"os"

	"k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Return true if the event name is the name of
// the pods's node (reading the env var NODE_NAME)
func EventIsForThisNode(meta v1.Object) bool {
	createdNodeName := meta.GetName()
	podNodeName := os.Getenv("NODE_NAME")
	// Only reconcile is it's for this pod
	return createdNodeName == podNodeName
}
