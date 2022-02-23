/*
Copyright The Kubernetes NMState Authors.


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package daemonset

import (
	"fmt"
	"reflect"

	"github.com/onsi/gomega/format"
	"github.com/onsi/gomega/types"

	v1 "k8s.io/api/apps/v1"
)

func BeReady() types.GomegaMatcher {
	return &BeReadyMatcher{}
}

type BeReadyMatcher struct {
	obtainedDaemonSet *v1.DaemonSet
}

func (matcher *BeReadyMatcher) Match(obtained interface{}) (success bool, err error) {
	obtainedDaemonset, ok := obtained.(v1.DaemonSet)

	if !ok {
		return false, fmt.Errorf(
			"daemonset.IsReady matcher expects a v1.DaemonSet %v %v",
			reflect.TypeOf(obtained),
			reflect.TypeOf(obtainedDaemonset),
		)
	}

	matcher.obtainedDaemonSet = &obtainedDaemonset
	return matcher.expectedNumberOfPods() == matcher.availableNumberOfPods(), nil
}

func (matcher *BeReadyMatcher) FailureMessage(actual interface{}) (message string) {
	return matcher.message("to equal")
}

func (matcher *BeReadyMatcher) NegatedFailureMessage(actual interface{}) (message string) {
	return matcher.message("to not equal")
}

func (matcher *BeReadyMatcher) message(message string) string {
	return format.Message(
		matcher.expectedNumberOfPods(),
		fmt.Sprintf("daemonset.Status.DesiredNumberScheduled %v daemonset.Status.NumberAvailable", message),
		matcher.availableNumberOfPods(),
	)
}

func (matcher *BeReadyMatcher) expectedNumberOfPods() int32 {
	if matcher.obtainedDaemonSet == nil {
		return 0
	}
	return matcher.obtainedDaemonSet.Status.DesiredNumberScheduled
}

func (matcher *BeReadyMatcher) availableNumberOfPods() int32 {
	if matcher.obtainedDaemonSet == nil {
		return 0
	}
	return matcher.obtainedDaemonSet.Status.NumberAvailable
}
