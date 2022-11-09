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
	"errors"
	"fmt"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
)

// Client represents the client grpc connection to the ContainerRuntimerInterface
type Client struct {
	CRIConn              *grpc.ClientConn              // A grpc client connection to CRI
	RuntimeServiceClient v1alpha2.RuntimeServiceClient // A RuntimeService climent
}

// CRIClient is an intstance of the Client for the CRI connection
var CRIClient Client

// CRIClientDialRetry is the amount of time to wait before retrying
var CRIClientDialRetry = 30 * time.Second

// CRIMaxConnectionRetry is the maximum number of connection retries.
var CRIMaxConnectionRetry = 3

// CRINewClientTimeout is the timeout for making a new client.
var CRINewClientTimeout = 90 * time.Second

// NewCRIClient returns a new client connection to the ContainerRuntimeInterface or an error
func NewCRIClient(criSock string, clientOpts ...grpc.DialOption) (*Client, error) {
	var err error
	ctx, cancel := context.WithTimeout(context.Background(), CRINewClientTimeout)
	defer cancel()
	for i := 0; i < CRIMaxConnectionRetry; i++ {
		CRIClient.CRIConn, err = grpc.DialContext(ctx, criSock, grpc.WithInsecure())
		if err != nil || CRIClient.CRIConn == nil {
			var errMsg string
			if err == nil {
				errMsg = "No error returned, but CRIClient.CRIConn is nil"
			} else {
				errMsg = err.Error()
			}
			log.Errorf("Waiting on connection to CRI socket: %s: %s", criSock, errMsg)
			time.Sleep(CRIClientDialRetry)
		} else {
			log.Infof("Connected to CRI: %s", criSock)
			CRIClient.RuntimeServiceClient = v1alpha2.NewRuntimeServiceClient(CRIClient.CRIConn)
			return &CRIClient, nil
		}
	}
	return &CRIClient, err
}

// Connected returns true if the CRI connection is up.
func (cri *Client) Connected() bool {
	return cri.CRIConn != nil
}

// Close closes the connection to the CRI.
func (cri *Client) Close() error {
	if cri.Connected() {
		if err := cri.CRIConn.Close(); err != nil {
			return err
		}
		cri.CRIConn = nil
		return nil
	}
	return nil
}

// ListContainers lists all the containers in the Container Runtime.
func (cri *Client) ListContainers(ctx context.Context, req *v1alpha2.ListContainersRequest) (*v1alpha2.ListContainersResponse, error) {
	return CRIClient.RuntimeServiceClient.ListContainers(ctx, req)
}

var knownPaths [3]string = [3]string{"/var/run/dockershim.sock", "/run/containerd/containerd.sock", "/run/crio/crio.sock"}

// ChooseCRIPath chooses an appropriate unix domain socket path to the CRI interface.
// This is done according to the ordering described for the crictl command.
func (cri *Client) ChooseCRIPath() (string, error) {
	for _, path := range knownPaths {
		_, err := os.Stat(path)
		if err == nil {
			retval := fmt.Sprintf("unix:///%s", path)
			return retval, nil
		}
	}
	return "", errors.New("Could not find path for CRI runtime from knownPaths")
}

// GetContainerInfo gets current status of all the containers on this server using CRI interface.
// The result is a map of ID to a structure containing the ID, Name, and State.
func (cri *Client) GetContainerInfo(ctx context.Context) (map[string]*ContainerInfo, error) {
	result := make(map[string]*ContainerInfo)

	path, err := cri.ChooseCRIPath()
	if err != nil {
		return result, err
	}
	client, err := NewCRIClient(path)
	if err != nil {
		return result, err
	}
	req := &v1alpha2.ListContainersRequest{}
	rep, err := client.ListContainers(context.Background(), req)
	if err != nil {
		return result, err
	}
	for _, cont := range rep.Containers {
		info := &ContainerInfo{
			ID:    cont.Id,
			Name:  cont.Metadata.Name,
			State: cont.State,
		}
		result[cont.Id] = info
	}
	err = client.Close()
	if err != nil {
		log.Infof("close error: %s", err)
	}
	return result, nil
}
