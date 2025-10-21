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

package controllers

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	rbac "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/openshift/cluster-network-operator/pkg/render"

	openshiftoperatorv1 "github.com/openshift/api/operator/v1"

	"github.com/nmstate/kubernetes-nmstate/api/names"
	"github.com/nmstate/kubernetes-nmstate/api/shared"
	nmstatev1 "github.com/nmstate/kubernetes-nmstate/api/v1"
	"github.com/nmstate/kubernetes-nmstate/pkg/cluster"
	"github.com/nmstate/kubernetes-nmstate/pkg/environment"
	nmstaterenderer "github.com/nmstate/kubernetes-nmstate/pkg/render"
)

const (
	nmstateOperatorFieldOwner = client.FieldOwner("nmstate-operator")
)

// NMStateReconciler reconciles a NMState object
type NMStateReconciler struct {
	client.Client
	APIClient   client.Client
	Log         logr.Logger
	Scheme      *runtime.Scheme
	deployments []client.ObjectKey
	daemonSets  []client.ObjectKey
}

// +kubebuilder:rbac:groups="",resources=services;endpoints;persistentvolumeclaims;events;configmaps;secrets;pods,verbs="*"
// ,namespace="{{ .OperatorNamespace }}"
// +kubebuilder:rbac:groups=apps,resources=deployments;daemonsets;replicasets;statefulsets,verbs="*",namespace="{{ .OperatorNamespace }}"
// +kubebuilder:rbac:groups=policy,resources=poddisruptionbudgets,verbs="*",namespace="{{ .OperatorNamespace }}"
// +kubebuilder:rbac:groups=admissionregistration.k8s.io,resources=mutatingwebhookconfigurations,verbs="*"
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterroles;clusterrolebindings;rolebindings;roles,verbs="*"
// +kubebuilder:rbac:groups=nmstate.io,resources="*",verbs="*"
// +kubebuilder:rbac:groups=apiextensions.k8s.io,resources="*",verbs="*"
// +kubebuilder:rbac:groups=apps,resources=deployments;daemonsets;replicasets;statefulsets,verbs="*"
// +kubebuilder:rbac:groups="",resources=serviceaccounts;configmaps;namespaces,verbs="*"
// +kubebuilder:rbac:groups="",resources=nodes,verbs=list;get
// +kubebuilder:rbac:groups="console.openshift.io",resources=consoleplugins,verbs="*"
// +kubebuilder:rbac:groups="operator.openshift.io",resources=consoles,verbs=list;get;watch;update
// +kubebuilder:rbac:groups="monitoring.coreos.com",resources=servicemonitors,verbs=list;get;watch;update;create;patch
// +kubebuilder:rbac:groups="networking.k8s.io",resources=networkpolicies,verbs="*"

func (r *NMStateReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = r.Log.WithValues("nmstate", req.NamespacedName)
	r.Log.Info("Starting Reconcile")
	instance := &nmstatev1.NMState{}
	err := r.Client.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile req.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the req.
		return ctrl.Result{}, err
	}

	// Fetch the NMState instance
	instanceList := &nmstatev1.NMStateList{}
	if err := r.Client.List(ctx, instanceList, &client.ListOptions{}); err != nil {
		err = errors.Wrap(err, "failed listing all NMState instances")
		r.setDegradedCondition(ctx, instance, shared.NmstateInternalError, err.Error())
		return ctrl.Result{}, err
	}

	// We only want one instance of NMState. Ignore anything after that.
	if len(instanceList.Items) > 0 {
		if len(instanceList.Items) > 1 {
			sort.Slice(instanceList.Items, func(i, j int) bool {
				return instanceList.Items[j].CreationTimestamp.After(instanceList.Items[i].CreationTimestamp.Time)
			})
		}
		if instanceList.Items[0].Name != req.Name {
			r.Log.Info("Ignoring NMState.nmstate.io because one already exists and does not match existing name")
			err = r.Client.Delete(ctx, instance, &client.DeleteOptions{})
			if err != nil {
				r.Log.Error(err, "failed to remove NMState.nmstate.io instance")
			}
			return ctrl.Result{}, nil
		}
	}

	r.deployments = []client.ObjectKey{}
	r.daemonSets = []client.ObjectKey{}

	if err := r.applyManifests(instance, ctx); err != nil {
		r.setDegradedCondition(ctx, instance, shared.NmstateInternalError, err.Error())
		return ctrl.Result{}, err
	}

	if err := r.cleanupObsoleteResources(ctx); err != nil {
		return ctrl.Result{}, err
	}

	if err := r.reconcileStatus(ctx, instance); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed reconciling status: %w", err)
	}

	r.Log.Info("Reconcile complete.")
	return ctrl.Result{}, nil
}

