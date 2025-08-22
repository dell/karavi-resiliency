# Karavi Resilience Integration Tests

## Overview

This directory contains integration tests for the Karavi Resiliency project.
These tests are designed to validate the functionality of the project in a realistic Kubernetes environment.

Several methods are implemented for simulating various environment failures during the tests.
As a method of simulating node and network failure [custom scripts](../../test/sh) will be used to take the network interface
and/or kubelet daemon offline.
`iptables` entries are also used as a less intrusive method of simulating a network failure.

## Test Structure

The integration tests are written using the [godog framework](https://github.com/cucumber/godog).
The test scenarios can be found under the `features` folder in the [`integration.feature`](./features/integration.feature) file.
Refer to [`integration_steps_test.go`](./integration_steps_test.go) for individual step implementation.

## Dependencies

- A Kubernetes or OpenShift cluster
- A Container Storage Modules CSI driver installed on the cluster with karavi-resiliency/podmon enabled.
- A Dell storage system corresponding with the installed CSI driver.
- `pstcli` - for PowerStore Metro volumes only
    - A command line tool used to test PowerStore metro volumes with the `powerstore-metro-integration` [Makefile](./Makefile) target.
    - It is used to get the iSCSI IP addresses for PowerStore arrays in order to drop packets from those addresses, simulating a broken network connection, or a storage array that is offline or unavailable.
    - Executable downloads available, [here](https://www.dell.com/support/home/en-us/drivers/driversdetails?driverId=NNTWN).

## Running the Tests
### Test Setup

These tests rely on a custom test image, `podmontest`.
Navigate to [`karavi-resiliency/test/podmontest`](../../test/podmontest), and compile the code, build the image, and push it to your image registry.
```bash
REGISTRY=your-registry.com make build docker push
```

### Test Execution
To run the integration tests, reference the [Makefile](./Makefile) in this directory. There are several targets for running different scenarios based on the CSI driver and its feature set.

Certain configuration information is passed via environment variables. The tests require `ssh` access to the cluster worker nodes in order to simulate node failure via [custom scripts](../../test/sh), and a registry from which to pull the `podmontest` image.
- `NODE_USER`: the user of the Kubernetes worker node.
- `PASSWORD`: the password for the user of the Kubernetes worker node.
- `REGISTRY_HOST`: the registry from which a Kubernetes node can pull the `podmontest` test image. Should be the same as `REGISTRY` provided when building `podmontest`.

> Test execution example:
> ```bash
> NODE_USER=user PASSWORD=password REGISTRY_HOST=your-registry.com make unity-integration-test
> ```
