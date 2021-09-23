package reconcile

import (
	"path/filepath"

	"github.com/secustor/renovate-operator/api/v1alpha1"
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
				Volumes: append(renovateStandardVolumes(renovateCR), //TODO fix volumes
					corev1.Volume{
						Name: VolumeRawConfig,
						VolumeSource: corev1.VolumeSource{
							ConfigMap: &corev1.ConfigMapVolumeSource{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: renovateCR.Name,
								},
							},
						},
					}),
				InitContainers: []corev1.Container{
					renovateContainer(renovateCR, []string{"--write-discovered-repos", filepath.Join(DirRenovateBase, "repositories.json")}),
				},
				Containers: []corev1.Container{ //TODO replace with custom program
					{
						Name:       "template-config",
						Image:      "dwdraju/alpine-curl-jq", //TODO use pinned image
						WorkingDir: DirRawConfig,
						Command:    []string{"/bin/sh", "-c"},
						Args: []string{
							"jq -s \".[0] * .[1][$(JOB_COMPLETION_INDEX)]\" renovate.json batches > " + FileRenovateConfigOutput + "; cat " + FileRenovateConfigOutput + "; exit 0",
						},
						VolumeMounts: []corev1.VolumeMount{
							{
								Name:      VolumeWorkDir,
								ReadOnly:  true,
								MountPath: DirRenovateBase,
							},
						},
					},
				},
			},
		},
	}
}
