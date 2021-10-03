package main

import (
	"context"
	"fmt"
	"os"

	renovatev1alpha1 "github.com/secustor/renovate-operator/api/v1alpha1"
	"github.com/secustor/renovate-operator/cmd/shipper/config"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(renovatev1alpha1.AddToScheme(scheme))
}

func main() {
	ctx := context.Background()
	logging := log.FromContext(ctx)

	shipperConfig, err := config.GetConfig()
	if err != nil {
		logging.Error(err, "Failed to get shipper configuration")
		panic(err.Error())
	}

	logging = logging.WithValues("namespace", shipperConfig.Namespace, "name", shipperConfig.Name)

	renovateCrName := types.NamespacedName{
		Namespace: shipperConfig.Namespace,
		Name:      shipperConfig.Name,
	}

	kubeConfig, err := rest.InClusterConfig()
	if err != nil {
		logging.Error(err, "Failed to initialize cluster internal kubeconfig")
		if shipperConfig.KubeConfigPath != "" {
			externalKubeConfigErr := fmt.Errorf("")
			kubeConfig, externalKubeConfigErr = clientcmd.BuildConfigFromFlags("", shipperConfig.KubeConfigPath)
			if externalKubeConfigErr != nil {
				logging.Error(externalKubeConfigErr, "Failed to read defined kubeconfig")
				panic(externalKubeConfigErr.Error())
			}
		} else {
			logging.Error(err, "Failed to read internal Kubernetes client config and outside kubeconfig defined")
			panic(err.Error())
		}
	}

	cl, err := client.New(kubeConfig, client.Options{
		Scheme: scheme,
	})
	if err != nil {
		logging.Error(err, "Failed to create Kubernetes client")
		panic(err.Error())
	}

	renovateCR := &renovatev1alpha1.Renovate{}
	err = cl.Get(ctx, renovateCrName, renovateCR)
	if err != nil {
		logging.Error(err, "Failed to retrieve Renovate instance")
		panic(err.Error())
	}

	readBytes, err := os.ReadFile(shipperConfig.FilePath)
	if err != nil {
		logging.Error(err, "Failed to read file", "file", shipperConfig.FilePath)
		panic(err.Error())
	}

	repositories := &[]renovatev1alpha1.RepositoryPath{}
	err = json.Unmarshal(readBytes, repositories)
	if err != nil {
		logging.Error(err, "Failed to unmarshal json")
		panic(err.Error())
	}

	renovateCR.Status.DiscoveredRepositories = *repositories
	err = cl.Status().Update(ctx, renovateCR)
	if err != nil {
		logging.Error(err, "Failed to update status of Renovate instance")
		panic(err.Error())
	}
}
