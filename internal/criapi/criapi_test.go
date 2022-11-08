//go:build k8s
// +build k8s

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

package criapi

import (
	"context"
	"fmt"
	"k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
	"testing"
)

func TestListContainers(t *testing.T) {
	client, err := NewCRIClient("unix:/var/run/dockershim.sock")
	if err != nil {
		t.Errorf("NewCRIClient: %s", err)
	}
	req := &v1alpha2.ListContainersRequest{}
	rep, err := client.ListContainers(context.Background(), req)
	if err != nil {
		t.Errorf("ListContainers: %s", err)
	} else {
		for _, cont := range rep.Containers {
			fmt.Printf("container Id %s Name %s State %s\n", cont.Id, cont.Metadata.Name, cont.State)
		}
	}
	err = client.Close()
	if err != nil {
		t.Errorf("Close: %s", err)
	}
}

func TestGetContainerInfo(t *testing.T) {
	infoMap, err := CRIClient.GetContainerInfo(context.Background())
	if err != nil {
		t.Errorf("GetContainerInfo failed: %s", err)
	}
	for key, value := range infoMap {
		if key != value.ID {
			t.Error("key != value.ID")
		}
		fmt.Printf("ContainerInfo %+v\n", *value)
	}
}
