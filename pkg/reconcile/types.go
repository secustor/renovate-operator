package reconcile

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/go-logr/logr"
	"github.com/secustor/renovate-operator/api/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
)

type Parameters struct {
	RenovateCR v1alpha1.Renovate
	Client     client.Client
	Scheme     *runtime.Scheme
	Ctx        context.Context
	Req        ctrl.Request
	Logger     logr.Logger
}
