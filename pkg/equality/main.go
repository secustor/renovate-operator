package equality

import (
	"reflect"

	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/equality"
)

func NeedsUpdateServiceAccount(a, b v1.ServiceAccount) bool {
	equal := reflect.DeepEqual(a.Labels, b.Labels)
	return !equal
}

func NeedsUpdateCronJob(a, b batchv1.CronJob) bool {
	equal := reflect.DeepEqual(a.Labels, b.Labels) &&
		equality.Semantic.DeepDerivative(a.Spec, b.Spec)
	return !equal
}

func NeedsUpdateRoleBinding(a, b rbacv1.RoleBinding) bool {
	equal := reflect.DeepEqual(a.Labels, b.Labels) &&
		equality.Semantic.DeepDerivative(a.RoleRef, b.RoleRef) &&
		equality.Semantic.DeepDerivative(a.Subjects, b.Subjects)
	return !equal
}

func NeedsUpdateRole(a, b rbacv1.Role) bool {
	equal := reflect.DeepEqual(a.Labels, b.Labels) &&
		reflect.DeepEqual(a.Rules, b.Rules)
	return !equal
}
