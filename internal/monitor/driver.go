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
	log "github.com/sirupsen/logrus"
	"os"
)

type drivertype interface {
	GetDriverMountDir(volumeHandle, pvName, podUUID string) string
	GetDriverBlockDev(volumeHandle, pvName, podUUID string) string
}

//Driver is an instance of the drivertype interface to provide driver specific functions.
var Driver drivertype

//VxflexDriver provides a Driver instance for the PowerFlex (VxFlex) architecture.
type VxflexDriver struct {
}

//GetDriverMountDir returns the Vxflex private mount directory.
func (d *VxflexDriver) GetDriverMountDir(volumeHandle, pvName, podUUID string) string {
	privateMountDir := os.Getenv("X_CSI_PRIVATE_MOUNT_DIR")
	if privateMountDir == "" {
		privateMountDir = "/var/lib/kubelet/plugins/vxflexos.emc.dell.com/disks"
	}
	privateMountDir = fmt.Sprintf("%s/%s", privateMountDir, volumeHandle)
	log.Infof("privateMountDir: %s", privateMountDir)
	return privateMountDir
}

// GetDriverBlockDev Returns the block device used for a PV by a pod.
func (d *VxflexDriver) GetDriverBlockDev(volumeHandle, pvName, podUUID string) string {
	privateBlockDev := fmt.Sprintf("/var/lib/kubelet/plugins/kubernetes.io/csi/volumeDevices/publish/%s/%s", pvName, podUUID)
	log.Infof("privateBlockDev: %s", privateBlockDev)
	return privateBlockDev
}
