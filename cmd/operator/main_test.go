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

package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// Embedding Manager makes unexpected controller setup calls fail the test:
// discovery failure must stop setup before a controller can be registered.
type discoveryTestManager struct {
	manager.Manager
	config *rest.Config
}

func (m *discoveryTestManager) GetConfig() *rest.Config    { return m.config }
func (m *discoveryTestManager) GetScheme() *runtime.Scheme { return scheme }
func (m *discoveryTestManager) GetRESTMapper() meta.RESTMapper {
	return meta.NewDefaultRESTMapper(nil)
}

func TestSetupOperatorControllerStopsOnDiscoveryFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/apis/security.openshift.io/v1" {
			t.Errorf("unexpected discovery request: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()
	mgr := &discoveryTestManager{config: &rest.Config{Host: server.URL}}
	err := setupOperatorController(mgr)
	if err == nil || !strings.Contains(err.Error(), "failed detecting cluster platform") {
		t.Fatalf("expected platform detection to stop controller setup, got %v", err)
	}
	if !apierrors.IsServiceUnavailable(err) {
		t.Fatalf("expected underlying discovery error to be preserved, got %v", err)
	}
}
