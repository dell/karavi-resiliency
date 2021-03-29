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
namespace="vxflexos"
once=0

for param in $*; do
  case $param in
  "--ns")
    shift
    namespace=$1
    shift
    ;;
  "--once")
    shift
    once=1
    ;;
  "--help")
    shift
    echo "parameters: --ns driver_namespace [ --label podmon_label ]"
    exit
    ;;
  esac
done

alias k=kubectl
while true;
do
	date
	k get nodes -o wide
	k get nodes -o custom-columns=NAME:.metadata.name,TAINTS:.spec.taints
	k get leases -n $namespace
	k get pods -n $namespace -o wide
	k get pods -l podmon.dellemc.com/driver -A -o wide
	if [ $once -gt 0 ]; then exit 0; fi
	sleep 5
done
