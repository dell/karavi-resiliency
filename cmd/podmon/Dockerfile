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
FROM registry.access.redhat.com/ubi8/ubi:8.6-943
LABEL vendor="Dell Inc." \
      name="csm-resiliency" \
      summary="Dell Container Storage Modules (CSM) for Resiliency" \
      description="Makes Kubernetes applications, including those that utilize persistent storage, more resilient to various failures" \
      version="2.0.0" \
      license="Apache-2.0"
RUN dnf -y install openssl && \
    dnf -y install nettle && \
    dnf -y install gnutls && \
    dnf -y install systemd
COPY podmon /podmon
ENTRYPOINT [ "/podmon" ]
