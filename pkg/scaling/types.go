package scaling

import "github.com/secustor/renovate-operator/api/v1alpha1"

type Batch struct {
	Repositories []v1alpha1.RepositoryPath `json:"repositories"`
}
