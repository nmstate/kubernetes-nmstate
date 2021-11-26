package helper

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/nmstate/kubernetes-nmstate/api/shared"
	nmstatev1beta1 "github.com/nmstate/kubernetes-nmstate/api/v1beta1"
	"github.com/nmstate/kubernetes-nmstate/pkg/names"
	"github.com/nmstate/kubernetes-nmstate/pkg/nmstatectl"
	"github.com/nmstate/kubernetes-nmstate/pkg/probe"
)

var (
	log = logf.Log.WithName("client")
)

const defaultGwRetrieveTimeout = 120 * time.Second
const defaultGwProbeTimeout = 120 * time.Second
const apiServerProbeTimeout = 120 * time.Second

func InitializeNodeNetworkState(client client.Client, node *corev1.Node) (*nmstatev1beta1.NodeNetworkState, error) {
	ownerRefList := []metav1.OwnerReference{{Name: node.ObjectMeta.Name, Kind: "Node", APIVersion: "v1", UID: node.UID}}

	nodeNetworkState := nmstatev1beta1.NodeNetworkState{
		// Create NodeNetworkState for this node
		ObjectMeta: metav1.ObjectMeta{
			Name:            node.ObjectMeta.Name,
			OwnerReferences: ownerRefList,
			Labels:          names.IncludeRelationshipLabels(nil),
		},
	}

	err := client.Create(context.TODO(), &nodeNetworkState)
	if err != nil {
		return nil, fmt.Errorf("error creating NodeNetworkState: %v, %+v", err, nodeNetworkState)
	}

	return &nodeNetworkState, nil
}

func CreateOrUpdateNodeNetworkState(client client.Client, node *corev1.Node, observedState shared.State, nns *nmstatev1beta1.NodeNetworkState) error {
	if nns == nil {
		var err error
		nns, err = InitializeNodeNetworkState(client, node)
		if err != nil {
			return err
		}
	}
	return UpdateCurrentState(client, nns, observedState)
}

func UpdateCurrentState(client client.Client, nodeNetworkState *nmstatev1beta1.NodeNetworkState, observedState shared.State) error {

	if observedState.String() == nodeNetworkState.Status.CurrentState.String() {
		log.Info("Skipping NodeNetworkState update, node network configuration not changed")
		return nil
	}

	nodeNetworkState.Status.CurrentState = observedState
	nodeNetworkState.Status.LastSuccessfulUpdateTime = metav1.Time{Time: time.Now()}

	err := client.Status().Update(context.Background(), nodeNetworkState)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return errors.Wrap(err, "Request object not found, could have been deleted after reconcile request")
		} else {
			return errors.Wrap(err, "Error updating nodeNetworkState")
		}
	}

	return nil
}

func rollback(client client.Client, probes []probe.Probe, cause error) error {
	message := fmt.Sprintf("rolling back desired state configuration: %s", cause)
	err := nmstatectl.Rollback()
	if err != nil {
		return errors.Wrap(err, message)
	}

	// wait for system to settle after rollback
	probesErr := probe.Run(client, probes)
	if probesErr != nil {
		return errors.Wrap(errors.Wrap(err, "failed running probes after rollback"), message)
	}
	return errors.New(message)
}

func ApplyDesiredState(client client.Client, desiredState shared.State) (string, error) {
	if len(string(desiredState.Raw)) == 0 {
		return "Ignoring empty desired state", nil
	}

	out, err := EnableVlanFiltering(desiredState)
	if err != nil {
		return out, fmt.Errorf("failed to enable vlan filtering via nmcli: %s", err.Error())
	}

	// Before apply we get the probes that are working fine, they should be
	// working fine after apply
	probes := probe.Select(client)

	// commit timeout doubles the default gw ping probe and check API server
	// connectivity timeout, to
	// ensure the Checkpoint is alive before rolling it back
	// https://nmstate.github.io/cli_guide#manual-transaction-control
	setOutput, err := nmstatectl.Set(desiredState, (defaultGwProbeTimeout+apiServerProbeTimeout)*2)
	if err != nil {
		return setOutput, err
	}

	err = probe.Run(client, probes)
	if err != nil {
		return "", rollback(client, probes, errors.Wrap(err, "failed runnig probes after network changes"))
	}

	commitOutput, err := nmstatectl.Commit()
	if err != nil {
		// We cannot rollback if commit fails, just return the error
		return commitOutput, err
	}

	commandOutput := fmt.Sprintf("setOutput: %s \n", setOutput)
	return commandOutput, nil
}
