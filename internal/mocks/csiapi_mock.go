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

package mocks

import (
	"context"
	"errors"
	"fmt"

	"github.com/container-storage-interface/spec/lib/go/csi"
	csiext "github.com/dell/dell-csi-extensions/podmon"
)

// CSIMock of csiapi.CSIApi
type CSIMock struct {
	InducedErrors struct {
		NotConnected                   bool
		ControllerUnpublishVolume      bool
		NodeUnpublishVolume            bool
		NodeUnstageVolume              bool
		ValidateVolumeHostConnectivity bool
		Close                          bool
		NodeUnpublishNFSShareNotFound  bool
		NodeUnstageNFSShareNotFound    bool
	}
	ValidateVolumeHostConnectivityResponse struct {
		Connected     bool
		IosInProgress bool
	}
}

// Connected is a mock implementation of csiapi.CSIApi.Connected
func (mock *CSIMock) Connected() bool {
	return !(mock.InducedErrors.NotConnected)
}

// Close is a mock implementation of csiapi.CSIApi.Close
func (mock *CSIMock) Close() error {
	if mock.InducedErrors.Close {
		return fmt.Errorf("induced error for Close")
	}
	return nil
}

// ControllerUnpublishVolume is a mock implementation of csiapi.CSIApi.ControllerUnpublishVolume
func (mock *CSIMock) ControllerUnpublishVolume(_ context.Context, _ *csi.ControllerUnpublishVolumeRequest) (*csi.ControllerUnpublishVolumeResponse, error) {
	rep := &csi.ControllerUnpublishVolumeResponse{}
	if mock.InducedErrors.ControllerUnpublishVolume {
		return rep, errors.New("ControllerUnpublishedVolume induced error")
	}
	return rep, nil
}

// NodeUnpublishVolume is a mock implementation of csiapi.CSIApi.NodeUnpublishVolume
func (mock *CSIMock) NodeUnpublishVolume(_ context.Context, _ *csi.NodeUnpublishVolumeRequest) (*csi.NodeUnpublishVolumeResponse, error) {
	rep := &csi.NodeUnpublishVolumeResponse{}
	if mock.InducedErrors.NodeUnpublishVolume {
		return rep, errors.New("NodeUnpublishedVolume induced error")
	}
	if mock.InducedErrors.NodeUnpublishNFSShareNotFound {
		return rep, errors.New("NFS Share for filesystem not found")
	}
	return rep, nil
}

// NodeUnstageVolume is a mock implementation of csiapi.CSIApi.NodeUnstageVolume
func (mock *CSIMock) NodeUnstageVolume(_ context.Context, _ *csi.NodeUnstageVolumeRequest) (*csi.NodeUnstageVolumeResponse, error) {
	rep := &csi.NodeUnstageVolumeResponse{}
	if mock.InducedErrors.NodeUnstageVolume {
		return rep, errors.New("NodeUnstageedVolume induced error")
	}
	if mock.InducedErrors.NodeUnstageNFSShareNotFound {
		return rep, errors.New("NFS Share for filesystem not found")
	}
	return rep, nil
}

// ValidateVolumeHostConnectivity is a mock implementation of csiapi.CSIApi.ValidateVolumeHostConnectivity
func (mock *CSIMock) ValidateVolumeHostConnectivity(_ context.Context, _ *csiext.ValidateVolumeHostConnectivityRequest) (*csiext.ValidateVolumeHostConnectivityResponse, error) {
	rep := &csiext.ValidateVolumeHostConnectivityResponse{}
	if mock.InducedErrors.ValidateVolumeHostConnectivity {
		return rep, errors.New("ValidateVolumeHostConnectivity induced error")
	}
	rep.Connected = mock.ValidateVolumeHostConnectivityResponse.Connected
	rep.IosInProgress = mock.ValidateVolumeHostConnectivityResponse.IosInProgress
	return rep, nil
}
