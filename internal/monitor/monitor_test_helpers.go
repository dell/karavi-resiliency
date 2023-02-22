/*
* Copyright (c) 2021-2023 Dell Inc., or its subsidiaries. All Rights Reserved.
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

package monitor

import (
	"fmt"

	"github.com/stretchr/testify/assert"
)

// AssertExpectedAndActual is a helper function to allow the step function to call
// assertion functions where you want to compare an expected and an actual value.
func AssertExpectedAndActual(a ExpectedAndActualAssertion, expected, actual interface{}, msgAndArgs ...interface{}) error {
	var t Asserter
	a(&t, expected, actual, msgAndArgs...)
	return t.err
}

// ExpectedAndActualAssertion represents an assert function that tests an actual value to an expected value
type ExpectedAndActualAssertion func(t assert.TestingT, expected, actual interface{}, msgAndArgs ...interface{}) bool

// AssertActual is a helper function to allow the step function to call
// assertion functions where you want to compare an actual value to a
// predefined state like nil, empty or true/false.
func AssertActual(a ActualAssertion, actual interface{}, msgAndArgs ...interface{}) error {
	var t Asserter
	a(&t, actual, msgAndArgs...)
	return t.err
}

// ActualAssertion represents an assert function that tests the value of a function
type ActualAssertion func(t assert.TestingT, actual interface{}, msgAndArgs ...interface{}) bool

// Asserter is used to be able to retrieve the error reported by the called assertion
type Asserter struct {
	err error
}

// Errorf is used by the called assertion to report an error
func (a *Asserter) Errorf(format string, args ...interface{}) {
	a.err = fmt.Errorf(format, args...)
}
