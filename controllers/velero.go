package controllers

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/phuongnd96/multi-verse/pkg/common"
	"github.com/phuongnd96/multi-verse/pkg/poll"
	log "github.com/sirupsen/logrus"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var (
	TargetClusterName = ""
	SourceClusterName = ""
	BackUpLocation    = ""
)

func deployVelero(ctx context.Context, clientset kubernetes.Interface, secret *v1.Secret, externalRestConfig *rest.Config) error {
	name := "velero"
	namespace := "velero"
	helm_driver := ""
	var err error
	u, err := url.Parse(externalRestConfig.Host)
	if err != nil {
		return err
	}
	caPath := "/tmp/" + strings.ReplaceAll(u.Host, ".", "")
	bearerToken := string(secret.Data["token"])
	caData := secret.Data["ca.crt"]
	err = os.WriteFile(caPath, caData, 0644)
	if err != nil {
		return err
	}
	os.Setenv("HELM_KUBETOKEN", bearerToken)
	os.Setenv("HELM_KUBEAPISERVER", externalRestConfig.Host)
	os.Setenv("HELM_KUBECAFILE", caPath)
	settings := cli.New()
	actionConfig := new(action.Configuration)
	_, err = clientset.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err != nil && errors.IsNotFound(err) {
		ns := &v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespace,
			},
		}
		clientset.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
		// Install
		if err := actionConfig.Init(settings.RESTClientGetter(), namespace, helm_driver, log.Printf); err != nil {
			return err
		}
		client := action.NewInstall(actionConfig)
		client.ReleaseName = name
		client.Namespace = namespace
		chartPath, err := client.LocateChart("https://github.com/vmware-tanzu/helm-charts/releases/download/velero-2.31.3/velero-2.31.3.tgz", settings)
		if err != nil {
			return err
		}
		chart, err := loader.Load(chartPath)
		if err != nil {
			return err
		}
		vals := map[string]interface{}{
			"configuration": map[string]interface{}{
				"provider": "gcp",
				"backupStorageLocation": map[string]interface{}{
					"bucket": common.VeleroBackupBucket,
					"name":   "default",
					"config": map[string]interface{}{
						"serviceAccount": common.VeleroSaId,
					},
				},
			},
			"initContainers": []map[string]interface{}{
				{
					"name":  "velero-plugin-for-gcp",
					"image": "velero/velero-plugin-for-gcp:v1.5.0",
					"volumeMounts": []map[string]interface{}{
						{
							"mountPath": "/target",
							"name":      "plugins",
						},
					},
				},
			},
			"credentials": map[string]interface{}{
				"useSecret": false,
			},
			"serviceAccount": map[string]interface{}{
				"server": map[string]interface{}{
					"annotations": map[string]interface{}{
						"iam.gke.io/gcp-service-account": common.VeleroSaId,
					},
				},
			},
		}
		rl, err := client.Run(chart, vals)
		fmt.Printf("rl: %v\n", rl)
		if err != nil {
			return err
		}
	}
	return nil
}

func createBackUp(ctx context.Context, externalRestConfig *rest.Config, namespaces []string, name string) error {
	var err error
	veleroNamespace := "velero"
	dynamicClient, err := dynamic.NewForConfig(externalRestConfig)
	if err != nil {
		return err
	}
	gvr := schema.GroupVersionResource{
		Group:    "velero.io",
		Version:  "v1",
		Resource: "backups",
	}
	_, err = dynamicClient.Resource(gvr).Namespace(veleroNamespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil && errors.IsNotFound(err) {
		b := &unstructured.Unstructured{}
		b.SetUnstructuredContent(map[string]interface{}{
			"apiVersion": "velero.io/v1",
			"kind":       "Backup",
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": veleroNamespace,
			},
			"spec": map[string]interface{}{
				"csiSnapshotTimeout":     "10m0s",
				"defaultVolumesToRestic": false,
				"hooks":                  map[string]interface{}{},
				"includedNamespaces":     namespaces,
				"metadata":               map[string]interface{}{},
				"storageLocation":        "default",
				"ttl":                    "720h0m0s",
				"volumeSnapshotLocations": []string{
					"default",
				},
				"labelSelector": map[string]interface{}{
					"matchExpressions": []map[string]interface{}{
						{
							"key":      "backup",
							"operator": "NotIn",
							"values": []string{
								"ignore",
							},
						},
					},
				},
			},
		})
		_, err := dynamicClient.Resource(gvr).Namespace(veleroNamespace).Create(ctx, b, metav1.CreateOptions{})
		if err != nil {
			return err
		}
		return nil
	}
	return nil
}

