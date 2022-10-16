package gcloud

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"reflect"
	"time"

	common "github.com/phuongnd96/multi-verse/pkg/common"
	"github.com/phuongnd96/multi-verse/pkg/poll"
	log "github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/container/v1"
	"google.golang.org/api/option"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func (g *Client) GetContainerClusterStatus(ctx context.Context, projectID string, name string, location string) (string, error) {
	t, _ := g.getToken(ctx)
	containerService, _ := container.NewService(ctx, option.WithHTTPClient(oauth2.NewClient(ctx, t)))
	selLInk := fmt.Sprintf("projects/%s/locations/%s/clusters/%s", projectID, location, name)
	c, err := container.NewProjectsLocationsClustersService(containerService).Get(selLInk).Do()
	if err != nil || reflect.ValueOf(c).IsZero() {
		return common.NotCreatedStatus, nil
	}
	if !reflect.ValueOf(c).IsZero() && c.Status == "RUNNING" {
		log.WithFields(log.Fields{
			"name": name,
		}).Info("Container cluster is running")
		return common.RunningStatus, nil
	}
	return common.NotCreatedStatus, nil
}

// GetGKEClientset return a *rest.Config can be used to connect to a kubernetes
func (g *Client) GetGKEClientset(ctx context.Context, projectID string, gkeClusterName string, gkeLocation string) (*rest.Config, error) {
	// TODO: refactor to use client
	var err error
	creds, err := google.FindDefaultCredentials(ctx, container.CloudPlatformScope)
	if err != nil {
		return nil, err
	}
	ts := creds.TokenSource
	err = poll.Wait(ctx, func(ctx context.Context) (bool, error) {
		gkeService, err := container.NewService(ctx, option.WithHTTPClient(oauth2.NewClient(ctx, creds.TokenSource)))
		if err != nil {
			return false, err
		}
		name := fmt.Sprintf("projects/%s/locations/%s/clusters/%s", projectID, gkeLocation, gkeClusterName)
		cluster, err := container.NewProjectsLocationsClustersService(gkeService).Get(name).Do()
		// No error or cluster is not yet created
		if err != nil || reflect.ValueOf(cluster.Endpoint).IsZero() {
			log.WithFields(log.Fields{
				"gkeClusterName": gkeClusterName,
			}).Info("Wating for cluster")
			return false, nil
		}
		if !reflect.ValueOf(cluster).IsZero() && cluster.Status == "RUNNING" {
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		return nil, err
	}
	// Done wait. Cluster is ensured to be created successfully.
	gkeService, _ := container.NewService(ctx, option.WithHTTPClient(oauth2.NewClient(ctx, creds.TokenSource)))
	name := fmt.Sprintf("projects/%s/locations/%s/clusters/%s", projectID, gkeLocation, gkeClusterName)
	cluster, _ := container.NewProjectsLocationsClustersService(gkeService).Get(name).Do()
	fmt.Printf("Cluster created successfully. Remote cluster endpoint: " + cluster.Endpoint + "\n")
	log.WithFields(log.Fields{
		"cluster.Endpoint": cluster.Endpoint,
		"gkeClusterName":   gkeClusterName,
	}).Info("Cluster create successfully")
	_, err = base64.StdEncoding.DecodeString(cluster.MasterAuth.ClusterCaCertificate)
	if err != nil {
		return nil, fmt.Errorf("failed to decode cluster CA cert: %s", err)
	}

	config := &rest.Config{
		Host: "https://" + cluster.Endpoint,
		TLSClientConfig: rest.TLSClientConfig{
			// CAData:   capem,
			Insecure: true,
		},
	}
	config.Wrap(func(rt http.RoundTripper) http.RoundTripper {
		return &oauth2.Transport{
			Source: ts,
			Base:   rt,
		}
	})

	remoteClientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to initialise remoteClientset from config: %s", err)
	}
	// Wait until cluster is fully functional
	err = dryRun(remoteClientSet)
	if err != nil {
		return nil, err
	}
	return config, nil
}

// GetGKEClientsetWithTimeout construct wait for GetGKEClientset to be successed or returned, default 10 minutes.
func (g *Client) GetGKEClientsetWithTimeout(projectID string, gkeClusterName string, gkeLocation string) (*rest.Config, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()
	remoteRestConfig, err := g.GetGKEClientset(ctx, projectID, gkeClusterName, gkeLocation)
	return remoteRestConfig, err
}

func dryRun(clientset *kubernetes.Clientset) error {
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Minute)
	defer cancel()
	err := poll.Wait(ctx, func(ctx context.Context) (bool, error) {
		log.WithTime(time.Now()).Info("Starting dry run")
		ns, err := clientset.CoreV1().Namespaces().Get(ctx, "default", metav1.GetOptions{})
		log.WithFields(log.Fields{
			"namespace": ns.Name,
		}).Info("Get default namespace to make sure cluster is functional")
		if err != nil {
			fmt.Printf("dryrun err: %v\n", err)
			return false, nil
		}
		log.Info("Dry run success, cluster is functional")
		return true, nil
	})
	if err != nil {
		return fmt.Errorf("failed to dry run, remote cluster is created but not usable %s", err)
	}
	return nil
}
