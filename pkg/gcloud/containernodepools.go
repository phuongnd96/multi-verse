package gcloud

import (
	"context"
	"fmt"
	"reflect"

	common "github.com/phuongnd96/multi-verse/pkg/common"
	log "github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	"google.golang.org/api/container/v1"
	"google.golang.org/api/option"
)

func (g *Client) GetContainerNodepoolStatus(ctx context.Context, projectID string, name string, clusterName string, location string) (string, error) {
	t, _ := g.getToken(ctx)
	containerService, _ := container.NewService(ctx, option.WithHTTPClient(oauth2.NewClient(ctx, t)))
	selLInk := fmt.Sprintf("projects/%s/locations/%s/clusters/%s/nodePools/%s", projectID, location, clusterName, name)
	c, err := container.NewProjectsLocationsClustersNodePoolsService(containerService).Get(selLInk).Do()
	if err != nil || reflect.ValueOf(c).IsZero() {
		return common.NotCreatedStatus, nil
	}
	if !reflect.ValueOf(c).IsZero() && c.Status == "RUNNING" {
		log.WithFields(log.Fields{
			"name": name,
		}).Info("Container nodepool is running")
		return common.RunningStatus, nil
	}
	return common.NotCreatedStatus, nil
}