func (r *NMStateReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&nmstatev1.NMState{}).
		Owns(&appsv1.Deployment{}).
		Owns(&appsv1.DaemonSet{}).
		Complete(r)
}

func (r *NMStateReconciler) applyManifests(instance *nmstatev1.NMState, ctx context.Context) error {
	if err := r.applyCRDs(ctx, instance); err != nil {
		return errors.Wrap(err, "failed applying CRDs")
	}

	if err := r.applyNamespace(ctx, instance); err != nil {
		return errors.Wrap(err, "failed applying Namespace")
	}

	if err := r.applyRBAC(ctx, instance); err != nil {
		return errors.Wrap(err, "failed applying RBAC")
	}

	if err := r.applyNetworkPolicies(ctx, instance); err != nil {
		return errors.Wrap(err, "failed applying network policies")
	}

	if err := r.applyHandler(ctx, instance); err != nil {
		return errors.Wrap(err, "failed applying Handler")
	}

	isOpenShift, err := cluster.IsOpenShift(r.APIClient)

	_, errUIPluginPathExists := os.Stat(filepath.Join(names.ManifestDir, "kubernetes-nmstate", "openshift", "ui-plugin"))
	if err == nil && isOpenShift && errUIPluginPathExists == nil {
		if err = r.applyOpenshiftUIPlugin(ctx, instance); err != nil {
			return errors.Wrap(err, "failed applying UI Plugin")
		}
		if err = r.patchOpenshiftConsolePlugin(ctx); err != nil {
			return errors.Wrap(err, "failed enabling the plugin in cluster's console")
		}
	} else if err != nil {
		r.Log.Info("Warning: could not determine if running on OpenShift")
	}
	return nil
}

func (r *NMStateReconciler) applyCRDs(ctx context.Context, instance *nmstatev1.NMState) error {
	data := render.MakeRenderData()
	return r.renderAndApply(ctx, instance, data, "crds", false)
}

func (r *NMStateReconciler) applyNamespace(ctx context.Context, instance *nmstatev1.NMState) error {
	data := render.MakeRenderData()
	data.Data["HandlerNamespace"] = os.Getenv("HANDLER_NAMESPACE")
	data.Data["HandlerPrefix"] = os.Getenv("HANDLER_PREFIX")
	return r.renderAndApply(ctx, instance, data, "namespace", false)
}

func (r *NMStateReconciler) applyNetworkPolicies(ctx context.Context, instance *nmstatev1.NMState) error {
	data := render.MakeRenderData()
	data.Data["HandlerNamespace"] = os.Getenv("HANDLER_NAMESPACE")
	data.Data["OperatorNamespace"] = os.Getenv("OPERATOR_NAMESPACE")
	data.Data["PluginNamespace"] = os.Getenv("HANDLER_NAMESPACE")

	isOpenShift, err := cluster.IsOpenShift(r.APIClient)
	if err != nil {
		return err
	}
	data.Data["IsOpenShift"] = isOpenShift

	return r.renderAndApply(ctx, instance, data, "netpol", true)
}

func (r *NMStateReconciler) applyRBAC(ctx context.Context, instance *nmstatev1.NMState) error {
	data := render.MakeRenderData()
	data.Data["HandlerNamespace"] = os.Getenv("HANDLER_NAMESPACE")
	data.Data["HandlerImage"] = os.Getenv("RELATED_IMAGE_HANDLER_IMAGE")
	data.Data["HandlerPullPolicy"] = os.Getenv("HANDLER_IMAGE_PULL_POLICY")
	data.Data["HandlerPrefix"] = os.Getenv("HANDLER_PREFIX")

	if err := setClusterReaderExist(ctx, r.Client, data); err != nil {
		return errors.Wrap(err, "failed checking if cluster-reader ClusterRole exists")
	}

	isOpenShift, err := cluster.IsOpenShift(r.APIClient)
	if err != nil {
		return err
	}
	data.Data["IsOpenShift"] = isOpenShift

	return r.renderAndApply(ctx, instance, data, "rbac", true)
}

