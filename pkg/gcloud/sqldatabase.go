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

// GetSQLDatabaseStatus return sqlinstance state and error if any
func (g *Client) GetSQLDatabaseStatus(ctx context.Context, projectID string, instanceName string, databaseName string) (string, error) {
	t, _ := g.getToken(ctx)
	sqlService, _ := sqladmin.NewService(ctx, option.WithHTTPClient(oauth2.NewClient(ctx, t)))
	i, err := sqladmin.NewDatabasesService(sqlService).Get(projectID, instanceName, databaseName).Do()
	if err != nil || reflect.ValueOf(i).IsZero() {
		return common.NotCreatedStatus, nil
	}
	if !reflect.ValueOf(i).IsZero() {
		log.WithFields(log.Fields{
			"instanceName": instanceName,
			"databaseName": databaseName,
		}).Info("SQLDatabase is running")
		return common.RunningStatus, nil
	}
	return common.NotCreatedStatus, nil
}
