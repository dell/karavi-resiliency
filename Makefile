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
all:
	(cd cmd/podmon; make clean build-base-image build podman push)

check:
	@scripts/check.sh ./internal/monitor ./internal/k8sapi ./internal/csiapi ./internal/criapi ./cmd/podmon  

unit-test:
	(cd cmd/podmon; make unit-test)
