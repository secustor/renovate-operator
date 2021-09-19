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
	"github.com/go-logr/logr"
	renovatev1alpha1 "github.com/secustor/renovate-operator/api/v1alpha1"
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
	logging := log.FromContext(ctx).WithValues("Namespace", req.Namespace, "Name", req.Name)

	// Fetch the Renovate instance.
	renovateCR := &renovatev1alpha1.Renovate{}
	err := r.Get(ctx, req.NamespacedName, renovateCR)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if result, err := r.setupDiscovery(ctx, req, renovateCR, logging); result != nil || err != nil {
		return *result, err
	}

	batches, err := r.createBatches(renovateCR)
	if err != nil {
		return ctrl.Result{}, err
	}

	if result, err := r.setupCCM(ctx, req, renovateCR, batches, logging); result != nil || err != nil {
		return *result, err
	}

	if result, err := r.setupExecution(ctx, req, renovateCR, batches, logging); result != nil || err != nil {
		return *result, err
	}

	return ctrl.Result{}, nil
}

func (r *RenovateReconciler) setupCCM(ctx context.Context, req ctrl.Request, renovate *renovatev1alpha1.Renovate, batches []Batch, logging logr.Logger) (*ctrl.Result, error) {
	// coordination config map
	currentCCM := &corev1.ConfigMap{}
	expectedCCM, creationErr := r.createCCM(renovate, batches)
	if creationErr != nil {
		return nil, creationErr
	}

	// ensure that the CCM exists
	err := r.Get(ctx, req.NamespacedName, currentCCM)
	if err != nil {
		if errors.IsNotFound(err) {
			if err = r.Create(ctx, expectedCCM); err != nil {
				logging.Error(err, "Failed to create ControlConfigMap")
				return nil, err
			}
			logging.Info("Created ControlConfigMap")
			return &ctrl.Result{Requeue: true}, nil
		}
		return nil, err
	}

	// update CCM if necessary
	if !equality.Semantic.DeepDerivative(expectedCCM.Data, currentCCM.Data) {
		logging.Info("Updating base config")
		err := r.Update(ctx, expectedCCM)
		if err != nil {
			logging.Error(err, "Failed to update base config")
			return &ctrl.Result{}, err
		}
		logging.Info("Updated base config")
		return &ctrl.Result{Requeue: true}, nil
	}
	return nil, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *RenovateReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&renovatev1alpha1.Renovate{}).
		Owns(&batchv1.CronJob{}).
		Owns(&corev1.ConfigMap{}).
		Complete(r)
}

