//  Copyright © 2021-2024 Dell Inc. or its subsidiaries. All Rights Reserved.
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

package ssh_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"podmon/test/ssh"
	"podmon/test/ssh/mocks"

	"github.com/golang/mock/gomock"
)

func TestCommandExecution_Run(t *testing.T) {
	info := ssh.AccessInfo{
		Hostname: "host123",
		Port:     "22",
		Username: "user",
		Password: "passwd",
	}

	type checkFn func(*testing.T, []string, []string, []string, []string, error, error)
	check := func(fns ...checkFn) []checkFn { return fns }

	verifyThisOutput := func(t *testing.T,
		expectedOutput []string, actualOutput []string,
		expectedErrors []string, actualErrors []string,
		_ error, _ error,
	) {
		if len(expectedOutput) > 0 {
			if len(expectedOutput) != len(actualOutput) {
				t.Fatalf("expected output '%s', but received '%s'",
					strings.Join(expectedOutput, ","), strings.Join(actualOutput, ","))
			}

			for idx, val := range expectedOutput {
				if actualOutput[idx] != val {
					t.Fatalf("expected output '%s', but received '%s'",
						strings.Join(expectedOutput, ","), strings.Join(actualOutput, ","))
				}
			}
		}

		if len(expectedErrors) > 0 {
			if len(expectedErrors) != len(actualErrors) {
				t.Fatalf("expected errors '%s', but received '%s'",
					strings.Join(expectedErrors, ","), strings.Join(actualErrors, ","))
			}

			for idx, val := range expectedErrors {
				if actualErrors[idx] != val {
					t.Fatalf("expected errors '%s', but received '%s'",
						strings.Join(expectedErrors, ","), strings.Join(actualErrors, ","))
				}
			}
		}
	}

	verifyReturnedErrorSimilar := func(t *testing.T,
		_ []string, _ []string,
		_ []string, _ []string,
		expectedError error, actualError error,
	) {
		if expectedError != nil && !strings.Contains(actualError.Error(), expectedError.Error()) {
			t.Fatalf("expected error was '%s', but actual error was '%s'",
				expectedError.Error(), actualError.Error())
		}
	}

	testCases := map[string]func(t *testing.T) (ssh.CommandExecution, string, []checkFn, []string, []string, error){
		// Basic test case
		"success": func(*testing.T) (ssh.CommandExecution, string, []checkFn, []string, []string, error) {
			now := time.Now().String()
			ctrl := gomock.NewController(t)
			mockSessionWrapper := mocks.NewMockSessionWrapper(ctrl)
			mockClientWrapper := mocks.NewMockClientWrapper(ctrl)
			mockClientWrapper.EXPECT().GetSession(gomock.Any()).Return(mockSessionWrapper, nil)
			mockClientWrapper.EXPECT().Close().AnyTimes()
			mockSessionWrapper.EXPECT().Close().AnyTimes()
			mockSessionWrapper.EXPECT().CombinedOutput(gomock.Any()).Return([]byte(now), nil)
			client := ssh.CommandExecution{
				AccessInfo: &info,
				SSHWrapper: mockClientWrapper,
			}
			return client, "date", check(verifyThisOutput), []string{now}, nil, nil
		},
		// Case where command takes too long and timeout expires
		"timeout": func(*testing.T) (ssh.CommandExecution, string, []checkFn, []string, []string, error) {
			ctrl := gomock.NewController(t)
			mockSessionWrapper := mocks.NewMockSessionWrapper(ctrl)
			mockClientWrapper := mocks.NewMockClientWrapper(ctrl)
			mockClientWrapper.EXPECT().GetSession(gomock.Any()).Return(mockSessionWrapper, nil)
			mockClientWrapper.EXPECT().Close().AnyTimes()
			mockSessionWrapper.EXPECT().Close().AnyTimes()
			mockSessionWrapper.EXPECT().CombinedOutput(gomock.Any()).Return(nil, nil).Do(func(_ string) {
				time.Sleep(1 * time.Second)
			})

			client := ssh.CommandExecution{
				AccessInfo: &info,
				SSHWrapper: mockClientWrapper,
				Timeout:    10 * time.Millisecond,
			}
			return client, "date", check(verifyReturnedErrorSimilar), nil, nil, fmt.Errorf("command 'date' on host host123 timed out")
		},
		// Case where session create fails
		"session-create-failure": func(*testing.T) (ssh.CommandExecution, string, []checkFn, []string, []string, error) {
			ctrl := gomock.NewController(t)
			mockClientWrapper := mocks.NewMockClientWrapper(ctrl)
			mockClientWrapper.EXPECT().GetSession(gomock.Any()).Return(nil, fmt.Errorf("could not create a session"))

			client := ssh.CommandExecution{
				AccessInfo: &info,
				SSHWrapper: mockClientWrapper,
				Timeout:    10 * time.Millisecond,
			}
			return client, "date", check(verifyReturnedErrorSimilar), nil, nil, fmt.Errorf("could not connect to host123:22: could not create a session")
		},
		// Case command takes some time
		"long-running-command": func(*testing.T) (ssh.CommandExecution, string, []checkFn, []string, []string, error) {
			now := time.Now().String()
			ctrl := gomock.NewController(t)
			mockSessionWrapper := mocks.NewMockSessionWrapper(ctrl)
			mockClientWrapper := mocks.NewMockClientWrapper(ctrl)
			mockClientWrapper.EXPECT().GetSession(gomock.Any()).Return(mockSessionWrapper, nil)
			mockClientWrapper.EXPECT().Close().AnyTimes()
			mockSessionWrapper.EXPECT().Close().AnyTimes()
			mockSessionWrapper.EXPECT().CombinedOutput(gomock.Any()).DoAndReturn(func(string) ([]byte, error) {
				time.Sleep(2 * time.Second)
				return []byte(now), nil
			})
			client := ssh.CommandExecution{
				AccessInfo: &info,
				SSHWrapper: mockClientWrapper,
			}
			return client, "date", check(verifyThisOutput), []string{now}, nil, nil
		},
		// Case command takes some time
		"command-fails": func(*testing.T) (ssh.CommandExecution, string, []checkFn, []string, []string, error) {
			anError := fmt.Errorf("this command failed for some reason")
			ctrl := gomock.NewController(t)
			mockSessionWrapper := mocks.NewMockSessionWrapper(ctrl)
			mockClientWrapper := mocks.NewMockClientWrapper(ctrl)
			mockClientWrapper.EXPECT().GetSession(gomock.Any()).Return(mockSessionWrapper, nil)
			mockClientWrapper.EXPECT().Close().AnyTimes()
			mockSessionWrapper.EXPECT().Close().AnyTimes()
			mockSessionWrapper.EXPECT().CombinedOutput(gomock.Any()).Return(nil, anError)

			client := ssh.CommandExecution{
				AccessInfo: &info,
				SSHWrapper: mockClientWrapper,
			}
			return client, "failing command", check(verifyReturnedErrorSimilar), nil, nil, anError
		},
		// Case where we try to use a real SSH connection, should fail to reach the fake host
		"new-wrapper": func(*testing.T) (ssh.CommandExecution, string, []checkFn, []string, []string, error) {
			client := ssh.CommandExecution{
				AccessInfo: &info,
				SSHWrapper: ssh.NewWrapper(&info),
			}
			return client, "date", check(verifyReturnedErrorSimilar), nil, nil, fmt.Errorf("could not connect to host123:22: could not create a session")
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			client, commandString, validators, expectedOutput, expectedErrors, expectedErr := tc(t)
			if len(validators) == 0 {
				t.Skipf("Skipping %s because there are no checks in place", name)
			}

			actualErr := client.Run(commandString)
			output := client.GetOutput()
			errors := client.GetErrors()
			client.HasError()

			for _, validate := range validators {
				validate(t, expectedOutput, output, expectedErrors, errors, expectedErr, actualErr)
			}
		})
	}
}

