package scaling

import (
	"github.com/secustor/renovate-operator/api/v1alpha1"
)

func CreateBatches(renovate v1alpha1.Renovate) ([]Batch, error) {
	repositories := renovate.Status.DiscoveredRepositories

	var batches []Batch

	switch renovate.Spec.ScalingSpec.ScalingStrategy {
	case v1alpha1.ScalingStrategy_SIZE:
		limit := renovate.Spec.ScalingSpec.Size
		for i := 0; i < len(repositories); i += limit {
			repositoryBatch := repositories[i:min(i+limit, len(repositories))]
			batches = append(batches, Batch{Repositories: repositoryBatch})
		}
		break
	case v1alpha1.ScalingStrategy_NONE:
	default:
		batches = append(batches, Batch{Repositories: repositories})
	}

	return batches, nil
}
