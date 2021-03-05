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

# This is script is for programmatic invocation of a shell script
echo "$(date +"%Y-%m-%d %H:%M:%S")" -- invoking script $* >>/tmp/invoke.log
nohup $* &
