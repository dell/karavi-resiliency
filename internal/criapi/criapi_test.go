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
