package controllers

import (
	"context"
	"log"
	"net/url"
	"os"
	"strings"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func (r *UniverseReconciler) deployNginxIngress(ctx context.Context, clientset kubernetes.Interface, secret *v1.Secret, externalRestConfig *rest.Config) error {
	name := "nginx-ingress-controller"
	namespace := "nginx-ingress"
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
		if err := actionConfig.Init(settings.RESTClientGetter(), namespace, helm_driver, log.Printf); err != nil {
			return err
		}
		client := action.NewInstall(actionConfig)
		client.ReleaseName = name
		client.Namespace = namespace
		chartPath, err := client.LocateChart("https://charts.bitnami.com/bitnami/nginx-ingress-controller-9.1.2.tgz", settings)
		if err != nil {
			return err
		}
		chart, err := loader.Load(chartPath)
		if err != nil {
			return err
		}
		vals := map[string]interface{}{
			"publishService": map[string]interface{}{
				"enabled": true,
			},
		}
		_, err = client.Run(chart, vals)
		if err != nil {
			return err
		}
		// fmt.Printf("rl: %v\n", rl)
	}
	return nil
}
