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

package readiness

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-logr/logr"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/nmstate/kubernetes-nmstate/pkg/environment"
)

var log = logf.Log.WithName("webhook-readiness")

// CheckerConfig holds configuration for webhook readiness checks
type CheckerConfig struct {
	Namespace      string
	ServiceName    string
	DeploymentName string
	Timeout        time.Duration
	CheckInterval  time.Duration
}

// NewCheckerConfig creates a CheckerConfig from environment variables
func NewCheckerConfig(timeout time.Duration) CheckerConfig {
	namespace := environment.GetEnvVar("WEBHOOK_NAMESPACE", os.Getenv("HANDLER_NAMESPACE"))
	prefix := environment.GetEnvVar("HANDLER_PREFIX", "")
	serviceName := prefix + "nmstate-webhook"
	deploymentName := prefix + "nmstate-webhook"

	checkIntervalStr := environment.GetEnvVar("WEBHOOK_READINESS_CHECK_INTERVAL", "5s")
	checkInterval, err := time.ParseDuration(checkIntervalStr)
	if err != nil {
		log.Info("Invalid WEBHOOK_READINESS_CHECK_INTERVAL, using default 5s", "error", err)
		checkInterval = 5 * time.Second
	}

	return CheckerConfig{
		Namespace:      namespace,
		ServiceName:    serviceName,
		DeploymentName: deploymentName,
		Timeout:        timeout,
		CheckInterval:  checkInterval,
	}
}

// WaitForWebhookReady performs a three-stage check to verify webhook readiness:
// 1. Check if webhook pods are ready
// 2. Check if webhook service has available endpoints
// 3. Attempt connection to webhook service (TLS validation skipped for connectivity check only)
//
// Returns true if webhook is ready, false if any stage fails or timeout is reached.
// The caller should decide how to handle the false result (e.g., fail-open, retry, abort).
func WaitForWebhookReady(ctx context.Context, cli client.Client, config CheckerConfig) bool {
	logger := log.WithName("WaitForWebhookReady").WithValues(
		"namespace", config.Namespace,
		"service", config.ServiceName,
		"timeout", config.Timeout.String())

	logger.Info("Starting webhook readiness check")

	timeoutCtx, cancel := context.WithTimeout(ctx, config.Timeout)
	defer cancel()

	// Stage 1: Check webhook pod readiness
	if !waitForWebhookPodsReady(timeoutCtx, cli, config) {
		logger.Info("Webhook pods not ready within timeout")
		return false
	}

	// Stage 2: Check webhook service endpoints
	if !waitForWebhookEndpoints(timeoutCtx, cli, config) {
		logger.Info("Webhook service endpoints not ready within timeout")
		return false
	}

	// Stage 3: Test webhook connectivity and TLS certificate
	if !testWebhookConnection(timeoutCtx, cli, config) {
		logger.Info("Webhook connection test failed within timeout")
		return false
	}

	logger.Info("Webhook is ready")
	return true
}

// waitForWebhookPodsReady checks if webhook deployment has ready replicas
func waitForWebhookPodsReady(ctx context.Context, cli client.Client, config CheckerConfig) bool {
	logger := log.WithName("waitForWebhookPodsReady")
	logger.Info("Stage 1/3: Checking webhook pod readiness")

	deploymentKey := types.NamespacedName{
		Namespace: config.Namespace,
		Name:      config.DeploymentName,
	}

	err := wait.PollUntilContextTimeout(ctx, config.CheckInterval, config.Timeout, true, func(ctx context.Context) (bool, error) {
		deployment := &appsv1.Deployment{}
		if err := cli.Get(ctx, deploymentKey, deployment); err != nil {
			logger.V(1).Info("Failed to get webhook deployment, retrying...", "error", err)
			return false, nil // Keep trying
		}

		if deployment.Status.ReadyReplicas > 0 {
			logger.Info("Webhook pods are ready", "readyReplicas", deployment.Status.ReadyReplicas)
			return true, nil
		}

		logger.V(1).Info("Webhook pods not ready yet",
			"readyReplicas", deployment.Status.ReadyReplicas,
			"replicas", deployment.Status.Replicas)
		return false, nil
	})

	return err == nil
}

