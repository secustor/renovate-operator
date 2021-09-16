/*
Copyright 2021 Sebastian Poxhofer.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/json"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"strconv"

	renovatev1alpha1 "github.com/secustor/renovate-operator/api/v1alpha1"
)

// RenovateReconciler reconciles a Renovate object
type RenovateReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=renovate.renovatebot.com,resources=renovates,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=renovate.renovatebot.com,resources=renovates/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=renovate.renovatebot.com,resources=renovates/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Renovate object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.9.2/pkg/reconcile
func (r *RenovateReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logging := log.FromContext(ctx).WithValues("CronJob.Namespace", req.Namespace, "CronJob.Name", req.Name)

	// Fetch the Renovate instance.
	renovateCR := &renovatev1alpha1.Renovate{}
	err := r.Get(ctx, req.NamespacedName, renovateCR)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// coordination config map
	currentCCM := &corev1.ConfigMap{}
	expectedCCM, creationErr := r.createCCM(renovateCR)
	if creationErr != nil {
		return ctrl.Result{}, creationErr
	}

	// ensure that the CCM exists
	err = r.Get(ctx, req.NamespacedName, currentCCM)
	if err != nil {
		if errors.IsNotFound(err) {
			if err = r.Create(ctx, expectedCCM); err != nil {
				logging.Error(err, "Failed to create ControlConfigMap")
				return ctrl.Result{}, err
			}
			logging.Info("Created ControlConfigMap")
			return ctrl.Result{Requeue: true}, nil
		}
		return ctrl.Result{}, err
	}

	// update CCM if necessary
	if !equality.Semantic.DeepDerivative(expectedCCM.Data, currentCCM.Data) {
		logging.Info("Updating CronJob")
		err := r.Update(ctx, expectedCCM)
		if err != nil {
			logging.Error(err, "Failed to update CronJob")
			return ctrl.Result{}, err
		}
		logging.Info("Updated CronJob")
		return ctrl.Result{Requeue: true}, nil
	}

	// create expected cronjob for comparison and creation
	currentCronJob := &batchv1.CronJob{}
	expectedCronJob, cjCreationErr := r.createCronJob(renovateCR)
	if cjCreationErr != nil {
		return ctrl.Result{}, cjCreationErr
	}

	// ensure cronJob exists
	err = r.Get(ctx, req.NamespacedName, currentCronJob)
	if err != nil {
		if errors.IsNotFound(err) {
			if err = r.Create(ctx, expectedCronJob); err != nil {
				logging.Error(err, "Failed to create Cronjob")
				return ctrl.Result{}, err
			}
			logging.Info("Created Cronjob")
			return ctrl.Result{Requeue: true}, nil
		}
		return ctrl.Result{}, err
	}

	// update if necessary
	if !equality.Semantic.DeepDerivative(expectedCronJob.Spec, currentCronJob.Spec) {
		logging.Info("Updating CronJob")
		err := r.Update(ctx, expectedCronJob)
		if err != nil {
			logging.Error(err, "Failed to update CronJob")
			return ctrl.Result{}, err
		}
		logging.Info("Updated CronJob")
		return ctrl.Result{Requeue: true}, nil
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *RenovateReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&renovatev1alpha1.Renovate{}).
		Complete(r)
}

func (r *RenovateReconciler) createCronJob(renovate *renovatev1alpha1.Renovate) (*batchv1.CronJob, error) {
	containerVars := []corev1.EnvVar{
		{
			Name:  "LOG_LEVEL",
			Value: string(renovate.Spec.Logging.Level),
		},
		{
			Name:  "RENOVATE_DRY_RUN",
			Value: strconv.FormatBool(*renovate.Spec.DryRun),
		},
		{
			Name:  "RENOVATE_BASE_DIR",
			Value: "/tmp/renovate/",
		},
		{
			Name:  "RENOVATE_AUTODISCOVER",
			Value: "true",
		},
		{
			Name:      "RENOVATE_TOKEN",
			ValueFrom: &renovate.Spec.Platform.Token,
		},
	}
	if renovate.Spec.GithubTokenSelector.Size() != 0 {
		containerVars = append(containerVars, corev1.EnvVar{
			Name:      "GITHUB_COM_TOKEN",
			ValueFrom: &renovate.Spec.GithubTokenSelector,
		})
	}
	if renovate.Spec.SharedCache.Enabled && renovate.Spec.SharedCache.Type == renovatev1alpha1.SharedCacheTypes_REDIS {
		containerVars = append(containerVars, corev1.EnvVar{
			Name:  "RENOVATE_REDIS_URL",
			Value: renovate.Spec.SharedCache.RedisConfig.Url,
		})
	}

	cronJob := &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      renovate.Name,
			Namespace: renovate.Namespace,
		},
		Spec: batchv1.CronJobSpec{
			Schedule:          renovate.Spec.Schedule,
			ConcurrencyPolicy: "Forbid",
			Suspend:           renovate.Spec.Suspend,
			JobTemplate: batchv1.JobTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:      renovate.Name,
					Namespace: renovate.Namespace,
				},
				Spec: batchv1.JobSpec{

					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Name:      renovate.Name,
							Namespace: renovate.Namespace,
						},
						Spec: corev1.PodSpec{
							RestartPolicy: corev1.RestartPolicyNever,
							Volumes: []corev1.Volume{
								{
									Name: "workdir",
									VolumeSource: corev1.VolumeSource{
										EmptyDir: &corev1.EmptyDirVolumeSource{},
									},
								},
								{
									Name: "config",
									VolumeSource: corev1.VolumeSource{
										EmptyDir: &corev1.EmptyDirVolumeSource{},
									},
								},
								{
									Name: "rawConfigs",
									VolumeSource: corev1.VolumeSource{
										ConfigMap: &corev1.ConfigMapVolumeSource{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: renovate.Name,
											},
										},
									},
								},
							},
							InitContainers: []corev1.Container{
								{
									Name:       "templateConfig",
									Image:      "imega/jq:1.6",
									WorkingDir: "/tmp/rawConfigs",
									Command:    []string{"jq"},
									Args: []string{
										"-s",
										"'.[0] * .[1]'",
										"base", "${JOB_COMPLETION_INDEX}", ">", "config.json",
									},
									VolumeMounts: []corev1.VolumeMount{
										{
											Name:      "config",
											MountPath: "/etc/config/renovate",
										},
										{
											Name:      "rawConfigs",
											ReadOnly:  true,
											MountPath: "/tmp/rawConfigs",
										},
									},
								},
							},
							Containers: []corev1.Container{
								{
									Name:  "renovate",
									Image: "renovate/renovate:" + renovate.Spec.RenovateVersion,
									Env:   containerVars,
									// TODO add config Volumes
									VolumeMounts: []corev1.VolumeMount{
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

	if err := controllerutil.SetControllerReference(renovate, cronJob, r.Scheme); err != nil {
		return nil, err
	}

	return cronJob, nil
}

func (r *RenovateReconciler) createCCM(renovate *renovatev1alpha1.Renovate) (*corev1.ConfigMap, error) {
	baseConfig, err := json.Marshal(renovatev1alpha1.RenovateConfig{
		Onboarding:       *renovate.Spec.RenovateAppConfig.OnBoarding,
		PrHourlyLimit:    renovate.Spec.RenovateAppConfig.PrHourlyLimit,
		OnboardingConfig: renovate.Spec.RenovateAppConfig.OnBoardingConfig,
		AddLabels:        renovate.Spec.RenovateAppConfig.AddLabels,
	})
	if err != nil {
		return nil, err
	}

	data := map[string]string{
		"base": string(baseConfig),
	}
	// TODO

	newConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      renovate.Name,
			Namespace: renovate.Namespace,
		},
		Data: data,
	}
	if err := controllerutil.SetControllerReference(renovate, newConfigMap, r.Scheme); err != nil {
		return nil, err
	}
	return newConfigMap, nil
}