// nolint: funlen
func (r *NMStateReconciler) applyHandler(ctx context.Context, instance *nmstatev1.NMState) error {
	data := render.MakeRenderData()
	// Register ToYaml template method
	data.Funcs["toYaml"] = nmstaterenderer.ToYaml
	// Prepare defaults
	masterExistsNoScheduleTolerations := []corev1.Toleration{
		{
			Key:      "node-role.kubernetes.io/master",
			Operator: corev1.TolerationOpExists,
			Effect:   corev1.TaintEffectNoSchedule,
		},
		{
			Key:      "node-role.kubernetes.io/control-plane",
			Operator: corev1.TolerationOpExists,
			Effect:   corev1.TaintEffectNoSchedule,
		},
	}
	operatorExistsToleration := corev1.Toleration{
		Key:      "",
		Operator: corev1.TolerationOpExists,
	}
	handlerTolerations := instance.Spec.Tolerations
	if handlerTolerations == nil {
		handlerTolerations = []corev1.Toleration{operatorExistsToleration}
	}
	handlerAffinity := instance.Spec.Affinity
	if handlerAffinity == nil {
		handlerAffinity = &corev1.Affinity{}
	}

	nodeSelector := instance.Spec.NodeSelector
	if nodeSelector == nil {
		nodeSelector = map[string]string{
			corev1.LabelOSStable: "linux",
		}
	}
	infraNodeSelector := instance.Spec.InfraNodeSelector
	if infraNodeSelector == nil {
		infraNodeSelector = map[string]string{
			corev1.LabelOSStable: "linux",
		}
	}

	infraTolerations := instance.Spec.InfraTolerations
	if infraTolerations == nil {
		infraTolerations = masterExistsNoScheduleTolerations
	}

	infraAffinity := instance.Spec.InfraAffinity
	if infraAffinity == nil {
		infraAffinity = &corev1.Affinity{
			NodeAffinity: &corev1.NodeAffinity{
				PreferredDuringSchedulingIgnoredDuringExecution: []corev1.PreferredSchedulingTerm{
					{
						Weight: 10,
						Preference: corev1.NodeSelectorTerm{
							MatchExpressions: []corev1.NodeSelectorRequirement{
								{
									Key:      "node-role.kubernetes.io/control-plane",
									Operator: corev1.NodeSelectorOpExists,
								},
							},
						},
					},
					{
						Weight: 1,
						Preference: corev1.NodeSelectorTerm{
							MatchExpressions: []corev1.NodeSelectorRequirement{
								{
									Key:      "node-role.kubernetes.io/master",
									Operator: corev1.NodeSelectorOpExists,
								},
							},
						},
					},
				},
			},
		}
	}

	webhookReplicaCountMin, webhookReplicaCountDesired, err := r.webhookReplicaCount(ctx, infraNodeSelector, infraTolerations)
	if err != nil {
		return fmt.Errorf("could not get min replica count for webhook: %w", err)
	}

	selfSignConfiguration := instance.Spec.SelfSignConfiguration
	if selfSignConfiguration == nil {
		selfSignConfiguration = &nmstatev1.SelfSignConfiguration{
			CARotateInterval:    "8760h0m0s",
			CAOverlapInterval:   "24h0m0s",
			CertRotateInterval:  "4380h0m0s",
			CertOverlapInterval: "24h0m0s",
		}
	}

	probeConfig := instance.Spec.ProbeConfiguration

	metricsConfig := instance.Spec.MetricsConfiguration
	if metricsConfig.BindAddress == "" {
		metricsConfig = nmstatev1.NMStateMetricsConfiguration{
			BindAddress: ":8089",
		}
	}

	logLevelHandlerCommandArg := ""
	handlerReadinessProbeExtraArg := ""
	if instance.Spec.LogLevel == shared.LogLevelDebug {
		logLevelHandlerCommandArg = "debug"
		handlerReadinessProbeExtraArg = "-vv"
	}

	data.Data["HandlerNamespace"] = os.Getenv("HANDLER_NAMESPACE")
	data.Data["HandlerImage"] = os.Getenv("RELATED_IMAGE_HANDLER_IMAGE")
	data.Data["HandlerPullPolicy"] = os.Getenv("HANDLER_IMAGE_PULL_POLICY")
	data.Data["HandlerPrefix"] = os.Getenv("HANDLER_PREFIX")
	data.Data["MonitoringNamespace"] = os.Getenv("MONITORING_NAMESPACE")
	data.Data["KubeRBACProxyImage"] = os.Getenv("KUBE_RBAC_PROXY_IMAGE")
	data.Data["InfraNodeSelector"] = infraNodeSelector
	data.Data["InfraTolerations"] = infraTolerations
	data.Data["WebhookAffinity"] = infraAffinity
	data.Data["WebhookReplicas"] = webhookReplicaCountDesired
	data.Data["WebhookMinReplicas"] = webhookReplicaCountMin
	data.Data["HandlerNodeSelector"] = nodeSelector
	data.Data["HandlerTolerations"] = handlerTolerations
	data.Data["HandlerAffinity"] = handlerAffinity
	data.Data["SelfSignConfiguration"] = selfSignConfiguration
	data.Data["ProbeConfiguration"] = probeConfig
	data.Data["MetricsConfiguration"] = metricsConfig
	data.Data["LogLevelHandlerCommandArg"] = logLevelHandlerCommandArg
	data.Data["HandlerReadinessProbeExtraArg"] = handlerReadinessProbeExtraArg

	isOpenShift, err := cluster.IsOpenShift(r.APIClient)
	if err != nil {
		return err
	}
	data.Data["IsOpenShift"] = isOpenShift

	return r.renderAndApply(ctx, instance, data, "handler", true)
}

