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
PATCH=58
VERSION?="v$(MAJOR).$(MINOR).$(PATCH)"
REGISTRY?="${REGISTRY_HOST}:${REGISTRY_PORT}/podmontest"

clean:
	go clean ./...

build:
	rm -f podmontest
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -tags podmontest -ldflags '-w' -o podmontest *.go

docker:  build
	 docker build -t podmontest  -f Dockerfile  --network host .

podman:
	podman build -t "$(REGISTRY):$(VERSION)" --label commit=$(shell git log --max-count 1 --format="%H") .

push:
	podman push "$(REGISTRY):$(VERSION)"
