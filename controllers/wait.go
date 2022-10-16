package controllers

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	"github.com/phuongnd96/multi-verse/pkg/common"
	"github.com/phuongnd96/multi-verse/pkg/gcloud"
	"github.com/phuongnd96/multi-verse/pkg/poll"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func waitForContainerCluster(projectID string, log logr.Logger, name string, location string) error {
	ctx, cancel := context.WithTimeout(context.Background(), common.GlobalResourceTimeOutMinutes*time.Minute)
	defer log.Info("End wait for containercluster", "name", name)
	defer cancel()
	c := gcloud.Client{}
	var err error
	var state string
	return poll.Wait(ctx, func(ctx context.Context) (bool, error) {
		state, err = c.GetContainerClusterStatus(ctx, projectID, name, location)
		if err != nil {
			return false, err
		}
		if state == common.RunningStatus {
			return true, nil
		}
		log.Info("Wating for containercluster", "state", state)
		return false, nil
	})
}

func waitForContainerNodePool(resource client.Object, projectID string, log logr.Logger, name string, clusterName string, location string) error {
	ctx, cancel := context.WithTimeout(context.Background(), common.GlobalResourceTimeOutMinutes*time.Minute)
	defer log.Info("End wait for containernodepool", "name", name)
	defer cancel()
	c := gcloud.Client{}
	var err error
	var state string
	return poll.Wait(ctx, func(ctx context.Context) (bool, error) {
		state, err = c.GetContainerNodepoolStatus(ctx, projectID, name, clusterName, location)
		if err != nil {
			return false, err
		}
		if state == common.RunningStatus {
			return true, nil
		}
		log.Info("Wating for containerNodePool", "state", state)
		return false, nil
	})
}

// continously query google cloud api until resource is running or deadline exceed, default common.GlobalResourceTimeOutMinutes minutes.
func waitForComputeNetwork(projectID string, log logr.Logger, name string) error {
	ctx, cancel := context.WithTimeout(context.Background(), common.GlobalResourceTimeOutMinutes*time.Minute)
	defer log.Info("End wait for computenetwork", "name", name)
	defer cancel()
	c := gcloud.Client{}
	var err error
	var state string
	return poll.Wait(ctx, func(ctx context.Context) (bool, error) {
		state, err = c.GetComputeNetworkStatus(ctx, projectID, name)
		if err != nil {
			return false, err
		}
		if state == common.RunningStatus {
			return true, nil
		}
		log.Info("Wating for computeNetwork", "state", state)
		return false, nil
	})
}

// continously query google cloud api until resource is running or deadline exceed, default common.GlobalResourceTimeOutMinutes minutes.
func waitForComputeSubnetwork(projectID string, log logr.Logger, name string, location string) error {
	ctx, cancel := context.WithTimeout(context.Background(), common.GlobalResourceTimeOutMinutes*time.Minute)
	defer log.Info("End wait for computesubnetwork", "name", name)
	defer cancel()
	c := gcloud.Client{}
	var err error
	var state string
	return poll.Wait(ctx, func(ctx context.Context) (bool, error) {
		state, err = c.GetComputeSubNetworkStatus(ctx, projectID, name, location)
		if err != nil {
			return false, err
		}
		if state == common.RunningStatus {
			return true, nil
		}
		log.Info("Wating for computeSubNetwork", "state", state)
		return false, nil
	})
}

// continously query google cloud api until resource is running or deadline exceed, default common.GlobalResourceTimeOutMinutes minutes.
func waitForSQLInstance(resource client.Object, projectID string, log logr.Logger, name string) error {
	ctx, cancel := context.WithTimeout(context.Background(), common.GlobalResourceTimeOutMinutes*time.Minute)
	defer log.Info("End wait for sqlinstance", "name", name)
	defer cancel()
	c := gcloud.Client{}
	var err error
	var state string
	return poll.Wait(ctx, func(ctx context.Context) (bool, error) {
		state, err = c.GetSQLInstanceStatus(ctx, projectID, name)
		if err != nil {
			return false, err
		}
		if state == common.RunningStatus {
			return true, nil
		}
		log.Info("Wating for sqlInstance", "state", state)
		return false, nil
	})
}

// continously query google cloud api until resource is running or deadline exceed, default common.GlobalResourceTimeOutMinutes minutes.
func waitForSQLDatabase(resource client.Object, projectID string, log logr.Logger, name string, instanceName string) error {
	ctx, cancel := context.WithTimeout(context.Background(), common.GlobalResourceTimeOutMinutes*time.Minute)
	defer log.Info("End wait for sqldatabase", "name", name)
	defer cancel()
	c := gcloud.Client{}
	var err error
	var state string
	return poll.Wait(ctx, func(ctx context.Context) (bool, error) {
		state, err = c.GetSQLDatabaseStatus(ctx, projectID, instanceName, name)
		if err != nil {
			return false, err
		}
		if state == common.RunningStatus {
			return true, nil
		}
		log.Info("Wating for sqlDatabase", "state", state)
		return false, nil
	})
}

// continously query google cloud api until resource is running or deadline exceed, default common.GlobalResourceTimeOutMinutes minutes.
func waitForComputeRouter(resource client.Object, projectID string, log logr.Logger, name string, region string) error {
	ctx, cancel := context.WithTimeout(context.Background(), common.GlobalResourceTimeOutMinutes*time.Minute)
	defer log.Info("End wait for computerouter", "name", name)
	defer cancel()
	c := gcloud.Client{}
	var err error
	var state string
	return poll.Wait(ctx, func(ctx context.Context) (bool, error) {
		state, err = c.GetComputeRouterStatus(ctx, projectID, name, region)
		if err != nil {
			return false, err
		}
		if state == common.RunningStatus {
			return true, nil
		}
		log.Info("Wating for computeRouter", "state", state)
		return false, nil
	})
}

// continously query google cloud api until resource is running or deadline exceed, default common.GlobalResourceTimeOutMinutes minutes.
func waitForComputeRouterNAT(resource client.Object, projectID string, log logr.Logger, name string, region string) error {
	ctx, cancel := context.WithTimeout(context.Background(), common.GlobalResourceTimeOutMinutes*time.Minute)
	defer log.Info("End wait for computerouter", "name", name)
	defer cancel()
	c := gcloud.Client{}
	var err error
	var state string
	return poll.Wait(ctx, func(ctx context.Context) (bool, error) {
		state, err = c.GetComputeRouterStatus(ctx, projectID, name, region)
		if err != nil {
			return false, err
		}
		if state == common.RunningStatus {
			return true, nil
		}
		log.Info("Wating for computeRouterNAT", "state", state)
		return false, nil
	})
}

// continously query google cloud api until resource is running or deadline exceed, default common.GlobalResourceTimeOutMinutes minutes.
func waitForStorageBucket(resource client.Object, log logr.Logger, name string) error {
	ctx, cancel := context.WithTimeout(context.Background(), common.GlobalResourceTimeOutMinutes*time.Minute)
	defer log.Info("End wait for storageBucket", "name", name)
	defer cancel()
	c := gcloud.Client{}
	var err error
	var state string
	return poll.Wait(ctx, func(ctx context.Context) (bool, error) {
		state, err = c.GetStorageBucketStatus(ctx, name)
		if err != nil {
			return false, err
		}
		if state == common.RunningStatus {
			return true, nil
		}
		log.Info("Wating for storageBucket", "state", state)
		return false, nil
	})
}
