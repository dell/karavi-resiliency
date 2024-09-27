/*
* Copyright (c) 2021-2023 Dell Inc., or its subsidiaries. All Rights Reserved.
*
* Licensed under the Apache License, Version 2.0 (the "License");
* you may not use this file except in compliance with the License.
* You may obtain a copy of the License at
*
* http://www.apache.org/licenses/LICENSE-2.0
*
* Unless required by applicable law or agreed to in writing, software
* distributed under the License is distributed on an "AS IS" BASIS,
* WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
* See the License for the specific language governing permissions and
* limitations under the License.
 */

package criapi

import (
	"context"

	v1 "k8s.io/cri-api/pkg/apis/runtime/v1"
)

// ContainerInfo is the information obtained for each container:
//
//	ID is the ContainerID that will match the ID in the Pod's container list.
//	Name is the name of the container.
//	State is the ContainerState.
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
