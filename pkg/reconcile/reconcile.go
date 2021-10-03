package reconcile

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/secustor/renovate-operator/pkg/equality"
	"github.com/secustor/renovate-operator/pkg/scaling"
	v1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func Reconcile(parameters Parameters) (*ctrl.Result, error) {
	batches, err := scaling.CreateBatches(parameters.RenovateCR)
	if err != nil {
		return &ctrl.Result{}, err
	}

	if result, err := SetupCCM(parameters, batches); result != nil || err != nil {
		return result, err
	}

	if result, err := SetupDiscovery(parameters); result != nil || err != nil {
		return result, err
	}

	// do not set up workers if no batches are not ready yet
	if batches == nil {
		return &ctrl.Result{}, nil
	}

	if result, err := SetupWorker(parameters, batches); result != nil || err != nil {
		return result, err
	}

	return &ctrl.Result{}, nil
}

func reconcileCronjob(ctx context.Context, client client.Client, expectedObject *v1.CronJob, logging logr.Logger) (*ctrl.Result, error) {
	logging = logging.WithValues("cronJob", expectedObject.Name)
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

	if equality.NeedsUpdateCronJob(*expectedObject, currentObject) {
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

func reconcileServiceAccount(ctx context.Context, client client.Client, expectedObject corev1.ServiceAccount, logging logr.Logger) (*ctrl.Result, error) {
	logging = logging.WithValues("serviceAccount", expectedObject.Name)
	key := types.NamespacedName{
		Namespace: expectedObject.Namespace,
		Name:      expectedObject.Name,
	}

	currentObject := corev1.ServiceAccount{}
	err := client.Get(ctx, key, &currentObject)
	if err != nil {
		if errors.IsNotFound(err) {
			if err = client.Create(ctx, &expectedObject); err != nil {
				logging.Error(err, "Failed to create ServiceAccount")
				return &ctrl.Result{}, err
			}
			logging.Info("Created ServiceAccount")
			return &ctrl.Result{Requeue: true}, nil
		}
		return &ctrl.Result{}, err
	}

	if equality.NeedsUpdateServiceAccount(expectedObject, currentObject) {
		logging.Info("Updating ServiceAccount")
		err := client.Update(ctx, &expectedObject)
		if err != nil {
			logging.Error(err, "Failed to update ServiceAccount")
			return &ctrl.Result{}, err
		}
		logging.Info("Updated ServiceAccount")
		return &ctrl.Result{Requeue: true}, nil
	}
	return nil, nil
}

func reconcileRole(ctx context.Context, client client.Client, expectedObject rbacv1.Role, logging logr.Logger) (*ctrl.Result, error) {
	key := types.NamespacedName{
		Namespace: expectedObject.Namespace,
		Name:      expectedObject.Name,
	}

	currentObject := rbacv1.Role{}
	err := client.Get(ctx, key, &currentObject)
	if err != nil {
		if errors.IsNotFound(err) {
			if err = client.Create(ctx, &expectedObject); err != nil {
				logging.Error(err, "Failed to create Role")
				return &ctrl.Result{}, err
			}
			logging.Info("Created Role")
			return &ctrl.Result{Requeue: true}, nil
		}
		return &ctrl.Result{}, err
	}

	if equality.NeedsUpdateRole(currentObject, expectedObject) {
		logging.Info("Updating Role")
		err := client.Update(ctx, &expectedObject)
		if err != nil {
			logging.Error(err, "Failed to update Role")
			return &ctrl.Result{}, err
		}
		logging.Info("Updated Role")
		return &ctrl.Result{Requeue: true}, nil
	}
	return nil, nil
}

func reconcileRoleBinding(ctx context.Context, client client.Client, expectedObject *rbacv1.RoleBinding, logging logr.Logger) (*ctrl.Result, error) {
	logging = logging.WithValues("roleBinding", expectedObject.Name)
	key := types.NamespacedName{
		Namespace: expectedObject.Namespace,
		Name:      expectedObject.Name,
	}

	currentObject := rbacv1.RoleBinding{}
	err := client.Get(ctx, key, &currentObject)
	if err != nil {
		if errors.IsNotFound(err) {
			if err = client.Create(ctx, expectedObject); err != nil {
				logging.Error(err, "Failed to create RoleBinding")
				return &ctrl.Result{}, err
			}
			logging.Info("Created Role")
			return &ctrl.Result{Requeue: true}, nil
		}
		return &ctrl.Result{}, err
	}

	if equality.NeedsUpdateRoleBinding(*expectedObject, currentObject) {
		logging.Info("Updating RoleBinding")
		err := client.Update(ctx, expectedObject)
		if err != nil {
			logging.Error(err, "Failed to update RoleBinding")
			return &ctrl.Result{}, err
		}
		logging.Info("Updated RoleBinding")
		return &ctrl.Result{Requeue: true}, nil
	}
	return nil, nil
}
