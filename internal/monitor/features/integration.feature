Feature: Integration Test
  As a podmon developer
  I want test podmon in a kubernetes environment
  So that it is known to work on various pod clean up cases and give consistent results

  @int-setup-check
  Scenario Outline: Validate that we have a valid k8s configuration for the integration tests
    Given a kubernetes <kubeConfig>
    Then these CSI driver <driverNames> are configured on the system
    And there is a <namespace> in the cluster
    And there are driver pods in <namespace> with this <name> prefix
    Examples:
      | kubeConfig | driverNames                | namespace  | name       |
      | ""         | "csi-vxflexos.dellemc.com" | "vxflexos" | "vxflexos" |


  @integration
  Scenario Outline: Basic node failover testing using podmontest
    Given a kubernetes <kubeConfig>
    And <podsPerNode> pods per node with <nVol> volumes and <nDev> devices using <driverType>
    When I fail <workers> worker nodes and <primary> primary nodes with <failure> failure for <failSecs> seconds
    Then validate that all pods are running within <runSecs> seconds
    And the taints for the failed nodes are removed within <nodeCleanSecs> seconds
    Then finally cleanup everything

    Examples:
      | kubeConfig | podsPerNode | nVol  | nDev  | driverType | workers     | primary | failure         | failSecs | runSecs | nodeCleanSecs |
      | ""         | "1-2"       | "1-2" | "1-2" | "vxflexos" | "one-third" | "zero"  | "interfacedown" | 10       | 10      | 10            |