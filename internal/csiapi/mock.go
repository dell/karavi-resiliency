package csiapi

import (
	"context"
	"errors"
	"fmt"
	"github.com/container-storage-interface/spec/lib/go/csi"
	csiext "github.com/dell/dell-csi-extensions/podmon"
)

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

func (mock *CSIMock) Connected() bool {
	if mock.InducedErrors.NotConnected {
		return false
	}
	return true
}

func (mock *CSIMock) Close() error {
	if mock.InducedErrors.Close {
		return fmt.Errorf("induced error for Close")
	}
	return nil
}

func (mock *CSIMock) ControllerUnpublishVolume(ctx context.Context, req *csi.ControllerUnpublishVolumeRequest) (*csi.ControllerUnpublishVolumeResponse, error) {
	rep := &csi.ControllerUnpublishVolumeResponse{}
	if mock.InducedErrors.ControllerUnpublishVolume {
		return rep, errors.New("ControllerUnpublishedVolume induced error")
	}
	return rep, nil
}

func (mock *CSIMock) NodeUnpublishVolume(ctx context.Context, req *csi.NodeUnpublishVolumeRequest) (*csi.NodeUnpublishVolumeResponse, error) {
	rep := &csi.NodeUnpublishVolumeResponse{}
	if mock.InducedErrors.NodeUnpublishVolume {
		return rep, errors.New("NodeUnpublishedVolume induced error")
	}
	return rep, nil
}

func (mock *CSIMock) NodeUnstageVolume(ctx context.Context, req *csi.NodeUnstageVolumeRequest) (*csi.NodeUnstageVolumeResponse, error) {
	rep := &csi.NodeUnstageVolumeResponse{}
	if mock.InducedErrors.NodeUnstageVolume {
		return rep, errors.New("NodeUnstageedVolume induced error")
	}
	return rep, nil
}

func (mock *CSIMock) ValidateVolumeHostConnectivity(ctx context.Context, req *csiext.ValidateVolumeHostConnectivityRequest) (*csiext.ValidateVolumeHostConnectivityResponse, error) {
	rep := &csiext.ValidateVolumeHostConnectivityResponse{}
	if mock.InducedErrors.ValidateVolumeHostConnectivity {
		return rep, errors.New("ValidateVolumeHostConnectivity induced error")
	}
	rep.Connected = mock.ValidateVolumeHostConnectivityResponse.Connected
	rep.IosInProgress = mock.ValidateVolumeHostConnectivityResponse.IosInProgress
	return rep, nil
}
