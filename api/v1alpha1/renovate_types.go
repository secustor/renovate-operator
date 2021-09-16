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

package v1alpha1

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

//type ScalingStrategy string
//
//const (
//	ScalingStrategy_NONE      = "none"
//	ScalingStrategy_FILTERS   = "filters"
//	ScalingStrategy_SIZE      = "size"
//	ScalingStrategy_AUTOMATIC = "automatic"
//)
//
//type Scaling struct {
//	ScalingStrategy ScalingStrategy `json:"strategy,omitempty"`
//	Filters         []string        `json:"filters,omitempty"`
//	Size            int             `json:"size,omitempty"`
//	// TODO add shared cache
//
//}

type Platform struct {
	PlatformType PlatformTypes   `json:"type"`
	Endpoint     string          `json:"endpoint,string"`
	Token        v1.EnvVarSource `json:"token"`
}

//+kubebuilder:validation:Enum=github;gitlab
type PlatformTypes string

const (
	PlatformType_GITHUB = "github"
	PlatformType_GITLAB = "gitlab"
	// TODO add additional platforms
)

//+kubebuilder:validation:Enum=trace;debug;info;warn;error;fatal
type LogLevel string

const (
	LogLevel_TRACE = "trace"
	LogLevel_DEBUG = "debug"
	LogLevel_INFO  = "info"
	LogLevel_WARN  = "warn"
	LogLevel_ERROR = "error"
	LogLevel_FATAL = "fatal"
)

type LoggingSettings struct {
	Level LogLevel `json:"level,omitempty"`
}

//+kubebuilder:validation:Enum=redis
type SharedCacheTypes string

const (
	SharedCacheTypes_REDIS = "redis"
	//TODO add RXWX volume option
)

type SharedCacheRedisConfig struct {
	Url string `json:"url"`
}

type SharedCache struct {
	Enabled bool `json:"enabled"`

	Type        SharedCacheTypes       `json:"type,omitempty"`
	RedisConfig SharedCacheRedisConfig `json:"redis,omitempty"`
}

type RenovateAppConfig struct {
	Platform Platform        `json:"platform"`
	Logging  LoggingSettings `json:"logging,omitempty"`

	//+kubebuilder:default:="27.7.0"
	RenovateVersion string `json:"version,omitempty"`

	//+kubebuilder:default:=false
	DryRun *bool `json:"dryRun,omitempty"`

	//+kubebuilder:default:=true
	OnBoarding *bool `json:"onBoarding,omitempty"`

	//+kubebuilder:default:="{"extends": ["config:base"]}"
	OnBoardingConfig map[string]interface{} `json:"onBoardingConfig,omitempty,inline"`

	//+kubebuilder:default:=10
	PrHourlyLimit int `json:"prHourlyLimit,omitempty"`

	AddLabels []string `json:"addLabels,omitempty"`

	GithubTokenSelector v1.EnvVarSource `json:"githubToken,omitempty"`
}

// RenovateSpec defines the desired state of Renovate
type RenovateSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	RenovateAppConfig RenovateAppConfig `json:"renovate"`

	//+kubebuilder:validation:Optional
	//+kubebuilder:default:=false
	Suspend *bool `json:"suspend"`

	Schedule string `json:"schedule"`

	//Scaling             Scaling              `json:"scaling,omitempty"`
	Logging LoggingSettings `json:"logging,omitempty"`

	//+kubebuilder:validation:Optional
	SharedCache SharedCache `json:"sharedCache"`
	// TODO add imageOverride
}

// RenovateStatus defines the observed state of Renovate
type RenovateStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Suspended",type=boolean,JSONPath=`.spec.suspend`
//+kubebuilder:printcolumn:name="DryRun",type=boolean,JSONPath=`.spec.dryRun`
//+kubebuilder:printcolumn:name="Version",type=string,JSONPath=`.spec.version`
type Renovate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RenovateSpec   `json:"spec,omitempty"`
	Status RenovateStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// RenovateList contains a list of Renovate
type RenovateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Renovate `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Renovate{}, &RenovateList{})
}

type RenovateConfig struct {
	Onboarding       bool                   `json:"onboarding"`
	PrHourlyLimit    int                    `json:"prHourlyLimit"`
	OnboardingConfig map[string]interface{} `json:"onboardingConfig,inline"`
	Platform         PlatformTypes          `json:"platform"`
	Endpoint         string                 `json:"endpoint"`
	AddLabels        []string               `json:"addLabels"`
}
