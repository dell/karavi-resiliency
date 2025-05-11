/*
* Copyright (c) 2021-2025 Dell Inc., or its subsidiaries. All Rights Reserved.
*
* Licensed under the Apache License, Version 2.0 (the "License");
* you may not use this file except in compliance with the License.
* You may obtain a copy of the License at
*
* http://www.apache.org/licenses/LICENSE-2.0
*
* Unless required by applicable law or agreed to in writing, software
* distributed under the License is distributed on an "AS IS" BASIS,
* WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
* See the License for the specific language governing permissions and
* limitations under the License.
 */
 
#!/bin/bash

# This script runs the longevity tests for CSI-Drivers that support Resiliency Module with OpenShift Virtualization

# Execution:
# ./longevity_opvirt.sh --driver <driver> --iterations <iterations> --isOCPVirt <true/false>
# E.g. for CSI-PowerFlex driver:
# ./longevity_opvirt.sh --driver powerflex --iterations 10 --isOCPVirt true

# Arguments passed during the script execution:
# driver: Name of the CSI-DRiver, value: powerflex/powerstore/powermax/powerscale
# iterations: Number of iterations, value: 10,20,30...
# isOCPVirt: Boolena value, value: true/false

# Default values (optional)
driver=""
iterations=0
isOCPVirt=false


# Function to comment out lines matching a pattern
comment_out() {
  pattern=$1
  sed -i "/$pattern/ s/^/# /" run.integration
}

# Function to uncomment lines matching a pattern
uncomment() {
  pattern=$1
  sed -i "/$pattern/ s/^# //" run.integration
}

# Parse arguments
while [[ $# -gt 0 ]]; do
  key="$1"

  case $key in
    --driver)
      driver="$2"
      shift # past argument
      shift # past value
      ;;
    --iterations)
      iterations="$2"
      shift
      shift
      ;;
    --isOCPVirt)
      isOCPVirt="$2"
      shift
      shift
      ;;
    *)    # unknown option
      echo "Unknown option: $1"
      exit 1
      ;;
  esac
done

# Debug print (optional)
echo "Driver: $driver"
echo "Iterations: $iterations"
echo "Is OCP Virt: $isOCPVirt"

# Verify OPS environent and OpenShift virtualization operator installation
if [[ $isOCPVirt == true ]]; then

  if kubectl get clusterversions.config.openshift.io &>/dev/null; then
    echo "OpenShift (OCP) cluster identified."
    
    virtualization_operator_version=$(oc describe kubevirt.kubevirt.io kubevirt-kubevirt-hyperconverged -n openshift-cnv | grep "Operator Version" | awk '{print $3}')
    if [[ -z "$virtualization_operator_version" ]]; then
        print_fail "Openshift Virtualization operator not found on the cluster."
        exit 1
    else
        echo "OpenShift Virtualization Operator Version: $virtualization_operator_version"

        # update run.integration to execute E2E tests for virtualized workloads
        sed -i 's/make "\${storage_type}-integration-test"/make "\${storage_type}-vm-integration-test"/' run.integration
    fi
  else
    echo "Given cluster is not an OpenShift (OCP) cluster, these tests are not applicable."
    exit 1
  fi  
fi

# Replace default configurations in run.integration script
comment_out "source"
original_iterations=$(grep -oP '^ITERATIONS=\K\d+' run.integration)
sed -i "s/^ITERATIONS=$original_iterations/ITERATIONS=$iterations/" run.integration

if [[ $driver == "powerflex" ]]; then
  comment_out "unity"
  comment_out "powerscale"
  comment_out "powerstore"
  comment_out "powermax"
elif [[ $driver == "unity" ]]; then
  comment_out "powerflex"
  comment_out "powerscale"
  comment_out "powerstore"
  comment_out "powermax"
elif [[ $driver == "powerscale" ]]; then
  comment_out "powerflex"
  comment_out "unity"
  comment_out "powerstore"
  comment_out "powermax"
elif [[ $driver == "powerstore" ]]; then
  comment_out "powerflex"
  comment_out "unity"
  comment_out "powerscale"
  comment_out "powermax"
elif [[ $driver == "powermax" ]]; then
  comment_out "powerflex"
  comment_out "unity"
  comment_out "powerscale"
  comment_out "powerstore"
fi

# Execute E2E tests only for the specific driver
export NODE_USER='root' && export PASSWORD='dangerous' && export OPENSHIFT_BASTION='$driver_node' && export REGISTRY_HOST='10.247.98.98' && export REGISTRY_PORT='5000' && export PODMON_VERSION='nightly'
sh run.integration | tee karavi-resiliency-int-test.log; exit \${PIPESTATUS[0]}

returnCode=$?
if [[ $returnCode == 0 ]]; then
    echo "Resiliency Longevity test(s) passed for $driver driver."
fi

# Revert the changes done in run.integration
uncomment "source"
sed -i "s/^ITERATIONS=$iterations/ITERATIONS=$original_iterations/" run.integration

if [[ $driver == "powerflex" ]]; then
  uncomment "unity"
  uncomment "powerscale"
  uncomment "powerstore"
  uncomment "powermax"
elif [[ $driver == "unity" ]]; then
  uncomment "powerflex"
  uncomment "powerscale"
  uncomment "powerstore"
  uncomment "powermax"
elif [[ $driver == "powerscale" ]]; then
  uncomment "powerflex"
  uncomment "unity"
  uncomment "powerstore"
  uncomment "powermax"
elif [[ $driver == "powerstore" ]]; then
  uncomment "powerflex"
  uncomment "unity"
  uncomment "powerscale"
  uncomment "powermax"
elif [[ $driver == "powermax" ]]; then
  uncomment "powerflex"
  uncomment "unity"
  uncomment "powerscale"
  uncomment "powerstore"
fi

sed -i 's/make "\${storage_type}-vm-integration-test"/make "\${storage_type}-integration-test"/' run.integration

exit $returnCode