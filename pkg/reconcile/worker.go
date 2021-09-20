package reconcile

import (
	"github.com/secustor/renovate-operator/pkg/scaling"
	v1 "k8s.io/api/batch/v1"
	v12 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	v13 "k8s.io/apimachinery/pkg/apis/meta/v1"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func createCronJob(parameter Parameters, batches []scaling.Batch) (*v1.CronJob, error) {
	renovate := parameter.RenovateCR
	containerVars := []v12.EnvVar{
		{
			Name:  "LOG_LEVEL",
			Value: string(renovate.Spec.Logging.Level),
		},
		{
			Name:  "RENOVATE_BASE_DIR",
			Value: "/tmp/renovate/",
		},
		{
			Name:  "RENOVATE_CONFIG_FILE",
			Value: "/etc/config/renovate/config.json",
		},
		{
			Name:      "RENOVATE_TOKEN",
			ValueFrom: &renovate.Spec.RenovateAppConfig.Platform.Token,
		},
	}
	if renovate.Spec.RenovateAppConfig.GithubTokenSelector.Size() != 0 {
		containerVars = append(containerVars, v12.EnvVar{
			Name:      "GITHUB_COM_TOKEN",
			ValueFrom: &renovate.Spec.RenovateAppConfig.GithubTokenSelector,
		})
	}
	completionMode := v1.IndexedCompletion
	completions := int32(len(batches))
	parallelism := renovate.Spec.ScalingSpec.MaxWorkers
	cronJob := &v1.CronJob{
		ObjectMeta: v13.ObjectMeta{
			Name:      renovate.Name,
			Namespace: renovate.Namespace,
		},
		Spec: v1.CronJobSpec{
			Schedule:          renovate.Spec.Schedule,
			ConcurrencyPolicy: "Forbid",
			Suspend:           renovate.Spec.Suspend,
			JobTemplate: v1.JobTemplateSpec{
				ObjectMeta: v13.ObjectMeta{
					Name:      renovate.Name,
					Namespace: renovate.Namespace,
				},
				Spec: v1.JobSpec{
					CompletionMode: &completionMode,
					Completions:    &completions,
					Parallelism:    &parallelism,
					Template: v12.PodTemplateSpec{
						ObjectMeta: v13.ObjectMeta{
							Name:      renovate.Name,
							Namespace: renovate.Namespace,
						},
						Spec: v12.PodSpec{
							RestartPolicy: v12.RestartPolicyNever,
							Volumes: []v12.Volume{
								{
									Name: "workdir",
									VolumeSource: v12.VolumeSource{
										EmptyDir: &v12.EmptyDirVolumeSource{},
									},
								},
								{
									Name: "config",
									VolumeSource: v12.VolumeSource{
										EmptyDir: &v12.EmptyDirVolumeSource{},
									},
								},
								{
									Name: "raw-configs",
									VolumeSource: v12.VolumeSource{
										ConfigMap: &v12.ConfigMapVolumeSource{
											LocalObjectReference: v12.LocalObjectReference{
												Name: renovate.Name,
											},
										},
									},
								},
							},
							InitContainers: []v12.Container{
								{
									Name:       "template-config",
									Image:      "dwdraju/alpine-curl-jq",
									WorkingDir: "/tmp/rawConfigs",
									Command:    []string{"/bin/sh", "-c"},
									Args: []string{
										"jq -s \".[0] * .[1][$(JOB_COMPLETION_INDEX)]\" base batches > /etc/config/renovate/config.json; cat /etc/config/renovate/config.json; exit 0",
									},
									VolumeMounts: []v12.VolumeMount{
										{
											Name:      "config",
											MountPath: "/etc/config/renovate",
										},
										{
											Name:      "raw-configs",
											ReadOnly:  true,
											MountPath: "/tmp/rawConfigs",
										},
									},
								},
							},
							Containers: []v12.Container{
								{
									Name:  "renovate",
									Image: "renovate/renovate:" + renovate.Spec.RenovateAppConfig.RenovateVersion,
									Env:   containerVars,
									// TODO add config Volumes
									VolumeMounts: []v12.VolumeMount{
										{
											Name:      "workdir",
											MountPath: "/tmp/renovate/",
										},
										{
											Name:      "config",
											ReadOnly:  true,
											MountPath: "/etc/config/renovate",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	if err := controllerutil.SetControllerReference(&renovate, cronJob, parameter.Scheme); err != nil {
		return nil, err
	}

	return cronJob, nil
}

func SetupExecution(parameters Parameters, batches []scaling.Batch) (*controllerruntime.Result, error) {
	logging := parameters.Logger
	// create expected cronjob for comparison and creation
	currentCronJob := &v1.CronJob{}
	expectedCronJob, cjCreationErr := createCronJob(parameters, batches)
	if cjCreationErr != nil {
		return &controllerruntime.Result{}, cjCreationErr
	}

	// ensure cronJob exists
	err := parameters.Client.Get(parameters.Ctx, parameters.Req.NamespacedName, currentCronJob)
	if err != nil {
		if errors.IsNotFound(err) {
			if err = parameters.Client.Create(parameters.Ctx, expectedCronJob); err != nil {
				logging.Error(err, "Failed to create Cronjob")
				return &controllerruntime.Result{}, err
			}
			logging.Info("Created Cronjob")
			return &controllerruntime.Result{Requeue: true}, nil
		}
		return &controllerruntime.Result{}, err
	}

	// update if necessary
	if !equality.Semantic.DeepDerivative(expectedCronJob.Spec, currentCronJob.Spec) {
		logging.Info("Updating CronJob")
		err := parameters.Client.Update(parameters.Ctx, expectedCronJob)
		if err != nil {
			logging.Error(err, "Failed to update CronJob")
			return &controllerruntime.Result{}, err
		}
		logging.Info("Updated CronJob")
		return &controllerruntime.Result{Requeue: true}, nil
	}
	return nil, nil
}
