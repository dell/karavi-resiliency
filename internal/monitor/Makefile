# Copyright (c) 2021-2023 Dell Inc., or its subsidiaries. All Rights Reserved.
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
# VM-Integration Tests
powerstore-vm-integration-test:
	RESILIENCY_VM_INT_TEST="true" \
	RESILIENCY_TEST_CLEANUP="true" \
	POLL_K8S="true" \
	SCRIPTS_DIR="../../test/sh" \
	go test -timeout 6h -test.v -test.run "^\QTestOcpVirtPowerStoreCheck\E|\QTestOcpVirtPowerStoreIntegration\E"

powerscale-vm-integration-test:
	RESILIENCY_VM_INT_TEST="true" \
	RESILIENCY_TEST_CLEANUP="true" \
	POLL_K8S="true" \
	SCRIPTS_DIR="../../test/sh" \
	go test -timeout 6h -test.v -test.run "^\QTestOcpVirtPowerScaleCheck\E|\QTestOcpVirtPowerScaleIntegration\E"

powerflex-vm-integration-test:
	RESILIENCY_VM_INT_TEST="true" \
	RESILIENCY_TEST_CLEANUP="true" \
	POLL_K8S="true" \
	SCRIPTS_DIR="../../test/sh" \
	go test -timeout 6h -test.v -test.run "^\QTestOcpVirtPowerFlexCheck\E|\QTestOcpVirtPowerFlexIntegration\E"

powermax-vm-integration-test:
	RESILIENCY_VM_INT_TEST="true" \
	RESILIENCY_TEST_CLEANUP="true" \
	POLL_K8S="true" \
	SCRIPTS_DIR="../../test/sh" \
	go test -timeout 6h -test.v -test.run "^\QTestOcpVirtPowerMaxCheck\E|\QTestOcpVirtPowerMaxIntegration\E"

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

powerstore-integration-test:
	RESILIENCY_INT_TEST="true" \
	RESILIENCY_TEST_CLEANUP="true" \
	POLL_K8S="true" \
	SCRIPTS_DIR="../../test/sh" \
	go test -timeout 6h -test.v -test.run "^\QTestPowerStoreFirstCheck\E|\QTestPowerStoreIntegration\E"

powerstore-metro-integration-test:
	RESILIENCY_INT_TEST="true" \
	RESILIENCY_TEST_CLEANUP="true" \
	POLL_K8S="true" \
	SCRIPTS_DIR="../../test/sh" \
	POWERSTORE_METRO="true" \
	go test -timeout 6h -test.v -test.run "^\QTestPowerStoreFirstCheck\E|\QTestPowerStoreMetroIntegration\E"

powermax-integration-test:
	RESILIENCY_INT_TEST="true" \
	RESILIENCY_TEST_CLEANUP="true" \
	POLL_K8S="true" \
	SCRIPTS_DIR="../../test/sh" \
	go test -timeout 6h -test.v -test.run "^\QTestPowerMaxFirstCheck\E|\QTestPowerMaxIntegration\E"

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

powerstore-short-integration-test:
	RESILIENCY_SHORT_INT_TEST="true" \
	RESILIENCY_TEST_CLEANUP="true" \
	POLL_K8S="true" \
	SCRIPTS_DIR="../../test/sh" \
	go test -timeout 6h -test.v -test.run "^\QTestPowerStoreShortCheck\E|\QTestPowerStoreShortIntegration\E"

powermax-short-integration-test:
	RESILIENCY_SHORT_INT_TEST="true" \
	RESILIENCY_TEST_CLEANUP="true" \
	POLL_K8S="true" \
	SCRIPTS_DIR="../../test/sh" \
	go test -timeout 6h -test.v -test.run "^\QTestPowerMaxShortCheck\E|\QTestPowerMaxShortIntegration\E"

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

powerstore-array-interface-test:
	RESILIENCY_INT_TEST="true" \
	RESILIENCY_TEST_CLEANUP="true" \
	POLL_K8S="true" \
	SCRIPTS_DIR="../../test/sh" \
	INTERFACE_A=${INTERFACE1} \
	INTERFACE_B=${INTERFACE2} \
	INTERFACE_C=${INTERFACE3} \
	go test -timeout 6h -test.v -test.run "^\QTestPowerStoreFirstCheck\E|\QTestPowerStoreArrayInterfaceDown\E"

powermax-array-interface-test:
	RESILIENCY_INT_TEST="true" \
	RESILIENCY_TEST_CLEANUP="true" \
	POLL_K8S="true" \
	SCRIPTS_DIR="../../test/sh" \
	INTERFACE_A=${INTERFACE1} \
	INTERFACE_B=${INTERFACE2} \
	INTERFACE_C=${INTERFACE3} \
	go test -timeout 6h -test.v -test.run "^\QTestPowerMaxFirstCheck\E|\QTestPowerMaxArrayInterfaceDown\E"

powerflex-sanity-test:
	RESILIENCY_SANITY_TEST="true" \
	RESILIENCY_TEST_CLEANUP="true" \
	POLL_K8S="true" \
	SCRIPTS_DIR="../../test/sh" \
	go test -timeout 6h -test.v -test.run "^\QTestPowerFlexSanityCheck\E|\QTestPowerFlexSanityTest\E"

powerstore-sanity-test:
	RESILIENCY_SANITY_TEST="true" \
	RESILIENCY_TEST_CLEANUP="true" \
	POLL_K8S="true" \
	SCRIPTS_DIR="../../test/sh" \
	go test -timeout 6h -test.v -test.run "^\QTestPowerStoreSanityCheck\E|\QTestPowerStoreSanityTest\E"

powermax-sanity-test:
	RESILIENCY_SANITY_TEST="true" \
	RESILIENCY_TEST_CLEANUP="true" \
	POLL_K8S="true" \
	SCRIPTS_DIR="../../test/sh" \
	go test -timeout 6h -test.v -test.run "^\QTestPowerMaxSanityCheck\E|\QTestPowerMaxSanityTest\E"

powerscale-sanity-test:
	RESILIENCY_SANITY_TEST="true" \
	RESILIENCY_TEST_CLEANUP="true" \
	POLL_K8S="true" \
	SCRIPTS_DIR="../../test/sh" \
	go test -timeout 6h -test.v -test.run "^\QTestPowerScaleSanityCheck\E|\QTestPowerScaleSanityTest\E"

powerstore-metro-resiliency-integration-test:
	RESILIENCY_SHORT_INT_TEST="true" \
	RESILIENCY_TEST_CLEANUP="true" \
	POLL_K8S="true" \
	SCRIPTS_DIR="../../test/sh" \
	go test -timeout 6h -test.v -test.run "^\QTestPowerStoreShortCheck\E|\QTestPowerStoreMetroResiliencyIntegration\E"