func TestCommandExecution_ScpRun(t *testing.T) {
	info := ssh.AccessInfo{
		Hostname: "host123",
		Port:     "22",
		Username: "user",
		Password: "passwd",
	}

	type checkFn func(*testing.T, error, error)
	check := func(fns ...checkFn) []checkFn { return fns }

	verifyThisOutput := func(t *testing.T, expectedError error, actualError error) {
		if expectedError != nil && actualError == nil {
			t.Fatalf("Expect error %v, but did not get it", expectedError)
		} else if expectedError == nil && actualError != nil {
			t.Fatalf("Did not expect error, but got %v", actualError)
		} else if expectedError != nil && actualError != nil && expectedError.Error() != actualError.Error() {
			t.Fatalf("Expected error %v, but got %v", expectedError, actualError)
		}
	}

	justVerifyTheresAnError := func(t *testing.T, expectedError error, actualError error) {
		if expectedError != nil && actualError == nil {
			t.Fatal("Expect an error but did not get it")
		}
	}

	testCases := map[string]func(t *testing.T) (context.Context, ssh.CommandExecution, string, string, []checkFn, error){
		// Basic test case
		"success": func(*testing.T) (context.Context, ssh.CommandExecution, string, string, []checkFn, error) {
			ctx := context.Background()
			ctrl := gomock.NewController(t)
			mockSessionWrapper := mocks.NewMockSessionWrapper(ctrl)
			mockClientWrapper := mocks.NewMockClientWrapper(ctrl)
			mockClientWrapper.EXPECT().GetSession(gomock.Any()).Return(mockSessionWrapper, nil)
			mockClientWrapper.EXPECT().Copy(gomock.Any(), gomock.Any(), "/file2", gomock.Any())
			mockClientWrapper.EXPECT().Close().AnyTimes()
			mockSessionWrapper.EXPECT().Close().AnyTimes()
			client := ssh.CommandExecution{
				AccessInfo: &info,
				SSHWrapper: mockClientWrapper,
			}
			return ctx, client, "client_test.go", "/file2", check(verifyThisOutput), nil
		},
		// Bad source file
		"source-file-not-found": func(*testing.T) (context.Context, ssh.CommandExecution, string, string, []checkFn, error) {
			ctx := context.Background()
			ctrl := gomock.NewController(t)
			mockSessionWrapper := mocks.NewMockSessionWrapper(ctrl)
			mockClientWrapper := mocks.NewMockClientWrapper(ctrl)
			mockClientWrapper.EXPECT().GetSession(gomock.Any()).Return(mockSessionWrapper, nil)
			mockClientWrapper.EXPECT().Close().AnyTimes()
			mockSessionWrapper.EXPECT().Close().AnyTimes()
			client := ssh.CommandExecution{
				AccessInfo: &info,
				SSHWrapper: mockClientWrapper,
			}
			return ctx, client, "bogus", "/file2", check(justVerifyTheresAnError), fmt.Errorf("file not found")
		},
		// Copy fails
		"copy-fails": func(*testing.T) (context.Context, ssh.CommandExecution, string, string, []checkFn, error) {
			ctx := context.Background()
			ctrl := gomock.NewController(t)
			mockSessionWrapper := mocks.NewMockSessionWrapper(ctrl)
			mockClientWrapper := mocks.NewMockClientWrapper(ctrl)
			mockClientWrapper.EXPECT().GetSession(gomock.Any()).Return(mockSessionWrapper, nil)
			mockClientWrapper.EXPECT().Copy(gomock.Any(), gomock.Any(), "/file2", gomock.Any()).Return(fmt.Errorf("copy failed"))
			mockClientWrapper.EXPECT().Close().AnyTimes()
			mockSessionWrapper.EXPECT().Close().AnyTimes()
			client := ssh.CommandExecution{
				AccessInfo: &info,
				SSHWrapper: mockClientWrapper,
			}
			return ctx, client, "client_test.go", "/file2", check(verifyThisOutput), fmt.Errorf("copy failed")
		},
		// GetSession fails
		"get-session-fails": func(*testing.T) (context.Context, ssh.CommandExecution, string, string, []checkFn, error) {
			ctx := context.Background()
			ctrl := gomock.NewController(t)
			mockSessionWrapper := mocks.NewMockSessionWrapper(ctrl)
			mockClientWrapper := mocks.NewMockClientWrapper(ctrl)
			mockClientWrapper.EXPECT().GetSession(gomock.Any()).Return(nil, fmt.Errorf("could not create session"))
			mockClientWrapper.EXPECT().Close().AnyTimes()
			mockSessionWrapper.EXPECT().Close().AnyTimes()
			client := ssh.CommandExecution{
				AccessInfo: &info,
				SSHWrapper: mockClientWrapper,
			}
			return ctx, client, "client_test.go", "/file2", check(verifyThisOutput), fmt.Errorf("could not create session")
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			ctx, client, src, dst, validators, expectedErr := tc(t)
			if len(validators) == 0 {
				t.Skipf("Skipping %s because there are no checks in place", name)
			}

			actualErr := client.Copy(ctx, src, dst)

			for _, validate := range validators {
				validate(t, expectedErr, actualErr)
			}
		})
	}
}

