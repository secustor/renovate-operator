package reconcile

import (
	"github.com/secustor/renovate-operator/api/v1alpha1"
	shipperconfig "github.com/secustor/renovate-operator/cmd/shipper/config"
	"github.com/secustor/renovate-operator/pkg/metadata"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func SetupDiscovery(parameters Parameters) (*controllerruntime.Result, error) {
	logging := parameters.Logger

	// ServiceAccount
	expectedServiceAccount, err := createServiceAccount(parameters)
	if err != nil {
		return nil, err
	}
	if result, reconcileErr := reconcileServiceAccount(parameters.Ctx, parameters.Client, expectedServiceAccount, logging); result != nil || err != nil {
		return result, reconcileErr
	}

	// Role
	expectedRole, err := createRole(parameters)
	if err != nil {
		return nil, err
	}
	if result, reconcileErr := reconcileRole(parameters.Ctx, parameters.Client, expectedRole, logging); result != nil || err != nil {
		return result, reconcileErr
	}

	// RoleBinding
	expectedRoleBinding := rbacv1.RoleBinding{
		ObjectMeta: metadata.GenericMetaData(parameters.Req),
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount", //TODO replace with dynamic call?
				APIGroup:  expectedServiceAccount.GroupVersionKind().Group,
				Name:      expectedServiceAccount.Name,
				Namespace: expectedServiceAccount.Namespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			Kind:     "Role", //TODO replace with dynamic call?
			APIGroup: expectedRole.GroupVersionKind().Group,
			Name:     expectedRole.Name,
		},
	}
	if err := controllerutil.SetControllerReference(&parameters.RenovateCR, &expectedRole, parameters.Scheme); err != nil {
		return nil, err
	}
	if result, reconcileErr := reconcileRoleBinding(parameters.Ctx, parameters.Client, &expectedRoleBinding, logging); result != nil || err != nil {
		return result, reconcileErr
	}

	// Cronjob
	jobSpec := createDiscoveryJobSpec(parameters.RenovateCR, expectedServiceAccount.Name)

	expectedCronJob, cjCreationErr := createDiscoveryCronJob(parameters, jobSpec)
	if cjCreationErr != nil {
		return &controllerruntime.Result{}, cjCreationErr
	}

	return reconcileCronjob(parameters.Ctx, parameters.Client, expectedCronJob, logging)
}

func createRole(parameters Parameters) (rbacv1.Role, error) {
	expectedRole := rbacv1.Role{
		ObjectMeta: metadata.GenericMetaData(parameters.Req),
		Rules: []rbacv1.PolicyRule{
			{
				Resources: []string{rbacv1.ResourceAll},
				Verbs:     v1.Verbs{"get", "list", "update", "patch"},
				APIGroups: []string{v1alpha1.GroupVersion.Group},
			},
		},
	}

	if err := controllerutil.SetControllerReference(&parameters.RenovateCR, &expectedRole, parameters.Scheme); err != nil {
		return rbacv1.Role{}, err
	}
	return expectedRole, nil
}

func createServiceAccount(parameters Parameters) (corev1.ServiceAccount, error) {
	expectedServiceAccount := corev1.ServiceAccount{
		ObjectMeta: metadata.GenericMetaData(parameters.Req),
	}

	if err := controllerutil.SetControllerReference(&parameters.RenovateCR, &expectedServiceAccount, parameters.Scheme); err != nil {
		return corev1.ServiceAccount{}, err
	}
	return expectedServiceAccount, nil
}

func createDiscoveryCronJob(parameters Parameters, jobSpec batchv1.JobSpec) (*batchv1.CronJob, error) {
	cronJob := batchv1.CronJob{
		ObjectMeta: metadata.DiscoveryMetaData(parameters.Req),
		Spec: batchv1.CronJobSpec{
			Schedule:          parameters.RenovateCR.Spec.RenovateDiscoveryConfig.Schedule,
			ConcurrencyPolicy: batchv1.ForbidConcurrent,
			Suspend:           parameters.RenovateCR.Spec.Suspend,
			JobTemplate: batchv1.JobTemplateSpec{
				Spec: jobSpec,
			},
		},
	}

	if err := controllerutil.SetControllerReference(&parameters.RenovateCR, &cronJob, parameters.Scheme); err != nil {
		return nil, err
	}
	return &cronJob, nil
}

func createDiscoveryJobSpec(renovateCR v1alpha1.Renovate, serviceAccountName string) batchv1.JobSpec {
	return batchv1.JobSpec{
		Template: corev1.PodTemplateSpec{
			Spec: corev1.PodSpec{
				ServiceAccountName: serviceAccountName,
				RestartPolicy:      corev1.RestartPolicyNever,
				Volumes: renovateStandardVolumes(renovateCR, corev1.VolumeSource{
					ConfigMap: &corev1.ConfigMapVolumeSource{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: renovateCR.Name,
						},
					},
				}),
				InitContainers: []corev1.Container{
					renovateContainer(renovateCR, []corev1.EnvVar{
						{
							Name:  "RENOVATE_AUTODISCOVER",
							Value: "true",
						},
					}, []string{"--write-discovered-repos", FileRenovateConfigOutput}),
				},
				Containers: []corev1.Container{
					{
						Name:  "shipper",
						Image: "shipper:0.1.0", //TODO allow overwrite
						Env: []corev1.EnvVar{
							{
								Name:  shipperconfig.EnvRenovateCrName,
								Value: renovateCR.Name,
							},
							{
								Name:  shipperconfig.EnvRenovateCrNamespace,
								Value: renovateCR.Namespace,
							},
							{
								Name:  shipperconfig.EnvRenovateOutputFile,
								Value: FileRenovateConfigOutput,
							},
						},
						VolumeMounts: []corev1.VolumeMount{
							{
								Name:      VolumeWorkDir,
								MountPath: DirRenovateBase,
							},
						},
					},
				},
			},
		},
	}
}
