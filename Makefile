# Copyright (c) 2021-2024 Dell Inc., or its subsidiaries. All Rights Reserved.
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

MAJOR=1
MINOR=12
PATCH=0
VERSION?="v$(MAJOR).$(MINOR).$(PATCH)"
REGISTRY?="${REGISTRY_HOST}:${REGISTRY_PORT}/podmon"

all: clean podman push

unit-test:
	(cd cmd/podmon; make unit-test)

clean:
	go clean ./...

build:
	GOOS=linux CGO_ENABLED=0 go build -o podmon ./cmd/podmon/

podman: download-csm-common
	$(eval include csm-common.mk)
	podman build --no-cache -t "$(REGISTRY):$(VERSION)" --build-arg GOIMAGE=$(DEFAULT_GOIMAGE) --build-arg BASEIMAGE=$(CSM_BASEIMAGE) -f ./Dockerfile --label commit=$(shell git log --max-count 1 --format="%H") .

push:
	podman push "$(REGISTRY):$(VERSION)"

download-csm-common:
	curl -O -L https://raw.githubusercontent.com/dell/csm/main/config/csm-common.mk

.PHONY: actions
actions: ## Run all the github action checks that run on a pull_request creation
	act -l | grep -v ^Stage | grep pull_request | grep -v image_security_scan | awk '{print $$2}' | while read WF; do act pull_request --no-cache-server --platform ubuntu-latest=ghcr.io/catthehacker/ubuntu:act-latest --job "$${WF}"; done

.PHONY: check
check: ## Echo instructions to run one specific workflow locally
	@echo "GitHub Workflows can be run locally with the following command:"
	@echo "act pull_request --no-cache-server --platform ubuntu-latest=ghcr.io/catthehacker/ubuntu:act-latest --job <jobid>"
	@echo
	@echo "Where '<jobid>' is a Job ID returned by the command:"
	@echo "act -l"
	@echo
	@echo "NOTE: if act if not installed, it can be from https://github.com/nektos/act"