func (r *NMStateReconciler) applyOpenshiftUIPlugin(ctx context.Context, instance *nmstatev1.NMState) error {
	data := render.MakeRenderData()
	data.Funcs["toYaml"] = nmstaterenderer.ToYaml
	data.Data["PluginNamespace"] = environment.GetEnvVar("HANDLER_NAMESPACE", "openshift-nmstate")
	data.Data["PluginName"] = environment.GetEnvVar("PLUGIN_NAME", "nmstate-console-plugin")
	data.Data["PluginImage"] = environment.GetEnvVar("PLUGIN_IMAGE", "quay.io/nmstate/nmstate-console-plugin:release-1.0.0")
	data.Data["PluginPort"] = environment.GetEnvVar("PLUGIN_PORT", "9443")

	// if not set in the NMState CR, these entries are nil
	data.Data["InfraNodeSelector"] = instance.Spec.InfraNodeSelector
	data.Data["InfraTolerations"] = instance.Spec.InfraTolerations
	data.Data["InfraAffinity"] = instance.Spec.InfraAffinity

	return r.renderAndApply(ctx, instance, data, filepath.Join("openshift", "ui-plugin"), true)
}

func (r *NMStateReconciler) patchOpenshiftConsolePlugin(ctx context.Context) error {
	// Enable console plugin for nmstate-console if not already enabled
	pluginName := environment.GetEnvVar("PLUGIN_NAME", "nmstate-console-plugin")
	consoleKey := client.ObjectKey{Name: "cluster"}
	consoleObj := &openshiftoperatorv1.Console{}
	if err := r.Client.Get(ctx, consoleKey, consoleObj); err != nil {
		r.Log.Error(err, "Could not get consoles.operator.openshift.io resource")
		return err
	}

	if !stringInSlice(pluginName, consoleObj.Spec.Plugins) {
		r.Log.Info("Enabling kubevirt plugin in Console")
		consoleObj.Spec.Plugins = append(consoleObj.Spec.Plugins, pluginName)
		err := r.Client.Update(ctx, consoleObj)
		if err != nil {
			r.Log.Error(err, fmt.Sprintf("Could not update resource - APIVersion: %s, Kind: %s, Name: %s",
				consoleObj.APIVersion, consoleObj.Kind, consoleObj.Name))
			return err
		}
	}
	return nil
}

func (r *NMStateReconciler) cleanupObsoleteResources(ctx context.Context) error {
	isOpenShift, err := cluster.IsOpenShift(r.APIClient)
	if err != nil {
		return err
	}
	// We are no longer using cert-manager at openshift, let's remove it
	if isOpenShift {
		err = r.Client.Delete(ctx, &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: os.Getenv("HANDLER_NAMESPACE"),
				Name:      os.Getenv("HANDLER_PREFIX") + "nmstate-cert-manager",
			},
		})
		if err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed deleting obsolete cert-manager deployment at openshift: %w", err)
		}

		// Remove the non openshift secret
		err = r.Client.Delete(ctx, &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: os.Getenv("HANDLER_NAMESPACE"),
				Name:      os.Getenv("HANDLER_PREFIX") + "nmstate-webhook",
			},
		})
		if err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed deleting old webhook secret at openshift: %w", err)
		}
	}
	return nil
}