// waitForWebhookEndpoints checks if webhook service has available endpoints
func waitForWebhookEndpoints(ctx context.Context, cli client.Client, config CheckerConfig) bool {
	logger := log.WithName("waitForWebhookEndpoints")
	logger.Info("Stage 2/3: Checking webhook service endpoints")

	endpointsKey := types.NamespacedName{
		Namespace: config.Namespace,
		Name:      config.ServiceName,
	}

	err := wait.PollUntilContextTimeout(ctx, config.CheckInterval, config.Timeout, true, func(ctx context.Context) (bool, error) {
		endpoints := &corev1.Endpoints{}
		if err := cli.Get(ctx, endpointsKey, endpoints); err != nil {
			logger.V(1).Info("Failed to get webhook service endpoints, retrying...", "error", err)
			return false, nil // Keep trying
		}

		// Check if there are any available addresses
		for _, subset := range endpoints.Subsets {
			if len(subset.Addresses) > 0 {
				logger.Info("Webhook service endpoints are available",
					"addresses", len(subset.Addresses))
				return true, nil
			}
		}

		logger.V(1).Info("Webhook service has no available endpoints yet")
		return false, nil
	})

	return err == nil
}

// testWebhookConnection attempts to connect to the webhook service and validate its TLS certificate.
// If TLS validation fails, it logs detailed certificate information but continues (does not fail the check).
// This allows the handler to proceed even when certificates are not yet properly configured, while providing
// diagnostic information for debugging certificate issues.
func testWebhookConnection(ctx context.Context, cli client.Client, config CheckerConfig) bool {
	logger := log.WithName("testWebhookConnection")
	logger.Info("Stage 3/3: Testing webhook connectivity and TLS certificate")

	// Construct webhook URL
	webhookURL := fmt.Sprintf("https://%s.%s.svc:443/readyz",
		config.ServiceName, config.Namespace)

	// Get CA bundle from webhook configuration
	caBundle := getCABundleFromWebhookConfig(ctx, cli, config)

	// Create cert pool if CA bundle is available
	var rootCAs *x509.CertPool
	if len(caBundle) > 0 {
		rootCAs = x509.NewCertPool()
		if !rootCAs.AppendCertsFromPEM(caBundle) {
			logger.Info("Warning: Failed to parse CA bundle from webhook configuration")
			rootCAs = nil
		} else {
			logger.V(1).Info("Using CA bundle from webhook configuration for TLS validation")
		}
	} else {
		logger.V(1).Info("CA bundle not available - certificate validation may fail (this is expected during startup)")
	}

	err := wait.PollUntilContextTimeout(ctx, config.CheckInterval, config.Timeout, true, func(ctx context.Context) (bool, error) {
		// Capture peer certificates for logging
		var peerCerts []*x509.Certificate

		// Create TLS config that captures certificates
		tlsConfig := &tls.Config{
			MinVersion: tls.VersionTLS12,
			RootCAs:    rootCAs, // nil if no CA bundle (will fail validation but we handle that below)
			VerifyPeerCertificate: func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
				// Capture certificates for potential logging
				for _, rawCert := range rawCerts {
					cert, err := x509.ParseCertificate(rawCert)
					if err == nil {
						peerCerts = append(peerCerts, cert)
					}
				}
				// Don't fail here - let normal validation proceed
				return nil
			},
		}

		// Create HTTP client with TLS validation enabled
		client := &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: tlsConfig,
			},
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, webhookURL, http.NoBody)
		if err != nil {
			logger.V(1).Info("Failed to create request", "error", err)
			return false, nil
		}

		resp, err := client.Do(req)
		if err != nil {
			// Check if this is a TLS/certificate error
			if isTLSError(err) {
				logger.Info("TLS certificate validation failed - logging certificate details",
					"error", err.Error())
				logCertificateDetails(logger, peerCerts, err)
				// Continue - don't fail the readiness check for TLS errors
				// The handler can proceed and status updates will be retried
				logger.V(1).Info("Continuing despite TLS error (will retry)")
				return false, nil // Keep trying
			}
			// Non-TLS errors: log and retry
			logger.V(1).Info("Failed to connect to webhook, retrying...", "error", err)
			return false, nil // Keep trying
		}
		defer resp.Body.Close()

		logger.Info("Successfully connected to webhook with valid TLS certificate", "statusCode", resp.StatusCode)
		return true, nil
	})

	return err == nil
}

