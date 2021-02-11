package csiapi

import (
	"context"
	csi "github.com/container-storage-interface/spec/lib/go/csi"
	csiext "github.com/dell/dell-csi-extensions/podmon"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"time"
)

type CSIApiStruct struct {
	DriverConn       *grpc.ClientConn     // A grpc client connection to the driver
	PodmonClient     csiext.PodmonClient  // A grpc CSIPodmonClient
	ControllerClient csi.ControllerClient // A grpc CSI ControllerClient
	NodeClient       csi.NodeClient       // A grpc CSI NodeClient
}

var CSIClient CSIApiStruct
var CSIClientDialRetry = 30 * time.Second

func NewCSIClient(csiSock string, clientOpts ...grpc.DialOption) (CSIApi, error) {
	var err error
	for {
		// Wait on the driver. It will not open its unix socket until it has become leader.
		CSIClient.DriverConn, err = grpc.DialContext(context.Background(), csiSock, clientOpts...)
		log.Debugf("grpc.Dial returned %v %v", CSIClient.DriverConn, err)
		if err != nil || CSIClient.DriverConn == nil {
			var errMsg string
			if err == nil {
				errMsg = "No error returned, but CSIClient.DriverConn is nil"
			} else {
				errMsg = err.Error()
			}
			log.Errorf("Waiting on connection to driver csi.sock: %s", errMsg)
			time.Sleep(CSIClientDialRetry)
		} else {
			break
		}
	}
	log.Infof("Connected to driver: %s", csiSock)
	CSIClient.PodmonClient = csiext.NewPodmonClient(CSIClient.DriverConn)
	CSIClient.ControllerClient = csi.NewControllerClient(CSIClient.DriverConn)
	CSIClient.NodeClient = csi.NewNodeClient(CSIClient.DriverConn)
	return &CSIClient, nil
}

func (csi *CSIApiStruct) Connected() bool {
	return csi.DriverConn != nil
}

func (csi *CSIApiStruct) Close() error {
	if csi.Connected() {
		return csi.DriverConn.Close()
	}
	return nil
}

func (csi *CSIApiStruct) ControllerUnpublishVolume(ctx context.Context, req *csi.ControllerUnpublishVolumeRequest) (*csi.ControllerUnpublishVolumeResponse, error) {
	return CSIClient.ControllerClient.ControllerUnpublishVolume(ctx, req)
}

func (csi *CSIApiStruct) NodeUnpublishVolume(ctx context.Context, req *csi.NodeUnpublishVolumeRequest) (*csi.NodeUnpublishVolumeResponse, error) {
	return CSIClient.NodeClient.NodeUnpublishVolume(ctx, req)
}

func (csi *CSIApiStruct) NodeUnstageVolume(ctx context.Context, req *csi.NodeUnstageVolumeRequest) (*csi.NodeUnstageVolumeResponse, error) {
	return CSIClient.NodeClient.NodeUnstageVolume(ctx, req)
}

func (csi *CSIApiStruct) ValidateVolumeHostConnectivity(ctx context.Context, req *csiext.ValidateVolumeHostConnectivityRequest) (*csiext.ValidateVolumeHostConnectivityResponse, error) {
	return CSIClient.PodmonClient.ValidateVolumeHostConnectivity(ctx, req)
}
