package reconcile

import (
	"github.com/secustor/renovate-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/equality"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

func SetupDiscovery(parameter Parameters) (*controllerruntime.Result, error) {
	//TODO replace hardcoded
	expectedStatus := v1alpha1.RenovateStatus{DiscoveredRepositories: []v1alpha1.RepositoryPath{
		"secustor/terraform-pessimistic-version-constraint",
		"secustor/renovate_lock_file_2",
		"secustor/renovate-bot-testbed-apollo",
		"secustor/renovate_terraform_lockfile",
	}}
	if !equality.Semantic.DeepEqual(parameter.RenovateCR.Status, expectedStatus) {
		parameter.RenovateCR.Status = expectedStatus
		_ = parameter.Client.Status().Update(parameter.Ctx, &parameter.RenovateCR)
		return &controllerruntime.Result{}, nil
	}
	return nil, nil
}
