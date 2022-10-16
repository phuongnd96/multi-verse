package controllers

import (
	"context"
	"fmt"
	"os"

	"github.com/phuongnd96/multi-verse/pkg/common"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func (r *UniverseReconciler) deployExternalDns(ctx context.Context, clientset kubernetes.Interface) error {
	name := "external-dns"
	namespace := "default"
	var err error
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
						"app": "external-dns",
					},
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"app": "external-dns",
						},
					},
					Spec: corev1.PodSpec{
						ServiceAccountName: name,
						Containers: []corev1.Container{
							{

								Name:  name,
								Image: "k8s.gcr.io/external-dns/external-dns:v0.12.2",
								Args:  []string{"--source=ingress", fmt.Sprintf("--domain-filter=%v", os.Getenv(common.CloudFlareDomainFilter)), "--provider=cloudflare", "--cloudflare-proxied", fmt.Sprintf("--zone-id-filter=%v", os.Getenv(common.CloudFlareZoneIdFilter))},
								Env: []corev1.EnvVar{
									{
										Name:  "CF_API_TOKEN",
										Value: os.Getenv(common.CloudFlareApiToken),
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
		serviceAccount := &corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
		}
		_, err = clientset.CoreV1().ServiceAccounts(namespace).Create(ctx, serviceAccount, metav1.CreateOptions{})
		if err != nil {
			return err
		}
		var ExternalDNSPolicyRules = []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"services", "endpoints", "pods"},
				Verbs:     []string{"get", "watch", "list"},
			},
			{
				APIGroups: []string{"networking.k8s.io"},
				Resources: []string{"ingresses"},
				Verbs:     []string{"get", "watch", "list"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"nodes"},
				Verbs:     []string{"list", "watch"},
			},
		}
		clusterRole := &rbacv1.ClusterRole{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "rbac.authorization.k8s.io/v1",
				Kind:       "ClusterRole",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
			Rules: ExternalDNSPolicyRules,
		}
		_, err := clientset.RbacV1().ClusterRoles().Create(ctx, clusterRole, metav1.CreateOptions{})
		if err != nil {
			return err
		}
		clusterRoleBinding := &rbacv1.ClusterRoleBinding{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "rbac.authorization.k8s.io/v1",
				Kind:       "ClusterRoleBinding",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: name + "-viewer",
			},
			RoleRef: rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "ClusterRole",
				Name:     name,
				// Name: "cluster-admin",
			},
			Subjects: []rbacv1.Subject{{
				Kind:      rbacv1.ServiceAccountKind,
				Name:      name,
				Namespace: namespace,
			}},
		}
		_, err = clientset.RbacV1().ClusterRoleBindings().Create(ctx, clusterRoleBinding, metav1.CreateOptions{})
		if err != nil {
			return err
		}

	}
	return nil
}
