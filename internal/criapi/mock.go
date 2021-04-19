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
	"errors"
	"k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
)

type MockClient struct {
	InducedErrors struct {
		GetContainerInfo bool
	}
	MockContainerInfos map[string]*ContainerInfo
}

func (mock *MockClient) Initialize() {
	mock.MockContainerInfos = make(map[string]*ContainerInfo)
}


func (mock *MockClient) Connected() bool {
	return true
}

func (mock *MockClient) Close() error {
	return errors.New("unimplemented")
}

func (mock *MockClient) ListContainers(ctx context.Context, req *v1alpha2.ListContainersRequest) (*v1alpha2.ListContainersResponse, error) {
	return nil, errors.New("unimplemented")
}

// ChoseCRIPath chooses an appropriate unix domain socket path to the CRI interface.
func (mock *MockClient) ChooseCRIPath() (string, error) {
	return "", errors.New("unimplemented")
}

// GetContainerInfo gets current status of all the containers on this server using CRI interface.
// The result is a map of ID to a structure containing the ID, Name, and State.
func (mock *MockClient) GetContainerInfo(ctx context.Context) (map[string]*ContainerInfo, error) {
	if mock.InducedErrors.GetContainerInfo {
		return mock.MockContainerInfos, errors.New("GetContainerInfo induced error")
	}
	return mock.MockContainerInfos, nil
}
