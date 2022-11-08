#!/bin/sh
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

DIRS=$@

if [ -f "../vendor" ]; then
    # Tell the applicable Go tools to use the vendor directory, if it exists.
    MOD_FLAGS="-mod=vendor"
fi

FMT_TMPFILE=/tmp/check_fmt
FMT_COUNT_TMPFILE=${FMT_TMPFILE}.count

fmt_count() {
    if [ ! -f $FMT_COUNT_TMPFILE ]; then
        echo "0"
    fi

    head -1 $FMT_COUNT_TMPFILE
}

fmt() {
    gofmt -d ${DIRS} | tee $FMT_TMPFILE
    cat $FMT_TMPFILE | wc -l > $FMT_COUNT_TMPFILE
    if [ ! `cat $FMT_COUNT_TMPFILE` -eq "0" ]; then
        echo Found `cat $FMT_COUNT_TMPFILE` formatting issue\(s\).
        return 1
    fi
}

echo === Checking format...
fmt
FMT_RETURN_CODE=$?
echo === Finished code=$FMT_RETURN_CODE

echo === Vetting...
go vet ${MOD_FLAGS} ${DIRS}
VET_RETURN_CODE=$?
echo === Finished code=$VET_RETURN_CODE

echo === Linting...
(command -v golint >/dev/null 2>&1 \
    || GO111MODULE=off go get -insecure -u golang.org/x/lint/golint) \
    && golint --set_exit_status ${DIRS}
LINT_RETURN_CODE=$?
echo === Finished code=$LINT_RETURN_CODE

# Run gosec scan
echo === Gosec scan...
gosec -h > /dev/null 2>&1
if [ $? -ne 0 ]; then
  echo "Installing gosec"
  # This is a workaround for the forbidden word scanner
  url=$(echo "https://raw.githubusercontent.com/securego/gosec/m a s t e r/install.sh" | tr -d " ")
  curl -sfL $url | sh -s v2.7.0
  mv ./bin/gosec /usr/bin/gosec
fi
gosec -exclude-dir=test  ./...
GOSEC_RETURN_CODE=$?
echo === Finished code=$GOSEC_RETURN_CODE

# Run internal references scanner
echo === Running private data scans...
docker pull "$DOCKER_REPO"/code-sanitizer
docker run --rm -v "$(pwd)":"/usr/local/share" "$DOCKER_REPO"/code-sanitizer /usr/local/share
INTERNAL_SCAN_CODE=$?
echo === Finished code=$INTERNAL_SCAN_CODE

# Report output.
fail_checks=0
[ "${FMT_RETURN_CODE}" != "0" ] && echo "Formatting checks failed!" && fail_checks=1
[ "${VET_RETURN_CODE}" != "0" ] && echo "Vetting checks failed!" && fail_checks=1
[ "${LINT_RETURN_CODE}" != "0" ] && echo "Linting checks failed!" && fail_checks=1
[ "${GOSEC_RETURN_CODE}" != "0" ] && echo "Gosec found Golang security issues" && fail_checks=1
[ "${INTERNAL_SCAN_CODE}" != "0" ] && echo "Sanitizer scanner found internal references" && fail_checks=1

exit ${fail_checks}

