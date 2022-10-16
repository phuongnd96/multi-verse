package controllers

import (
	"context"

	"github.com/go-logr/logr"
	multiversev1alpha1 "github.com/phuongnd96/multi-verse/api/v1alpha1"
)

func (r *UniverseReconciler) finalizeUniverse(ctx context.Context, reqLogger logr.Logger, u *multiversev1alpha1.Universe) error {
	// TODO(user): Add the cleanup steps that the operator
	// needs to do before the CR can be deleted. Examples
	// of finalizers include performing backups and deleting
	// resources that are not owned by this CR, like a PVC.
	reqLogger.Info("Successfully finalized universe")
	return nil
}
