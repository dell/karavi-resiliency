package criapi

import (
	"context"
	v1 "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
)

type ContainerInfo struct {
	ID    string
	Name  string
	State v1.ContainerState
}

type CRIAPI interface {
	Connected() bool
	Close() error
	ListContainers(ctx context.Context, req *v1.ListContainersRequest) (*v1.ListContainersResponse, error)
	GetContainerInfo(ctx context.Context) (map[string]*ContainerInfo, error)
}
