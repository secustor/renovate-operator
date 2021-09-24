package shipper

import (
	"context"
	"os"

	renovatev1alpha1 "github.com/secustor/renovate-operator/api/v1alpha1"
	"github.com/secustor/renovate-operator/cmd/shipper/config"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

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
		logging.Error(err, "Failed to read Kubernetes client config")
		panic(err.Error())
	}

	cl, err := client.New(kubeConfig, client.Options{})
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

	var repositories []renovatev1alpha1.RepositoryPath
	err = json.Unmarshal(readBytes, repositories)
	if err != nil {
		logging.Error(err, "Failed to unmarshal json")
		panic(err.Error())
	}

	renovateCR.Status.DiscoveredRepositories = repositories
	err = cl.Status().Update(ctx, renovateCR)
	if err != nil {
		logging.Error(err, "Failed to update status of Renovate instance")
		panic(err.Error())
	}
}
