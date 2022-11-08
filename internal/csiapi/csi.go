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

package csiapi

import (
	"context"
	"time"

	csi "github.com/container-storage-interface/spec/lib/go/csi"
	csiext "github.com/dell/dell-csi-extensions/podmon"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

//Client holds clients related to CSI access
type Client struct {
	DriverConn       *grpc.ClientConn     // A grpc client connection to the driver
	PodmonClient     csiext.PodmonClient  // A grpc CSIPodmonClient
	ControllerClient csi.ControllerClient // A grpc CSI ControllerClient
	NodeClient       csi.NodeClient       // A grpc CSI NodeClient
}

//CSIClient is reference to CSI Client
var CSIClient Client

//CSIClientDialRetry is timeout after failure to connect to the CSI Driver
var CSIClientDialRetry = 30 * time.Second

//NewCSIClient returns a new CSIApi interface
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

//Connected returns true if there is non-nil driver connection
func (csi *Client) Connected() bool {
	return csi.DriverConn != nil
}

//Close will close connections on the driver connection, if it exists
func (csi *Client) Close() error {
	if csi.Connected() {
		return csi.DriverConn.Close()
	}
	return nil
}

//ControllerUnpublishVolume calls the UnpublishVolume in the controller
func (csi *Client) ControllerUnpublishVolume(ctx context.Context, req *csi.ControllerUnpublishVolumeRequest) (*csi.ControllerUnpublishVolumeResponse, error) {
	return CSIClient.ControllerClient.ControllerUnpublishVolume(ctx, req)
}

//NodeUnpublishVolume calls the UnpublishVolume in the node
func (csi *Client) NodeUnpublishVolume(ctx context.Context, req *csi.NodeUnpublishVolumeRequest) (*csi.NodeUnpublishVolumeResponse, error) {
	return CSIClient.NodeClient.NodeUnpublishVolume(ctx, req)
}

//NodeUnstageVolume calls UnstageVolume in the node
func (csi *Client) NodeUnstageVolume(ctx context.Context, req *csi.NodeUnstageVolumeRequest) (*csi.NodeUnstageVolumeResponse, error) {
	return CSIClient.NodeClient.NodeUnstageVolume(ctx, req)
}

//ValidateVolumeHostConnectivity calls the ValidateVolumeHostConnectivity in the podmon client
func (csi *Client) ValidateVolumeHostConnectivity(ctx context.Context, req *csiext.ValidateVolumeHostConnectivityRequest) (*csiext.ValidateVolumeHostConnectivityResponse, error) {
	return CSIClient.PodmonClient.ValidateVolumeHostConnectivity(ctx, req)
}
