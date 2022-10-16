package controllers

import (
	"context"

	kccNetworkv1beta1 "github.com/GoogleCloudPlatform/k8s-config-connector/pkg/clients/generated/apis/compute/v1beta1"
	kccContainerClusterv1beta1 "github.com/GoogleCloudPlatform/k8s-config-connector/pkg/clients/generated/apis/container/v1beta1"
	kccIAMv1beta1 "github.com/GoogleCloudPlatform/k8s-config-connector/pkg/clients/generated/apis/iam/v1beta1"
	kccSQLv1beta1 "github.com/GoogleCloudPlatform/k8s-config-connector/pkg/clients/generated/apis/sql/v1beta1"
	kccStoragev1beta1 "github.com/GoogleCloudPlatform/k8s-config-connector/pkg/clients/generated/apis/storage/v1beta1"

	multiversev1alpha1 "github.com/phuongnd96/multi-verse/api/v1alpha1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (reconciler *UniverseReconciler) updateStatus(ctx context.Context, resourceType string, u *multiversev1alpha1.Universe, progress string, name string) (error, bool) {

	if len(u.Status.Resources) == 0 {
		// Nothing created
		// Init
		u.Status.Resources = append(u.Status.Resources, multiversev1alpha1.ResourceStatus{
			Kind:   resourceType,
			Name:   name,
			Status: progress,
		})
		reconciler.Status().Update(ctx, u, &client.UpdateOptions{})
		return nil, true
	}
	resourceCreated := 0
	for _, r := range u.Status.Resources {
		if r.Kind == resourceType && r.Name == name {
			resourceCreated = 1
		}
	}
	if resourceCreated == 0 {
		// Create
		u.Status.Resources = append(u.Status.Resources, multiversev1alpha1.ResourceStatus{
			Kind:   resourceType,
			Name:   name,
			Status: progress,
		})
		reconciler.Status().Update(ctx, u, &client.UpdateOptions{})
		return nil, true
	}
	if len(u.Status.Resources) > 0 {
		for index, r := range u.Status.Resources {
			if r.Kind == resourceType && r.Name == name {
				// Replace
				u.Status.Resources[index] = multiversev1alpha1.ResourceStatus{
					Kind:   resourceType,
					Name:   name,
					Status: progress,
				}
				reconciler.Status().Update(ctx, u, &client.UpdateOptions{})
				return nil, true
			}
		}
	}
	return nil, true
}

// getComputeNetworkForUniverse return a list of computenetwork managed by universe
func (r *UniverseReconciler) getComputeNetworksForUniverse(ctx context.Context, u *multiversev1alpha1.Universe, selector map[string]string) ([]kccNetworkv1beta1.ComputeNetwork, error) {
	list := &kccNetworkv1beta1.ComputeNetworkList{}
	err := r.List(ctx, list, client.MatchingLabelsSelector{Selector: labels.SelectorFromSet(selector)})
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

func (r *UniverseReconciler) getComputeSubNetworksForUniverse(ctx context.Context, u *multiversev1alpha1.Universe, selector map[string]string) ([]kccNetworkv1beta1.ComputeSubnetwork, error) {
	list := &kccNetworkv1beta1.ComputeSubnetworkList{}
	err := r.List(ctx, list, client.MatchingLabelsSelector{Selector: labels.SelectorFromSet(selector)})
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

func (r *UniverseReconciler) getContainerClustersForUniverse(ctx context.Context, u *multiversev1alpha1.Universe, selector map[string]string) ([]kccContainerClusterv1beta1.ContainerCluster, error) {
	list := &kccContainerClusterv1beta1.ContainerClusterList{}
	err := r.List(ctx, list, client.MatchingLabelsSelector{Selector: labels.SelectorFromSet(selector)})
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

func (r *UniverseReconciler) getContainerNodePoolsForUniverse(ctx context.Context, u *multiversev1alpha1.Universe, selector map[string]string) ([]kccContainerClusterv1beta1.ContainerNodePool, error) {
	list := &kccContainerClusterv1beta1.ContainerNodePoolList{}
	err := r.List(ctx, list, client.MatchingLabelsSelector{Selector: labels.SelectorFromSet(selector)})
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

func (r *UniverseReconciler) getSQLInstancesForUniverse(ctx context.Context, u *multiversev1alpha1.Universe, selector map[string]string) ([]kccSQLv1beta1.SQLInstance, error) {
	list := &kccSQLv1beta1.SQLInstanceList{}
	err := r.List(ctx, list, client.MatchingLabelsSelector{Selector: labels.SelectorFromSet(selector)})
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

func (r *UniverseReconciler) getSQLDatabasesForUniverse(ctx context.Context, u *multiversev1alpha1.Universe, selector map[string]string) ([]kccSQLv1beta1.SQLDatabase, error) {
	list := &kccSQLv1beta1.SQLDatabaseList{}
	err := r.List(ctx, list, client.MatchingLabelsSelector{Selector: labels.SelectorFromSet(selector)})
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

func (r *UniverseReconciler) getComputeRoutersForUniverse(ctx context.Context, u *multiversev1alpha1.Universe, selector map[string]string) ([]kccNetworkv1beta1.ComputeRouter, error) {
	list := &kccNetworkv1beta1.ComputeRouterList{}
	err := r.List(ctx, list, client.MatchingLabelsSelector{Selector: labels.SelectorFromSet(selector)})
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

func (r *UniverseReconciler) getComputeRouterNATsForUniverse(ctx context.Context, u *multiversev1alpha1.Universe, selector map[string]string) ([]kccNetworkv1beta1.ComputeRouterNAT, error) {
	list := &kccNetworkv1beta1.ComputeRouterNATList{}
	err := r.List(ctx, list, client.MatchingLabelsSelector{Selector: labels.SelectorFromSet(selector)})
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

func (r *UniverseReconciler) getStorageBucketsForUniverse(ctx context.Context, u *multiversev1alpha1.Universe, selector map[string]string) ([]kccStoragev1beta1.StorageBucket, error) {
	list := &kccStoragev1beta1.StorageBucketList{}
	err := r.List(ctx, list, client.MatchingLabelsSelector{Selector: labels.SelectorFromSet(selector)})
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

func (r *UniverseReconciler) getIAMServiceAccountsForUniverse(ctx context.Context, u *multiversev1alpha1.Universe, selector map[string]string) ([]kccIAMv1beta1.IAMServiceAccount, error) {
	list := &kccIAMv1beta1.IAMServiceAccountList{}
	err := r.List(ctx, list, client.MatchingLabelsSelector{Selector: labels.SelectorFromSet(selector)})
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}
