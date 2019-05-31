package e2e

import (
	"context"
	"time"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"

	nmstatev1 "github.com/nmstate/kubernetes-nmstate/pkg/apis/nmstate/v1"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
)

func findInterface(name string, interfaces nmstatev1.State) bool {
	//TODO: State is a interface{} we will have to convert it to json
	//      and look for the interface name
	return false
}

func waitInterface(namespace string, nodeName string, name string, mustExist bool) error {

	return wait.PollImmediate(5*time.Second, 50*time.Second, func() (bool, error) {
		var err error
		state := nmstatev1.NodeNetworkState{}
		err = framework.Global.Client.Get(context.TODO(), types.NamespacedName{Name: nodeName, Namespace: namespace}, &state)
		if err != nil {
			return false, err
		}
		exist := findInterface(name, state.CurrentState)
		return exist == mustExist, nil
	})

}

func waitInterfaceCreated(namespace string, nodeName string, name string) error {
	return waitInterface(namespace, nodeName, name, true)
}

func waitInterfaceDeleted(namespace string, nodeName string, name string) error {
	return waitInterface(namespace, nodeName, name, false)
}