// webhookReplicaCount returns the number of replicas for the nmstate webhook
// deployment based on the underlying infrastructure topology. It returns 2
// values (and error):
// 1. min. number of replicas
// 2. number of desired replicas
// 3. error
//
//nolint:gocritic
func (r *NMStateReconciler) webhookReplicaCount(
	ctx context.Context,
	nodeSelector map[string]string,
	tolerations []corev1.Toleration,
) (int, int, error) {
	const (
		multiNodeClusterReplicaCountDesired = 2
		multiNodeClusterReplicaMinCount     = 1

		singleNodeClusterReplicaCountDesired = 1
		singleNodeClusterReplicaMinCount     = 0
	)

	nodes := corev1.NodeList{}
	err := r.APIClient.List(ctx, &nodes, client.MatchingLabels(nodeSelector))
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get nodes: %w", err)
	}

	infraNodes := 0
	for _, node := range nodes.Items {
		for _, toleration := range tolerations {
			for _, nodeTaint := range node.Spec.Taints {
				if nodeTaint.Key == toleration.Key &&
					nodeTaint.Value == toleration.Value &&
					nodeTaint.Effect == toleration.Effect {
					infraNodes++
				}
			}
		}
	}

	if infraNodes > 1 {
		return multiNodeClusterReplicaMinCount, multiNodeClusterReplicaCountDesired, nil
	} else {
		return singleNodeClusterReplicaMinCount, singleNodeClusterReplicaCountDesired, nil
	}
}

func (r *NMStateReconciler) reconcileStatus(ctx context.Context, instance *nmstatev1.NMState) error {
	progressing := []string{}
	for _, deploymentKey := range r.deployments {
		deployment := &appsv1.Deployment{}
		if err := r.Client.Get(ctx, deploymentKey, deployment); err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}
			return errors.Wrap(err, "failed to get deployment")
		}
		if deployment.Status.UnavailableReplicas > 0 {
			progressing = append(progressing, fmt.Sprintf(
				"Deployment %q is not available (awaiting %d nodes)",
				deployment.Name, deployment.Status.UnavailableReplicas))
		} else if deployment.Status.AvailableReplicas == 0 && (deployment.Spec.Replicas == nil || *deployment.Spec.Replicas > 0) {
			progressing = append(progressing, fmt.Sprintf(
				"Deployment %q is not yet scheduled on any nodes", deployment.Name))
		} else if deployment.Status.ObservedGeneration < deployment.Generation {
			progressing = append(progressing, fmt.Sprintf(
				"Deployment %q update is being processed (generation %d, observed generation %d)",
				deployment.Name, deployment.Generation, deployment.Status.ObservedGeneration))
		}
	}
	for _, daemonSetKey := range r.daemonSets {
		daemonSet := &appsv1.DaemonSet{}
		if err := r.Client.Get(ctx, daemonSetKey, daemonSet); err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}
			return errors.Wrap(err, "failed to get daemonset")
		}
		if daemonSet.Status.NumberUnavailable > 0 {
			progressing = append(progressing, fmt.Sprintf(
				"DaemonSet %q is not available (awaiting %d nodes)",
				daemonSet.Name, daemonSet.Status.NumberUnavailable))
		} else if daemonSet.Status.NumberAvailable == 0 && daemonSet.Status.DesiredNumberScheduled != 0 {
			progressing = append(progressing, fmt.Sprintf("DaemonSet %q is not yet scheduled on any nodes", daemonSet.Name))
		} else if daemonSet.Status.UpdatedNumberScheduled < daemonSet.Status.DesiredNumberScheduled {
			progressing = append(progressing, fmt.Sprintf("DaemonSet %q update is rolling out (%d out of %d updated)",
				daemonSet.Name, daemonSet.Status.UpdatedNumberScheduled, daemonSet.Status.DesiredNumberScheduled))
		} else if daemonSet.Generation > daemonSet.Status.ObservedGeneration {
			progressing = append(progressing, fmt.Sprintf("DaemonSet %q update is being processed (generation %d, observed generation %d)",
				daemonSet.Name, daemonSet.Generation, daemonSet.Status.ObservedGeneration))
		}
	}

	if len(progressing) > 0 {
		instance.Status.Conditions.Set(
			shared.NmstateConditionProgressing,
			corev1.ConditionTrue,
			shared.NmstateDeploying,
			strings.Join(progressing, "\n"),
		)
		instance.Status.Conditions.Set(
			shared.NmstateConditionAvailable,
			corev1.ConditionFalse,
			shared.NmstateDeploying,
			"Deploying is in progress",
		)
		// Clear any previous degraded condition when progressing
		instance.Status.Conditions.Set(
			shared.NmstateConditionDegraded,
			corev1.ConditionFalse,
			shared.NmstateDeploying,
			"Deploying is in progress",
		)
	} else {
		instance.Status.Conditions.Set(
			shared.NmstateConditionProgressing,
			corev1.ConditionFalse,
			shared.NmstateSuccessfullyDeployed,
			"All components are available and ready",
		)
		instance.Status.Conditions.Set(
			shared.NmstateConditionAvailable,
			corev1.ConditionTrue,
			shared.NmstateSuccessfullyDeployed,
			"All components are available and ready",
		)
		// Clear any previous degraded condition when available
		instance.Status.Conditions.Set(
			shared.NmstateConditionDegraded,
			corev1.ConditionFalse,
			shared.NmstateSuccessfullyDeployed,
			"All components are available and ready",
		)
	}

	// Clear managed fields before applying server-side apply to status subresource
	instance.SetManagedFields(nil)

	return r.Client.Status().Patch(ctx, instance, client.Apply, nmstateOperatorFieldOwner)
}

