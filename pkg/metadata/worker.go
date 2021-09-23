package metadata

import (
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

func WorkerMetaData(request ctrl.Request) v1.ObjectMeta {
	return v1.ObjectMeta{
		Name:      WorkerName(request),
		Namespace: request.Namespace,
	}
}

func WorkerName(request ctrl.Request) string {
	return buildName(request.Name, workerGroupName)
}
