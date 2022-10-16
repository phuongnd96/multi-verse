package gcloud

import (
	"context"
	"reflect"

	common "github.com/phuongnd96/multi-verse/pkg/common"
	log "github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	"google.golang.org/api/option"
	"google.golang.org/api/storage/v1"
)

// GetStorageBucketStatus return storagebucket state and error if any
func (g *Client) GetStorageBucketStatus(ctx context.Context, name string) (string, error) {
	t, _ := g.getToken(ctx)
	storageService, _ := storage.NewService(ctx, option.WithHTTPClient(oauth2.NewClient(ctx, t)))
	snw, err := storage.NewBucketsService(storageService).Get(name).Do()
	if err != nil || reflect.ValueOf(snw).IsZero() {
		return common.NotCreatedStatus, nil
	}
	if !reflect.ValueOf(snw).IsZero() && snw.TimeCreated != "" {
		log.WithFields(log.Fields{
			"name": name,
		}).Info("SQLInstance is running")
		return common.RunningStatus, nil
	}
	return common.NotCreatedStatus, nil
}
