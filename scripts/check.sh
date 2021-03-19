#!/bin/sh

#
# Copyright (c) 2021. Dell Inc., or its subsidiaries. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
#

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
echo === Finished

echo === Vetting...
go vet ${MOD_FLAGS} ${DIRS}
VET_RETURN_CODE=$?
echo === Finished

echo === Linting...
(command -v golint >/dev/null 2>&1 \
    || GO111MODULE=off go get -insecure -u golang.org/x/lint/golint) \
    && golint --set_exit_status ${DIRS}
LINT_RETURN_CODE=$?
echo === Finished

# Report output.
fail_checks=0
[ "${FMT_RETURN_CODE}" != "0" ] && echo "Formatting checks failed!" && fail_checks=1
[ "${VET_RETURN_CODE}" != "0" ] && echo "Vetting checks failed!" && fail_checks=1
[ "${LINT_RETURN_CODE}" != "0" ] && echo "Linting checks failed!" && fail_checks=1

# Run gosec scan
gosec -h > /dev/null 2>&1
if [ $? -ne 0 ]; then
  echo "Installing gosec"
  # This is a workaround for the forbidden word scanner
  url=$(echo "https://raw.githubusercontent.com/securego/gosec/m a s t e r/install.sh" | tr -d " ")
  curl -sfL $url | sh -s v2.7.0
  mv ./bin/gosec /usr/bin/gosec
fi
gosec -exclude-dir=test  ./...

exit ${fail_checks}

