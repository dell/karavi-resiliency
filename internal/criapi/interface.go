/*
 * Copyright (c) 2021. Dell Inc., or its subsidiaries. All Rights Reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 */

package criapi

import (
	"context"
	v1 "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
)

// ContainerInfo is the information obtained for each container:
//   ID is the ContainerID that will match the ID in the Pod's container list.
//   Name is the name of the container.
//   State is the ContainerState.
type ContainerInfo struct {
	ID    string
	Name  string
	State v1.ContainerState
}

// CRIAPI is an interface for retrieving information about containers using the Container Runtime Interface
// that crictl uses.
type CRIAPI interface {
	Connected() bool
	Close() error
	ListContainers(ctx context.Context, req *v1.ListContainersRequest) (*v1.ListContainersResponse, error)
	GetContainerInfo(ctx context.Context) (map[string]*ContainerInfo, error)
}
