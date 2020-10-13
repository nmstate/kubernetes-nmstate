package enactmentstatus

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/pkg/errors"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"

	nmstate "github.com/nmstate/kubernetes-nmstate/api/shared"
	nmstatev1beta1 "github.com/nmstate/kubernetes-nmstate/api/v1beta1"
)

var (
	log = logf.Log.WithName("enactmentstatus")
)

func Update(client client.Client, key types.NamespacedName, statusSetter func(*nmstate.NodeNetworkConfigurationEnactmentStatus)) error {
	logger := log.WithValues("enactment", key.Name)
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		instance := &nmstatev1beta1.NodeNetworkConfigurationEnactment{}
		err := client.Get(context.TODO(), key, instance)
		if err != nil {
			return errors.Wrap(err, "getting enactment failed")
		}

		statusSetter(&instance.Status)

		logger.Info(fmt.Sprintf("status: %+v", instance.Status))

		err = client.Status().Update(context.TODO(), instance)
		if err != nil {
			return err
		}

		// Wait until enactment has being updated at the node
		expectedStatus := instance.Status
		return wait.PollImmediate(1*time.Second, 30*time.Second, func() (bool, error) {
			err = client.Get(context.TODO(), key, instance)
			if err != nil {
				return false, err
			}

			isEqual := reflect.DeepEqual(expectedStatus, instance.Status)
			logger.Info(fmt.Sprintf("enactment updated at the node: %t", isEqual))
			return isEqual, nil
		})
	})
}
