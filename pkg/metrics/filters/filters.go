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

// Package filters provides authentication and authorization filters for the metrics server.
// This is a local implementation of the controller-runtime metrics filters for projects
// that don't have the filters package vendored.
package filters

import (
	"net/http"
	"strings"

	"github.com/go-logr/logr"
	authenticationv1 "k8s.io/api/authentication/v1"
	authorizationv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
)

// WithAuthenticationAndAuthorization provides a metrics.Filter for authentication and authorization.
// Metrics will be authenticated (via TokenReviews) and authorized (via SubjectAccessReviews) with the kube-apiserver.
//
// For the authentication and authorization the controller needs a ClusterRole with the following rules:
//   - apiGroups: authentication.k8s.io
//     resources:
//   - tokenreviews
//     verbs:
//   - create
//   - apiGroups: authorization.k8s.io
//     resources:
//   - subjectaccessreviews
//     verbs:
//   - create
//
// To scrape metrics (e.g. via Prometheus), the client needs a ClusterRole with the following rule:
//   - nonResourceURLs: "/metrics"
//     verbs:
//   - get
func WithAuthenticationAndAuthorization(config *rest.Config, httpClient *http.Client) (metricsserver.Filter, error) {
	clientset, err := kubernetes.NewForConfigAndClient(config, httpClient)
	if err != nil {
		return nil, err
	}

	return func(log logr.Logger, handler http.Handler) (http.Handler, error) {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			ctx := req.Context()

			// Extract bearer token from Authorization header
			authHeader := req.Header.Get("Authorization")
			if authHeader == "" {
				log.V(4).Info("No Authorization header found")
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// Check for Bearer token
			const bearerPrefix = "Bearer "
			if !strings.HasPrefix(authHeader, bearerPrefix) {
				log.V(4).Info("Authorization header is not a Bearer token")
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			token := strings.TrimPrefix(authHeader, bearerPrefix)

			// Authenticate via TokenReview
			tokenReview := &authenticationv1.TokenReview{
				Spec: authenticationv1.TokenReviewSpec{
					Token: token,
				},
			}

			result, err := clientset.AuthenticationV1().TokenReviews().Create(ctx, tokenReview, metav1.CreateOptions{})
			if err != nil {
				log.Error(err, "TokenReview failed")
				http.Error(w, "Authentication failed", http.StatusInternalServerError)
				return
			}

			if !result.Status.Authenticated {
				log.V(4).Info("Token is not authenticated", "error", result.Status.Error)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// Authorize via SubjectAccessReview
			sar := &authorizationv1.SubjectAccessReview{
				Spec: authorizationv1.SubjectAccessReviewSpec{
					User:   result.Status.User.Username,
					Groups: result.Status.User.Groups,
					NonResourceAttributes: &authorizationv1.NonResourceAttributes{
						Path: req.URL.Path,
						Verb: strings.ToLower(req.Method),
					},
				},
			}

			sarResult, err := clientset.AuthorizationV1().SubjectAccessReviews().Create(ctx, sar, metav1.CreateOptions{})
			if err != nil {
				log.Error(err, "SubjectAccessReview failed")
				http.Error(w, "Authorization failed", http.StatusInternalServerError)
				return
			}

			if !sarResult.Status.Allowed {
				log.V(4).Info("Request not authorized", "user", result.Status.User.Username, "reason", sarResult.Status.Reason)
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			// Serve the request
			handler.ServeHTTP(w, req)
		}), nil
	}, nil
}
