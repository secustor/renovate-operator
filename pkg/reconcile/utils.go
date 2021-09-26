package reconcile

import (
	renovatev1alpha1 "github.com/secustor/renovate-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

func renovateContainer(renovate renovatev1alpha1.Renovate, additionalEnVars []corev1.EnvVar, additionalArgs []string) corev1.Container {
	return corev1.Container{
		Name:  "renovate",
		Image: renovateContainerImage(renovate),
		Args:  additionalArgs,
		Env:   append(renovateEnvVars(renovate), additionalEnVars...),
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      VolumeWorkDir,
				MountPath: DirRenovateBase,
			},
			{
				Name:      VolumeConfig,
				ReadOnly:  true,
				MountPath: DirRenovateConfig,
			},
		},
	}
}

func renovateEnvVars(renovate renovatev1alpha1.Renovate) []corev1.EnvVar {
	containerVars := []corev1.EnvVar{
		{
			Name:  "LOG_LEVEL",
			Value: string(renovate.Spec.Logging.Level),
		},
		{
			Name:  "RENOVATE_BASE_DIR",
			Value: DirRenovateBase,
		},
		{
			Name:  "RENOVATE_CONFIG_FILE",
			Value: FileRenovateConfig,
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
	return containerVars
}

func renovateContainerImage(renovate renovatev1alpha1.Renovate) string {
	return "renovate/renovate:" + renovate.Spec.RenovateAppConfig.RenovateVersion
}

func renovateStandardVolumes(renovate renovatev1alpha1.Renovate, volumeConfigVolumeSource corev1.VolumeSource) []corev1.Volume {
	return []corev1.Volume{
		{
			Name: VolumeWorkDir,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
		{
			Name:         VolumeConfig,
			VolumeSource: volumeConfigVolumeSource,
		},
	}
}
