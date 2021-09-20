package reconcile

import (
	"github.com/secustor/renovate-operator/pkg/scaling"
	ctrl "sigs.k8s.io/controller-runtime"
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

	if result, err := SetupExecution(parameters, batches); result != nil || err != nil {
		return result, err
	}

	return &ctrl.Result{}, nil
}
