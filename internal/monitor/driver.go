/*
 *
 * Copyright © 2021-2024 Dell Inc. or its subsidiaries. All Rights Reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

// Copyright © 2021-2023 Dell Inc. or its subsidiaries. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package monitor

import (
	"crypto/sha256"
	"fmt"
	"os"
	"podmon/internal/tools"
	"strings"

	log "github.com/sirupsen/logrus"
)

type drivertype interface {
	GetDriverName() string
	GetDriverMountDir(volumeHandle, pvName, podUUID string) string
	GetDriverBlockDev(volumeHandle, pvName, podUUID string) string
	GetStagingMountDir(volumeHandle, pvName string) string
	GetStagingMountDirAfter125(volumeHandle, pvName string) string
	GetStagingBlockDir(volumeHandle, pvName string) string
	NodeUnpublishExcludedError(err error) bool
	NodeUnstageExcludedError(err error) bool
	FinalCleanup(rawBlock bool, volumeHandle, pvName, podUUID string) error
}

// Driver is an instance of the drivertype interface to provide driver specific functions.
var Driver drivertype

// VxflexDriver provides a Driver instance for the PowerFlex (VxFlex) architecture.
type VxflexDriver struct{}

// GetDriverName returns the driver name string
func (d *VxflexDriver) GetDriverName() string {
	return "vxflexos"
}

// GetDriverMountDir returns the Vxflex private mount directory.
func (d *VxflexDriver) GetDriverMountDir(volumeHandle, _, _ string) string {
	privateMountDir := os.Getenv("X_CSI_PRIVATE_MOUNT_DIR")
	if privateMountDir == "" {
		privateMountDir = "/var/lib/kubelet/plugins/vxflexos.emc.dell.com/disks"
	}
	privateMountDir = fmt.Sprintf("%s/%s", privateMountDir, volumeHandle)
	log.Debugf("privateMountDir: %s", privateMountDir)
	return privateMountDir
}

// GetDriverBlockDev Returns the block device used for a PV by a pod.
func (d *VxflexDriver) GetDriverBlockDev(_, pvName, podUUID string) string {
	privateBlockDev := fmt.Sprintf("/var/lib/kubelet/plugins/kubernetes.io/csi/volumeDevices/publish/%s/%s", pvName, podUUID)
	log.Debugf("privateBlockDev: %s", privateBlockDev)
	return privateBlockDev
}

// GetStagingMountDir Returns the staging directory used by NodeUnstage for a mount device.
func (d *VxflexDriver) GetStagingMountDir(volumeHandle, _ string) string {
	stagingMountDir := os.Getenv("X_CSI_PRIVATE_MOUNT_DIR")
	if stagingMountDir == "" {
		stagingMountDir = "/var/lib/kubelet/plugins/vxflexos.emc.dell.com/disks"
	}
	stagingMountDir = fmt.Sprintf("%s/%s", stagingMountDir, volumeHandle)
	log.Debugf("stagingMountDir: %s", stagingMountDir)
	return stagingMountDir
}

// GetStagingMountDirAfter125 Returns the staging directory used by NodeUnstage for a mount device.
func (d *VxflexDriver) GetStagingMountDirAfter125(volumeHandle, _ string) string {
	result := sha256.Sum256([]byte(fmt.Sprintf("%s", volumeHandle)))
	volSha := fmt.Sprintf("%x", result)

	stagingMountDir := os.Getenv("X_CSI_PRIVATE_MOUNT_DIR")
	if stagingMountDir == "" {
		stagingMountDir = "/var/lib/kubelet/plugins/vxflexos.emc.dell.com/disks"
	}
	stagingMountDir = fmt.Sprintf("%s/%s", stagingMountDir, volSha)
	log.Debugf("stagingMountDev: %s", stagingMountDir)
	return stagingMountDir
}

// GetStagingBlockDir Returns the staging directory used by NodeUnstage for a block device.
func (d *VxflexDriver) GetStagingBlockDir(_, pvName string) string {
	stagingBlockDir := fmt.Sprintf("/var/lib/kubelet/plugins/kubernetes.io/csi/volumeDevices/staging/%s", pvName)
	log.Debugf("stagingBlockDir: %s", stagingBlockDir)
	return stagingBlockDir
}

// NodeUnpublishExcludedError filters out NodeUnpublish errors that should be excluded
func (d *VxflexDriver) NodeUnpublishExcludedError(_ error) bool {
	return false
}

// NodeUnstageExcludedError filters out NodeStage errors that should be excluded
func (d *VxflexDriver) NodeUnstageExcludedError(_ error) bool {
	return false
}

// FinalCleanup handles any driver specific final cleanup.
func (d *VxflexDriver) FinalCleanup(_ bool, _, _, _ string) error {
	return nil
}

// UnityDriver provides a Driver instance for the Unity architecture.
type UnityDriver struct{}

// GetDriverName returns the driver name string
func (d *UnityDriver) GetDriverName() string {
	return "unity"
}

// GetDriverMountDir returns the Unity private mount directory.
func (d *UnityDriver) GetDriverMountDir(_, pvName, podUUID string) string {
	privateMountDir := fmt.Sprintf("/var/lib/kubelet/pods/%s/volumes/kubernetes.io~csi/%s/mount", podUUID, pvName)
	log.Debugf("privateMountDir: %s", privateMountDir)
	return privateMountDir
}

// GetDriverBlockDev Returns the block device used for a PV by a pod.
func (d *UnityDriver) GetDriverBlockDev(_, pvName, podUUID string) string {
	privateBlockDev := fmt.Sprintf("/var/lib/kubelet/plugins/kubernetes.io/csi/volumeDevices/publish/%s/%s", pvName, podUUID)
	log.Debugf("privateBlockDev: %s", privateBlockDev)
	return privateBlockDev
}

// GetStagingMountDir Returns the staging directory used by NodeUnstage for a mount device.
func (d *UnityDriver) GetStagingMountDir(_, pvName string) string {
	stagingMountDir := fmt.Sprintf("/var/lib/kubelet/plugins/kubernetes.io/csi/pv/%s/globalmount", pvName)
	log.Debugf("stagingMountDev: %s", stagingMountDir)
	return stagingMountDir
}

// GetStagingMountDirAfter125 Returns the staging directory used by NodeUnstage for a mount device.
func (d *UnityDriver) GetStagingMountDirAfter125(volumeHandle, _ string) string {
	result := sha256.Sum256([]byte(fmt.Sprintf("%s", volumeHandle)))
	volSha := fmt.Sprintf("%x", result)

	stagingMountDir := fmt.Sprintf("/var/lib/kubelet/plugins/kubernetes.io/csi/csi-unity.dellemc.com/%s/globalmount", volSha)
	log.Debugf("stagingMountDev: %s", stagingMountDir)
	return stagingMountDir
}

// GetStagingBlockDir Returns the staging directory used by NodeUnstage for a block device.
func (d *UnityDriver) GetStagingBlockDir(_, pvName string) string {
	stagingBlockDir := fmt.Sprintf("/var/lib/kubelet/plugins/kubernetes.io/csi/volumeDevices/staging/%s", pvName)
	log.Debugf("stagingBlockDir: %s", stagingBlockDir)
	return stagingBlockDir
}

// NodeUnpublishExcludedError filters out NodeUnpublish errors that should be excluded
func (d *UnityDriver) NodeUnpublishExcludedError(err error) bool {
	if strings.Contains(err.Error(), "NFS Share for filesystem") && strings.Contains(err.Error(), "not found") {
		log.Infof("Ignored error: %s", err)
		return true
	}
	return false
}

// NodeUnstageExcludedError filters out NodeStage errors that should be excluded
func (d *UnityDriver) NodeUnstageExcludedError(err error) bool {
	if strings.Contains(err.Error(), "NFS Share for filesystem") && strings.Contains(err.Error(), "not found") {
		log.Infof("Ignored error: %s", err)
		return true
	}
	return false
}

// FinalCleanup handles any driver specific final cleanup.
func (d *UnityDriver) FinalCleanup(rawBlock bool, _, pvName, podUUID string) error {
	if rawBlock { // Do this cleanup on raw device block
		loopBackDev, err := getLoopBackDevice(pvName)
		if err != nil || loopBackDev == "" {
			// nothing to clean
			return err
		}

		_, err = deleteLoopBackDevice(loopBackDev)
		if err != nil {
			log.Infof("error deleting loopback device: %s", loopBackDev)
			return err
		}

		blockDev := fmt.Sprintf("/var/lib/kubelet/plugins/kubernetes.io/csi/volumeDevices/%s/dev/%s", pvName, podUUID)
		err = unMountPath(blockDev, 1)
		log.Infof("unmount block device in FinalCleanup path: %s error: %s", blockDev, err)

		err = RemoveDev(blockDev)
		log.Infof("remove block device FinalCleanup path: %s error: %s", blockDev, err)
	}
	return nil
}

var (
	getLoopBackDevice    = tools.GetLoopBackDevice
	deleteLoopBackDevice = tools.DeleteLoopBackDevice
	unMountPath          = tools.Unmount
)

// PScaleDriver provides a Driver instance for the PowerScale architecture.
type PScaleDriver struct{}

// GetDriverName returns the driver name string
func (d *PScaleDriver) GetDriverName() string {
	return "isilon"
}

// GetDriverMountDir returns the PowerScale private mount directory.
func (d *PScaleDriver) GetDriverMountDir(_, pvName, podUUID string) string {
	privateMountDir := os.Getenv("X_CSI_PRIVATE_MOUNT_DIR")
	if privateMountDir == "" {
		privateMountDir = "/var/lib/kubelet"
	}
	privateMountDir = fmt.Sprintf("%s/pods/%s/volumes/kubernetes.io~csi/%s/mount", privateMountDir, podUUID, pvName)
	log.Infof("privateMountDir: %s", privateMountDir)
	return privateMountDir
}

// GetDriverBlockDev Returns the block device used for a PV by a pod.
func (d *PScaleDriver) GetDriverBlockDev(_, _, _ string) string {
	return ""
}

// GetStagingMountDir Returns the staging directory used by NodeUnstage for a mount device.
func (d *PScaleDriver) GetStagingMountDir(_, _ string) string {
	return ""
}

// GetStagingMountDirAfter125 Returns the staging directory used by NodeUnstage for a mount device.
func (d *PScaleDriver) GetStagingMountDirAfter125(_, _ string) string {
	return ""
}

// GetStagingBlockDir Returns the staging directory used by NodeUnstage for a block device.
func (d *PScaleDriver) GetStagingBlockDir(_, _ string) string {
	return ""
}

// NodeUnpublishExcludedError filters out NodeUnpublish errors that should be excluded
func (d *PScaleDriver) NodeUnpublishExcludedError(_ error) bool {
	return false
}

// NodeUnstageExcludedError filters out NodeStage errors that should be excluded
func (d *PScaleDriver) NodeUnstageExcludedError(_ error) bool {
	return false
}

// FinalCleanup handles any driver specific final cleanup.
func (d *PScaleDriver) FinalCleanup(_ bool, _, _, _ string) error {
	return nil
}

// PStoreDriver provides a Driver instance for the Powerstore architecture.
type PStoreDriver struct{}

// GetDriverName returns the driver name string
func (d *PStoreDriver) GetDriverName() string {
	return "powerstore"
}

// GetDriverMountDir returns the mount directory used for a PV by a pod.
func (d *PStoreDriver) GetDriverMountDir(_, pvName, podUUID string) string {
	privateMountDir := getPrivateMountDir("/var/lib/kubelet")
	privateMountDir = fmt.Sprintf("%s/pods/%s/volumes/kubernetes.io~csi/%s/mount", privateMountDir, podUUID, pvName)
	log.Debugf("privateMountDir: %s", privateMountDir)
	return privateMountDir
}

// GetDriverBlockDev Returns the block device used for a PV by a pod.
func (d *PStoreDriver) GetDriverBlockDev(_, pvName, podUUID string) string {
	privateMountDir := getPrivateMountDir("/var/lib/kubelet")
	privateBlockDev := fmt.Sprintf("%s/plugins/kubernetes.io/csi/volumeDevices/publish/%s/%s", privateMountDir, pvName, podUUID)
	log.Debugf("privateBlockDev: %s", privateBlockDev)
	return privateBlockDev
}

// GetStagingMountDir Returns the staging directory used by NodeUnstage for a mount device.
func (d *PStoreDriver) GetStagingMountDir(_, pvName string) string {
	privateMountDir := getPrivateMountDir("/var/lib/kubelet")
	stagingMountDev := fmt.Sprintf("%s/plugins/kubernetes.io/csi/pv/%s/globalmount", privateMountDir, pvName)
	log.Debugf("stagingMountDev: %s", stagingMountDev)
	return stagingMountDev
}

// GetStagingMountDirAfter125 Returns the staging directory used by NodeUnstage for a mount device.
func (d *PStoreDriver) GetStagingMountDirAfter125(volumeHandle, _ string) string {
	result := sha256.Sum256([]byte(fmt.Sprintf("%s", volumeHandle)))
	volSha := fmt.Sprintf("%x", result)

	stagingMountDir := fmt.Sprintf("/var/lib/kubelet/plugins/kubernetes.io/csi/csi-powerstore.dellemc.com/%s/globalmount", volSha)
	log.Debugf("stagingMountDev: %s", stagingMountDir)
	return stagingMountDir
}

// GetStagingBlockDir Returns the staging directory used by NodeUnstage for a block device.
func (d *PStoreDriver) GetStagingBlockDir(_, pvName string) string {
	privateMountDir := getPrivateMountDir("/var/lib/kubelet")
	stagingBlockDir := fmt.Sprintf("%s/plugins/kubernetes.io/csi/volumeDevices/staging/%s", privateMountDir, pvName)
	log.Debugf("stagingBlockDir: %s", stagingBlockDir)
	return stagingBlockDir
}

// NodeUnpublishExcludedError filters out NodeUnpublish errors that should be excluded
func (d *PStoreDriver) NodeUnpublishExcludedError(_ error) bool {
	return false
}

// NodeUnstageExcludedError filters out NodeStage errors that should be excluded
func (d *PStoreDriver) NodeUnstageExcludedError(_ error) bool {
	return false
}

// FinalCleanup handles any driver specific final cleanup.
func (d *PStoreDriver) FinalCleanup(_ bool, _, _, _ string) error {
	return nil
}

// PMaxDriver provides a Driver instance for the PowerMax architecture.
type PMaxDriver struct{}

// GetDriverName returns the driver name string
func (d *PMaxDriver) GetDriverName() string {
	return "powermax"
}

// GetDriverMountDir returns the mount directory used for a PV by a pod.
func (d *PMaxDriver) GetDriverMountDir(_, pvName, podUUID string) string {
	privateMountDir := getPrivateMountDir("/var/lib/kubelet")
	privateMountDir = fmt.Sprintf("%s/pods/%s/volumes/kubernetes.io~csi/%s/mount", privateMountDir, podUUID, pvName)
	log.Debugf("privateMountDir: %s", privateMountDir)
	return privateMountDir
}

// GetDriverBlockDev Returns the block device used for a PV by a pod.
func (d *PMaxDriver) GetDriverBlockDev(_, pvName, podUUID string) string {
	privateMountDir := getPrivateMountDir("/var/lib/kubelet")
	privateBlockDev := fmt.Sprintf("%s/plugins/kubernetes.io/csi/volumeDevices/publish/%s/%s", privateMountDir, pvName, podUUID)
	log.Debugf("privateBlockDev: %s", privateBlockDev)
	return privateBlockDev
}

// GetStagingMountDir Returns the staging directory used by NodeUnstage for a mount device.
func (d *PMaxDriver) GetStagingMountDir(_, pvName string) string {
	privateMountDir := getPrivateMountDir("/var/lib/kubelet")
	stagingMountDev := fmt.Sprintf("%s/plugins/kubernetes.io/csi/pv/%s/globalmount", privateMountDir, pvName)
	log.Debugf("stagingMountDev: %s", stagingMountDev)
	return stagingMountDev
}

// GetStagingMountDirAfter125 Returns the staging directory used by NodeUnstage for a mount device.
func (d *PMaxDriver) GetStagingMountDirAfter125(volumeHandle, _ string) string {
	result := sha256.Sum256([]byte(fmt.Sprintf("%s", volumeHandle)))
	volSha := fmt.Sprintf("%x", result)

	stagingMountDir := fmt.Sprintf("/var/lib/kubelet/plugins/kubernetes.io/csi/csi-powermax.dellemc.com/%s/globalmount", volSha)
	log.Debugf("stagingMountDev: %s", stagingMountDir)
	return stagingMountDir
}

// GetStagingBlockDir Returns the staging directory used by NodeUnstage for a block device.
func (d *PMaxDriver) GetStagingBlockDir(_, pvName string) string {
	privateMountDir := getPrivateMountDir("/var/lib/kubelet")
	stagingBlockDir := fmt.Sprintf("%s/plugins/kubernetes.io/csi/volumeDevices/staging/%s", privateMountDir, pvName)
	log.Debugf("stagingBlockDir: %s", stagingBlockDir)
	return stagingBlockDir
}

// NodeUnpublishExcludedError filters out NodeUnpublish errors that should be excluded
func (d *PMaxDriver) NodeUnpublishExcludedError(_ error) bool {
	return false
}

// NodeUnstageExcludedError filters out NodeStage errors that should be excluded
func (d *PMaxDriver) NodeUnstageExcludedError(_ error) bool {
	return false
}

// FinalCleanup handles any driver specific final cleanup.
func (d *PMaxDriver) FinalCleanup(_ bool, _, _, _ string) error {
	return nil
}

func getPrivateMountDir(defaultDir string) string {
	privateMountDir := os.Getenv("X_CSI_PRIVATE_MOUNT_DIR")
	if privateMountDir == "" {
		log.Debugf("Returning defaultDir: %s", defaultDir)
		return defaultDir
	}
	return privateMountDir
}
