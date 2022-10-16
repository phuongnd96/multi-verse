package gcloud

import (
	"context"
	"reflect"

	common "github.com/phuongnd96/multi-verse/pkg/common"
	log "github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	"google.golang.org/api/option"
	"google.golang.org/api/sqladmin/v1"
)

// GetSQLInstanceStatus return sqlinstance state and error if any
func (g *Client) GetSQLInstanceStatus(ctx context.Context, projectID string, name string) (string, error) {
	t, _ := g.getToken(ctx)
	sqlService, _ := sqladmin.NewService(ctx, option.WithHTTPClient(oauth2.NewClient(ctx, t)))
	i, err := sqladmin.NewInstancesService(sqlService).Get(projectID, name).Do()
	if err != nil || reflect.ValueOf(i).IsZero() {
		return common.NotCreatedStatus, nil
	}
	if !reflect.ValueOf(i).IsZero() && i.State == "RUNNABLE" {
		log.WithFields(log.Fields{
			"name": name,
		}).Info("SQLInstance is running")
		return common.RunningStatus, nil
	}
	return common.NotCreatedStatus, nil
}

// sqladmin.NewInstancesService(sqlService).RestoreBackup()