func (r *NMStateReconciler) setDegradedCondition(
	ctx context.Context,
	instance *nmstatev1.NMState,
	reason shared.ConditionReason,
	message string,
) error {
	instance.Status.Conditions.Set(
		shared.NmstateConditionDegraded,
		corev1.ConditionTrue,
		reason,
		message,
	)
	instance.Status.Conditions.Set(
		shared.NmstateConditionAvailable,
		corev1.ConditionFalse,
		reason,
		message,
	)
	instance.Status.Conditions.Set(
		shared.NmstateConditionProgressing,
		corev1.ConditionFalse,
		reason,
		message,
	)
	return r.Client.Status().Update(ctx, instance)
}

func (r *NMStateReconciler) renderAndApply(
	ctx context.Context,
	instance *nmstatev1.NMState,
	data render.RenderData,
	sourceDirectory string,
	setControllerReference bool,
) error {
	var err error

	sourceFullDirectory := filepath.Join(names.ManifestDir, "kubernetes-nmstate", sourceDirectory)
	objs, err := render.RenderDir(sourceFullDirectory, &data)
	if err != nil {
		return errors.Wrapf(err, "failed to render kubernetes-nmstate %s", sourceDirectory)
	}

	// If no file found in directory - return error
	if len(objs) == 0 {
		return fmt.Errorf("no manifests rendered from %s", sourceFullDirectory)
	}

	for _, obj := range objs {
		// RenderDir seems to add an extra null entry to the list. It appears to be because of the
		// nested templates. This just makes sure we don't try to apply an empty obj.
		if obj.GetName() == "" {
			continue
		}
		if setControllerReference {
			// Set the controller reference. When the CR is removed, it will remove the CRDs as well
			err = controllerutil.SetControllerReference(instance, obj, r.Scheme)
			if err != nil {
				return errors.Wrap(err, "failed to set owner reference")
			}
			if obj.GetKind() == "Deployment" {
				r.deployments = append(r.deployments, client.ObjectKeyFromObject(obj))
			} else if obj.GetKind() == "DaemonSet" {
				r.daemonSets = append(r.daemonSets, client.ObjectKeyFromObject(obj))
			}
		}
		// Use server-side apply
		err := r.Patch(ctx, obj, client.Apply, nmstateOperatorFieldOwner)
		if err != nil {
			// If the CRD doesn't exist (e.g., ServiceMonitor), log a warning and continue
			if meta.IsNoMatchError(err) {
				r.Log.Info("Skipping resource because CRD is not available", "kind", obj.GetKind(), "name", obj.GetName(), "error", err.Error())
				continue
			}
			return errors.Wrap(err, "failed applying server-side apply on nmstate owned resource")
		}
	}
	return nil
}

func setClusterReaderExist(ctx context.Context, c client.Client, data render.RenderData) error {
	var clusterReader rbac.ClusterRole
	key := types.NamespacedName{Name: "cluster-reader"}
	err := c.Get(ctx, key, &clusterReader)

	found := true
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}
		found = false
	}

	data.Data["ClusterReaderExists"] = found
	return nil
}

func stringInSlice(a string, list []string) bool {
	return slices.Contains(list, a)
}
