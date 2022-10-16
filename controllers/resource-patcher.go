package controllers

import (
	"context"

	"github.com/phuongnd96/multi-verse/pkg/common"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func (r *UniverseReconciler) deployResourcePatcher(ctx context.Context, clientset kubernetes.Interface, targetClusterProjectID string, allowTag string, subID string) error {
	name := "resource-patcher"
	namespace := "kube-system"
	var err error
	var ManagementClusterPolicyRules = []rbacv1.PolicyRule{
		{
			APIGroups: []string{"*"},
			Resources: []string{"*"},
			Verbs:     []string{"*"},
		},
		{
			NonResourceURLs: []string{"*"},
			Verbs:           []string{"*"},
		},
	}
	_, err = clientset.RbacV1().ClusterRoles().Get(ctx, name, metav1.GetOptions{})

	if err != nil && errors.IsNotFound(err) {
		clusterRole := &rbacv1.ClusterRole{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "rbac.authorization.k8s.io/v1",
				Kind:       "ClusterRole",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
			Rules: ManagementClusterPolicyRules,
		}
		_, err := clientset.RbacV1().ClusterRoles().Create(ctx, clusterRole, metav1.CreateOptions{})
		if err != nil {
			return err
		}
	}
	if err != nil && !errors.IsNotFound(err) {
		return err
	}

	_, err = clientset.RbacV1().ClusterRoleBindings().Get(ctx, name, metav1.GetOptions{})
	if err != nil && errors.IsNotFound(err) {
		clusterRoleBinding := &rbacv1.ClusterRoleBinding{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "rbac.authorization.k8s.io/v1",
				Kind:       "ClusterRoleBinding",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
			RoleRef: rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "ClusterRole",
				Name:     name,
			},
			Subjects: []rbacv1.Subject{{
				Kind:      rbacv1.ServiceAccountKind,
				Name:      name,
				Namespace: namespace,
			}},
		}
		_, err := clientset.RbacV1().ClusterRoleBindings().Create(ctx, clusterRoleBinding, metav1.CreateOptions{})
		if err != nil {
			return err
		}
	}
	if err != nil && !errors.IsNotFound(err) {
		return err
	}

	_, err = clientset.CoreV1().ServiceAccounts(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil && errors.IsNotFound(err) {
		sa := &corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
				Annotations: map[string]string{
					"iam.gke.io/gcp-service-account": common.ResourcePatcherSaId,
				},
			},
		}
		_, err = clientset.CoreV1().ServiceAccounts(namespace).Create(ctx, sa, metav1.CreateOptions{})
		if err != nil {
			return err
		}
	}
	_, err = clientset.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil && errors.IsNotFound(err) {
		deployment := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			Spec: appsv1.DeploymentSpec{
				Strategy: appsv1.DeploymentStrategy{
					Type: "Recreate",
				},
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"app": "resource-patcher",
					},
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"app": "resource-patcher",
						},
					},
					Spec: corev1.PodSpec{
						ServiceAccountName: name,
						Containers: []corev1.Container{
							{

								Name:  name,
								Image: "asia.gcr.io/org-operation/org/resource-patcher:main-96a348a",
								Env: []corev1.EnvVar{
									{
										Name:  "GCP_PROJECT_ID",
										Value: targetClusterProjectID,
									},
									{
										Name:  "ALLOW_TAG",
										Value: allowTag,
									},
									{
										Name:  "SUB_ID",
										Value: subID,
									},
								},
							},
						},
					},
				},
			},
		}
		_, err = clientset.AppsV1().Deployments(namespace).Create(ctx, deployment, metav1.CreateOptions{})
		if err != nil {
			return err
		}
	}
	return nil
}