// getCABundleFromWebhookConfig retrieves the CA bundle from the MutatingWebhookConfiguration.
// Returns the CA bundle bytes or nil if not found/not yet injected.
func getCABundleFromWebhookConfig(ctx context.Context, cli client.Client, config CheckerConfig) []byte {
	logger := log.WithName("getCABundleFromWebhookConfig")

	// Construct webhook configuration name using the same pattern as the service
	prefix := environment.GetEnvVar("HANDLER_PREFIX", "")
	webhookConfigName := prefix + "nmstate"

	webhookConfig := &admissionregistrationv1.MutatingWebhookConfiguration{}
	err := cli.Get(ctx, types.NamespacedName{Name: webhookConfigName}, webhookConfig)
	if err != nil {
		logger.V(1).Info("Failed to get MutatingWebhookConfiguration", "name", webhookConfigName, "error", err)
		return nil
	}

	// Find the webhook that matches our service
	for i := range webhookConfig.Webhooks {
		webhook := &webhookConfig.Webhooks[i]
		if webhook.ClientConfig.Service != nil &&
			webhook.ClientConfig.Service.Name == config.ServiceName &&
			webhook.ClientConfig.Service.Namespace == config.Namespace {
			if len(webhook.ClientConfig.CABundle) > 0 {
				logger.V(1).Info("Found CA bundle in webhook configuration",
					"webhook", webhook.Name,
					"caBundleSize", len(webhook.ClientConfig.CABundle))
				return webhook.ClientConfig.CABundle
			}
			logger.V(1).Info("CA bundle not yet injected in webhook configuration", "webhook", webhook.Name)
			return nil
		}
	}

	logger.V(1).Info("Webhook not found in MutatingWebhookConfiguration",
		"service", config.ServiceName,
		"namespace", config.Namespace)
	return nil
}

// isTLSError checks if an error is related to TLS certificate validation
func isTLSError(err error) bool {
	if err == nil {
		return false
	}

	errStr := strings.ToLower(err.Error())
	tlsErrorPatterns := []string{
		"x509:",
		"certificate",
		"tls:",
		"unknown authority",
		"crypto/rsa: verification error",
		"certificate signed by unknown authority",
		"certificate has expired",
		"certificate is not valid",
	}

	for _, pattern := range tlsErrorPatterns {
		if strings.Contains(errStr, pattern) {
			return true
		}
	}

	return false
}

// logCertificateDetails logs comprehensive certificate information at Warning level
func logCertificateDetails(logger logr.Logger, certs []*x509.Certificate, err error) {
	if len(certs) == 0 {
		logger.Info("No certificates available to log (connection may have failed before TLS handshake)")
		return
	}

	logger = logger.WithName("certificate-details")
	logger.Info("TLS certificate validation failed - logging certificate details for debugging",
		"error", err.Error(),
		"certificateCount", len(certs))

	for i, cert := range certs {
		certType := "leaf"
		if i > 0 {
			certType = fmt.Sprintf("intermediate-%d", i)
		}

		// Extract DNS SANs
		dnsNames := cert.DNSNames
		if len(dnsNames) == 0 {
			dnsNames = []string{"<none>"}
		}

		// Format validity period
		now := time.Now()
		var validityStatus string
		if now.Before(cert.NotBefore) {
			validityStatus = "NOT_YET_VALID"
		} else if now.After(cert.NotAfter) {
			validityStatus = "EXPIRED"
		} else {
			validityStatus = "VALID"
		}

		// Log certificate details with structured fields
		logger.Info(fmt.Sprintf("Certificate #%d (%s)", i, certType),
			"subject", cert.Subject.String(),
			"issuer", cert.Issuer.String(),
			"serialNumber", cert.SerialNumber.String(),
			"notBefore", cert.NotBefore.Format(time.RFC3339),
			"notAfter", cert.NotAfter.Format(time.RFC3339),
			"validityStatus", validityStatus,
			"dnsNames", strings.Join(dnsNames, ", "),
			"signatureAlgorithm", cert.SignatureAlgorithm.String(),
			"publicKeyAlgorithm", cert.PublicKeyAlgorithm.String(),
			"version", cert.Version,
			"isCA", cert.IsCA,
		)

		// Log certificate in PEM format for easy inspection
		certPEM := pem.EncodeToMemory(&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: cert.Raw,
		})
		logger.V(1).Info(fmt.Sprintf("Certificate #%d PEM", i), "pem", string(certPEM))
	}
}
