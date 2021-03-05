Feature: Integration Test
  As a podmon developer
  I want test podmon in a kubernetes environment
  So that it is known to work on various pod clean up cases and give consistent results

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


  @integration
  Scenario Outline: Basic node failover testing using podmontest
    Given a kubernetes <kubeConfig>
    And <podsPerNode> pods per node with <nVol> volumes and <nDev> devices using <driverType> and <storageClass> in <deploySecs>
    Then validate that all pods are running within <deploySecs> seconds
    When I fail <workers> worker nodes and <primary> primary nodes with <failure> failure for <failSecs> seconds
    Then validate that all pods are running within <runSecs> seconds
    And the taints for the failed nodes are removed within <nodeCleanSecs> seconds
    Then finally cleanup everything

    Examples:
      | kubeConfig | podsPerNode | nVol  | nDev  | driverType | storageClass | workers     | primary | failure         | failSecs | deploySecs | runSecs | nodeCleanSecs |
      | ""         | "2-4"       | "1-2" | "1-2" | "vxflexos" | "vxflexos"   | "one-third" | "zero"  | "interfacedown" | 120      | 60         | 120     | 120           |
