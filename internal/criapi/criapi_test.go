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
	"net"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
	v1 "k8s.io/cri-api/pkg/apis/runtime/v1"
)

// Mocking the grpc.DialContext function
var dialContextMock = func(ctx context.Context, target string, _ ...grpc.DialOption) (*grpc.ClientConn, error) {
	_ = target // Explicitly ignore the unused parameter
	lis := bufconn.Listen(1024 * 1024)
	s := grpc.NewServer()
	v1.RegisterRuntimeServiceServer(s, &mockRuntimeServiceServer{})
	go func() {
		if err := s.Serve(lis); err != nil {
			panic(err)
		}
	}()
	return grpc.DialContext(ctx, "", grpc.WithContextDialer(bufconnDialer(lis)), grpc.WithInsecure())
}

func bufconnDialer(lis *bufconn.Listener) func(context.Context, string) (net.Conn, error) {
	return func(context.Context, string) (net.Conn, error) {
		return lis.Dial()
	}
}

type mockRuntimeServiceServer struct {
	v1.UnimplementedRuntimeServiceServer
}

func (s *mockRuntimeServiceServer) ListContainers(_ context.Context, _ *v1.ListContainersRequest) (*v1.ListContainersResponse, error) {
	// Return a mock response
	return &v1.ListContainersResponse{
		Containers: []*v1.Container{
			{
				Id: "test-container-id",
				Metadata: &v1.ContainerMetadata{
					Name: "test-container",
				},
				State: v1.ContainerState_CONTAINER_RUNNING,
			},
		},
	}, nil
}

func TestGetGrpcDialContext(t *testing.T) {
	// Mock context and target
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	target := "unix:///var/run/dockershim.sock"

	// Call the function
	conn, err := getGrpcDialContext(ctx, target, grpc.WithInsecure())

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, conn)

	// Close the connection
	err = conn.Close()
	assert.NoError(t, err)
}

func TestNewCRIClient_Success(t *testing.T) {
	// Override the getGrpcDialContext function with mock
	getGrpcDialContext = dialContextMock

	client, err := NewCRIClient("unix:///var/run/dockershim.sock")
	assert.NoError(t, err)
	assert.NotNil(t, client)
	assert.NotNil(t, client.CRIConn)
	assert.NotNil(t, client.RuntimeServiceClient)
}

func TestNewCRIClient_Failure(t *testing.T) {
	// Override the getGrpcDialContext function to simulate failure
	getGrpcDialContext = func(_ context.Context, _ string, _ ...grpc.DialOption) (*grpc.ClientConn, error) {
		return nil, errors.New("failed to connect")
	}

	// Reduce retry count and sleep interval for the test
	originalRetry := CRIMaxConnectionRetry
	originalSleep := CRIClientDialRetry
	CRIMaxConnectionRetry = 2
	CRIClientDialRetry = 1 * time.Second
	defer func() {
		CRIMaxConnectionRetry = originalRetry
		CRIClientDialRetry = originalSleep
	}()

	_, err := NewCRIClient("unix:///var/run/dockershim.sock")
	assert.Error(t, err)
	//	assert.Nil(t, client)
}

func TestNewCRIClient_NilConnNoError(t *testing.T) {
	// Simulate retries with nil connection and no error
	retryCount := 0
	getGrpcDialContext = func(_ context.Context, _ string, _ ...grpc.DialOption) (*grpc.ClientConn, error) {
		retryCount++
		if retryCount < 2 {
			return nil, nil
		}
		return nil, errors.New("failed to connect after retries")
	}

	// Reduce retry count and sleep interval for the test
	originalRetry := CRIMaxConnectionRetry
	originalSleep := CRIClientDialRetry
	CRIMaxConnectionRetry = 2
	CRIClientDialRetry = 1 * time.Second
	defer func() {
		CRIMaxConnectionRetry = originalRetry
		CRIClientDialRetry = originalSleep
	}()

	_, err := NewCRIClient("unix:///var/run/dockershim.sock")
	assert.Error(t, err)
	//	assert.Nil(t, client)
}

func TestClient_Connected(t *testing.T) {
	client := &Client{}

	// Test when CRIConn is nil
	assert.False(t, client.Connected())

	// Test when CRIConn is not nil
	client.CRIConn = &grpc.ClientConn{}
	assert.True(t, client.Connected())
}

