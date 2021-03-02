Feature: Integration Test
  As a podmon developer
  I want test podmon in a kubernetes environment
  So that it is known to work on various pod clean up cases and give consistent results

  @int-setup-check
  Scenario Outline: Validate that we can k8s configuration that can be used for integration testing
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
    And <podsPerNode> pods per node with <nVol> volumes and <nDev> devices each
    When I fail <nodes> nodes and <masters> master nodes with <failure> failure for <failSecs> seconds
    Then all pods are running within <runSecs> seconds
    And the taints for the failed nodes are removed within <nodeCleanSecs> seconds

    Examples:
      | kubeConfig | podsPerNode | nVol  | nDev  | nodes       | masters | failure         | failSecs | runSecs | nodeCleanSecs |
      | ""         | "1-15"      | "1-8" | "1-4" | "one-third" | "zero"  | "interfacedown" | 150      | 300     | 800           |
#      | ""         | "1-15"      | "1-8" | "1-4" | "one"       | "one"   | "reboot"        | 150 | 300     | 800           |
#      | ""         | "1-15"      | "1-8" | "1-4" | "all"       | "all"   | "kubelet"       | 150 | 300     | 800           |
#      | ""         | "1-15"      | "1-8" | "1-4" | "two"       | "all"   | "kubelet"       | 150 | 300     | 800           |
#      | ""         | "1-15"      | "1-8" | "1-4" | "three"     | "all"   | "kubelet"       | 150 | 300     | 800           |
#      | ""         | "1-15"      | "1-8" | "1-4" | "five"      | "all"   | "kubelet"       | 150 | 300     | 800           |
#      | ""         | "1-15"      | "1-8" | "1-4" | "two-third" | "all"   | "kubelet"       | 150 | 300     | 800           |

#  Scenario Outline: Cascading multinode failure
#    Given a kubernetes <kubeConfig>
#    And <podsPerNode> pods per node with <nVol> volumes and <nDev> devices each
#    When I fail <nodes> nodes and <masters> master nodes with <failure> failure for <failSecs> seconds
#    And I wait <waitsec> and fail <othernodes> nodes
#    Then all pods are running within <runSecs> seconds
#    And the taints for the failed nodes are removed within <nodeCleanSecs> seconds
#    Examples:
#      | podsPerNode | nVol | nDev | nodes       | masters | failure         | runSecs | nodeCleanSecs |
#      | "1-15"        | "1-8"  | "1-4"  | "one-third" | "zero"  | "interfacedown" | 300     | 800           |
#      | "1-15"        | "1-8"  | "1-4"  | "one"       | "one"   | "reboot"        | 300     | 800           |
#      | "1-15"        | "1-8"  | "1-4"  | "all"       | "all"   | "kubelet"       | 300     | 800           |
#      | "1-15"        | "1-8"  | "1-4"  | "two"       | "all"   | "kubelet"       | 300     | 800           |
#      | "1-15"        | "1-8"  | "1-4"  | "three"     | "all"   | "kubelet"       | 300     | 800           |
#      | "1-15"        | "1-8"  | "1-4"  | "five"      | "all"   | "kubelet"       | 300     | 800           |
#      | "1-15"        | "1-8"  | "1-4"  | "two-third" | "all"   | "kubelet"       | 300     | 800           |
# 
# Scenario Outline:  Postgres deployment similar to Basic case above
#    Given a kubernetes <kubeConfig>
#    And <podsPerNode> pods per node with <nVol> volumes and <nDev> devices each
#    When I fail <nodes> nodes and <masters> master nodes with <failure> failure for <failSecs> seconds
#    And I wait <waitsec> and fail <othernodes> nodes
#    Then all pods are running within <runSecs> seconds
#    And the taints for the failed nodes are removed within <nodeCleanSecs> seconds
#    Examples:
#      | podsPerNode | nVol | nDev | nodes       | masters | failure         | runSecs | nodeCleanSecs |
#      | "1-15"        | "1-8"  | "1-4"  | "one-third" | "zero"  | "interfacedown" | 300     | 800           |
#      | "1-15"        | "1-8"  | "1-4"  | "one"       | "one"   | "reboot"        | 300     | 800           |
#      | "1-15"        | "1-8"  | "1-4"  | "all"       | "all"   | "kubelet"       | 300     | 800           |
#      | "1-15"        | "1-8"  | "1-4"  | "two"       | "all"   | "kubelet"       | 300     | 800           |
#      | "1-15"        | "1-8"  | "1-4"  | "three"     | "all"   | "kubelet"       | 300     | 800           |
#      | "1-15"        | "1-8"  | "1-4"  | "five"      | "all"   | "kubelet"       | 300     | 800           |
#      | "1-15"        | "1-8"  | "1-4"  | "two-third" | "all"   | "kubelet"       | 300     | 800           |
