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
	"reflect"
	"testing"
	"time"

	"google.golang.org/grpc"
	v1 "k8s.io/cri-api/pkg/apis/runtime/v1"
)

func TestNewCRIClient(t *testing.T) {
	tests := []struct {
		name    string
		criSock string
		wantErr bool
	}{
		{
			name:    "Valid connection",
			criSock: "unix:///var/run/dockershim.sock",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewCRIClient(tt.criSock)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewCRIClient() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewCRIClient_withMocking(t *testing.T) {
	copyGetGrpcDialContext := getGrpcDialContext
	copyCRIClientDialRetry := CRIClientDialRetry
	getGrpcDialContext = func(ctx context.Context, target string, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
		return nil, nil
	}
	CRIClientDialRetry = 1 * time.Second

	defer func() {
		getGrpcDialContext = copyGetGrpcDialContext
		CRIClientDialRetry = copyCRIClientDialRetry
	}()

	tests := []struct {
		name    string
		criSock string
		wantErr bool
	}{
		{
			name:    "Valid connection",
			criSock: "unix:///var/run/dockershim.sock",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewCRIClient(tt.criSock)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewCRIClient() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestClient_Connected(t *testing.T) {
	tests := []struct {
		name    string
		criConn *grpc.ClientConn
		want    bool
	}{
		{
			name:    "CRIConn is nil",
			criConn: nil,
			want:    false,
		},
		{
			name:    "CRIConn is not nil",
			criConn: &grpc.ClientConn{},
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cri := &Client{
				CRIConn: tt.criConn,
			}
			if got := cri.Connected(); got != tt.want {
				t.Errorf("Connected() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClient_Close(t *testing.T) {
	tests := []struct {
		name          string
		criConn       *grpc.ClientConn
		expectedError error
	}{
		{
			name:          "CRIConn is nil",
			criConn:       nil,
			expectedError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cri := &Client{
				CRIConn: tt.criConn,
			}
			if got := cri.Close(); got != tt.expectedError {
				t.Errorf("Close() = %v, want %v", got, tt.expectedError)
			}
		})
	}
}

func TestClient_ListContainers(t *testing.T) {
	tests := []struct {
		name          string
		criConn       *grpc.ClientConn
		expectedError error
		expectedRep   *v1.ListContainersResponse
	}{
		{
			name:          "CRIConn is nil",
			criConn:       nil,
			expectedError: errors.New("CRIConn is nil"),
			expectedRep:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cri := &Client{
				CRIConn: tt.criConn,
			}
			ctx := context.Background()
			req := &v1.ListContainersRequest{}
			rep, err := cri.ListContainers(ctx, req)
			if (err != nil) != (tt.expectedError != nil) {
				t.Errorf("ListContainers() error = %v, expectedError %v", err, tt.expectedError)
				return
			}
			if !reflect.DeepEqual(rep, tt.expectedRep) {
				t.Errorf("ListContainers() = %v, want %v", rep, tt.expectedRep)
			}
		})
	}
}

func TestClient_ChooseCRIPath(t *testing.T) {
	tests := []struct {
		name          string
		criConn       *grpc.ClientConn
		expectedPath  string
		expectedError error
	}{
		{
			name:          "CRIConn is nil",
			criConn:       nil,
			expectedPath:  "",
			expectedError: errors.New("Could not find path for CRI runtime from knownPaths"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cri := &Client{
				CRIConn: tt.criConn,
			}
			path, err := cri.ChooseCRIPath()
			if (err != nil) != (tt.expectedError != nil) {
				t.Errorf("ChooseCRIPath() error = %v, expectedError %v", err, tt.expectedError)
				return
			}
			if path != tt.expectedPath {
				t.Errorf("ChooseCRIPath() = %v, want %v", path, tt.expectedPath)
			}
		})
	}
}

func TestClient_GetContainerInfo(t *testing.T) {
	tests := []struct {
		name          string
		criConn       *grpc.ClientConn
		expectedError error
		expectedRep   map[string]*ContainerInfo
	}{
		{
			name:          "CRIConn is not nil",
			criConn:       &grpc.ClientConn{},
			expectedError: errors.New("Could not find path for CRI runtime from knownPaths"),
			expectedRep:   map[string]*ContainerInfo{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cri := &Client{
				CRIConn: tt.criConn,
			}
			ctx := context.Background()
			rep, err := cri.GetContainerInfo(ctx)
			if (err != nil) != (tt.expectedError != nil) {
				t.Errorf("GetContainerInfo() error = %v, expectedError %v", err, tt.expectedError)
				return
			}
			if !reflect.DeepEqual(rep, tt.expectedRep) {
				t.Errorf("GetContainerInfo() = %v, want %v", rep, tt.expectedRep)
			}
		})
	}
}
