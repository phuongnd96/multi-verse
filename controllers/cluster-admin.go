package controllers

import (
	"context"
	"fmt"
	"time"

	"github.com/phuongnd96/multi-verse/pkg/poll"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// CreateClusterAdminToken create a service account in remote cluster and get its token
func (r *UniverseReconciler) CreateClusterAdminToken(ctx context.Context, clientset kubernetes.Interface, namespace string, name string) (*v1.Secret, error) {
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
	var err error
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
			return nil, err
		}
	}
	if err != nil && !errors.IsNotFound(err) {
		return nil, err
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
			return nil, err
		}
	}
	if err != nil && !errors.IsNotFound(err) {
		return nil, err
	}

	_, err = clientset.CoreV1().ServiceAccounts(namespace).Get(ctx, name, metav1.GetOptions{})

	if err != nil && errors.IsNotFound(err) {
		serviceAccount := &v1.ServiceAccount{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "ServiceAccount",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
		}
		createdServiceAccount, err := clientset.CoreV1().ServiceAccounts(namespace).Create(ctx, serviceAccount, metav1.CreateOptions{})
		if err != nil {
			return nil, err
		}
		fmt.Printf("createdServiceAccount: %v\n", createdServiceAccount)
	}
	if err != nil && !errors.IsNotFound(err) {
		return nil, err
	}

	_, err = clientset.CoreV1().Secrets(namespace).Get(ctx, name+"sasecret", metav1.GetOptions{})
	if err != nil && errors.IsNotFound(err) {
		serviceAccountSecret := &v1.Secret{
			Type: "kubernetes.io/service-account-token",
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "Secret",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      name + "sasecret",
				Namespace: namespace,
				Annotations: map[string]string{
					"kubernetes.io/service-account.name": name,
				},
			},
		}
		_, err := clientset.CoreV1().Secrets(namespace).Create(ctx, serviceAccountSecret, metav1.CreateOptions{})
		if err != nil {
			return nil, err
		}
	}
	if err != nil && !errors.IsNotFound(err) {
		return nil, err
	}
	ctxWithTImeOut, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()
	// Get secret with data filled by K8S Secret Controller
	createdServiceAccountSecret := &v1.Secret{}
	// createdServiceAccountSecret, _ := clientset.CoreV1().Secrets(namespace).Get(ctx, name+"sasecret", metav1.GetOptions{})
	poll.Wait(ctxWithTImeOut, func(ctx context.Context) (bool, error) {
		createdServiceAccountSecret, err = clientset.CoreV1().Secrets(namespace).Get(ctx, name+"sasecret", metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		if !isNil(createdServiceAccountSecret) {
			return true, nil
		}
		return false, nil
	})
	return createdServiceAccountSecret, nil
}

func CreateAdminUser(ctx context.Context, externalRestConfig *rest.Config, userEmail string) error {
	name := "pr-admin"
	clientset, err := kubernetes.NewForConfig(externalRestConfig)
	if err != nil {
		return err
	}
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
				Kind: rbacv1.UserKind,
				Name: userEmail,
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
	return nil
}
