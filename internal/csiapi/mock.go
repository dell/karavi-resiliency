package csiapi

import (
	"context"
	"errors"
	"fmt"
	"github.com/container-storage-interface/spec/lib/go/csi"
	csiext "github.com/dell/dell-csi-extensions/podmon"
)

//CSIMock of csiapi.CSIApi
type CSIMock struct {
	InducedErrors struct {
		NotConnected                   bool
		ControllerUnpublishVolume      bool
		NodeUnpublishVolume            bool
		NodeUnstageVolume              bool
		ValidateVolumeHostConnectivity bool
		Close                          bool
	}
	ValidateVolumeHostConnectivityResponse struct {
		Connected     bool
		IosInProgress bool
	}
}

//Connected is a mock implementation of csiapi.CSIApi.Connected
func (mock *CSIMock) Connected() bool {
	return !(mock.InducedErrors.NotConnected)
}

//Close is a mock implementation of csiapi.CSIApi.Close
func (mock *CSIMock) Close() error {
	if mock.InducedErrors.Close {
		return fmt.Errorf("induced error for Close")
	}
	return nil
}

//ControllerUnpublishVolume is a mock implementation of csiapi.CSIApi.ControllerUnpublishVolume
func (mock *CSIMock) ControllerUnpublishVolume(ctx context.Context, req *csi.ControllerUnpublishVolumeRequest) (*csi.ControllerUnpublishVolumeResponse, error) {
	rep := &csi.ControllerUnpublishVolumeResponse{}
	if mock.InducedErrors.ControllerUnpublishVolume {
		return rep, errors.New("ControllerUnpublishedVolume induced error")
	}
	return rep, nil
}

//NodeUnpublishVolume is a mock implementation of csiapi.CSIApi.NodeUnpublishVolume
func (mock *CSIMock) NodeUnpublishVolume(ctx context.Context, req *csi.NodeUnpublishVolumeRequest) (*csi.NodeUnpublishVolumeResponse, error) {
	rep := &csi.NodeUnpublishVolumeResponse{}
	if mock.InducedErrors.NodeUnpublishVolume {
		return rep, errors.New("NodeUnpublishedVolume induced error")
	}
	return rep, nil
}

//NodeUnstageVolume is a mock implementation of csiapi.CSIApi.NodeUnstageVolume
func (mock *CSIMock) NodeUnstageVolume(ctx context.Context, req *csi.NodeUnstageVolumeRequest) (*csi.NodeUnstageVolumeResponse, error) {
	rep := &csi.NodeUnstageVolumeResponse{}
	if mock.InducedErrors.NodeUnstageVolume {
		return rep, errors.New("NodeUnstageedVolume induced error")
	}
	return rep, nil
}

//ValidateVolumeHostConnectivity is a mock implementation of csiapi.CSIApi.ValidateVolumeHostConnectivity
func (mock *CSIMock) ValidateVolumeHostConnectivity(ctx context.Context, req *csiext.ValidateVolumeHostConnectivityRequest) (*csiext.ValidateVolumeHostConnectivityResponse, error) {
	rep := &csiext.ValidateVolumeHostConnectivityResponse{}
	if mock.InducedErrors.ValidateVolumeHostConnectivity {
		return rep, errors.New("ValidateVolumeHostConnectivity induced error")
	}
	rep.Connected = mock.ValidateVolumeHostConnectivityResponse.Connected
	rep.IosInProgress = mock.ValidateVolumeHostConnectivityResponse.IosInProgress
	return rep, nil
}
