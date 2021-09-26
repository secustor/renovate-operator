package reconcile

import (
	"github.com/secustor/renovate-operator/pkg/metadata"
	"github.com/secustor/renovate-operator/pkg/scaling"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func createWorkerCronJob(parameter Parameters, batches []scaling.Batch) (*batchv1.CronJob, error) {
	renovateCR := parameter.RenovateCR

	completionMode := batchv1.IndexedCompletion
	completions := int32(len(batches))
	parallelism := renovateCR.Spec.ScalingSpec.MaxWorkers
	cronJob := &batchv1.CronJob{
		ObjectMeta: metadata.WorkerMetaData(parameter.Req),
		Spec: batchv1.CronJobSpec{
			Schedule:          renovateCR.Spec.Schedule,
			ConcurrencyPolicy: batchv1.ForbidConcurrent,
			Suspend:           renovateCR.Spec.Suspend,
			JobTemplate: batchv1.JobTemplateSpec{
				Spec: batchv1.JobSpec{
					CompletionMode: &completionMode,
					Completions:    &completions,
					Parallelism:    &parallelism,
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							RestartPolicy: corev1.RestartPolicyNever,
							Volumes: append(renovateStandardVolumes(renovateCR, corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							}),
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
								{
									Name:       "template-config",
									Image:      "dwdraju/alpine-curl-jq", //TODO use pinned image
									WorkingDir: DirRawConfig,
									Command:    []string{"/bin/sh", "-c"},
									Args: []string{
										"jq -s \".[0] * .[1][$(JOB_COMPLETION_INDEX)]\" renovate.json batches > " + FileRenovateConfig + "; cat " + FileRenovateConfig + "; exit 0",
									},
									VolumeMounts: []corev1.VolumeMount{
										{
											Name:      VolumeConfig,
											MountPath: DirRenovateConfig,
										},
										{
											Name:      VolumeRawConfig,
											ReadOnly:  true,
											MountPath: DirRawConfig,
										},
									},
								},
							},
							Containers: []corev1.Container{
								renovateContainer(renovateCR, []corev1.EnvVar{}, []string{}),
							},
						},
					},
				},
			},
		},
	}

	if err := controllerutil.SetControllerReference(&renovateCR, cronJob, parameter.Scheme); err != nil {
		return nil, err
	}

	return cronJob, nil
}

func SetupWorker(parameters Parameters, batches []scaling.Batch) (*controllerruntime.Result, error) {
	logging := parameters.Logger.WithValues("cronJob", metadata.WorkerName(parameters.Req))
	// create expected cronjob for comparison and creation
	expectedCronJob, cjCreationErr := createWorkerCronJob(parameters, batches)
	if cjCreationErr != nil {
		return &controllerruntime.Result{}, cjCreationErr
	}

	return reconcileCronjob(parameters.Ctx, parameters.Client, expectedCronJob, logging)
}
