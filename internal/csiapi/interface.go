package csiapi

import (
	"context"
	csi "github.com/container-storage-interface/spec/lib/go/csi"
	csiext "github.com/dell/dell-csi-extensions/podmon"
)

type CSIApi interface {
	// Returns if the podmon is connected to the driver
	Connected() bool
	Close() error
	ControllerUnpublishVolume(context.Context, *csi.ControllerUnpublishVolumeRequest) (*csi.ControllerUnpublishVolumeResponse, error)
	NodeUnstageVolume(context.Context, *csi.NodeUnstageVolumeRequest) (*csi.NodeUnstageVolumeResponse, error)
	NodeUnpublishVolume(context.Context, *csi.NodeUnpublishVolumeRequest) (*csi.NodeUnpublishVolumeResponse, error)
	ValidateVolumeHostConnectivity(context.Context, *csiext.ValidateVolumeHostConnectivityRequest) (*csiext.ValidateVolumeHostConnectivityResponse, error)
}
