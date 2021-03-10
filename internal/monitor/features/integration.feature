Feature: Integration Test
  As a karavi-resiliency developer
  I want to test karavi-resiliency in a kubernetes environment
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
  Scenario Outline: Basic node failover testing using test StatefulSet pods (node interface down)
    Given a kubernetes <kubeConfig>
    And <podsPerNode> pods per node with <nVol> volumes and <nDev> devices using <driverType> and <storageClass> in <deploySecs>
    Then validate that all pods are running within <deploySecs> seconds
    When I fail <workers> worker nodes and <primary> primary nodes with <failure> failure for <failSecs> seconds
    Then validate that all pods are running within <runSecs> seconds
    And the taints for the failed nodes are removed within <nodeCleanSecs> seconds
    Then finally cleanup everything

    Examples:
      | kubeConfig | podsPerNode | nVol  | nDev  | driverType | storageClass | workers     | primary | failure         | failSecs | deploySecs | runSecs | nodeCleanSecs |
      # Small number of pods, increasing number of vols and devs
      | ""         | "1-2"       | "1-1" | "1-1" | "vxflexos" | "vxflexos"   | "one-third" | "zero"  | "interfacedown" | 120      | 60         | 120     | 120           |
      | ""         | "1-2"       | "2-2" | "2-2" | "vxflexos" | "vxflexos"   | "one-third" | "zero"  | "interfacedown" | 120      | 60         | 120     | 120           |
      | ""         | "1-2"       | "4-4" | "4-4" | "vxflexos" | "vxflexos"   | "one-third" | "zero"  | "interfacedown" | 120      | 60         | 120     | 120           |
      # Slightly more pods, increasing number of vols and devs
      | ""         | "3-5"       | "1-1" | "1-1" | "vxflexos" | "vxflexos"   | "one-third" | "zero"  | "interfacedown" | 120      | 60         | 180     | 240           |
      | ""         | "3-5"       | "2-2" | "2-2" | "vxflexos" | "vxflexos"   | "one-third" | "zero"  | "interfacedown" | 120      | 60         | 180     | 240           |
      | ""         | "3-5"       | "4-4" | "4-4" | "vxflexos" | "vxflexos"   | "one-third" | "zero"  | "interfacedown" | 120      | 60         | 180     | 240           |

  @integration
  Scenario Outline: Basic node failover testing using test StatefulSet pods (node slow reboots)
    Given a kubernetes <kubeConfig>
    And <podsPerNode> pods per node with <nVol> volumes and <nDev> devices using <driverType> and <storageClass> in <deploySecs>
    Then validate that all pods are running within <deploySecs> seconds
    When I fail <workers> worker nodes and <primary> primary nodes with <failure> failure for <failSecs> seconds
    Then validate that all pods are running within <runSecs> seconds
    And the taints for the failed nodes are removed within <nodeCleanSecs> seconds
    Then finally cleanup everything

    Examples:
      | kubeConfig | podsPerNode | nVol  | nDev  | driverType | storageClass | workers     | primary | failure  | failSecs | deploySecs | runSecs | nodeCleanSecs |
      # Small number of pods, increasing number of vols and devs
      | ""         | "1-2"       | "1-1" | "1-1" | "vxflexos" | "vxflexos"   | "one-third" | "zero"  | "reboot" | 120      | 60         | 120     | 120           |
      | ""         | "1-2"       | "2-2" | "2-2" | "vxflexos" | "vxflexos"   | "one-third" | "zero"  | "reboot" | 120      | 60         | 120     | 120           |
      | ""         | "1-2"       | "4-4" | "4-4" | "vxflexos" | "vxflexos"   | "one-third" | "zero"  | "reboot" | 120      | 60         | 120     | 120           |
      # Slightly more pods, increasing number of vols and devs
      | ""         | "3-5"       | "1-1" | "1-1" | "vxflexos" | "vxflexos"   | "one-third" | "zero"  | "reboot" | 120      | 60         | 180     | 240           |
      | ""         | "3-5"       | "2-2" | "2-2" | "vxflexos" | "vxflexos"   | "one-third" | "zero"  | "reboot" | 120      | 60         | 180     | 240           |
      | ""         | "3-5"       | "4-4" | "4-4" | "vxflexos" | "vxflexos"   | "one-third" | "zero"  | "reboot" | 120      | 60         | 180     | 240           |
