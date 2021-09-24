package config

import (
	"fmt"
	"os"
)

type config struct {
	Name      string
	Namespace string
	FilePath  string
}

const (
	EnvRenovateCrName      = "SHIPPER_RENOVATE_CR_NAME"
	EnvRenovateCrNamespace = "SHIPPER_RENOVATE_CR_NAMESPACE"
	EnvRenovateOutputFile  = "SHIPPER_RENOVATE_OUTPUT_FILE"
)

func GetConfig() (*config, error) {
	aConfig := &config{}

	if err := setEnvVariable(aConfig, EnvRenovateCrName); err != nil {
		return aConfig, err
	}
	if err := setEnvVariable(aConfig, EnvRenovateCrNamespace); err != nil {
		return aConfig, err
	}
	if err := setEnvVariable(aConfig, EnvRenovateOutputFile); err != nil {
		return aConfig, err
	}
	return aConfig, nil
}

func setEnvVariable(aConfig *config, envVariable string) error {
	if value, isSet := os.LookupEnv(envVariable); isSet {
		aConfig.Name = value
	} else {
		return fmt.Errorf("environment variable %s is not defined", envVariable)
	}
	return nil
}
