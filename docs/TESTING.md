<!--
Copyright (c) 2021 Dell Inc., or its subsidiaries. All Rights Reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0
-->

# Testing Methlodolog and Results

A three tier testing methodology is used for Karavi Resiliency:

1. Unit testing with high coverage (>90% statement) tests the program logic and is especially used to test the error paths by injecting faults.
2. An integration test describes test scenarios in Gherkin that sets up specific testing scenarios executed against a Kubernetes test cluster. The tests use ranges for many of the parameters to add an element of "chaos testing".
3. Script based testing supports longevity testing in a kubernetes cluster. For example, one test repeatedly fails three different lists of nodes in succession and is used to fail 1/3 of the cluster's worker nodes on a cyclic basis and repeat indefinitely. This test collect statistics on length of time for pod evacuation, pod recovery, and node cleanup.