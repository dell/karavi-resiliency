# Integration test
The Karavi Resiliency integration uses a [Gherkin](https://cucumber.io/docs/gherkin) feature file to define the integration test cases in BDD format. It uses [Godog](https://github.com/cucumber/godog) to run the defined tests. The integration tests, unlike the unit tests, are expected to work with a real Kubernetes cluster. The prerequisites are listed below. 

The tests will deploy [test pods](../test/podmontest) on the system and run the failure scenarios. These failure scenarios require that systems are accessible using username/password credentials. Also note, that the tests will bring down network interfaces and/or reboot any of the Kubernetes cluster's nodes.

# Prerequisites

You will need to have the following deployed in your Kubernetes cluster:
* A Karavi Resiliency supported CSI Driver (see the appropriate documentation for the driver).
* CSI Driver should have the podmon sidecar enabled (see [Getting Started Guide](./GETTING_STARTED_GUIDE.md)).
* Storage classes created using the supported CSI Driver. Specific driver names are called out in the test scenarios.

You will need to find out the following before running the test:
* Kubeconfig for your cluster(s). By default, the test will look in `<user-homedir>/.kube/config`.
* Username and password of the nodes (at least the worker nodes).

These environmental variables need to be set:

| Variable | Required | Description | Set to |
|----------|----------|-------------|--------|
| RESILIENCY_INT_TEST | Yes |  This is a flag for running the test. Since the go test will run any _test.go file, we need a way for the integration test to be run only when specifically needed. This variable achieves this requirement. | "true" |
| NODE_USER | Yes | The username to use for scp'ing failure test scripts and ssh'ing to invoke the failure scripts. _It is assumed all hosts can be accessible with the same username_ | _Appropriate value for your test hosts_ |
| PASSWORD | Yes | The password to use for scp'ing failure test scripts and ssh'ing to invoke the failure scripts. _It is assumed all hosts can be accessible with the same password_ | _Appropriate value for your test hosts_ |
| SCRIPTS_DIR | Yes | The full path to the Karavi Resiliency test scripts from the machine that you are invoking the integration test. | For example if you've cloned the karavi-resiliency repo to /workspace/karavi-resiliency, then this value should be _/workspace/karavi-resiliency/test/sh_ | 
| POLL_K8S | No |When enabled, will run a background poller that dumps status of the nodes and test pods.  | "true" |

# Running 

Before running, make sure no test pods are running in the cluster. For example, if you rerun the test, the test would not know about test pods that may have been left over from a previous run. So, check for any before starting, example command:

```shell
 kubectl get pods -l podmon.dellemc.com/driver=csi-vxflexos -A -o wide
```

After validating that the cluster is clean, go to the internal/monitor directory, example:
```shell
cd /workspace/karavi-resiliency/internal/monitor
```

Set your environmental variables (one time):
```shell
export RESILIENCY_INT_TEST="true"
export NODE_USER="username"
export PASSWORD="password"
export SCRIPTS_DIR="/workspace/karavi-resiliency/test/sh"
# Optionally:
export POLL_K8S="true"
```

Invoke the make rule:
```shell
make integration-test
```

# Feature file

The test configuration is specified in a [Gherkin BDD](https://cucumber.io/docs/gherkin/) format in the [integration.feature](../internal/monitor/features/integration.feature) file. Each test `Scenario Outline` captures an overarching test condition we would like to put the system under for testing. The goal is for each `Scenario Outline` to be as human-readable as possible. For each `Scenario Outline`, there are `Examples` used to fill in different parameters. Each example is a separate test and all the steps (e.g., `When ...`, `And ...`, `Then ...`) for the `Scenario Outline` will be executed.

## Anatomy Of A Test

Let's take the setup test and deconstruct it in plain language:
```gherkin
  @int-setup-check
  Scenario Outline: Validate that we have a valid k8s configuration for the integration tests
    Given a kubernetes <kubeConfig>
    And test environmental variables are set
    And these CSI driver <driverNames> are configured on the system
    And these storageClasses <storageClasses> exist in the cluster
    And there is a <namespace> in the cluster
    And there are driver pods in <namespace> with this <name> prefix
    And can logon to nodes and drop test scripts
    Examples:
      | kubeConfig | driverNames                | namespace  | name       | storageClasses          |
      | ""         | "csi-vxflexos.dellemc.com" | "vxflexos" | "vxflexos" | "vxflexos,vxflexos-xfs" |
```

The `@int-setup-check` is an annotation used in our test runner to have this run as a special test. The other scenarios will have `@integration` as the annotation to indicate that they will run scenarios on the cluster. In this specific case, as the `Scenario Outline` describes we want to validate the Kubernetes cluster is good and ready to use.

Under the `Examples:` we have the following which represents the testing parameters. These will be referenced in one or more steps:

`| kubeConfig | driverNames | namespace | name | storageClasses |`

Looking at the steps, the first step `Given a Kubernetes <kubeConfig>` uses the `kubeConfig` parameter as a file path value to the kubeconfig file. If an empty string is used, then the default `<user-homedir>/.kube/config` is assumed to the path. 

Each of the subsequent steps will be executed per `Example` row. So, if we put it all together, then the setup test does the following:
* Tries connecting to the Kubernetes cluster using the kubeconfig found at `kubeConfig`.
* Checks if the required environmental variables are set.
* Checks if the specified driverName is installed in the cluster.
* Checks if the specified comma delimited string of `storageClasses` exist in the cluster.
* Checks if there is a specified `namespace` existing in the cluster. This should be the namespace associated with the installed CSI driver.
* Checks if the driver pods in the `namespace` have the expected CSI driver pods running, and the podmon sidecar is running.
* Checks if it can access each node in the cluster. If so, it will copy over the scripts from `SCRIPTS_DIR` into a directory on the node.

If the setup test does not pass for any reason, the subsequent `@integration` test scenarios will not be run.

Let's now take a look at an actual testing scenario:

```gherkin
  @integration
  Scenario Outline: Basic node failover testing using test StatefulSet pods (node interface down)
    Given a kubernetes <kubeConfig>
    And cluster is clean of test pods
    And wait <nodeCleanSecs> to see there are no taints
    And <podsPerNode> pods per node with <nVol> volumes and <nDev> devices using <driverType> and <storageClass> in <deploySecs>
    Then validate that all pods are running within <deploySecs> seconds
    When I fail <workers> worker nodes and <primary> primary nodes with <failure> failure for <failSecs> seconds
    Then validate that all pods are running within <runSecs> seconds
    And the taints for the failed nodes are removed within <nodeCleanSecs> seconds
    Then finally cleanup everything

    Examples:
      | kubeConfig | podsPerNode | nVol  | nDev  | driverType | storageClass | workers | primary | failure | failSecs | deploySecs | runSecs | nodeCleanSecs |
```

All the previous descriptions about the `Examples` apply, except that these are applied to a failure test scenario. In the above, the test does the following:

* Tries connecting to the Kubernetes cluster using the kubeconfig found at `kubeConfig`.
* Removes any existing test pods that may have been left over from a previous test.
* Waits `nodeCleanSecs` for all the nodes to be ready, and the podmon taints to be clear. This validation allows the test to run in a clean state.
* Deploys test pods with given number of pods, volumes, and devices. The volumes will be based on the `driverType`.
* Expects the test pods to be running with in `deploySecs`.
* Induces a failure of the specified `failure` type against a number of `workers` and `primary` nodes for a total of `failSecs`. The `workers` and `primary` values are either ratios written in English or numbers, e.g. "one-third", "1", "2". Whether ratio or number, ultimately a number of nodes will be determined to be failed. The test will randomly choose hosts to fulfill that number of failed nodes.
  * Accepted ratio values are: "one-fourth", "1/4", "one-third", "1/3", "one-half", "1/2", "two-thirds", "2/3".
* Expects the test pods to be running with in `deploySecs`. This is to validate that the fail over of the test pods (if any) occurred, and they are in good condition again after the node failure(s).
* Checks that all nodes are in the 'Ready' state and that the podmon taints are removed from the nodes after `nodeCleanSecs`.
* Deletes all test pods from the cluster.

# Troubleshooting

* In case of any missing prerequisites, such as a required environmental variables or a driver dependencies, the test will fail with an appropriate message pointing to what it expected that was missing.

* If you run into an error that points to a problem accessing a file or directory, check if the `SCRIPTS_DIR` is pointing the location of the Karavi Resiliency test scripts.

* If you see early errors in the tests indicating that it failed to create a session, check if you have the right credentials.

* If you see errors later on in the tests indicating that it failed to create a session, check if all the Kubernetes nodes are up and running. It could be that a node has not come back up after a test failure invocation.