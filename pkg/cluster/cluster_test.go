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

package cluster

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	apidiscoveryv2 "k8s.io/api/apidiscovery/v2"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

func TestIsOpenShift(t *testing.T) {
	for _, tc := range []struct {
		name      string
		status    int
		body      string
		openShift bool
		wantError bool
	}{
		{
			name: "OpenShift", status: http.StatusOK, openShift: true,
			body: `{"groupVersion":"security.openshift.io/v1","resources":[{"name":"securitycontextconstraints"}]}`,
		},
		{name: "Kubernetes", status: http.StatusNotFound},
		{name: "resource absent", status: http.StatusOK, body: `{"resources":[{"name":"other"}]}`},
		{name: "unavailable", status: http.StatusServiceUnavailable, wantError: true},
		{name: "forbidden", status: http.StatusForbidden, wantError: true},
		{name: "unauthorized", status: http.StatusUnauthorized, wantError: true},
		{name: "invalid response", status: http.StatusOK, body: `{`, wantError: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/apis/security.openshift.io/v1" {
					t.Errorf("unexpected discovery request: %s", r.URL.Path)
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer server.Close()
			client, err := discovery.NewDiscoveryClientForConfig(&rest.Config{Host: server.URL})
			if err != nil {
				t.Fatal(err)
			}
			got, err := IsOpenShift(client)
			if got != tc.openShift || (err != nil) != tc.wantError {
				t.Fatalf("IsOpenShift() = %t, %v; want %t, error=%t", got, err, tc.openShift, tc.wantError)
			}
			if tc.status == http.StatusForbidden && !apierrors.IsForbidden(err) {
				t.Fatalf("expected wrapped Forbidden error, got %v", err)
			}
		})
	}
}

// Reproduce the silent false negative from the REST mapper when aggregated
// discovery marks the OpenShift API stale during a control-plane restart.
func TestIsOpenShiftWithStaleAggregatedDiscovery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/apis/security.openshift.io/v1" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(metav1.APIResourceList{
				GroupVersion: "security.openshift.io/v1",
				APIResources: []metav1.APIResource{{Name: "securitycontextconstraints", Kind: "SecurityContextConstraints"}},
			})
			return
		}
		w.Header().Set("Content-Type", "application/json;g=apidiscovery.k8s.io;v=v2;as=APIGroupDiscoveryList")
		groups := apidiscoveryv2.APIGroupDiscoveryList{
			TypeMeta: metav1.TypeMeta{APIVersion: "apidiscovery.k8s.io/v2", Kind: "APIGroupDiscoveryList"},
		}
		switch r.URL.Path {
		case "/api":
			groups.Items = []apidiscoveryv2.APIGroupDiscovery{{Versions: []apidiscoveryv2.APIVersionDiscovery{{
				Version: "v1", Freshness: apidiscoveryv2.DiscoveryFreshnessCurrent,
			}}}}
		case "/apis":
			groups.Items = []apidiscoveryv2.APIGroupDiscovery{{
				ObjectMeta: metav1.ObjectMeta{Name: "security.openshift.io"},
				Versions: []apidiscoveryv2.APIVersionDiscovery{{
					Version: "v1", Freshness: apidiscoveryv2.DiscoveryFreshnessStale,
				}},
			}}
		default:
			t.Errorf("unexpected discovery request: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(groups)
	}))
	defer server.Close()
	cfg := &rest.Config{Host: server.URL}
	mapper, err := apiutil.NewDynamicRESTMapper(cfg, server.Client())
	if err != nil {
		t.Fatal(err)
	}
	_, err = mapper.ResourcesFor(schema.GroupVersionResource{
		Group: "security.openshift.io", Version: "v1", Resource: "securitycontextconstraints",
	})
	if !meta.IsNoMatchError(err) {
		t.Fatalf("expected stale discovery to produce a NoMatch error, got %v", err)
	}
	client, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if got, err := IsOpenShift(client); !got || err != nil {
		t.Fatalf("direct discovery should identify OpenShift despite stale aggregated discovery: %t, %v", got, err)
	}
}