func (r *RenovateReconciler) createCronJob(renovate *renovatev1alpha1.Renovate, batches []Batch) (*batchv1.CronJob, error) {
	containerVars := []corev1.EnvVar{
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
		containerVars = append(containerVars, corev1.EnvVar{
			Name:      "GITHUB_COM_TOKEN",
			ValueFrom: &renovate.Spec.RenovateAppConfig.GithubTokenSelector,
		})
	}
	completionMode := batchv1.IndexedCompletion
	completions := int32(len(batches))
	parallelism := renovate.Spec.ScalingSpec.MaxWorkers
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
					CompletionMode: &completionMode,
					Completions:    &completions,
					Parallelism:    &parallelism,
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
									Name: "raw-configs",
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
									Name:       "template-config",
									Image:      "dwdraju/alpine-curl-jq",
									WorkingDir: "/tmp/rawConfigs",
									Command:    []string{"/bin/sh", "-c"},
									Args: []string{
										"jq -s \".[0] * .[1][$(JOB_COMPLETION_INDEX)]\" base batches > /etc/config/renovate/config.json; cat /etc/config/renovate/config.json; exit 0",
									},
									VolumeMounts: []corev1.VolumeMount{
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
							Containers: []corev1.Container{
								{
									Name:  "renovate",
									Image: "renovate/renovate:" + renovate.Spec.RenovateAppConfig.RenovateVersion,
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

func (r *RenovateReconciler) createCCM(renovate *renovatev1alpha1.Renovate, batches []Batch) (*corev1.ConfigMap, error) {
	config := renovatev1alpha1.RenovateConfig{
		DryRun:        *renovate.Spec.RenovateAppConfig.DryRun,
		Onboarding:    *renovate.Spec.RenovateAppConfig.OnBoarding,
		PrHourlyLimit: renovate.Spec.RenovateAppConfig.PrHourlyLimit,
		//OnboardingConfig: renovate.Spec.RenovateAppConfig.OnBoardingConfig,
		AddLabels: renovate.Spec.RenovateAppConfig.AddLabels,
		Platform:  renovate.Spec.RenovateAppConfig.Platform.PlatformType,
		Endpoint:  renovate.Spec.RenovateAppConfig.Platform.Endpoint,
	}
	if renovate.Spec.SharedCache.Enabled && renovate.Spec.SharedCache.Type == renovatev1alpha1.SharedCacheTypes_REDIS {
		config.RedisUrl = renovate.Spec.SharedCache.RedisConfig.Url
	}

	baseConfig, err := json.Marshal(config)
	if err != nil {
		return nil, err
	}

	batchesString, err := json.Marshal(batches)
	if err != nil {
		return nil, err
	}
	if err != nil {
		return nil, err
	}
	data := map[string]string{
		"base":    string(baseConfig),
		"batches": string(batchesString),
	}

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

type Batch struct {
	Repositories []renovatev1alpha1.RepositoryPath `json:"repositories"`
}

func (r *RenovateReconciler) createBatches(renovate *renovatev1alpha1.Renovate) ([]Batch, error) {
	repositories := renovate.Status.DiscoveredRepositories

	var batches []Batch

	switch renovate.Spec.ScalingSpec.ScalingStrategy {
	case renovatev1alpha1.ScalingStrategy_SIZE:
		limit := renovate.Spec.ScalingSpec.Size
		for i := 0; i < len(repositories); i += limit {
			repositoryBatch := repositories[i:min(i+limit, len(repositories))]
			batches = append(batches, Batch{Repositories: repositoryBatch})
		}
		break
	case renovatev1alpha1.ScalingStrategy_NONE:
	default:
		batches = append(batches, Batch{Repositories: repositories})
	}

	return batches, nil
}

func (r *RenovateReconciler) setupExecution(ctx context.Context, req ctrl.Request, renovate *renovatev1alpha1.Renovate, batches []Batch, logging logr.Logger) (*ctrl.Result, error) {
	// create expected cronjob for comparison and creation
	currentCronJob := &batchv1.CronJob{}
	expectedCronJob, cjCreationErr := r.createCronJob(renovate, batches)
	if cjCreationErr != nil {
		return &ctrl.Result{}, cjCreationErr
	}

	// ensure cronJob exists
	err := r.Get(ctx, req.NamespacedName, currentCronJob)
	if err != nil {
		if errors.IsNotFound(err) {
			if err = r.Create(ctx, expectedCronJob); err != nil {
				logging.Error(err, "Failed to create Cronjob")
				return &ctrl.Result{}, err
			}
			logging.Info("Created Cronjob")
			return &ctrl.Result{Requeue: true}, nil
		}
		return &ctrl.Result{}, err
	}

	// update if necessary
	if !equality.Semantic.DeepDerivative(expectedCronJob.Spec, currentCronJob.Spec) {
		logging.Info("Updating CronJob")
		err := r.Update(ctx, expectedCronJob)
		if err != nil {
			logging.Error(err, "Failed to update CronJob")
			return &ctrl.Result{}, err
		}
		logging.Info("Updated CronJob")
		return &ctrl.Result{Requeue: true}, nil
	}
	return nil, nil
}

func (r *RenovateReconciler) setupDiscovery(ctx context.Context, req ctrl.Request, renovate *renovatev1alpha1.Renovate, logging logr.Logger) (*ctrl.Result, error) {
	//TODO replace hardcoded
	expectedStatus := renovatev1alpha1.RenovateStatus{DiscoveredRepositories: []renovatev1alpha1.RepositoryPath{
		"secustor/terraform-pessimistic-version-constraint",
		"secustor/renovate_lock_file_2",
		"secustor/renovate-bot-testbed-apollo",
		"secustor/renovate_terraform_lockfile",
	}}
	if !equality.Semantic.DeepEqual(renovate.Status, expectedStatus) {
		renovate.Status = expectedStatus
		_ = r.Status().Update(ctx, renovate)
		return &ctrl.Result{}, nil
	}
	return nil, nil
}

func min(a, b int) int {
	if a <= b {
		return a
	}
	return b
}
