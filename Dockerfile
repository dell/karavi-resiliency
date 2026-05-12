# Copyright (c) 2021-2026 Dell Inc., or its subsidiaries. All Rights Reserved.
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

ARG GOIMAGE
ARG BASEIMAGE
ARG VERSION="1.16.0"

# Build the module binary
FROM $GOIMAGE as builder

WORKDIR /workspace
COPY . /workspace

# Build the binary
RUN make build

# Stage to build the module image
FROM $BASEIMAGE AS final
ARG VERSION
LABEL vendor="Dell Technologies" \
      maintainer="Dell Technologies" \
      name="csm-resiliency" \
      summary="Dell Container Storage Modules (CSM) for Resiliency" \
      description="Makes Kubernetes applications, including those that utilize persistent storage, more resilient to various failures" \
      release="1.17.0" \
      version=$VERSION \
      license="Apache-2.0"

COPY licenses /licenses
COPY --from=builder /workspace/podmon /

ENTRYPOINT [ "/podmon" ]
