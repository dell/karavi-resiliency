/*
 * Copyright (c) 2022. Dell Inc., or its subsidiaries. All Rights Reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 */

package utils

import (
	"errors"
)

//UtilsMock is a mock structure used for testing
type UtilsMock struct {
	InducedErrors struct {
		GetLoopBackDevice    bool
		DeleteLoopBackDevice bool
		Unmount              bool
		Creat                bool
	}
}

// GetLoopBackDevice gets all the volume attachments in the K8S system
func (mock *UtilsMock) GetLoopBackDevice(pv string) (string, error) {
	if mock.InducedErrors.GetLoopBackDevice {
		return "", errors.New("induced GetLoopBackDevice error")
	}
	return pv, nil
}

// DeleteLoopBackDevice deletes a volume attachment by name.
func (mock *UtilsMock) DeleteLoopBackDevice(device string) ([]byte, error) {
	delSucc := []byte("loopbackdevice")
	if mock.InducedErrors.DeleteLoopBackDevice {
		return nil, errors.New("induced DeleteLoopBackDevice error")
	}
	return delSucc, nil
}

// Unmount is a wrapper around syscall.Unmount
func (mock *UtilsMock) Unmount(devName string, flags int) error {
	if mock.InducedErrors.Unmount {
		return errors.New("induced Unmount error")
	}
	return nil
}

// Creat is a wrapper around syscall.Creat
func (mock *UtilsMock) Creat(filepath string, flags int) (int, error) {
	if mock.InducedErrors.Creat {
		return 1, errors.New("induced Creat error")
	}
	return 0, nil
}
