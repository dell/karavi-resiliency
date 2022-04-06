/*
 * Copyright (c) 2021. Dell Inc., or its subsidiaries. All Rights Reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 */

package monitor

import (
	"fmt"
	"os"
	"podmon/internal/utils"
	"strings"

	log "github.com/sirupsen/logrus"
)

type drivertype interface {
	GetDriverName() string
	GetDriverMountDir(volumeHandle, pvName, podUUID string) string
	GetDriverBlockDev(volumeHandle, pvName, podUUID string) string
	GetStagingMountDir(volumeHandle, pvName string) string
	GetStagingBlockDir(volumeHandle, pvName string) string
	NodeUnpublishExcludedError(err error) bool
	NodeUnstageExcludedError(err error) bool
	FinalCleanup(rawBlock bool, volumeHandle, pvName, podUUID string) error
}

//Driver is an instance of the drivertype interface to provide driver specific functions.
var Driver drivertype

//VxflexDriver provides a Driver instance for the PowerFlex (VxFlex) architecture.
type VxflexDriver struct {
}

//GetDriverName returns the driver name string
func (d *VxflexDriver) GetDriverName() string {
	return "vxflexos"
}

//GetDriverMountDir returns the Vxflex private mount directory.
func (d *VxflexDriver) GetDriverMountDir(volumeHandle, pvName, podUUID string) string {
	privateMountDir := os.Getenv("X_CSI_PRIVATE_MOUNT_DIR")
	if privateMountDir == "" {
		privateMountDir = "/var/lib/kubelet/plugins/vxflexos.emc.dell.com/disks"
	}
	privateMountDir = fmt.Sprintf("%s/%s", privateMountDir, volumeHandle)
	log.Debugf("privateMountDir: %s", privateMountDir)
	return privateMountDir
}

// GetDriverBlockDev Returns the block device used for a PV by a pod.
func (d *VxflexDriver) GetDriverBlockDev(volumeHandle, pvName, podUUID string) string {
	privateBlockDev := fmt.Sprintf("/var/lib/kubelet/plugins/kubernetes.io/csi/volumeDevices/publish/%s/%s", pvName, podUUID)
	log.Debugf("privateBlockDev: %s", privateBlockDev)
	return privateBlockDev
}

// GetStagingMountDir Returns the staging directory used by NodeUnstage for a mount device.
func (d *VxflexDriver) GetStagingMountDir(volumeHandle, pvName string) string {
	stagingMountDir := os.Getenv("X_CSI_PRIVATE_MOUNT_DIR")
	if stagingMountDir == "" {
		stagingMountDir = "/var/lib/kubelet/plugins/vxflexos.emc.dell.com/disks"
	}
	stagingMountDir = fmt.Sprintf("%s/%s", stagingMountDir, volumeHandle)
	log.Debugf("stagingMountDir: %s", stagingMountDir)
	return stagingMountDir
}

// GetStagingBlockDir Returns the staging directory used by NodeUnstage for a block device.
func (d *VxflexDriver) GetStagingBlockDir(volumeHandle, pvName string) string {
	stagingBlockDir := fmt.Sprintf("/var/lib/kubelet/plugins/kubernetes.io/csi/volumeDevices/staging/%s", pvName)
	log.Debugf("stagingBlockDir: %s", stagingBlockDir)
	return stagingBlockDir
}

// NodeUnpublishExcludedError filters out NodeUnpublish errors that should be excluded
func (d *VxflexDriver) NodeUnpublishExcludedError(err error) bool {
	return false
}

// NodeUnstageExcludedError filters out NodeStage errors that should be excluded
func (d *VxflexDriver) NodeUnstageExcludedError(err error) bool {
	return false
}

// FinalCleanup handles any driver specific final cleanup.
func (d *VxflexDriver) FinalCleanup(rawBlock bool, volumeHandle, pvName, podUUID string) error {
	return nil
}

//UnityDriver provides a Driver instance for the Unity architecture.
type UnityDriver struct {
}

//GetDriverName returns the driver name string
func (d *UnityDriver) GetDriverName() string {
	return "unity"
}

//GetDriverMountDir returns the Unity private mount directory.
func (d *UnityDriver) GetDriverMountDir(volumeHandle, pvName, podUUID string) string {
	privateMountDir := fmt.Sprintf("/var/lib/kubelet/pods/%s/volumes/kubernetes.io~csi/%s/mount", podUUID, pvName)
	log.Debugf("privateMountDir: %s", privateMountDir)
	return privateMountDir
}

// GetDriverBlockDev Returns the block device used for a PV by a pod.
func (d *UnityDriver) GetDriverBlockDev(volumeHandle, pvName, podUUID string) string {
	privateBlockDev := fmt.Sprintf("/var/lib/kubelet/plugins/kubernetes.io/csi/volumeDevices/publish/%s/%s", pvName, podUUID)
	log.Debugf("privateBlockDev: %s", privateBlockDev)
	return privateBlockDev
}

// GetStagingMountDir Returns the staging directory used by NodeUnstage for a mount device.
func (d *UnityDriver) GetStagingMountDir(volumeHandle, pvName string) string {
	stagingMountDev := fmt.Sprintf("/var/lib/kubelet/plugins/kubernetes.io/csi/pv/%s/globalmount", pvName)
	log.Debugf("stagingMountDev: %s", stagingMountDev)
	return stagingMountDev
}

// GetStagingBlockDir Returns the staging directory used by NodeUnstage for a block device.
func (d *UnityDriver) GetStagingBlockDir(volumeHandle, pvName string) string {
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
func (d *UnityDriver) FinalCleanup(rawBlock bool, volumeHandle, pvName, podUUID string) error {
	if rawBlock { //Do this cleanup on raw device block
		blockDev := fmt.Sprintf("/var/lib/kubelet/plugins/kubernetes.io/csi/volumeDevices/%s/dev/%s", pvName, podUUID)

		loopBackDev, err := utils.GetLoopBackDevice(pvName)
		if err != nil || loopBackDev == "" {
			// nothing to clean
			log.Infof("LOOOOOOOOOOOOOOOOPPPPBACK DEVICE EMPTY")
			return nil
		}

		log.Infof("LOOOOOOOOOOOOOOOOPPPPBACK DEVICE: %s", loopBackDev)

		_, err = utils.DeleteLoopBackDevice(loopBackDev)
		if err != nil {
			log.Infof("error deleting loopback device: %s", loopBackDev)
			return nil
		}

		err = utils.Unmount(blockDev, 1)
		if err != nil {
			log.Infof("error in unmount block device in FinalCleanup: %s path: %s", err, blockDev)
		} else {
			log.Infof("sucessfully unmounted block device: %s", blockDev)
		}
		err = RemoveDev(blockDev)
		if err != nil {
			log.Infof("error remove block device FinalCleanup: %s path: %s", err, blockDev)
		} else {
			log.Infof("removed block device: %s", blockDev)
		}
	}
	return nil
}
