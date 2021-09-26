package config

import (
	"fmt"
	"os"
)

type config struct {
	Name           string
	Namespace      string
	FilePath       string
	KubeConfigPath string
}

const (
	EnvRenovateCrName      = "SHIPPER_RENOVATE_CR_NAME"
	EnvRenovateCrNamespace = "SHIPPER_RENOVATE_CR_NAMESPACE"
	EnvRenovateOutputFile  = "SHIPPER_RENOVATE_OUTPUT_FILE"
	EnvKubeConfig          = "KUBECONFIG"
)

func GetConfig() (*config, error) {
	aConfig := &config{}

	err := fmt.Errorf("") // initialize so we can overwrite
	if aConfig.Name, err = setEnvVariable(EnvRenovateCrName); err != nil {
		return aConfig, err
	}
	if aConfig.Namespace, err = setEnvVariable(EnvRenovateCrNamespace); err != nil {
		return aConfig, err
	}
	if aConfig.FilePath, err = setEnvVariable(EnvRenovateOutputFile); err != nil {
		return aConfig, err
	}

	//optionals
	aConfig.KubeConfigPath, _ = setEnvVariable(EnvKubeConfig)

	return aConfig, nil
}

func setEnvVariable(envVariable string) (string, error) {
	if value, isSet := os.LookupEnv(envVariable); isSet {
		return value, nil
	} else {
		return "", fmt.Errorf("environment variable %s is not defined", envVariable)
	}
}
