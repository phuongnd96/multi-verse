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
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/GoogleCloudPlatform/k8s-config-connector/pkg/clients/generated/apis/compute/v1beta1"
	kccContainerv1beta1 "github.com/GoogleCloudPlatform/k8s-config-connector/pkg/clients/generated/apis/container/v1beta1"
	kccIAMv1beta1 "github.com/GoogleCloudPlatform/k8s-config-connector/pkg/clients/generated/apis/iam/v1beta1"

	"github.com/GoogleCloudPlatform/k8s-config-connector/pkg/clients/generated/apis/k8s/v1alpha1"
	"github.com/go-logr/logr"
	"github.com/google/go-github/github"
	multiversev1alpha1 "github.com/phuongnd96/multi-verse/api/v1alpha1"
	common "github.com/phuongnd96/multi-verse/pkg/common"
	gh "github.com/phuongnd96/multi-verse/pkg/github"
)

// TriggerReconciler reconciles a Trigger object
type TriggerReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

//+kubebuilder:rbac:groups=multiverse.saga.dev,resources=triggers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=multiverse.saga.dev,resources=triggers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=multiverse.saga.dev,resources=triggers/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Trigger object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.1/pkg/reconcile
func (r *TriggerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// TODO(user): your logic here
	// Watch a list of github repo, query every x interval
	// Find for any PR with label preview
	// Create a new cluster
	// Comment back to that PR

	// 1. Parse credentialSecretRef to get github access token
	// 2. Parse requeueAfterSeconds to get interval to query to github
	// 3. Find for maching PR with label preview, gke, sql, gcs
	// 4. Create new universe with resource listed in pr label
	// 5. Comment back to the PR
	log := ctrllog.FromContext(ctx)
	var err error
	defer log.Info("End reconcile")
	foundTrigger := &multiversev1alpha1.Trigger{}
	err = r.Get(ctx, req.NamespacedName, foundTrigger)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			log.Info("Trigger resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get Trigger")
		return ctrl.Result{}, err
	}
	if foundTrigger.Status.ObservedGeneration == foundTrigger.GetObjectMeta().GetGeneration() {
		r.Recorder.Event(foundTrigger, "Normal", "Synced", fmt.Sprintf("Synced at %s", time.Now()))
	}
	foundGitSecret := &v1.Secret{}
	err = r.Get(ctx, types.NamespacedName{Namespace: req.Namespace, Name: foundTrigger.Spec.CredentialSecretRef}, foundGitSecret)
	if err != nil && errors.IsNotFound(err) {
		log.Error(err, "Can not find git credential")
	}
	gitToken := string(foundGitSecret.Data["token"])
	gitClient := gh.NewClient(gitToken)
	for _, repo := range foundTrigger.Spec.Respositories {
		owner := strings.Split(repo, "/")[0]
		repository := strings.Split(repo, "/")[1]
		prListOpen, _, err := gitClient.PullRequests.List(ctx, owner, repository, &github.PullRequestListOptions{
			State: "open",
		})
		if err != nil {
			log.Error(err, "Listing open pull request", "repo", repo)
		}
		for _, pr := range prListOpen {
			var labelNames []string
			var previewServiceList []string
			for _, label := range pr.Labels {
				labelNames = append(labelNames, *label.Name)
				r, _ := regexp.Compile(`svc\/.+`)
				if r.MatchString(label.GetName()) {
					previewServiceList = append(previewServiceList, foundTrigger.Spec.LabelToDeploymentMap[label.GetName()])
				}
			}
			if !stringInSlice("preview-created", labelNames) && stringInSlice("preview", labelNames) {
				log.Info("Detected preview request", "pr", *pr.Number, "repo", repo)
				err := r.NewUniverse(ctx, req.Namespace, log, repository, *pr.Number, foundTrigger.Spec.TargetNamespaces, foundTrigger.Spec.PrimaryCluster, previewServiceList, pr.GetAuthorAssociation())
				if err != nil {
					log.Error(err, "Create new unviverse", "name", makeUniverseName(repository, *pr.Number))
				}
				content := createOpenComment(makeUniverseName(repository, *pr.Number))
				payload := map[string]interface{}{
					"body":        content,
					"pull_number": pr.Number,
					"owner":       owner,
					"repo":        repository,
				}
				u := fmt.Sprintf("repos/%v/%v/issues/%d/comments", owner, repository, *pr.Number)
				req, err := gitClient.NewRequest("POST", u, payload)
				if err != nil {
					log.Error(err, "Construct comment request")
				}
				_, err = gitClient.Do(ctx, req, &payload)
				if err != nil {
					log.Error(err, "Create comment request")
				}
				newLabels := append(labelNames, "preview-created")
				for i, v := range newLabels {
					if v == "preview-closed" {
						newLabels = append(newLabels[:i], newLabels[i+1:]...)
						break
					}
				}
				_, _, err = gitClient.Issues.Edit(ctx, owner, repository, *pr.Number, &github.IssueRequest{
					Labels: &newLabels,
				})
				if err != nil {
					log.Error(err, "Update PR Label")
				}
			}
		}
		cronName := foundTrigger.Name + "-cron"
		schedule := fmt.Sprintf("*/%v * * * *", foundTrigger.Spec.RequeueAfterMinutes)
		foundCron := &batchv1.CronJob{}
		cronCommand := fmt.Sprintf("kubectl annotate --overwrite trigger %v triggers.multiverse.saga.dev/last-sync-timestamp=\"$(date)\"", foundTrigger.Name)
		err = r.Get(ctx, types.NamespacedName{Namespace: req.Namespace, Name: cronName}, foundCron)
		if err != nil && errors.IsNotFound(err) {
			cron := &batchv1.CronJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:      cronName,
					Namespace: req.Namespace,
				},
				Spec: batchv1.CronJobSpec{
					Schedule:          schedule,
					ConcurrencyPolicy: batchv1.ReplaceConcurrent,
					JobTemplate: batchv1.JobTemplateSpec{
						Spec: batchv1.JobSpec{
							Template: v1.PodTemplateSpec{
								Spec: v1.PodSpec{
									RestartPolicy: "OnFailure",
									Containers: []v1.Container{
										{
											Name:    cronName,
											Image:   "bitnami/kubectl:1.24.3-debian-11-r11",
											Command: []string{"sh", "-c", cronCommand},
										},
									},
									ServiceAccountName: cronName,
								},
							},
						},
					},
				},
			}
			serviceAccount := &v1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Name:      cronName,
					Namespace: req.Namespace,
				},
			}
			var policyRules = []rbacv1.PolicyRule{
				{
					APIGroups: []string{"multiverse.saga.dev"},
					Resources: []string{"triggers"},
					Verbs:     []string{"get", "watch", "list", "patch", "update"},
				},
			}
			role := &rbacv1.Role{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "rbac.authorization.k8s.io/v1",
					Kind:       "ClusterRole",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      cronName,
					Namespace: req.Namespace,
				},
				Rules: policyRules,
			}
			roleBinding := &rbacv1.RoleBinding{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "rbac.authorization.k8s.io/v1",
					Kind:       "ClusterRoleBinding",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      cronName,
					Namespace: req.Namespace,
				},
				RoleRef: rbacv1.RoleRef{
					APIGroup: "rbac.authorization.k8s.io",
					Kind:     "Role",
					Name:     role.Name,
				},
				Subjects: []rbacv1.Subject{{
					Kind:      rbacv1.ServiceAccountKind,
					Name:      cronName,
					Namespace: req.Namespace,
				}},
			}
			ctrl.SetControllerReference(foundTrigger, cron, r.Scheme)
			ctrl.SetControllerReference(foundTrigger, serviceAccount, r.Scheme)
			ctrl.SetControllerReference(foundTrigger, role, r.Scheme)
			ctrl.SetControllerReference(foundTrigger, roleBinding, r.Scheme)
			err = r.Create(ctx, serviceAccount, &client.CreateOptions{})
			if err != nil {
				log.Error(err, "Create cron sa")
			}
			err = r.Create(ctx, role, &client.CreateOptions{})
			if err != nil {
				log.Error(err, "Create cron role")
			}
			err = r.Create(ctx, roleBinding, &client.CreateOptions{})
			if err != nil {
				log.Error(err, "Create cron rolebinding")
			}
			err = r.Create(ctx, cron, &client.CreateOptions{})
			if err != nil {
				log.Error(err, "Create cron")
			}
		}
		// observed generate < spec generate on cluster, update
		foundTrigger.Status.ObservedGeneration = foundTrigger.GetObjectMeta().GetGeneration()
		r.Status().Update(ctx, foundTrigger, &client.UpdateOptions{})
		prListClosed, _, err := gitClient.PullRequests.List(ctx, owner, repository, &github.PullRequestListOptions{
			State: "closed",
		})
		if err != nil {
			log.Error(err, "Listing closed request", "repo", repo)
		}
		for _, pr := range prListClosed {
			var labelNames []string
			for _, label := range pr.Labels {
				labelNames = append(labelNames, *label.Name)
			}
			if !stringInSlice("preview-closed", labelNames) && stringInSlice("preview", labelNames) {
				foundUniverse := &multiversev1alpha1.Universe{}
				r.Get(ctx, types.NamespacedName{Name: makeUniverseName(repository, *pr.Number), Namespace: req.Namespace}, foundUniverse)
				// Preview created. Tearing down
				err = r.Delete(ctx, foundUniverse, &client.DeleteOptions{})
				if err != nil {
					log.Error(err, "Tearing down environment for preview request", "pr", *pr.Number, "repo", repo)
				}
				content := createCloseComment()
				payload := map[string]interface{}{
					"body":        content,
					"pull_number": pr.Number,
					"owner":       owner,
					"repo":        repository,
				}
				u := fmt.Sprintf("repos/%v/%v/issues/%d/comments", owner, repository, *pr.Number)
				req, err := gitClient.NewRequest("POST", u, payload)
				if err != nil {
					log.Error(err, "Construct teardown comment request")
				}
				_, err = gitClient.Do(ctx, req, &payload)
				if err != nil {
					log.Error(err, "Create teardown comment request")
				}
				newLabels := append(labelNames, "preview-closed")
				for i, v := range newLabels {
					if v == "preview-created" {
						newLabels = append(newLabels[:i], newLabels[i+1:]...)
						break
					}
				}
				_, _, err = gitClient.Issues.Edit(ctx, owner, repository, *pr.Number, &github.IssueRequest{
					Labels: &newLabels,
				})
				if err != nil {
					log.Error(err, "Update PR Label")
				}
			}

		}
	}

	return ctrl.Result{}, nil
}

