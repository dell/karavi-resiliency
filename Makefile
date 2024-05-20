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
MINOR=10
PATCH=0
VERSION?="v$(MAJOR).$(MINOR).$(PATCH)"
REGISTRY?="${REGISTRY_HOST}:${REGISTRY_PORT}/podmon"
BASEIMAGE?="resiliency-ubimicro:latest"

all: clean podman push

check:
	@scripts/check.sh ./internal/monitor ./internal/k8sapi ./internal/csiapi ./internal/criapi ./cmd/podmon  

unit-test:
	(cd cmd/podmon; make unit-test)

clean:
	go clean ./...

build:
	GOOS=linux CGO_ENABLED=0 go build -o podmon ./cmd/podmon/

build-base-image: download-csm-common
	$(eval include csm-common.mk)
	sh ./scripts/buildubimicro.sh $(DEFAULT_BASEIMAGE)

podman: build-base-image
	podman build --no-cache -t "$(REGISTRY):$(VERSION)" --build-arg GOIMAGE=$(DEFAULT_GOIMAGE) --build-arg BASEIMAGE=$(BASEIMAGE) -f ./Dockerfile --label commit=$(shell git log --max-count 1 --format="%H") .

push:
	podman push "$(REGISTRY):$(VERSION)"

download-csm-common:
	curl -O -L https://raw.githubusercontent.com/dell/csm/main/config/csm-common.mk
