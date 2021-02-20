package monitor

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"os"
)

type drivertype interface {
	GetDriverName() string
	GetDriverMountDir(volumeHandle, pvName, podUUID string) string
	GetDriverBlockDev(volumeHandle, pvName, podUUID string) string
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
	log.Infof("privateMountDir: %s", privateMountDir)
	return privateMountDir
}

// GetDriverBlockDev Returns the block device used for a PV by a pod.
func (d *VxflexDriver) GetDriverBlockDev(volumeHandle, pvName, podUUID string) string {
	privateBlockDev := fmt.Sprintf("/var/lib/kubelet/plugins/kubernetes.io/csi/volumeDevices/publish/%s/%s", pvName, podUUID)
	log.Infof("privateBlockDev: %s", privateBlockDev)
	return privateBlockDev
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
	privateMountDir := fmt.Sprintf("/var/lib/kubelet/plugins/kubernetes.io/csi/pv/%s/mount", pvName)
	log.Infof("privateMountDir: %s", privateMountDir)
	return privateMountDir
}

// GetDriverBlockDev Returns the block device used for a PV by a pod.
func (d *UnityDriver) GetDriverBlockDev(volumeHandle, pvName, podUUID string) string {
	privateBlockDev := fmt.Sprintf("/var/lib/kubelet/plugins/kubernetes.io/csi/volumeDevices/publish/%s/%s", pvName, podUUID)
	log.Infof("privateBlockDev: %s", privateBlockDev)
	return privateBlockDev
}
