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
MAJOR=0
MINOR=0
PATCH=54
VERSION?="v$(MAJOR).$(MINOR).$(PATCH)"
REGISTRY?="${REGISTRY_HOST}:${REGISTRY_PORT}/podmon"
BASEIMAGE?="resiliency-ubimicro:latest"

clean:
	go clean ./...

build:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64  go build -a -ldflags '-w' ./...

build-base-image: download-csm-common
	$(eval include csm-common.mk)
	sh ../../scripts/buildubimicro.sh $(DEFAULT_BASEIMAGE)

podman:
	podman build --no-cache -t "$(REGISTRY):$(VERSION)" --build-arg BASEIMAGE=$(BASEIMAGE) --label commit=$(shell git log --max-count 1 --format="%H") .

push:
	podman push "$(REGISTRY):$(VERSION)"

unit-test:
	go test -race -v -coverprofile=c.out ./...

godog:
	go clean -cache
	go test -v -coverprofile=c.out -test.run TestMain ./...

download-csm-common:
	curl -O -L https://raw.githubusercontent.com/dell/csm/main/config/csm-common.mk
