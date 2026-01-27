# Copyright © 2025-2026 Dell Inc. or its subsidiaries. All Rights Reserved.
#
# Dell Technologies, Dell and other trademarks are trademarks of Dell Inc.
# or its subsidiaries. Other trademarks may be trademarks of their respective 
# owners.

include images.mk

all: clean images

unit-test: go-code-tester
	GITHUB_OUTPUT=/dev/null \
	./go-code-tester 85 "." "podmon/test/ssh" "true" "" "" "./test|./internal/mocks|./core"

clean:
	go clean ./...
	rm -f csm-common.mk
	rm -rf vendor

build: generate vendor
	GOOS=linux CGO_ENABLED=0 go build -mod=vendor -o podmon ./cmd/podmon/ 

go-code-tester:
	git clone --depth 1 git@github.com:CSM/actions.git temp-repo
	cp temp-repo/go-code-tester/entrypoint.sh ./go-code-tester
	chmod +x go-code-tester
	rm -rf temp-repo
