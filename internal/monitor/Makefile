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

integration-test:
	RESILIENCY_INT_TEST="true" \
	RESILIENCY_TEST_CLEANUP="true" \
	K8S_POLL="true" \
	SCRIPTS_DIR="../../test/sh" \
	go test -timeout 2h -test.v -test.run "^\QTestFirstCheck\E|\QTestIntegration\E"
