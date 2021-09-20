package reconcile

import (
	"github.com/secustor/renovate-operator/api/v1alpha1"
	"github.com/secustor/renovate-operator/internal/config"
	"github.com/secustor/renovate-operator/pkg/scaling"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/json"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func SetupCCM(parameter Parameters, batches []scaling.Batch) (*controllerruntime.Result, error) {
	logging := parameter.Logger
	// coordination config map
	currentCCM := &corev1.ConfigMap{}
	expectedCCM, creationErr := createCCM(parameter, batches)
	if creationErr != nil {
		return nil, creationErr
	}

	// ensure that the CCM exists
	err := parameter.Client.Get(parameter.Ctx, parameter.Req.NamespacedName, currentCCM)
	if err != nil {
		if errors.IsNotFound(err) {
			if err = parameter.Client.Create(parameter.Ctx, expectedCCM); err != nil {
				logging.Error(err, "Failed to create ControlConfigMap")
				return nil, err
			}
			logging.Info("Created ControlConfigMap")
			return &controllerruntime.Result{Requeue: true}, nil
		}
		return nil, err
	}

	// update CCM if necessary
	if !equality.Semantic.DeepDerivative(expectedCCM.Data, currentCCM.Data) {
		logging.Info("Updating base config")
		err := parameter.Client.Update(parameter.Ctx, expectedCCM)
		if err != nil {
			logging.Error(err, "Failed to update base config")
			return &controllerruntime.Result{}, err
		}
		logging.Info("Updated base config")
		return &controllerruntime.Result{Requeue: true}, nil
	}
	return nil, nil
}

func createCCM(parameter Parameters, batches []scaling.Batch) (*corev1.ConfigMap, error) {
	renovate := parameter.RenovateCR
	renovateConfig := config.RenovateConfig{
		DryRun:        *renovate.Spec.RenovateAppConfig.DryRun,
		Onboarding:    *renovate.Spec.RenovateAppConfig.OnBoarding,
		PrHourlyLimit: renovate.Spec.RenovateAppConfig.PrHourlyLimit,
		//OnboardingConfig: renovate.Spec.RenovateAppConfig.OnBoardingConfig,
		AddLabels: renovate.Spec.RenovateAppConfig.AddLabels,
		Platform:  renovate.Spec.RenovateAppConfig.Platform.PlatformType,
		Endpoint:  renovate.Spec.RenovateAppConfig.Platform.Endpoint,
	}
	if renovate.Spec.SharedCache.Enabled && renovate.Spec.SharedCache.Type == v1alpha1.SharedCacheTypes_REDIS {
		renovateConfig.RedisUrl = renovate.Spec.SharedCache.RedisConfig.Url
	}

	baseConfig, err := json.Marshal(renovateConfig)
	if err != nil {
		return nil, err
	}

	batchesString, err := json.Marshal(batches)
	if err != nil {
		return nil, err
	}
	data := map[string]string{
		"base":    string(baseConfig),
		"batches": string(batchesString),
	}

	newConfigMap := &corev1.ConfigMap{
		ObjectMeta: v1.ObjectMeta{
			Name:      renovate.Name,
			Namespace: renovate.Namespace,
		},
		Data: data,
	}
	if err := controllerutil.SetControllerReference(&renovate, newConfigMap, parameter.Scheme); err != nil {
		return nil, err
	}
	return newConfigMap, nil
}
