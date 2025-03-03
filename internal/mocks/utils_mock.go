/*
* Copyright (c) 2023 Dell Inc., or its subsidiaries. All Rights Reserved.
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

package mocks

import (
	"errors"
)

// Mock is a mock structure used for testing
type Mock struct {
	InducedErrors struct {
		GetLoopBackDevice    bool
		DeleteLoopBackDevice bool
		Unmount              bool
		Creat                bool
	}
}

// GetLoopBackDevice gets the loopbackdevice for given pv
func (mock *Mock) GetLoopBackDevice(pv string) (string, error) {
	if mock.InducedErrors.GetLoopBackDevice {
		return "", errors.New("induced GetLoopBackDevice error")
	}
	return pv, nil
}

// DeleteLoopBackDevice deletes a loopbackdevice.
func (mock *Mock) DeleteLoopBackDevice(_ string) ([]byte, error) {
	delSucc := []byte("loopbackdevice")
	if mock.InducedErrors.DeleteLoopBackDevice {
		return nil, errors.New("induced DeleteLoopBackDevice error")
	}
	return delSucc, nil
}

// Unmount is a wrapper around syscall.Unmount
func (mock *Mock) Unmount(_ string, _ int) error {
	if mock.InducedErrors.Unmount {
		return errors.New("induced Unmount error")
	}
	return nil
}

// Creat is a wrapper around syscall.Creat
func (mock *Mock) Creat(_ string, _ int) (int, error) {
	if mock.InducedErrors.Creat {
		return 1, errors.New("induced Creat error")
	}
	return 0, nil
}
