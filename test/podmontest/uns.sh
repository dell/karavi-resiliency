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

instances=${instances:-4}
prefix="pmt"
remove_all=""

for param in $*
do
    case $param in
       "--instances")
          shift
          instances=$1
          shift
          ;;
       "--prefix")
          shift
          prefix=$1
          shift
          ;;
       "--all")
          shift
          remove_all=$1
          shift
          ;;
    esac
done

if [ "$remove_all"x != "x" ]; then
  instances=$(kubectl get pods -l "podmon.dellemc.com/driver=csi-${remove_all}" -A | grep -c "$prefix")
fi

i=1
while [ $i -le $instances ]; do
  helm delete -n "${prefix}"$i "${prefix}"$i &
  i=$((i + 1))
done
wait

i=1
while [ $i -le $instances ]; do
  kubectl delete namespace "${prefix}"$i &
  i=$((i + 1))
done
wait
