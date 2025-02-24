/*
* Copyright (c) 2021-2022 Dell Inc., or its subsidiaries. All Rights Reserved.
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

package mocks

import (
	"context"
	"errors"
	"podmon/internal/criapi"
	v1 "k8s.io/cri-api/pkg/apis/runtime/v1"
)

// MockClient is a mock client supporting the criapi.
type MockClient struct {
	InducedErrors struct {
		GetContainerInfo bool
	}
	MockContainerInfos map[string]*criapi.ContainerInfo
}

// Initialize initializes the MockClient.
func (mock *MockClient) Initialize() {
	mock.MockContainerInfos = make(map[string]*criapi.ContainerInfo)
}

// Connected returns true if connected.
func (mock *MockClient) Connected() bool {
	return true
}

// Close closes the mock client. This is unimplemented for the mock client.
func (mock *MockClient) Close() error {
	return errors.New("unimplemented")
}

// ListContainers would list individual containers but is not implemented for the mock client.
func (mock *MockClient) ListContainers(_ context.Context, _ *v1.ListContainersRequest) (*v1.ListContainersResponse, error) {
	return nil, errors.New("unimplemented")
}

// ChooseCRIPath chooses an appropriate unix domain socket path to the CRI interface. This is unimplemented for the mock client.
func (mock *MockClient) ChooseCRIPath() (string, error) {
	return "", errors.New("unimplemented")
}

// GetContainerInfo gets current status of all the containers on this server using CRI interface.
// The result is a map of ID to a structure containing the ID, Name, and State.
func (mock *MockClient) GetContainerInfo(_ context.Context) (map[string]*criapi.ContainerInfo, error) {
	if mock.InducedErrors.GetContainerInfo {
		return mock.MockContainerInfos, errors.New("GetContainerInfo induced error")
	}
	return mock.MockContainerInfos, nil
}
