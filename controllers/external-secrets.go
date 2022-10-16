package controllers

import (
	"context"
	"log"
	"net/url"
	"os"
	"strings"

	"github.com/phuongnd96/multi-verse/pkg/common"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func (r *UniverseReconciler) deployExternalSecret(ctx context.Context, clientset kubernetes.Interface, secret *v1.Secret, externalRestConfig *rest.Config) error {
	name := "external-secrets"
	namespace := "kube-system"
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

	_, err = clientset.AppsV1().Deployments(namespace).Get(ctx, "external-secrets-kubernetes-external-secrets", metav1.GetOptions{})
	if err != nil && errors.IsNotFound(err) {
		if err := actionConfig.Init(settings.RESTClientGetter(), namespace, helm_driver, log.Printf); err != nil {
			return err
		}
		client := action.NewInstall(actionConfig)
		client.ReleaseName = name
		client.Namespace = namespace
		chartPath, err := client.LocateChart("https://external-secrets.github.io/kubernetes-external-secrets/kubernetes-external-secrets-8.5.4.tgz", settings)
		if err != nil {
			return err
		}
		chart, err := loader.Load(chartPath)
		if err != nil {
			return err
		}
		vals := map[string]interface{}{
			"serviceAccount": map[string]interface{}{
				"annotations": map[string]interface{}{
					"iam.gke.io/gcp-service-account": common.ExternalSecretSaId,
				},
			},
		}
		_, err = client.Run(chart, vals)
		if err != nil {
			return err
		}
	}
	return nil
}
