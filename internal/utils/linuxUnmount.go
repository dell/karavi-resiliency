// +build test linux

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

package utils

import "syscall"

func Unmount(devName string, flags int) error {
	return syscall.Unmount(devName, flags)
}

func Creat(filepath string, flags int) (int, error) {
	return syscall.Creat(filepath, flags)
}
