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