func createOpenComment(name string) string {
	template := `## 🚀 Preview Environment 🚀
 Environment for pr **%s** was created successfully 🎉🎉🎉
 ### Notes
 * 📕The preview environment is open until this PR is **closed**. 
`
	return fmt.Sprintf(template, name)
}
func createCloseComment() string {
	template := `## 🚀 Preview Environment 🚀
Environment is closed.
`
	return fmt.Sprint(template)
}
func makeUniverseName(repo string, prNumber int) string {
	return repo + "-" + strconv.Itoa(prNumber)
}

func (r *TriggerReconciler) NewUniverse(ctx context.Context, namespace string, log logr.Logger, repo string, prNumber int, syncTargetNamespace []string, primaryCluster string, previewServiceList []string, prAdmin string) error {
	name := makeUniverseName(repo, prNumber)
	var err error
	foundUniverse := &multiversev1alpha1.Universe{}
	foundServiceAccount := &kccIAMv1beta1.IAMServiceAccount{}
	region := "asia-southeast1"
	routingMode := "REGIONAL"
	initialNodeCount := 1
	minMasterVersion := "1.23.8-gke.1900"
	workloadPool := namespace + ".svc.id.goog"
	networkingMode := "VPC_NATIVE"
	servicesSecondaryRangeName := "ip-range-service"
	clusterSecondaryRangeName := "ip-range-pod"
	machineType := "n1-standard-1"
	diskSizeGb := 50
	preemptible := true
	diskType := "pd-standard"
	autoscalingProfile := "BALANCED"
	minCpuPlatform := "Intel Haswell"
	autoCreateSubnetworks := false
	maxCpu := 30
	minCpu := 10
	maxMem := 30
	minMem := 8
	minNodeCount := 1
	maxNodeCount := 5
	enableBinaryAuthorization := false
	enableIntranodeVisibility := false
	enableShieldedNodes := true
	desription := "Created for " + name
	// Create cluster service account if not exists. All preview cluster use the same service account, delete cluster will not delete the service account
	err = r.Get(ctx, types.NamespacedName{Name: common.ClusterServiceAccountName, Namespace: namespace}, foundServiceAccount)
	if err != nil && errors.IsNotFound(err) {
		log.Info("Preview cluster service account not found, creating.")
		iamServiceAccount := &kccIAMv1beta1.IAMServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:      common.ClusterServiceAccountName,
				Namespace: namespace,
			},
		}
		err := r.Create(ctx, iamServiceAccount, &client.CreateOptions{})
		if err != nil {
			return err
		}
	}
	if err != nil {
		return err
	}
	err = r.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, foundUniverse)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("Creating a new Universe", "name", name)
			// Creating a new universe
			// pr-123-git_hash
			allowTag := "^pr-" + strconv.Itoa(prNumber) + ".*"
			universe := &multiversev1alpha1.Universe{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: namespace,
					Annotations: map[string]string{
						"multiverse.saga.dev/cluster-addons":   "external-dns,nginx-ingress,external-secrets,resource-patcher",
						"multiverse.saga.dev/repo":             repo,
						"multiverse.saga.dev/pr":               strconv.Itoa(prNumber),
						"multiverse.saga.dev/preview-services": strings.Join(previewServiceList, ","),
						"multiverse.saga.dev/allow-tag":        allowTag,
						"multiverse.saga.dev/pr-admin":         prAdmin,
					},
				},
				Spec: multiversev1alpha1.UniverseSpec{
					ProjectID:        namespace,
					TTL:              86400,
					TargetNamespaces: syncTargetNamespace,
					PrimaryCluster:   primaryCluster,
					Compute: multiversev1alpha1.Compute{
						ComputeRouters: []multiversev1alpha1.ComputeRouterTemplateSpec{
							{
								CustomObjectMeta: multiversev1alpha1.CustomObjectMeta{
									Name:      name,
									Namespace: namespace,
								},
								Spec: v1beta1.ComputeRouterSpec{
									Description: &desription,
									Region:      region,
									NetworkRef: v1alpha1.ResourceRef{
										Name: name,
									},
								},
							},
						},
						ComputeRouterNATs: []multiversev1alpha1.ComputeRouterNATTemplateSpec{
							{
								CustomObjectMeta: multiversev1alpha1.CustomObjectMeta{
									Name:      name,
									Namespace: namespace,
								},
								Spec: v1beta1.ComputeRouterNATSpec{
									Region: region,
									RouterRef: v1alpha1.ResourceRef{
										Name: name,
									},
									NatIpAllocateOption:           "AUTO_ONLY",
									SourceSubnetworkIpRangesToNat: "ALL_SUBNETWORKS_ALL_IP_RANGES",
								},
							},
						},
						ComputeNetworks: []multiversev1alpha1.ComputeNetworkTemplateSpec{
							{
								CustomObjectMeta: multiversev1alpha1.CustomObjectMeta{
									Name:      name,
									Namespace: namespace,
								},
								Spec: v1beta1.ComputeNetworkSpec{
									Description:           &desription,
									RoutingMode:           &routingMode,
									AutoCreateSubnetworks: &autoCreateSubnetworks,
								},
							},
						},
						ComputeSubnetworks: []multiversev1alpha1.ComputeSubnetworkTemplateSpec{
							{
								CustomObjectMeta: multiversev1alpha1.CustomObjectMeta{
									Name:      name,
									Namespace: namespace,
								},
								Spec: v1beta1.ComputeSubnetworkSpec{
									Description: &desription,
									IpCidrRange: "10.29.0.0/16",
									Region:      region,
									NetworkRef: v1alpha1.ResourceRef{
										Name: name,
									},
									SecondaryIpRange: []v1beta1.SubnetworkSecondaryIpRange{
										{
											IpCidrRange: "192.168.0.0/16",
											RangeName:   "ip-range-pod",
										},
										{
											IpCidrRange: "172.16.0.0/16",
											RangeName:   "ip-range-service",
										},
									},
								},
							},
						},
					},
					Container: multiversev1alpha1.Container{
						ContainerClusters: []multiversev1alpha1.ContainerClusterTemplateSpec{
							{
								CustomObjectMeta: multiversev1alpha1.CustomObjectMeta{
									Name:      name,
									Namespace: namespace,
									Annotations: map[string]string{
										"cnrm.cloud.google.com/remove-default-node-pool": "true",
									},
								},
								Spec: kccContainerv1beta1.ContainerClusterSpec{
									Description:      &desription,
									Location:         region,
									InitialNodeCount: &initialNodeCount,
									MinMasterVersion: &minMasterVersion,
									WorkloadIdentityConfig: &kccContainerv1beta1.ClusterWorkloadIdentityConfig{
										WorkloadPool: &workloadPool,
									},
									NetworkingMode: &networkingMode,
									NetworkRef: &v1alpha1.ResourceRef{
										Name: name,
									},
									SubnetworkRef: &v1alpha1.ResourceRef{
										Name: name,
									},
									IpAllocationPolicy: &kccContainerv1beta1.ClusterIpAllocationPolicy{
										ServicesSecondaryRangeName: &servicesSecondaryRangeName,
										ClusterSecondaryRangeName:  &clusterSecondaryRangeName,
									},
									ClusterAutoscaling: &kccContainerv1beta1.ClusterClusterAutoscaling{
										Enabled:            true,
										AutoscalingProfile: &autoscalingProfile,
										AutoProvisioningDefaults: &kccContainerv1beta1.ClusterAutoProvisioningDefaults{
											ServiceAccountRef: &v1alpha1.ResourceRef{
												Name: common.ClusterServiceAccountName,
											},
										},
										ResourceLimits: []kccContainerv1beta1.ClusterResourceLimits{
											{
												ResourceType: "cpu",
												Maximum:      &maxCpu,
												Minimum:      &minCpu,
											},
											{
												ResourceType: "memory",
												Maximum:      &maxMem,
												Minimum:      &minMem,
											},
										},
									},
									ReleaseChannel: &kccContainerv1beta1.ClusterReleaseChannel{
										Channel: "STABLE",
									},
									EnableBinaryAuthorization: &enableBinaryAuthorization,
									EnableIntranodeVisibility: &enableIntranodeVisibility,
									EnableShieldedNodes:       &enableShieldedNodes,
								},
							},
						},
						ContainerNodePools: []multiversev1alpha1.ContainerNodePoolTemplateSpec{
							{
								CustomObjectMeta: multiversev1alpha1.CustomObjectMeta{
									Name:      name,
									Namespace: namespace,
								},
								Spec: kccContainerv1beta1.ContainerNodePoolSpec{
									Location: region,
									Autoscaling: &kccContainerv1beta1.NodepoolAutoscaling{
										MinNodeCount: minNodeCount,
										MaxNodeCount: maxNodeCount,
									},
									InitialNodeCount: &initialNodeCount,
									NodeConfig: &kccContainerv1beta1.NodepoolNodeConfig{
										MachineType:    &machineType,
										DiskSizeGb:     &diskSizeGb,
										DiskType:       &diskType,
										Preemptible:    &preemptible,
										MinCpuPlatform: &minCpuPlatform,
										ServiceAccountRef: &v1alpha1.ResourceRef{
											Name: common.ClusterServiceAccountName,
										},
										OauthScopes: []string{
											"https://www.googleapis.com/auth/logging.write",
											"https://www.googleapis.com/auth/monitoring",
											"https://www.googleapis.com/auth/devstorage.read_only",
										},
										Metadata: map[string]string{
											"disable-legacy-endpoints": "true",
										},
									},
									ClusterRef: v1alpha1.ResourceRef{
										Name: name,
									},
								},
							},
						},
					},
				},
			}
			err := r.Create(ctx, universe, &client.CreateOptions{})
			if err != nil {
				return err
			}
			return nil
		}

		return err
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *TriggerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&multiversev1alpha1.Trigger{}).
		Complete(r)
}
