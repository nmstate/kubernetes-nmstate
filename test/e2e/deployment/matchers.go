package deployment

import (
	"fmt"

	"github.com/onsi/gomega/format"
	"github.com/onsi/gomega/types"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	deputils "k8s.io/kubectl/pkg/util/deployment"
)

func BeReady() types.GomegaMatcher {
	return &BeReadyMatcher{}
}

type BeReadyMatcher struct {
	obtainedDeployment *appsv1.Deployment
}

func (matcher *BeReadyMatcher) Match(obtained interface{}) (success bool, err error) {
	obtainedDeployment, ok := obtained.(appsv1.Deployment)

	if !ok {
		return false, fmt.Errorf("deployment.IsReady matcher expects a v1.Deployment")
	}

	matcher.obtainedDeployment = &obtainedDeployment

	cond := deputils.GetDeploymentCondition(matcher.obtainedDeployment.Status, appsv1.DeploymentAvailable)
	if cond == nil {
		return false, fmt.Errorf("deployment.Status does not contain the DeploymentAvailable condition")
	}

	return cond.Status == corev1.ConditionTrue, nil
}

func (matcher *BeReadyMatcher) FailureMessage(actual interface{}) (message string) {
	return matcher.message("to equal")
}

func (matcher *BeReadyMatcher) NegatedFailureMessage(actual interface{}) (message string) {
	return matcher.message("to not equal")
}

func (matcher *BeReadyMatcher) message(message string) string {
	cond := deputils.GetDeploymentCondition(matcher.obtainedDeployment.Status, appsv1.DeploymentAvailable)
	return format.Message(cond, fmt.Sprintf("deployment.Status.Condition %v corev1.ConditionTrue", message), corev1.ConditionTrue)
}
