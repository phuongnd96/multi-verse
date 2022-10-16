package helm

import (
	"context"
	"log"
	"net/url"
	"os"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func DeployHelm(ctx context.Context, clientset kubernetes.Interface, secret *v1.Secret, externalRestConfig *rest.Config, name string, namespace string, chartUrl string) error {
	helm_driver := ""
	var err error
	u, err := url.Parse(externalRestConfig.Host)
	if err != nil {
		return err
	}
	caPath := "/tmp/" + u.Host
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
		chartPath, err := client.LocateChart(chartUrl, settings)
		if err != nil {
			return err
		}
		chart, err := loader.Load(chartPath)
		if err != nil {
			panic(err)
		}
		_, err = client.Run(chart, nil)
		if err != nil {
			return err
		}
	}
	return nil
}
