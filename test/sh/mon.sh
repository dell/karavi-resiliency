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

alias k=kubectl
while true;
do
	date
	k get nodes -o wide
	k get nodes -o custom-columns=NAME:.metadata.name,TAINTS:.spec.taints
	k get leases $vn
	k get pods $vn -o wide
	k get pods -l podmon.dellemc.com/driver -A -o wide
	sleep 5
done
