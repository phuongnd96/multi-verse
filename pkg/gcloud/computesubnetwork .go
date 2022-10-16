package gcloud

import (
	"context"
	"reflect"

	common "github.com/phuongnd96/multi-verse/pkg/common"
	log "github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/option"
)

// getComputeNetwork return computenetwork state and error if any
func (g *Client) GetComputeSubNetworkStatus(ctx context.Context, projectID string, name string, region string) (string, error) {
	t, _ := g.getToken(ctx)
	computeService, _ := compute.NewService(ctx, option.WithHTTPClient(oauth2.NewClient(ctx, t)))
	snw, err := compute.NewSubnetworksService(computeService).Get(projectID, region, name).Do()
	if err != nil || reflect.ValueOf(snw).IsZero() {
		return common.NotCreatedStatus, nil
	}
	if !reflect.ValueOf(snw).IsZero() && snw.SelfLink != "" {
		log.WithFields(log.Fields{
			"name": name,
		}).Info("Compute subnetwork is running")
		return common.RunningStatus, nil
	}
	return common.NotCreatedStatus, nil
}