func TestCommandExecution_SendRequest(t *testing.T) {
	info := ssh.AccessInfo{
		Hostname: "host123",
		Port:     "22",
		Username: "user",
		Password: "passwd",
	}

	type checkFn func(*testing.T, error, error)
	check := func(fns ...checkFn) []checkFn { return fns }

	verifyThisOutput := func(t *testing.T, expectedError error, actualError error) {
		if expectedError != nil && actualError == nil {
			t.Fatalf("Expect error %v, but did not get it", expectedError)
		} else if expectedError == nil && actualError != nil {
			t.Fatalf("Did not expect error, but got %v", actualError)
		} else if expectedError != nil && actualError != nil && expectedError.Error() != actualError.Error() {
			t.Fatalf("Expected error %v, but got %v", expectedError, actualError)
		}
	}

	testCases := map[string]func(t *testing.T) (ssh.CommandExecution, string, []checkFn, error){
		// Basic test case
		"success": func(*testing.T) (ssh.CommandExecution, string, []checkFn, error) {
			ctrl := gomock.NewController(t)
			mockSessionWrapper := mocks.NewMockSessionWrapper(ctrl)
			mockClientWrapper := mocks.NewMockClientWrapper(ctrl)
			mockClientWrapper.EXPECT().GetSession(gomock.Any()).Return(mockSessionWrapper, nil)
			mockClientWrapper.EXPECT().Close().AnyTimes()
			mockSessionWrapper.EXPECT().Close().AnyTimes()
			mockSessionWrapper.EXPECT().SendRequest("exec", false, gomock.Any()).Return(false, nil)
			client := ssh.CommandExecution{
				AccessInfo: &info,
				SSHWrapper: mockClientWrapper,
			}
			return client, "date", check(verifyThisOutput), nil
		},
		// Session fails
		"session-failed": func(*testing.T) (ssh.CommandExecution, string, []checkFn, error) {
			ctrl := gomock.NewController(t)
			mockClientWrapper := mocks.NewMockClientWrapper(ctrl)
			mockClientWrapper.EXPECT().GetSession(gomock.Any()).Return(nil, fmt.Errorf("get session failed"))
			client := ssh.CommandExecution{
				AccessInfo: &info,
				SSHWrapper: mockClientWrapper,
			}
			return client, "date", check(verifyThisOutput), fmt.Errorf("get session failed")
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			client, commandStr, validators, expectedErr := tc(t)
			if len(validators) == 0 {
				t.Skipf("Skipping %s because there are no checks in place", name)
			}

			actualErr := client.SendRequest(commandStr)

			for _, validate := range validators {
				validate(t, expectedErr, actualErr)
			}
		})
	}
}