func TestClient_Close(t *testing.T) {
	client := &Client{}

	// Test when CRIConn is nil
	err := client.Close()
	assert.NoError(t, err)

	// Test when CRIConn is not nil
	lis := bufconn.Listen(1024 * 1024)
	conn, _ := grpc.DialContext(context.Background(), "", grpc.WithContextDialer(bufconnDialer(lis)), grpc.WithInsecure())
	client.CRIConn = conn

	err = client.Close()
	assert.NoError(t, err)
	assert.Nil(t, client.CRIConn)
}

func TestClient_ListContainers(t *testing.T) {
	// Mocking the grpc.DialContext function
	getGrpcDialContext = func(ctx context.Context, _ string, _ ...grpc.DialOption) (*grpc.ClientConn, error) {
		lis := bufconn.Listen(1024 * 1024)
		s := grpc.NewServer()
		v1.RegisterRuntimeServiceServer(s, &mockRuntimeServiceServer{})
		go func() {
			if err := s.Serve(lis); err != nil {
				panic(err)
			}
		}()
		return grpc.DialContext(ctx, "", grpc.WithContextDialer(bufconnDialer(lis)), grpc.WithInsecure())
	}

	client, err := NewCRIClient("unix:///var/run/dockershim.sock")
	assert.NoError(t, err)
	assert.NotNil(t, client)

	req := &v1.ListContainersRequest{}
	resp, err := client.ListContainers(context.Background(), req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Len(t, resp.Containers, 1)
	assert.Equal(t, "test-container-id", resp.Containers[0].Id)
	assert.Equal(t, "test-container", resp.Containers[0].Metadata.Name)
	assert.Equal(t, v1.ContainerState_CONTAINER_RUNNING, resp.Containers[0].State)
}

func TestChooseCRIPath_Success(t *testing.T) {
	// Override osStat with mockStat
	originalStat := osStat
	osStat = mockStat
	defer func() { osStat = originalStat }() // Restore original osStat after test

	client := &Client{}
	path, err := client.ChooseCRIPath()
	assert.NoError(t, err)
	assert.Equal(t, "unix:////var/run/dockershim.sock", path)
}

func TestChooseCRIPath_Failure(t *testing.T) {
	// Override osStat with a mock implementation that simulates all paths not existing
	originalStat := osStat
	osStat = func(_ string) (os.FileInfo, error) {
		return nil, os.ErrNotExist
	}
	defer func() { osStat = originalStat }() // Restore original osStat after test

	client := &Client{}
	path, err := client.ChooseCRIPath()
	assert.Error(t, err)
	assert.Equal(t, "", path)
	assert.Equal(t, "Could not find path for CRI runtime from knownPaths", err.Error())
}

var dialContextMockForGetContainerInfo = func(ctx context.Context, _ string, _ ...grpc.DialOption) (*grpc.ClientConn, error) {
	lis := bufconn.Listen(1024 * 1024)
	s := grpc.NewServer()
	v1.RegisterRuntimeServiceServer(s, &mockRuntimeServiceServer{})
	go func() {
		if err := s.Serve(lis); err != nil {
			panic(err)
		}
	}()
	return grpc.DialContext(ctx, "", grpc.WithContextDialer(bufconnDialer(lis)), grpc.WithInsecure())
}

func mockStat(path string) (os.FileInfo, error) {
	if path == "/var/run/dockershim.sock" || path == "/run/containerd/containerd.sock" || path == "/run/crio/crio.sock" {
		return nil, nil // Simulate that the file exists
	}
	return nil, os.ErrNotExist // Simulate that the file does not exist
}

func TestGetContainerInfo_Success(t *testing.T) {
	// Override osStat with mockStat
	originalStat := osStat
	osStat = mockStat
	defer func() { osStat = originalStat }() // Restore original osStat after test

	// Override the getGrpcDialContext function with our mock
	getGrpcDialContext = dialContextMockForGetContainerInfo

	client := &Client{}
	result, err := client.GetContainerInfo(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result, 1)
	assert.Equal(t, "test-container-id", result["test-container-id"].ID)
	assert.Equal(t, "test-container", result["test-container-id"].Name)
	assert.Equal(t, v1.ContainerState_CONTAINER_RUNNING, result["test-container-id"].State)
}

func TestGetContainerInfo_ChooseCRIPathFailure(t *testing.T) {
	// Override osStat with a mock implementation that simulates all paths not existing
	originalStat := osStat
	osStat = func(_ string) (os.FileInfo, error) {
		return nil, os.ErrNotExist
	}
	defer func() { osStat = originalStat }() // Restore original osStat after test

	client := &Client{}
	result, err := client.GetContainerInfo(context.Background())
	assert.Error(t, err)
	assert.Equal(t, "Could not find path for CRI runtime from knownPaths", err.Error())
	assert.Empty(t, result)
}
