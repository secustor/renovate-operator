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

type ScalingStrategy string

const (
	//ScalingStrategy_NONE A single batch be created and no parallelization will take place
	ScalingStrategy_NONE = "none"
	//ScalingStrategy_SIZE Create batches based on number of repositories. If 30 repositories have been found and size
	//is defined as 10, then 3 batches will be created.
	ScalingStrategy_SIZE = "size"
	//ScalingStrategy_FILTERS   = "filters"
	//ScalingStrategy_AUTOMATIC = "automatic"
)

type ScalingSpec struct {
	//+kubebuilder:validation:Enum=none;size
	//+kubebuilder:default:="none"
	ScalingStrategy ScalingStrategy `json:"strategy,omitempty"`

	//MaxWorkers Maximum number of parallel workers to start. A single worker will only process a single batch at maximum
	//+kubebuilder:default:=1
	MaxWorkers int32 `json:"maxWorkers"`

	//Size if ScalingStrategy
	Size int `json:"size,omitempty"`
}

type Platform struct {
	PlatformType PlatformTypes   `json:"type"`
	Endpoint     string          `json:"endpoint"`
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

	//OnBoardingConfig object `json:"onBoardingConfig,omitempty,inline"`

	//+kubebuilder:default:=10
	PrHourlyLimit int `json:"prHourlyLimit,omitempty"`

	AddLabels []string `json:"addLabels,omitempty"`

	GithubTokenSelector v1.EnvVarSource `json:"githubToken,omitempty"`
}

type RenovateDiscoveryConfig struct {
	//+kubebuilder:validation:Optional
	//+kubebuilder:default:="0 */2 * * *"
	Schedule string `json:"schedule"`
}

// RenovateSpec defines the desired state of Renovate
type RenovateSpec struct {
	RenovateAppConfig RenovateAppConfig `json:"renovate"`

	RenovateDiscoveryConfig RenovateDiscoveryConfig `json:"discovery,omitempty"`

	//+kubebuilder:validation:Optional
	//+kubebuilder:default:=false
	Suspend *bool `json:"suspend"`

	Schedule string `json:"schedule"`

	Logging LoggingSettings `json:"logging,omitempty"`

	//+kubebuilder:validation:Optional
	SharedCache SharedCache `json:"sharedCache"`

	//+kubebuilder:validation:Optional
	ScalingSpec ScalingSpec `json:"scaling"`
	// TODO add imageOverride
}

type RepositoryPath string

// RenovateStatus defines the observed state of Renovate
type RenovateStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	DiscoveredRepositories []RepositoryPath `json:"discoveredDepositories"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:shortName=ren
//+kubebuilder:printcolumn:name="Suspended",type=boolean,JSONPath=`.spec.suspend`
//+kubebuilder:printcolumn:name="DryRun",type=boolean,JSONPath=`.spec.renovate.dryRun`
//+kubebuilder:printcolumn:name="Version",type=string,JSONPath=`.spec.renovate.version`
//Renovate crd object
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
