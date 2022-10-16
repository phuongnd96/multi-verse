/*
Copyright 2022.

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
	"strings"
	"time"

	kccNetworkv1beta1 "github.com/GoogleCloudPlatform/k8s-config-connector/pkg/clients/generated/apis/compute/v1beta1"
	kccContainerClusterv1beta1 "github.com/GoogleCloudPlatform/k8s-config-connector/pkg/clients/generated/apis/container/v1beta1"
	kccIAMv1beta1 "github.com/GoogleCloudPlatform/k8s-config-connector/pkg/clients/generated/apis/iam/v1beta1"
	kccSQLv1beta1 "github.com/GoogleCloudPlatform/k8s-config-connector/pkg/clients/generated/apis/sql/v1beta1"
	kccStoragev1beta1 "github.com/GoogleCloudPlatform/k8s-config-connector/pkg/clients/generated/apis/storage/v1beta1"
	multiversev1alpha1 "github.com/phuongnd96/multi-verse/api/v1alpha1"
	"github.com/phuongnd96/multi-verse/pkg/common"
	"github.com/phuongnd96/multi-verse/pkg/gcloud"

	// gClient "github.com/phuongnd96/multi-verse/pkg/gcloud"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
)

// UniverseReconciler reconciles a Universe object
type UniverseReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=multiverse.saga.dev,resources=universes,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=multiverse.saga.dev,resources=universes/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=multiverse.saga.dev,resources=universes/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Universe object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
const universeFinalizer = "universes.multiverse.saga.dev/finalizer"

// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.1/pkg/reconcile
func (r *UniverseReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrllog.FromContext(ctx)
	defer log.Info("End reconcile", "timestamp", time.Now())
	universe := &multiversev1alpha1.Universe{}
	err := r.Get(ctx, req.NamespacedName, universe)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			log.Info("Universe resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get Universe")
		return ctrl.Result{}, err
	}
	if universe.Annotations["multiverse.saga.dev/repo"] != "" {
		log.Info("Updating status")
		universe.Status.Repository = universe.Annotations["multiverse.saga.dev/repo"]
		err = r.Status().Update(ctx, universe, &client.UpdateOptions{})
		if err != nil {
			log.Error(err, "Add repo annotation")
		}
	}
	if universe.Annotations["multiverse.saga.dev/pr"] != "" {
		log.Info("Updating status")
		// Fetch resource update from cluster
		r.Get(ctx, req.NamespacedName, universe)
		universe.Status.PR = universe.Annotations["multiverse.saga.dev/pr"]
		err = r.Status().Update(ctx, universe, &client.UpdateOptions{})
		if err != nil {
			log.Error(err, "Add pr annotation")
		}
	}
	// iam serviceaccount
	if len(universe.Spec.IAM.IAMServiceAccounts) > 0 {
		resourceType := "IAMServiceAccount"
		for _, elem := range universe.Spec.IAM.IAMServiceAccounts {
			foundResource := &kccIAMv1beta1.IAMServiceAccount{}
			err := r.Get(ctx, types.NamespacedName{Name: elem.Name, Namespace: elem.Namespace}, foundResource)
			if err != nil && errors.IsNotFound(err) {
				// Define a new resource
				log.Info("Creating a new resource", "resourceType", resourceType)
				newResource := &kccIAMv1beta1.IAMServiceAccount{
					ObjectMeta: metav1.ObjectMeta{
						Name:      elem.CustomObjectMeta.Name,
						Namespace: elem.CustomObjectMeta.Namespace,
						Labels:    addmap(elem.CustomObjectMeta.Labels, getLabelSelector(universe.Name)),
					},
					Spec: elem.Spec,
				}
				resource := newResource
				ctrl.SetControllerReference(universe, resource, r.Scheme)
				err = r.Create(ctx, resource)
				if err != nil {
					log.Error(err, "Failed to create new resource", "resource Type", resourceType)
					return ctrl.Result{}, err
				}
				log.Info("Update resource to creating", "resourceType", resourceType)
				err, _ = r.updateStatus(ctx, resourceType, universe, common.CreatingStatus, resource.Name)
				if err != nil {
					log.Info("Failed to update resource to creating", "resourceType", resourceType, "error", err)
				}
				return ctrl.Result{Requeue: true}, nil
			}

			// Syncing resource
			if universe.Status.ObservedGeneration < universe.GetObjectMeta().GetGeneration() && universe.Status.ObservedGeneration >= 1 {
				desiredSpec := elem.Spec
				// Sync spec
				log.Info("Syncing spec", "resourceType", resourceType)
				err, _ = r.updateStatus(ctx, "IAMServiceAccount", universe, common.ReconcilingStatus, elem.Name)
				if err != nil {
					log.Error(err, "Update storageBucket to reconciling")
				}
				// Sync all mutable field
				foundResource.Spec.Description = desiredSpec.Description
				foundResource.Spec.Disabled = desiredSpec.Disabled
				foundResource.Spec.DisplayName = desiredSpec.DisplayName

				err = r.Update(ctx, foundResource, &client.UpdateOptions{})
				if err != nil {
					log.Error(err, "Syncing spec", "resourceType", resourceType)
				}
			}
			// TODO: Wait for serviceaccount to be created

			// Update status to running
			err, _ = r.updateStatus(ctx, "IAMServiceAccount", universe, common.RunningStatus, elem.Name)
			if err != nil {
				log.Error(err, "Update IAMServiceAccount to runnning after waiting for IAMServiceAccount")
			}
		}
	}
	// Delete or scale to zero
	saList, err := r.getIAMServiceAccountsForUniverse(ctx, universe, getLabelSelector(universe.Name))
	if err != nil {
		log.Error(err, "Get IAMServiceAccount with label selector")
	}
	if len(saList) > len(universe.Spec.IAM.IAMServiceAccounts) {
		var isDelete bool
		log.Info("Removing Resource")
		for _, cExisting := range saList {
			name := cExisting.Name
			isDelete = true
			for _, cInSpec := range universe.Spec.IAM.IAMServiceAccounts {
				if cInSpec.Name == name {
					isDelete = false
				}
			}
			if isDelete {
				r.Delete(ctx, &cExisting, &client.DeleteOptions{})
			}
		}
	}

	// computenetwork
	if len(universe.Spec.Compute.ComputeNetworks) > 0 {
		resourceType := "ComputeNetwork"
		for _, network := range universe.Spec.Compute.ComputeNetworks {
			foundComputeNetwork := &kccNetworkv1beta1.ComputeNetwork{}
			// computeNetwork struct can be passed empty, but we use full struct so we can use it to create if resource not exists
			err := r.Get(ctx, types.NamespacedName{Name: network.Name, Namespace: network.Namespace}, foundComputeNetwork)
			if err != nil && errors.IsNotFound(err) {
				// Define a new resource
				log.Info("Creating a new resource", "resourceType", resourceType)
				computeNetwork := &kccNetworkv1beta1.ComputeNetwork{
					ObjectMeta: metav1.ObjectMeta{
						Name:      network.CustomObjectMeta.Name,
						Namespace: network.CustomObjectMeta.Namespace,
						Labels:    addmap(network.CustomObjectMeta.Labels, getLabelSelector(universe.Name)),
					},
					Spec: network.Spec,
				}
				ctrl.SetControllerReference(universe, computeNetwork, r.Scheme)
				err = r.Create(ctx, computeNetwork)
				if err != nil {
					log.Error(err, "Failed to create new resource", "resource Type", resourceType)
					return ctrl.Result{}, err
				}
				log.Info("Update resource to creating", "resourceType", resourceType)
				err, _ = r.updateStatus(ctx, resourceType, universe, common.CreatingStatus, computeNetwork.Name)
				if err != nil {
					log.Info("Failed to update resource to creating", "resourceType", resourceType, "error", err)
				}
				return ctrl.Result{Requeue: true}, nil
			}
			// No get error or resource is already exists
			err = waitForComputeNetwork(network.Namespace, log, network.Name)
			if err != nil {
				log.Error(err, "Wait for computenetwork to be running")
			}
			// computeNetwork is updated after r.Get, so it's the resource that is currently running in cluster
			// network is the resource get from universe spec
			foundSpec := foundComputeNetwork.Spec
			desiredSpec := network.Spec
			// Only sync on spec change
			if universe.Status.ObservedGeneration < universe.GetObjectMeta().GetGeneration() && universe.Status.ObservedGeneration >= 1 {
				log.Info("Syncing spec", "resourceType", resourceType)
				// Some field of kcc resource is immutable
				foundComputeNetwork.Spec = desiredSpec
				foundComputeNetwork.Spec.ResourceID = foundSpec.ResourceID
				foundComputeNetwork.Spec.Description = foundSpec.Description
				err = r.Update(ctx, foundComputeNetwork, &client.UpdateOptions{})
				if err != nil {
					log.Error(err, "Syncing spec", "resourceType", resourceType)
				}
			}
			err, _ = r.updateStatus(ctx, "ComputeNetwork", universe, common.RunningStatus, foundComputeNetwork.Name)
			if err != nil {
				log.Error(err, "Update computenetwork to runnning after waiting for computenetwork")
			}
		}
	}
	// Delete or scale to zero
	nwList, err := r.getComputeNetworksForUniverse(ctx, universe, getLabelSelector(universe.Name))
	if err != nil {
		log.Error(err, "Get ComputeNetwork with label selector")
	}
	if len(nwList) > len(universe.Spec.Compute.ComputeNetworks) {
		var isDelete bool
		log.Info("Removing Resource")
		for _, nwExisting := range nwList {
			name := nwExisting.Name
			isDelete = true
			for _, nwInSpec := range universe.Spec.Compute.ComputeNetworks {
				if nwInSpec.Name == name {
					isDelete = false
				}
			}
			if isDelete {
				r.Delete(ctx, &nwExisting, &client.DeleteOptions{})
			}
		}
	}
	// computesubnetwork
	if len(universe.Spec.Compute.ComputeSubnetworks) > 0 {
		resourceType := "ComputeSubNetwork"
		for _, subNetwork := range universe.Spec.Compute.ComputeSubnetworks {
			foundComputeSubNetwork := &kccNetworkv1beta1.ComputeSubnetwork{}
			err := r.Get(ctx, types.NamespacedName{Name: subNetwork.Name, Namespace: subNetwork.Namespace}, foundComputeSubNetwork)
			if err != nil && errors.IsNotFound(err) {
				// Define a new resource
				log.Info("Creating a new resource", "resourceType", resourceType)
				computeSubNetwork := &kccNetworkv1beta1.ComputeSubnetwork{
					ObjectMeta: metav1.ObjectMeta{
						Name:      subNetwork.CustomObjectMeta.Name,
						Namespace: subNetwork.CustomObjectMeta.Namespace,
						Labels:    addmap(subNetwork.CustomObjectMeta.Labels, getLabelSelector(universe.Name)),
					},
					Spec: subNetwork.Spec,
				}
				resource := computeSubNetwork
				ctrl.SetControllerReference(universe, resource, r.Scheme)
				err = r.Create(ctx, resource)
				if err != nil {
					log.Error(err, "Failed to create new resource", "resource Type", resourceType)
					return ctrl.Result{}, err
				}
				log.Info("Update resource to creating", "resourceType", resourceType)
				err, _ = r.updateStatus(ctx, resourceType, universe, common.CreatingStatus, subNetwork.Name)
				if err != nil {
					log.Info("Failed to update resource to creating", "resourceType", resourceType, "error", err)
				}
				return ctrl.Result{Requeue: true}, nil
			}
			err = waitForComputeSubnetwork(foundComputeSubNetwork.Namespace, log, foundComputeSubNetwork.Name, foundComputeSubNetwork.Spec.Region)
			if err != nil {
				log.Error(err, "Wait for computesubnetwork to be running")
			}
			// TODO: Handle sync resource
			// computeNetwork is updated after r.Get, so it's the resource that is currently running in cluster
			// network is the resource get from universe spec
			foundSpec := foundComputeSubNetwork.Spec
			desiredSpec := subNetwork.Spec
			// Only sync on spec change
			if universe.Status.ObservedGeneration < universe.GetObjectMeta().GetGeneration() && universe.Status.ObservedGeneration >= 1 {
				log.Info("Syncing spec", "resourceType", resourceType)
				// Some field of kcc resource is immutable
				foundComputeSubNetwork.Spec = desiredSpec
				foundComputeSubNetwork.Spec.ResourceID = foundSpec.ResourceID
				foundComputeSubNetwork.Spec.Description = foundSpec.Description
				foundComputeSubNetwork.Spec.Purpose = foundSpec.Purpose
				err = r.Update(ctx, foundComputeSubNetwork, &client.UpdateOptions{})
				if err != nil {
					log.Error(err, "Syncing spec", "resourceType", resourceType)
				}
			}
			// Update status to running
			err, _ = r.updateStatus(ctx, "ComputeSubNetwork", universe, common.RunningStatus, foundComputeSubNetwork.Name)
			if err != nil {
				log.Error(err, "Update computesubnetwork to runnning after waiting for computesubnetwork")
			}
		}
	}
	// Delete or scale to zero
	snwList, err := r.getComputeSubNetworksForUniverse(ctx, universe, getLabelSelector(universe.Name))
	if err != nil {
		log.Error(err, "Get ComputeNetwork with label selector")
	}
	if len(snwList) > len(universe.Spec.Compute.ComputeSubnetworks) {
		var isDelete bool
		log.Info("Removing Resource")
		for _, snwExisting := range nwList {
			name := snwExisting.Name
			isDelete = true
			for _, snwInSpec := range universe.Spec.Compute.ComputeSubnetworks {
				if snwInSpec.Name == name {
					isDelete = false
				}
			}
			if isDelete {
				r.Delete(ctx, &snwExisting, &client.DeleteOptions{})
			}
		}
	}
	// containercluster
	if len(universe.Spec.Container.ContainerClusters) > 0 {
		for _, cluster := range universe.Spec.Container.ContainerClusters {
			resourceType := "ContainerCluster"
			foundContainerCluster := &kccContainerClusterv1beta1.ContainerCluster{}
			err := r.Get(ctx, types.NamespacedName{Name: cluster.Name, Namespace: cluster.Namespace}, foundContainerCluster)
			if err != nil && errors.IsNotFound(err) {
				// Define a new resource
				log.Info("Creating a new resource", "resourceType", resourceType)
				containerCluster := &kccContainerClusterv1beta1.ContainerCluster{
					ObjectMeta: metav1.ObjectMeta{
						Name:        cluster.CustomObjectMeta.Name,
						Namespace:   cluster.CustomObjectMeta.Namespace,
						Annotations: cluster.CustomObjectMeta.Annotations,
						Labels:      addmap(cluster.CustomObjectMeta.Labels, getLabelSelector(universe.Name)),
					},
					Spec: cluster.Spec,
				}
				resource := containerCluster
				ctrl.SetControllerReference(universe, resource, r.Scheme)
				err = r.Create(ctx, resource)
				if err != nil {
					log.Error(err, "Failed to create new resource", "resource Type", resourceType)
					return ctrl.Result{}, err
				}
				log.Info("Update resource to creating", "resourceType", resourceType)
				err, _ = r.updateStatus(ctx, resourceType, universe, common.CreatingStatus, cluster.Name)
				if err != nil {
					log.Info("Failed to update resource to creating", "resourceType", resourceType, "error", err)
				}
				return ctrl.Result{Requeue: true}, nil
			}
			// computeNetwork is updated after r.Get, so it's the resource that is currently running in cluster
			// network is the resource get from universe spec
			// Only sync on spec change
			if universe.Status.ObservedGeneration < universe.GetObjectMeta().GetGeneration() && universe.Status.ObservedGeneration >= 1 {
				desiredSpec := cluster.Spec
				// Sync spec
				log.Info("Syncing spec", "resourceType", resourceType)
				err, _ = r.updateStatus(ctx, "ContainerCluster", universe, common.ReconcilingStatus, cluster.Name)
				if err != nil {
					log.Error(err, "Update containerCluster to reconciling")
				}
				// Some field of kcc resource is immutable
				// Sync all mutable field
				foundContainerCluster.Spec.AddonsConfig = desiredSpec.AddonsConfig
				foundContainerCluster.Spec.ClusterAutoscaling = desiredSpec.ClusterAutoscaling
				foundContainerCluster.Spec.ClusterTelemetry = desiredSpec.ClusterTelemetry
				foundContainerCluster.Spec.DatabaseEncryption = desiredSpec.DatabaseEncryption
				foundContainerCluster.Spec.DatapathProvider = desiredSpec.DatapathProvider
				foundContainerCluster.Spec.DefaultSnatStatus = desiredSpec.DefaultSnatStatus
				foundContainerCluster.Spec.EnableBinaryAuthorization = desiredSpec.EnableBinaryAuthorization
				foundContainerCluster.Spec.EnableIntranodeVisibility = desiredSpec.EnableIntranodeVisibility
				foundContainerCluster.Spec.EnableL4IlbSubsetting = desiredSpec.EnableL4IlbSubsetting
				foundContainerCluster.Spec.EnableLegacyAbac = desiredSpec.EnableLegacyAbac
				foundContainerCluster.Spec.EnableShieldedNodes = desiredSpec.EnableShieldedNodes
				foundContainerCluster.Spec.IdentityServiceConfig = desiredSpec.IdentityServiceConfig
				foundContainerCluster.Spec.LoggingConfig = desiredSpec.LoggingConfig
				foundContainerCluster.Spec.LoggingService = desiredSpec.LoggingService
				foundContainerCluster.Spec.MaintenancePolicy = desiredSpec.MaintenancePolicy
				foundContainerCluster.Spec.MasterAuthorizedNetworksConfig = desiredSpec.MasterAuthorizedNetworksConfig
				foundContainerCluster.Spec.MinMasterVersion = desiredSpec.MinMasterVersion
				foundContainerCluster.Spec.MonitoringConfig = desiredSpec.MonitoringConfig
				foundContainerCluster.Spec.MonitoringService = desiredSpec.MonitoringService
				foundContainerCluster.Spec.NetworkPolicy = desiredSpec.NetworkPolicy
				foundContainerCluster.Spec.NetworkRef = desiredSpec.NetworkRef
				foundContainerCluster.Spec.NodeLocations = desiredSpec.NodeLocations
				foundContainerCluster.Spec.NodeVersion = desiredSpec.NodeVersion
				foundContainerCluster.Spec.NotificationConfig = desiredSpec.NotificationConfig
				foundContainerCluster.Spec.PrivateClusterConfig = desiredSpec.PrivateClusterConfig
				foundContainerCluster.Spec.PrivateIpv6GoogleAccess = desiredSpec.PrivateIpv6GoogleAccess
				foundContainerCluster.Spec.ReleaseChannel = desiredSpec.ReleaseChannel
				foundContainerCluster.Spec.ResourceUsageExportConfig = desiredSpec.ResourceUsageExportConfig
				foundContainerCluster.Spec.SubnetworkRef = desiredSpec.SubnetworkRef
				foundContainerCluster.Spec.VerticalPodAutoscaling = desiredSpec.VerticalPodAutoscaling
				foundContainerCluster.Spec.WorkloadIdentityConfig = desiredSpec.WorkloadIdentityConfig
				// Merge existing annotation and updated annotations
				foundContainerCluster.Annotations = addmap(foundContainerCluster.Annotations, cluster.Annotations)
				err = r.Update(ctx, foundContainerCluster, &client.UpdateOptions{})
				if err != nil {
					log.Error(err, "Syncing spec", "resourceType", resourceType)
				}
			}
			err = waitForContainerCluster(foundContainerCluster.Namespace, log, foundContainerCluster.Name, foundContainerCluster.Spec.Location)
			if err != nil {
				log.Error(err, "Wait for containercluster to be running")
			}
			// Update status to running
			err, _ = r.updateStatus(ctx, "ContainerCluster", universe, common.RunningStatus, foundContainerCluster.Name)
			if err != nil {
				log.Error(err, "Update containerCluster to runnning after waiting for containerCluster")
			}
			gCloudClient := gcloud.Client{}
			// Create serviceaccount in remote cluster
			sourceClusterRestConfig, err := gCloudClient.GetGKEClientsetWithTimeout(universe.Spec.ProjectID, universe.Spec.PrimaryCluster, "asia-southeast1")
			if err != nil {
				log.Error(err, "Create rest config", "cluster", universe.Spec.PrimaryCluster)
			}
			sourceClusterClientSet, _ := kubernetes.NewForConfig(sourceClusterRestConfig)
			sourceClusterAdminSecret, err := r.CreateClusterAdminToken(ctx, sourceClusterClientSet, "kube-system", "temp")
			if err != nil {
				log.Error(err, "Create admin secret", "cluster", universe.Spec.PrimaryCluster)
			}
			log.Info("Deploying velero", "cluster", universe.Spec.PrimaryCluster)
			deployVelero(ctx, sourceClusterClientSet, sourceClusterAdminSecret, sourceClusterRestConfig)
			targetClusterRestConfig, err := gCloudClient.GetGKEClientsetWithTimeout(universe.Spec.ProjectID, foundContainerCluster.Name, "asia-southeast1")
			if err != nil {
				log.Error(err, "Create rest config", "cluster", foundContainerCluster.Name)
			}
			targetClusterClientSet, _ := kubernetes.NewForConfig(targetClusterRestConfig)
			targetClusterAdminSecret, err := r.CreateClusterAdminToken(ctx, targetClusterClientSet, "kube-system", "temp")
			if err != nil {
				log.Error(err, "Create admin secret", "cluster", foundContainerCluster.Name)
			}
			log.Info("Deploying velero", "cluster", foundContainerCluster.Name)
			deployVelero(ctx, targetClusterClientSet, targetClusterAdminSecret, targetClusterRestConfig)
			if universe.Annotations["multiverse.saga.dev/pr-admin"] != "" {
				log.Info("Adding pr admin", "name", universe.Annotations["multiverse.saga.dev/pr-admin"])
				err = CreateAdminUser(ctx, targetClusterRestConfig, universe.Annotations["multiverse.saga.dev/pr-admin"])
				if err != nil {
					log.Error(err, "adding clusterAdmin permssion", "cluster", foundContainerCluster.Name)
				}
			}
			if strings.Contains(universe.Annotations["multiverse.saga.dev/cluster-addons"], "resource-patcher") {
				log.Info("Deploying addons", "name", "resource-patcher")
				err = r.deployResourcePatcher(ctx, targetClusterClientSet, universe.Namespace, universe.Annotations["multiverse.saga.dev/allow-tag"], "gcr-sub")
				if err != nil {
					log.Error(err, "deploy addons", "addon", "resource-patcher")
				}
			}
			if strings.Contains(universe.Annotations["multiverse.saga.dev/cluster-addons"], "external-dns") {
				log.Info("Deploying addons", "name", "external-dns")
				err = r.deployExternalDns(ctx, targetClusterClientSet)
				if err != nil {
					log.Error(err, "deploy addons", "addon", "external-dns")
				}
			}
			if strings.Contains(universe.Annotations["multiverse.saga.dev/cluster-addons"], "nginx-ingress") {
				log.Info("Deploying addons", "name", "nginx-ingress")
				err = r.deployNginxIngress(ctx, targetClusterClientSet, targetClusterAdminSecret, targetClusterRestConfig)
				if err != nil {
					log.Error(err, "deploy addons", "addon", "nginx-ingress")
				}
			}
			if strings.Contains(universe.Annotations["multiverse.saga.dev/cluster-addons"], "external-secrets") {
				log.Info("Deploying addons", "name", "external-secrets")
				err = r.deployExternalSecret(ctx, targetClusterClientSet, targetClusterAdminSecret, targetClusterRestConfig)
				if err != nil {
					log.Error(err, "deploy addons", "addon", "external-secrets")
				}
			}
			for _, service := range strings.Split(universe.Annotations["multiverse.saga.dev/preview-services"], ",") {
				log.Info("Annotate preview service", "service", service)
				deploymentList, err := targetClusterClientSet.AppsV1().Deployments("").List(ctx, metav1.ListOptions{})
				if err != nil {
					log.Error(err, "Listing service in target cluster")
				}
				for _, deploy := range deploymentList.Items {
					if deploy.Name == service {
						deploy.Annotations = addmap(deploy.Annotations, map[string]string{
							"image-update": "true",
						})
						_, err := targetClusterClientSet.AppsV1().Deployments(deploy.Namespace).Update(ctx, &deploy, metav1.UpdateOptions{})
						if err != nil {
							log.Error(err, "Annote preview service", "service", service)
						}
					}
				}
			}
		}
	}
	// Delete or scale to zero
	cList, err := r.getContainerClustersForUniverse(ctx, universe, getLabelSelector(universe.Name))
	if err != nil {
		log.Error(err, "Get ContainerCluster with label selector")
	}
	if len(cList) > len(universe.Spec.Container.ContainerClusters) {
		var isDelete bool
		log.Info("Removing Resource")
		for _, cExisting := range cList {
			name := cExisting.Name
			isDelete = true
			for _, cInSpec := range universe.Spec.Container.ContainerClusters {
				if cInSpec.Name == name {
					isDelete = false
				}
			}
			if isDelete {
				r.Delete(ctx, &cExisting, &client.DeleteOptions{})
			}
		}
	}
	// containernodepools
	if len(universe.Spec.Container.ContainerNodePools) > 0 {
		for _, nodePool := range universe.Spec.Container.ContainerNodePools {
			foundContainerNodePool := &kccContainerClusterv1beta1.ContainerNodePool{}
			resourceType := "ContainerNodePool"
			err := r.Get(ctx, types.NamespacedName{Name: nodePool.Name, Namespace: nodePool.Namespace}, foundContainerNodePool)
			if err != nil && errors.IsNotFound(err) {
				// Define a new resource
				log.Info("Creating a new resource", "resourceType", resourceType)
				containerNodePool := &kccContainerClusterv1beta1.ContainerNodePool{
					ObjectMeta: metav1.ObjectMeta{
						Name:        nodePool.CustomObjectMeta.Name,
						Namespace:   nodePool.CustomObjectMeta.Namespace,
						Annotations: nodePool.CustomObjectMeta.Annotations,
						Labels:      addmap(nodePool.CustomObjectMeta.Labels, getLabelSelector(universe.Name)),
					},
					Spec: nodePool.Spec,
				}
				resource := containerNodePool
				ctrl.SetControllerReference(universe, resource, r.Scheme)
				err = r.Create(ctx, resource)
				if err != nil {
					log.Error(err, "Failed to create new resource", "resource Type", resourceType)
					return ctrl.Result{}, err
				}
				log.Info("Update resource to creating", "resourceType", resourceType)
				err, _ = r.updateStatus(ctx, resourceType, universe, common.CreatingStatus, nodePool.Name)
				if err != nil {
					log.Info("Failed to update resource to creating", "resourceType", resourceType, "error", err)
				}
				return ctrl.Result{Requeue: true}, nil
			}

			// Only sync when spec change
			if universe.Status.ObservedGeneration < universe.GetObjectMeta().GetGeneration() && universe.Status.ObservedGeneration >= 1 {
				desiredSpec := nodePool.Spec
				// Sync spec
				log.Info("Syncing spec", "resourceType", resourceType)
				err, _ = r.updateStatus(ctx, "ContainerNodePool", universe, common.ReconcilingStatus, nodePool.Name)
				if err != nil {
					log.Error(err, "Update containerNodePool to reconciling")
				}
				// Some field of kcc resource is immutable
				// Sync all mutable field
				foundContainerNodePool.Spec.Autoscaling = desiredSpec.Autoscaling
				foundContainerNodePool.Spec.Management = desiredSpec.Management
				foundContainerNodePool.Spec.NodeCount = desiredSpec.NodeCount
				foundContainerNodePool.Spec.NodeLocations = desiredSpec.NodeLocations
				foundContainerNodePool.Spec.UpgradeSettings = desiredSpec.UpgradeSettings
				foundContainerNodePool.Spec.Version = desiredSpec.Version
				// Merge existing annotation and updated annotations
				foundContainerNodePool.Annotations = addmap(foundContainerNodePool.Annotations, nodePool.Annotations)
				err = r.Update(ctx, foundContainerNodePool, &client.UpdateOptions{})
				if err != nil {
					log.Error(err, "Syncing spec", "resourceType", resourceType)
				}
			}
			err = waitForContainerNodePool(foundContainerNodePool, foundContainerNodePool.Namespace, log, foundContainerNodePool.Name, foundContainerNodePool.Spec.ClusterRef.Name, foundContainerNodePool.Spec.Location)
			if err != nil {
				log.Error(err, "Wait for containernodepool to be running")
			}
			// Update status to running
			// Multiple nodepools will override nodepool infront of it. However, if one of nodepool creation fails, pool behind it will never created.
			err, _ = r.updateStatus(ctx, "ContainerNodePool", universe, common.RunningStatus, foundContainerNodePool.Name)
			if err != nil {
				log.Error(err, "Update containerNodePool to runnning after waiting for containerNodePool")
			}
		}
	}
	// Delete or scale to zero
	cNodePool, err := r.getContainerNodePoolsForUniverse(ctx, universe, getLabelSelector(universe.Name))
	if err != nil {
		log.Error(err, "Get ContainerNodePool with label selector")
	}
	if len(cNodePool) > len(universe.Spec.Container.ContainerNodePools) {
		var isDelete bool
		log.Info("Removing Resource")
		for _, cExisting := range cNodePool {
			name := cExisting.Name
			isDelete = true
			for _, cInSpec := range universe.Spec.Container.ContainerNodePools {
				if cInSpec.Name == name {
					isDelete = false
				}
			}
			if isDelete {
				r.Delete(ctx, &cExisting, &client.DeleteOptions{})
			}
		}
	}
	// sqlinstance
	if len(universe.Spec.SQL.SQLInstances) > 0 {
		resourceType := "SQLInstance"
		for _, instance := range universe.Spec.SQL.SQLInstances {
			foundSQLInstance := &kccSQLv1beta1.SQLInstance{}

			err := r.Get(ctx, types.NamespacedName{Name: instance.Name, Namespace: instance.Namespace}, foundSQLInstance)
			if err != nil && errors.IsNotFound(err) {
				// Define a new resource
				log.Info("Creating a new resource", "resourceType", resourceType)
				sqlInstance := &kccSQLv1beta1.SQLInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name:      instance.CustomObjectMeta.Name,
						Namespace: instance.CustomObjectMeta.Namespace,
						Labels:    addmap(instance.CustomObjectMeta.Labels, getLabelSelector(universe.Name)),
					},
					Spec: instance.Spec,
				}
				resource := sqlInstance
				ctrl.SetControllerReference(universe, resource, r.Scheme)
				err = r.Create(ctx, resource)
				if err != nil {
					log.Error(err, "Failed to create new resource", "resource Type", resourceType)
					return ctrl.Result{}, err
				}
				log.Info("Update resource to creating", "resourceType", resourceType)
				err, _ = r.updateStatus(ctx, resourceType, universe, common.CreatingStatus, sqlInstance.Name)
				if err != nil {
					log.Info("Failed to update resource to creating", "resourceType", resourceType, "error", err)
				}
				return ctrl.Result{Requeue: true}, nil
			}
			// Syncing resource
			if universe.Status.ObservedGeneration < universe.GetObjectMeta().GetGeneration() && universe.Status.ObservedGeneration >= 1 {
				desiredSpec := instance.Spec
				// Sync spec
				log.Info("Syncing spec", "resourceType", resourceType)
				err, _ = r.updateStatus(ctx, "SQLInstance", universe, common.ReconcilingStatus, foundSQLInstance.Name)
				if err != nil {
					log.Error(err, "Update sqlInstance to reconciling")
				}
				// Some field of kcc resource is immutable
				// Sync all mutable field
				foundSQLInstance.Spec.EncryptionKMSCryptoKeyRef = desiredSpec.EncryptionKMSCryptoKeyRef
				foundSQLInstance.Spec.MasterInstanceRef = desiredSpec.MasterInstanceRef
				foundSQLInstance.Spec.ReplicaConfiguration = desiredSpec.ReplicaConfiguration
				foundSQLInstance.Spec.Settings = desiredSpec.Settings
				// Merge existing annotation and updated annotations
				foundSQLInstance.Annotations = addmap(foundSQLInstance.Annotations, instance.Annotations)
				err = r.Update(ctx, foundSQLInstance, &client.UpdateOptions{})
				if err != nil {
					log.Error(err, "Syncing spec", "resourceType", resourceType)
				}
			}
			err = waitForSQLInstance(foundSQLInstance, foundSQLInstance.Namespace, log, foundSQLInstance.Name)
			if err != nil {
				log.Error(err, "Wait for sqlinstance to be running")
			}

			// Update status to running
			err, _ = r.updateStatus(ctx, "SQLInstance", universe, common.RunningStatus, foundSQLInstance.Name)
			if err != nil {
				log.Error(err, "Update sqlInstance to runnning after waiting for sqlInstance")
			}
		}
	}
	// Delete or scale to zero
	sList, err := r.getSQLInstancesForUniverse(ctx, universe, getLabelSelector(universe.Name))
	if err != nil {
		log.Error(err, "Get SQLInstance with label selector")
	}
	if len(sList) > len(universe.Spec.SQL.SQLInstances) {
		var isDelete bool
		log.Info("Removing Resource")
		for _, cExisting := range sList {
			name := cExisting.Name
			isDelete = true
			for _, cInSpec := range universe.Spec.SQL.SQLInstances {
				if cInSpec.Name == name {
					isDelete = false
				}
			}
			if isDelete {
				r.Delete(ctx, &cExisting, &client.DeleteOptions{})
			}
		}
	}
	// sqldatabase
	if len(universe.Spec.SQL.SQLDatabases) > 0 {
		resourceType := "SQLDatabase"
		for _, db := range universe.Spec.SQL.SQLDatabases {
			foundDb := &kccSQLv1beta1.SQLDatabase{}
			err := r.Get(ctx, types.NamespacedName{Name: db.Name, Namespace: db.Namespace}, foundDb)
			if err != nil && errors.IsNotFound(err) {
				// Define a new resource
				log.Info("Creating a new resource", "resourceType", resourceType)
				sqlDb := &kccSQLv1beta1.SQLDatabase{
					ObjectMeta: metav1.ObjectMeta{
						Name:      db.CustomObjectMeta.Name,
						Namespace: db.CustomObjectMeta.Namespace,
						Labels:    addmap(db.CustomObjectMeta.Labels, getLabelSelector(universe.Name)),
					},
					Spec: db.Spec,
				}
				resource := sqlDb
				ctrl.SetControllerReference(universe, resource, r.Scheme)
				err = r.Create(ctx, resource)
				if err != nil {
					log.Error(err, "Failed to create new resource", "resource Type", resourceType)
					return ctrl.Result{}, err
				}
				log.Info("Update resource to creating", "resourceType", resourceType)
				err, _ = r.updateStatus(ctx, resourceType, universe, common.CreatingStatus, db.Name)
				if err != nil {
					log.Info("Failed to update resource to creating", "resourceType", resourceType, "error", err)
				}
				return ctrl.Result{Requeue: true}, nil
			}

			// Syncing resource
			if universe.Status.ObservedGeneration < universe.GetObjectMeta().GetGeneration() && universe.Status.ObservedGeneration >= 1 {
				desiredSpec := db.Spec
				// Sync spec
				log.Info("Syncing spec", "resourceType", resourceType)
				err, _ = r.updateStatus(ctx, "SQLDatabase", universe, common.ReconcilingStatus, db.Name)
				if err != nil {
					log.Error(err, "Update sqlDatabase to reconciling")
				}
				// Some field of kcc resource is immutable
				// Sync all mutable field
				foundDb.Spec.Charset = desiredSpec.Charset
				foundDb.Spec.Collation = desiredSpec.Collation
				// Merge existing annotation and updated annotations
				foundDb.Annotations = addmap(foundDb.Annotations, db.Annotations)
				err = r.Update(ctx, foundDb, &client.UpdateOptions{})
				if err != nil {
					log.Error(err, "Syncing spec", "resourceType", resourceType)
				}
			}
			err = waitForSQLDatabase(foundDb, foundDb.Namespace, log, foundDb.Name, foundDb.Spec.InstanceRef.Name)
			if err != nil {
				log.Error(err, "Wait for sqlDatabase to be running")
			}

			// Update status to running
			err, _ = r.updateStatus(ctx, "SQLDatabase", universe, common.RunningStatus, foundDb.Name)
			if err != nil {
				log.Error(err, "Update sqlDatabase to runnning after waiting for sqlDatabase")
			}
		}
	}
	// Delete or scale to zero
	dList, err := r.getSQLDatabasesForUniverse(ctx, universe, getLabelSelector(universe.Name))
	if err != nil {
		log.Error(err, "Get SQLInstance with label selector")
	}
	if len(dList) > len(universe.Spec.SQL.SQLDatabases) {
		var isDelete bool
		log.Info("Removing Resource")
		for _, cExisting := range sList {
			name := cExisting.Name
			isDelete = true
			for _, cInSpec := range universe.Spec.SQL.SQLDatabases {
				if cInSpec.Name == name {
					isDelete = false
				}
			}
			if isDelete {
				r.Delete(ctx, &cExisting, &client.DeleteOptions{})
			}
		}
	}
	// router
	if len(universe.Spec.Compute.ComputeRouters) > 0 {
		resourceType := "ComputeRouter"
		for _, router := range universe.Spec.Compute.ComputeRouters {
			foundRouter := &kccNetworkv1beta1.ComputeRouter{}
			err := r.Get(ctx, types.NamespacedName{Name: router.Name, Namespace: router.Namespace}, foundRouter)
			if err != nil && errors.IsNotFound(err) {
				// Define a new resource
				log.Info("Creating a new resource", "resourceType", resourceType)
				computeRouter := &kccNetworkv1beta1.ComputeRouter{
					ObjectMeta: metav1.ObjectMeta{
						Name:      router.CustomObjectMeta.Name,
						Namespace: router.CustomObjectMeta.Namespace,
						Labels:    addmap(router.CustomObjectMeta.Labels, getLabelSelector(universe.Name)),
					},
					Spec: router.Spec,
				}
				resource := computeRouter
				ctrl.SetControllerReference(universe, resource, r.Scheme)
				err = r.Create(ctx, resource)
				if err != nil {
					log.Error(err, "Failed to create new resource", "resource Type", resourceType)
					return ctrl.Result{}, err
				}
				log.Info("Update resource to creating", "resourceType", resourceType)
				err, _ = r.updateStatus(ctx, resourceType, universe, common.CreatingStatus, router.Name)
				if err != nil {
					log.Info("Failed to update resource to creating", "resourceType", resourceType, "error", err)
				}
				return ctrl.Result{Requeue: true}, nil
			}

			// Syncing resource
			if universe.Status.ObservedGeneration < universe.GetObjectMeta().GetGeneration() && universe.Status.ObservedGeneration >= 1 {
				desiredSpec := router.Spec
				// Sync spec
				log.Info("Syncing spec", "resourceType", resourceType)
				err, _ = r.updateStatus(ctx, "ComputeRouter", universe, common.ReconcilingStatus, foundRouter.Name)
				if err != nil {
					log.Error(err, "Update computeRouter to reconciling")
				}
				// Some field of kcc resource is immutable
				// Sync all mutable field
				foundRouter.Spec.Bgp = desiredSpec.Bgp
				foundRouter.Spec.Description = desiredSpec.Description
				foundRouter.Spec.NetworkRef = desiredSpec.NetworkRef

				err = r.Update(ctx, foundRouter, &client.UpdateOptions{})
				if err != nil {
					log.Error(err, "Syncing spec", "resourceType", resourceType)
				}
			}
			err = waitForComputeRouter(foundRouter, foundRouter.Namespace, log, foundRouter.Name, foundRouter.Spec.Region)
			if err != nil {
				log.Error(err, "Wait for computeRouter to be running")
			}

			// Update status to running
			err, _ = r.updateStatus(ctx, "ComputeRouter", universe, common.RunningStatus, foundRouter.Name)
			if err != nil {
				log.Error(err, "Update computeRouter to runnning after waiting for computeRouter")
			}
		}
	}
	// Delete or scale to zero
	rList, err := r.getComputeRoutersForUniverse(ctx, universe, getLabelSelector(universe.Name))
	if err != nil {
		log.Error(err, "Get ComputeRouter with label selector")
	}
	if len(rList) > len(universe.Spec.Compute.ComputeRouters) {
		var isDelete bool
		log.Info("Removing Resource")
		for _, cExisting := range sList {
			name := cExisting.Name
			isDelete = true
			for _, cInSpec := range universe.Spec.Compute.ComputeRouters {
				if cInSpec.Name == name {
					isDelete = false
				}
			}
			if isDelete {
				r.Delete(ctx, &cExisting, &client.DeleteOptions{})
			}
		}
	}
	// routernat
	if len(universe.Spec.Compute.ComputeRouterNATs) > 0 {
		resourceType := "ComputeRouterNAT"
		for _, routerNAT := range universe.Spec.Compute.ComputeRouterNATs {
			foundRouterNAT := &kccNetworkv1beta1.ComputeRouterNAT{}
			err := r.Get(ctx, types.NamespacedName{Name: routerNAT.Name, Namespace: routerNAT.Namespace}, foundRouterNAT)
			if err != nil && errors.IsNotFound(err) {
				// Define a new resource
				log.Info("Creating a new resource", "resourceType", resourceType)
				computeRouterNAT := &kccNetworkv1beta1.ComputeRouterNAT{
					ObjectMeta: metav1.ObjectMeta{
						Name:      routerNAT.CustomObjectMeta.Name,
						Namespace: routerNAT.CustomObjectMeta.Namespace,
						Labels:    addmap(routerNAT.CustomObjectMeta.Labels, getLabelSelector(universe.Name)),
					},
					Spec: routerNAT.Spec,
				}
				resource := computeRouterNAT
				ctrl.SetControllerReference(universe, resource, r.Scheme)
				err = r.Create(ctx, resource)
				if err != nil {
					log.Error(err, "Failed to create new resource", "resource Type", resourceType)
					return ctrl.Result{}, err
				}
				log.Info("Update resource to creating", "resourceType", resourceType)
				err, _ = r.updateStatus(ctx, resourceType, universe, common.CreatingStatus, computeRouterNAT.Name)
				if err != nil {
					log.Info("Failed to update resource to creating", "resourceType", resourceType, "error", err)
				}
				return ctrl.Result{Requeue: true}, nil
			}

			// Syncing resource
			if universe.Status.ObservedGeneration < universe.GetObjectMeta().GetGeneration() && universe.Status.ObservedGeneration >= 1 {
				desiredSpec := routerNAT.Spec
				// Sync spec
				log.Info("Syncing spec", "resourceType", resourceType)
				err, _ = r.updateStatus(ctx, "ComputeRouter", universe, common.ReconcilingStatus, foundRouterNAT.Name)
				if err != nil {
					log.Error(err, "Update computeRouterNAT to reconciling")
				}
				// Some field of kcc resource is immutable
				// Sync all mutable field
				foundRouterNAT.Spec.DrainNatIps = desiredSpec.DrainNatIps
				foundRouterNAT.Spec.EnableDynamicPortAllocation = desiredSpec.EnableDynamicPortAllocation
				foundRouterNAT.Spec.EnableEndpointIndependentMapping = desiredSpec.EnableEndpointIndependentMapping
				foundRouterNAT.Spec.IcmpIdleTimeoutSec = desiredSpec.IcmpIdleTimeoutSec
				foundRouterNAT.Spec.LogConfig = desiredSpec.LogConfig
				foundRouterNAT.Spec.MinPortsPerVm = desiredSpec.MinPortsPerVm
				foundRouterNAT.Spec.NatIpAllocateOption = desiredSpec.NatIpAllocateOption
				foundRouterNAT.Spec.NatIps = desiredSpec.NatIps
				foundRouterNAT.Spec.RouterRef = desiredSpec.RouterRef
				foundRouterNAT.Spec.SourceSubnetworkIpRangesToNat = desiredSpec.SourceSubnetworkIpRangesToNat
				foundRouterNAT.Spec.Subnetwork = desiredSpec.Subnetwork
				foundRouterNAT.Spec.TcpEstablishedIdleTimeoutSec = desiredSpec.TcpEstablishedIdleTimeoutSec
				foundRouterNAT.Spec.TcpTransitoryIdleTimeoutSec = desiredSpec.TcpTransitoryIdleTimeoutSec
				foundRouterNAT.Spec.UdpIdleTimeoutSec = desiredSpec.UdpIdleTimeoutSec

				err = r.Update(ctx, foundRouterNAT, &client.UpdateOptions{})
				if err != nil {
					log.Error(err, "Syncing spec", "resourceType", resourceType)
				}
			}
			err = waitForComputeRouterNAT(foundRouterNAT, foundRouterNAT.Namespace, log, foundRouterNAT.Name, foundRouterNAT.Spec.Region)
			if err != nil {
				log.Error(err, "Wait for computeRouter to be running")
			}

			// Update status to running
			err, _ = r.updateStatus(ctx, "ComputeRouterNAT", universe, common.RunningStatus, foundRouterNAT.Name)
			if err != nil {
				log.Error(err, "Update computeRouterNAT to runnning after waiting for computeRouterNAT")
			}
		}
	}
	// Delete or scale to zero
	rNatList, err := r.getComputeRouterNATsForUniverse(ctx, universe, getLabelSelector(universe.Name))
	if err != nil {
		log.Error(err, "Get ComputeRouterNAT with label selector")
	}
	if len(rNatList) > len(universe.Spec.Compute.ComputeRouterNATs) {
		var isDelete bool
		log.Info("Removing Resource")
		for _, cExisting := range rNatList {
			name := cExisting.Name
			isDelete = true
			for _, cInSpec := range universe.Spec.Compute.ComputeRouters {
				if cInSpec.Name == name {
					isDelete = false
				}
			}
			if isDelete {
				r.Delete(ctx, &cExisting, &client.DeleteOptions{})
			}
		}
	}

	// storagebucket
	if len(universe.Spec.Storage.StorageBuckets) > 0 {
		resourceType := "StorageBucket"
		for _, bucket := range universe.Spec.Storage.StorageBuckets {
			foundBucket := &kccStoragev1beta1.StorageBucket{}
			err := r.Get(ctx, types.NamespacedName{Name: bucket.Name, Namespace: bucket.Namespace}, foundBucket)
			if err != nil && errors.IsNotFound(err) {
				// Define a new resource
				log.Info("Creating a new resource", "resourceType", resourceType)
				storageBucket := &kccStoragev1beta1.StorageBucket{
					ObjectMeta: metav1.ObjectMeta{
						Name:      bucket.CustomObjectMeta.Name,
						Namespace: bucket.CustomObjectMeta.Namespace,
						Labels:    addmap(bucket.CustomObjectMeta.Labels, getLabelSelector(universe.Name)),
					},
					Spec: bucket.Spec,
				}
				resource := storageBucket
				ctrl.SetControllerReference(universe, resource, r.Scheme)
				err = r.Create(ctx, resource)
				if err != nil {
					log.Error(err, "Failed to create new resource", "resource Type", resourceType)
					return ctrl.Result{}, err
				}
				log.Info("Update resource to creating", "resourceType", resourceType)
				err, _ = r.updateStatus(ctx, resourceType, universe, common.CreatingStatus, storageBucket.Name)
				if err != nil {
					log.Info("Failed to update resource to creating", "resourceType", resourceType, "error", err)
				}
				return ctrl.Result{Requeue: true}, nil
			}

			// Syncing resource
			if universe.Status.ObservedGeneration < universe.GetObjectMeta().GetGeneration() && universe.Status.ObservedGeneration >= 1 {
				desiredSpec := bucket.Spec
				// Sync spec
				log.Info("Syncing spec", "resourceType", resourceType)
				err, _ = r.updateStatus(ctx, "StorageBucket", universe, common.ReconcilingStatus, foundBucket.Name)
				if err != nil {
					log.Error(err, "Update storageBucket to reconciling")
				}
				// Some field of kcc resource is immutable
				// Sync all mutable field
				foundBucket.Spec.BucketPolicyOnly = desiredSpec.BucketPolicyOnly
				foundBucket.Spec.Cors = desiredSpec.Cors
				foundBucket.Spec.DefaultEventBasedHold = desiredSpec.DefaultEventBasedHold
				foundBucket.Spec.Encryption = desiredSpec.Encryption
				foundBucket.Spec.LifecycleRule = desiredSpec.LifecycleRule
				foundBucket.Spec.Logging = desiredSpec.Logging
				foundBucket.Spec.PublicAccessPrevention = desiredSpec.PublicAccessPrevention
				foundBucket.Spec.RequesterPays = desiredSpec.RequesterPays
				foundBucket.Spec.RetentionPolicy = desiredSpec.RetentionPolicy
				foundBucket.Spec.StorageClass = desiredSpec.StorageClass
				foundBucket.Spec.UniformBucketLevelAccess = desiredSpec.UniformBucketLevelAccess
				foundBucket.Spec.Versioning = desiredSpec.Versioning
				foundBucket.Spec.Website = desiredSpec.Website

				err = r.Update(ctx, foundBucket, &client.UpdateOptions{})
				if err != nil {
					log.Error(err, "Syncing spec", "resourceType", resourceType)
				}
			}
			err = waitForStorageBucket(foundBucket, log, foundBucket.Name)
			if err != nil {
				log.Error(err, "Wait for storageBucket to be running")
			}

			// Update status to running
			err, _ = r.updateStatus(ctx, "StorageBucket", universe, common.RunningStatus, foundBucket.Name)
			if err != nil {
				log.Error(err, "Update storageBucket to runnning after waiting for computeRouterNAT")
			}
		}
	}
	// Delete or scale to zero
	rBList, err := r.getStorageBucketsForUniverse(ctx, universe, getLabelSelector(universe.Name))
	if err != nil {
		log.Error(err, "Get StorageBucket with label selector")
	}
	if len(rBList) > len(universe.Spec.Storage.StorageBuckets) {
		var isDelete bool
		log.Info("Removing Resource")
		for _, cExisting := range rBList {
			name := cExisting.Name
			isDelete = true
			for _, cInSpec := range universe.Spec.Storage.StorageBuckets {
				if cInSpec.Name == name {
					isDelete = false
				}
			}
			if isDelete {
				r.Delete(ctx, &cExisting, &client.DeleteOptions{})
			}
		}
	}

	// TODO
	// gcs
	// sqluser
	gCloudClient := gcloud.Client{}
	targetNamespaces := universe.Spec.TargetNamespaces
	// Create serviceaccount in remote cluster
	sourceClusterRestConfig, err := gCloudClient.GetGKEClientsetWithTimeout(universe.Spec.ProjectID, universe.Spec.PrimaryCluster, "asia-southeast1")
	if err != nil {
		log.Error(err, "Create rest config", "cluster", universe.Spec.PrimaryCluster)
	}
	if err != nil {
		log.Error(err, "Create admin secret", "cluster", universe.Spec.PrimaryCluster)
	}
	for _, cluster := range universe.Spec.Container.ContainerClusters {
		foundContainerCluster := &kccContainerClusterv1beta1.ContainerCluster{}
		err := r.Get(ctx, types.NamespacedName{Name: cluster.Name, Namespace: cluster.Namespace}, foundContainerCluster)
		if err != nil {
			log.Error(err, "Container cluster not found", "name", cluster.Name)
			return ctrl.Result{Requeue: true}, nil
		}
		targetClusterRestConfig, err := gCloudClient.GetGKEClientsetWithTimeout(universe.Spec.ProjectID, foundContainerCluster.Name, "asia-southeast1")
		if err != nil {
			log.Error(err, "Create rest config", "cluster", foundContainerCluster.Name)
		}
		targetClusterClientSet, _ := kubernetes.NewForConfig(targetClusterRestConfig)
		targetClusterAdminSecret, err := r.CreateClusterAdminToken(ctx, targetClusterClientSet, "kube-system", "temp")
		if err != nil {
			log.Error(err, "Create admin secret", "cluster", foundContainerCluster.Name)
		}
		err = createBackUp(ctx, sourceClusterRestConfig, targetNamespaces, universe.Name)
		if err != nil {
			log.Error(err, "Create backup", "targetNamespaces", targetNamespaces)
		}
		err = waitForBackUp(targetClusterRestConfig, universe.Name)
		if err != nil {
			log.Error(err, "Wait for backup availble", "cluster", foundContainerCluster.Name)
		}
		err = createRestore(ctx, targetClusterAdminSecret, targetClusterRestConfig, universe.Name, universe.Name)
		if err != nil {
			log.Error(err, "Create restore", "cluster", foundContainerCluster.Name)
		}
		err = waitForRestore(targetClusterRestConfig, universe.Name)
		if err != nil {
			log.Error(err, "Wait for restore availble", "cluster", foundContainerCluster.Name)
		}
		// Patch ingress
		for _, ns := range targetNamespaces {
			ingressPatch(ctx, targetClusterRestConfig, universe.Name, ns)
		}
	}

	// Check if the Universe instance is marked to be deleted, which is
	// indicated by the deletion timestamp being set.
	isUniverseMarkedToBeDeleted := universe.GetDeletionTimestamp() != nil
	if isUniverseMarkedToBeDeleted {
		if controllerutil.ContainsFinalizer(universe, universeFinalizer) {
			// Run finalization logic for universeFinalizer. If the
			// finalization logic fails, don't remove the finalizer so
			// that we can retry during the next reconciliation.
			if err := r.finalizeUniverse(ctx, log, universe); err != nil {
				return ctrl.Result{}, err
			}

			// Remove universeFinalizer. Once all finalizers have been
			// removed, the object will be deleted.
			controllerutil.RemoveFinalizer(universe, universeFinalizer)
			err := r.Update(ctx, universe)
			if err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}
	// Add finalizer for this CR
	if !controllerutil.ContainsFinalizer(universe, universeFinalizer) {
		controllerutil.AddFinalizer(universe, universeFinalizer)
		err = r.Update(ctx, universe)
		if err != nil {
			return ctrl.Result{}, err
		}
	}
	if universe.GetCreationTimestamp().Second()+universe.Spec.TTL < time.Now().Second() {
		log.Info("Resource TTL reached. Cleaning up.", "name", universe.Name)
		err = r.Delete(ctx, universe, &client.DeleteOptions{})
		if err != nil {
			return ctrl.Result{RequeueAfter: 10 * time.Minute}, err
		}
	}
	// observed generate < universe spec generate on cluster, update
	universe.Status.ObservedGeneration = universe.GetObjectMeta().GetGeneration()
	r.Status().Update(ctx, universe, &client.UpdateOptions{})
	return ctrl.Result{}, nil
}

func ingressPatch(ctx context.Context, externalRestConfig *rest.Config, suffix string, namespace string) error {
	clientset, err := kubernetes.NewForConfig(externalRestConfig)
	if err != nil {
		return err
	}
	ingressList, err := clientset.NetworkingV1().Ingresses(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}
	for _, ing := range ingressList.Items {
		for index, rule := range ing.Spec.Rules {
			ing.Spec.Rules[index].Host = getHostName(rule.Host, suffix)
			_, err = clientset.NetworkingV1().Ingresses(namespace).Update(ctx, &ing, metav1.UpdateOptions{})
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func getHostName(host string, suffix string) string {
	return strings.Split(host, ".")[0] + "-" + suffix + "." + strings.Join(strings.Split(host, ".")[1:], ".")
}

// SetupWithManager sets up the controller with the Manager.
func (r *UniverseReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&multiversev1alpha1.Universe{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: 10}).
		Complete(r)
}
