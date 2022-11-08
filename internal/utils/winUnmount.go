//go:build test || windows
// +build test windows

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

package utils

import "os"

// !!! This is not meant for production. This implementation is provided in case !!!
// !!! a developer wishes to run the unit tests from an IDE on Windows           !!!

func Unmount(devName string, flags int) error {
	return os.Remove(devName)
}

func Creat(filepath string, flags int) (int, error) {
	if file, err := os.Create(filepath); err != nil {
		return -1, err
	} else {
		return int(file.Fd()), err
	}
}
