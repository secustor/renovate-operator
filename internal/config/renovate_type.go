package config

import "github.com/secustor/renovate-operator/api/v1alpha1"

type RenovateConfig struct {
	Onboarding    bool   `json:"onboarding"`
	PrHourlyLimit int    `json:"prHourlyLimit"`
	DryRun        bool   `json:"dryRun"`
	RedisUrl      string `json:"redisUrl"`
	//OnboardingConfig string        `json:"onboardingConfig,inline"`
	Platform  v1alpha1.PlatformTypes `json:"platform"`
	Endpoint  string                 `json:"endpoint"`
	AddLabels []string               `json:"addLabels"`
}
