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
func (g *Client) GetComputeNetworkStatus(ctx context.Context, projectID string, name string) (string, error) {
	t, _ := g.getToken(ctx)
	computeService, _ := compute.NewService(ctx, option.WithHTTPClient(oauth2.NewClient(ctx, t)))
	network, err := compute.NewNetworksService(computeService).Get(projectID, name).Do()
	if err != nil || reflect.ValueOf(network).IsZero() {
		return common.NotCreatedStatus, nil
	}
	if !reflect.ValueOf(network).IsZero() && network.SelfLink != "" {
		log.WithFields(log.Fields{
			"name": name,
		}).Info("Compute network is running")
		return common.RunningStatus, nil
	}
	return common.NotCreatedStatus, nil
}
