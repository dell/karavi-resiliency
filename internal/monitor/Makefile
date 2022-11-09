# Copyright (c) 2021-2022 Dell Inc., or its subsidiaries. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
# Includes the following generated file to get semantic version information
.PHONY: clean unit-test gocover

clean:
	go clean -cache

unit-test:
	RESILIENCY_INT_TEST="false" \
	go test -v -coverprofile=c.out ./...

godog:
	go clean -cache
	RESILIENCY_INT_TEST="false" \
	go test -v -coverprofile=c.out -test.run TestGoDog ./...

gocover:
	go tool cover -html=c.out

powerflex-integration-test:
	RESILIENCY_INT_TEST="true" \
	RESILIENCY_TEST_CLEANUP="true" \
	POLL_K8S="true" \
	SCRIPTS_DIR="../../test/sh" \
	go test -timeout 6h -test.v -test.run "^\QTestPowerFlexFirstCheck\E|\QTestPowerFlexIntegration\E"

unity-integration-test:
	RESILIENCY_INT_TEST="true" \
	RESILIENCY_TEST_CLEANUP="true" \
	POLL_K8S="true" \
	SCRIPTS_DIR="../../test/sh" \
	go test -timeout 6h -test.v -test.run "^\QTestUnityFirstCheck\E|\QTestUnityIntegration\E"

powerscale-integration-test:
	RESILIENCY_INT_TEST="true" \
	RESILIENCY_TEST_CLEANUP="true" \
	POLL_K8S="true" \
	SCRIPTS_DIR="../../test/sh" \
	go test -timeout 6h -test.v -test.run "^\QTestPowerScaleFirstCheck\E|\QTestPowerScaleIntegration\E"


powerflex-short-integration-test:
	RESILIENCY_SHORT_INT_TEST="true" \
	RESILIENCY_TEST_CLEANUP="true" \
	POLL_K8S="true" \
	SCRIPTS_DIR="../../test/sh" \
	go test -timeout 6h -test.v -test.run "^\QTestPowerFlexShortCheck\E|\QTestPowerFlexShortIntegration\E"

unity-short-integration-test:
	RESILIENCY_SHORT_INT_TEST="true" \
	RESILIENCY_TEST_CLEANUP="true" \
	POLL_K8S="true" \
	SCRIPTS_DIR="../../test/sh" \
	go test -timeout 6h -test.v -test.run "^\QTestUnityShortCheck\E|\QTestUnityShortIntegration\E"

powerscale-short-integration-test:
	RESILIENCY_SHORT_INT_TEST="true" \
	RESILIENCY_TEST_CLEANUP="true" \
	POLL_K8S="true" \
	SCRIPTS_DIR="../../test/sh" \
	go test -timeout 6h -test.v -test.run "^\QTestPowerScaleShortCheck\E|\QTestPowerScaleShortIntegration\E"

powerflex-array-interface-test:
	RESILIENCY_INT_TEST="true" \
	RESILIENCY_TEST_CLEANUP="true" \
	POLL_K8S="true" \
	SCRIPTS_DIR="../../test/sh" \
	INTERFACE_A=${INTERFACE1} \
	go test -timeout 6h -test.v -test.run "^\QTestPowerFlexFirstCheck\E|\QTestPowerflexArrayInterfaceDown\E"

unity-array-interface-test:
	RESILIENCY_INT_TEST="true" \
	RESILIENCY_TEST_CLEANUP="true" \
	POLL_K8S="true" \
	SCRIPTS_DIR="../../test/sh" \
	INTERFACE_A=${INTERFACE1} \
	INTERFACE_B=${INTERFACE2} \
	go test -timeout 6h -test.v -test.run "^\QTestUnityFirstCheck\E|\QTestUnityArrayInterfaceDown\E"
