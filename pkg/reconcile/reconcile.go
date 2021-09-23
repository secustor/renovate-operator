package reconcile

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/secustor/renovate-operator/pkg/scaling"
	v1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func Reconcile(parameters Parameters) (*ctrl.Result, error) {
	if result, err := SetupDiscovery(parameters); result != nil || err != nil {
		return result, err
	}

	batches, err := scaling.CreateBatches(parameters.RenovateCR)
	if err != nil {
		return &ctrl.Result{}, err
	}

	if result, err := SetupCCM(parameters, batches); result != nil || err != nil {
		return result, err
	}

	if result, err := SetupWorker(parameters, batches); result != nil || err != nil {
		return result, err
	}

	return &ctrl.Result{}, nil
}

func reconcileCronjob(ctx context.Context, client client.Client, expectedObject *v1.CronJob, logging logr.Logger) (*ctrl.Result, error) {

	key := types.NamespacedName{
		Namespace: expectedObject.Namespace,
		Name:      expectedObject.Name,
	}

	currentObject := v1.CronJob{}
	err := client.Get(ctx, key, &currentObject)
	if err != nil {
		if errors.IsNotFound(err) {
			if err = client.Create(ctx, expectedObject); err != nil {
				logging.Error(err, "Failed to create Cronjob")
				return &ctrl.Result{}, err
			}
			logging.Info("Created Cronjob")
			return &ctrl.Result{Requeue: true}, nil
		}
		return &ctrl.Result{}, err
	}

	if !equality.Semantic.DeepDerivative(expectedObject.Spec, currentObject.Spec) {
		logging.Info("Updating CronJob")
		err := client.Update(ctx, expectedObject)
		if err != nil {
			logging.Error(err, "Failed to update CronJob")
			return &ctrl.Result{}, err
		}
		logging.Info("Updated CronJob")
		return &ctrl.Result{Requeue: true}, nil
	}
	return nil, nil
}
