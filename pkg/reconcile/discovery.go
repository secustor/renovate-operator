package reconcile

import (
	"github.com/secustor/renovate-operator/api/v1alpha1"
	shipperconfig "github.com/secustor/renovate-operator/cmd/shipper/config"
	"github.com/secustor/renovate-operator/pkg/metadata"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func SetupDiscovery(parameters Parameters) (*controllerruntime.Result, error) {
	logging := parameters.Logger.WithValues("cronJob", metadata.DiscoveryName(parameters.Req))

	jobSpec := createDiscoveryJobSpec(parameters.RenovateCR)

	expectedCronJob, cjCreationErr := createDiscoveryCronJob(parameters, jobSpec)
	if cjCreationErr != nil {
		return &controllerruntime.Result{}, cjCreationErr
	}

	return reconcileCronjob(parameters.Ctx, parameters.Client, expectedCronJob, logging)
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

func createDiscoveryJobSpec(renovateCR v1alpha1.Renovate) batchv1.JobSpec {
	return batchv1.JobSpec{
		Template: corev1.PodTemplateSpec{
			Spec: corev1.PodSpec{
				RestartPolicy: corev1.RestartPolicyNever,
				Volumes:       renovateStandardVolumes(renovateCR),
				InitContainers: []corev1.Container{
					renovateContainer(renovateCR, []string{"--write-discovered-repos", FileRenovateConfigOutput}),
				},
				Containers: []corev1.Container{
					{
						Name:  "shipper",
						Image: "localhost/shipper:latest", //TODO use pinned image
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
					},
				},
			},
		},
	}
}
