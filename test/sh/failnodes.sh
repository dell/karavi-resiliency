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

# Fails a list of nodes passed in as arguments.
# Assumes that bounce.ip is installed an appropriately configured on each node.

nodelist=""
seconds=240

for param in $*; do
  case $param in
  "--seconds")
    shift
    seconds=$1
    shift
    ;;
  *)
    nodelist="$nodelist $1"
    shift
  esac
done

for node in $nodelist
do
	echo bounce.ip  --seconds $seconds $node
	ssh $node nohup sh /root/bounce.ip --seconds $seconds &
done