func createRestore(ctx context.Context, secret *v1.Secret, externalRestConfig *rest.Config, name string, backUpName string) error {
	var err error
	veleroNamespace := "velero"
	dynamicClient, err := dynamic.NewForConfig(externalRestConfig)
	if err != nil {
		return err
	}
	gvr := schema.GroupVersionResource{
		Group:    "velero.io",
		Version:  "v1",
		Resource: "restores",
	}
	_, err = dynamicClient.Resource(gvr).Namespace(veleroNamespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil && errors.IsNotFound(err) {
		b := &unstructured.Unstructured{}
		b.SetUnstructuredContent(map[string]interface{}{
			"apiVersion": "velero.io/v1",
			"kind":       "Restore",
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": veleroNamespace,
			},
			"spec": map[string]interface{}{
				"backupName": backUpName,
				"excludedResources": []string{
					"nodes", "events", "events.events.k8s.io", "backups.velero.io", "restores.velero.io", "resticrepositories.velero.io",
				},
				"hooks": map[string]interface{}{},
				"includedNamespaces": []string{
					"*",
				},
			},
		})
		_, err := dynamicClient.Resource(gvr).Namespace(veleroNamespace).Create(ctx, b, metav1.CreateOptions{})
		if err != nil {
			return err
		}
		return nil
	}
	return nil
}

func waitForBackUp(externalRestConfig *rest.Config, name string) error {
	ctx, cancel := context.WithTimeout(context.Background(), common.GlobalResourceTimeOutMinutes*time.Minute)
	defer cancel()
	return poll.Wait(ctx, func(ctx context.Context) (bool, error) {
		log.WithFields(log.Fields{
			"name": name,
		}).Info("Waiting for backup")
		var err error
		veleroNamespace := "velero"
		dynamicClient, err := dynamic.NewForConfig(externalRestConfig)
		if err != nil {
			return false, err
		}
		gvr := schema.GroupVersionResource{
			Group:    "velero.io",
			Version:  "v1",
			Resource: "backups",
		}
		resource, err := dynamicClient.Resource(gvr).Namespace(veleroNamespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		var backup velerov1.Backup
		err = runtime.DefaultUnstructuredConverter.FromUnstructured(resource.UnstructuredContent(), &backup)
		if err != nil {
			return false, err
		}
		log.WithFields(log.Fields{
			"phase": backup.Status.Phase,
		}).Info("Waiting for backup")
		if backup.Status.Phase == "Completed" {
			return true, nil
		}
		return false, nil
	})
}

func waitForRestore(externalRestConfig *rest.Config, name string) error {
	ctx, cancel := context.WithTimeout(context.Background(), common.GlobalResourceTimeOutMinutes*time.Minute)
	defer cancel()
	return poll.Wait(ctx, func(ctx context.Context) (bool, error) {
		log.WithFields(log.Fields{
			"name": name,
		}).Info("Waiting for restore")
		var err error
		veleroNamespace := "velero"
		dynamicClient, err := dynamic.NewForConfig(externalRestConfig)
		if err != nil {
			return false, err
		}
		gvr := schema.GroupVersionResource{
			Group:    "velero.io",
			Version:  "v1",
			Resource: "restores",
		}
		resource, err := dynamicClient.Resource(gvr).Namespace(veleroNamespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return false, nil
		}
		var backup velerov1.Restore
		err = runtime.DefaultUnstructuredConverter.FromUnstructured(resource.UnstructuredContent(), &backup)
		if err != nil {
			return false, err
		}
		if backup.Status.Phase == "Completed" {
			return true, nil
		}
		return false, nil
	})
}
