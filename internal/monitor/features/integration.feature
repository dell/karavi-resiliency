Feature: Integration Test
  As a CSM for Resiliency developer
  I want to test CSM for Resiliency in a kubernetes environment
  So that it is known to work on various pod clean up cases and give consistent results

  @powerflex-int-setup-check
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

  @unity-int-setup-check
  Scenario Outline: Validate that we have a valid k8s configuration for the integration tests
    Given a kubernetes <kubeConfig>
    And test environmental variables are set
    And these CSI driver <driverNames> are configured on the system
    And these storageClasses <storageClasses> exist in the cluster
    And there is a <namespace> in the cluster
    And there are driver pods in <namespace> with this <name> prefix
    And can logon to nodes and drop test scripts
    Examples:
      | kubeConfig | driverNames             | namespace | name    | storageClasses          |
      | ""         | "csi-unity.dellemc.com" | "unity"   | "unity" | "unity-iscsi,unity-nfs" |

  @powerscale-int-setup-check
  Scenario Outline: Validate that we have a valid k8s configuration for the integration tests
    Given a kubernetes <kubeConfig>
    And test environmental variables are set
    And these CSI driver <driverNames> are configured on the system
    And these storageClasses <storageClasses> exist in the cluster
    And there is a <namespace> in the cluster
    And there are driver pods in <namespace> with this <name> prefix
    And can logon to nodes and drop test scripts
    Examples:
      | kubeConfig | driverNames             | namespace | name    | storageClasses          |
      | ""         | "csi-isilon.dellemc.com" | "isilon"   | "isilon" | "isilon" |
  
  @powerstore-int-setup-check
  Scenario Outline: Validate that we have a valid k8s configuration for the integration tests
    Given a kubernetes <kubeConfig>
    And test environmental variables are set
    And these CSI driver <driverNames> are configured on the system
    And these storageClasses <storageClasses> exist in the cluster
    And there is a <namespace> in the cluster
    And there are driver pods in <namespace> with this <name> prefix
    And can logon to nodes and drop test scripts
    Examples:
      | kubeConfig | driverNames                  | namespace      | name         | storageClasses                                       |
      | ""         | "csi-powerstore.dellemc.com" | "powerstore"   | "powerstore" | "powerstore-nfs,powerstore-iscsi,powerstore-nvmetcp" |

  @powermax-int-setup-check
  Scenario Outline: Validate that we have a valid k8s configuration for the integration tests
    Given a kubernetes <kubeConfig>
    And test environmental variables are set
    And these CSI driver <driverNames> are configured on the system
    And these storageClasses <storageClasses> exist in the cluster
    And there is a <namespace> in the cluster
    And there are driver pods in <namespace> with this <name> prefix
    And can logon to nodes and drop test scripts
    Examples:
      | kubeConfig | driverNames                  | namespace    | name       | storageClasses                |
      | ""         | "csi-powermax.dellemc.com"   | "powermax"   | "powermax" | "powermax-iscsi, powermax-fc" |

  @powerflex-integration
  Scenario Outline: Basic node failover testing using test StatefulSet pods (node interface down)
    Given a kubernetes <kubeConfig>
    And cluster is clean of test pods
    And wait <nodeCleanSecs> to see there are no taints
    And <podsPerNode> pods per node with <nVol> volumes and <nDev> devices using <driverType> and <storageClass> in <deploySecs>
    Then validate that all pods are running within <deploySecs> seconds
    When I fail <workers> worker nodes and <primary> primary nodes with <failure> failure for <failSecs> seconds
    Then validate that all pods are running within <runSecs> seconds
    And labeled pods are on a different node
    And the taints for the failed nodes are removed within <nodeCleanSecs> seconds
    Then finally cleanup everything

    Examples:
      | kubeConfig | podsPerNode | nVol  | nDev  | driverType | storageClass | workers     | primary | failure         | failSecs | deploySecs | runSecs | nodeCleanSecs |
      # Small number of pods, increasing number of vols and devs
      | ""         | "1-2"       | "1-1" | "1-1" | "vxflexos" | "vxflexos"   | "one-third" | "zero"  | "interfacedown" | 120      | 240        | 300     | 300           |
      | ""         | "1-2"       | "2-2" | "2-2" | "vxflexos" | "vxflexos"   | "one-third" | "zero"  | "interfacedown" | 120      | 240        | 300     | 300           |
      | ""         | "1-2"       | "4-4" | "4-4" | "vxflexos" | "vxflexos"   | "one-third" | "zero"  | "interfacedown" | 120      | 240        | 300     | 300           |
      # Slightly more pods, increasing number of vols and devs
      | ""         | "3-5"       | "1-1" | "1-1" | "vxflexos" | "vxflexos"   | "one-third" | "zero"  | "interfacedown" | 240      | 240        | 300     | 600           |
      | ""         | "3-5"       | "2-2" | "2-2" | "vxflexos" | "vxflexos"   | "one-third" | "zero"  | "interfacedown" | 240      | 240        | 300     | 600           |
      | ""         | "3-5"       | "4-4" | "4-4" | "vxflexos" | "vxflexos"   | "one-third" | "zero"  | "interfacedown" | 240      | 240        | 300     | 600           |

@powerflex-integration
  Scenario Outline: Basic node failover testing using test StatefulSet pods (node kubelet down)
    Given a kubernetes <kubeConfig>
    And cluster is clean of test pods
    And wait <nodeCleanSecs> to see there are no taints
    And <podsPerNode> pods per node with <nVol> volumes and <nDev> devices using <driverType> and <storageClass> in <deploySecs>
    Then validate that all pods are running within <deploySecs> seconds
    When I fail <workers> worker nodes and <primary> primary nodes with <failure> failure for <failSecs> seconds
    Then validate that all pods are running within <runSecs> seconds
    And labeled pods are on a different node
    And the taints for the failed nodes are removed within <nodeCleanSecs> seconds
    Then finally cleanup everything

    Examples:
      | kubeConfig | podsPerNode | nVol  | nDev  | driverType | storageClass | workers     | primary | failure       | failSecs | deploySecs | runSecs | nodeCleanSecs |
      # Small number of pods, increasing number of vols and devs
      | ""         | "1-2"       | "1-1" | "1-1" | "vxflexos" | "vxflexos"   | "one-third" | "zero"  | "kubeletdown" | 600      | 900        | 900     | 900           |
      | ""         | "1-2"       | "2-2" | "2-2" | "vxflexos" | "vxflexos"   | "one-third" | "zero"  | "kubeletdown" | 600      | 900        | 900     | 900           |
      | ""         | "1-2"       | "4-4" | "4-4" | "vxflexos" | "vxflexos"   | "one-third" | "zero"  | "kubeletdown" | 600      | 900        | 900     | 900           |
      # Slightly more pods, increasing number of vols and devs
      | ""         | "3-5"       | "1-1" | "1-1" | "vxflexos" | "vxflexos"   | "one-third" | "zero"  | "kubeletdown" | 600      | 900        | 900     | 900           |
      | ""         | "3-5"       | "2-2" | "2-2" | "vxflexos" | "vxflexos"   | "one-third" | "zero"  | "kubeletdown" | 600      | 900        | 900     | 900           |
      | ""         | "3-5"       | "4-4" | "4-4" | "vxflexos" | "vxflexos"   | "one-third" | "zero"  | "kubeletdown" | 600      | 900        | 900     | 900           |

  @unity-integration
  Scenario Outline: Basic node failover testing using test StatefulSet pods (node interface down)
    Given a kubernetes <kubeConfig>
    And cluster is clean of test pods
    And wait <nodeCleanSecs> to see there are no taints
    And <podsPerNode> pods per node with <nVol> volumes and <nDev> devices using <driverType> and <storageClass> in <deploySecs>
    Then validate that all pods are running within <deploySecs> seconds
    When I fail <workers> worker nodes and <primary> primary nodes with <failure> failure for <failSecs> seconds
    Then validate that all pods are running within <runSecs> seconds
    And labeled pods are on a different node
    And the taints for the failed nodes are removed within <nodeCleanSecs> seconds
    Then finally cleanup everything

    Examples:
      | kubeConfig | podsPerNode | nVol  | nDev  | driverType | storageClass  | workers     | primary | failure         | failSecs | deploySecs | runSecs | nodeCleanSecs |
      # Small number of pods, increasing number of vols and devs
      | ""         | "1-2"       | "1-1" | "1-1" | "unity"    | "unity-iscsi" | "one-third" | "zero"  | "interfacedown" | 600      | 900        | 900     | 900           |
      # Slightly more pods, increasing number of vols and devs
      | ""         | "3-5"       | "2-2" | "2-2" | "unity"    | "unity-iscsi" | "one-third" | "zero"  | "interfacedown" | 600      | 900        | 900     | 900           |

  @powerscale-integration
  Scenario Outline: Basic node failover testing using test StatefulSet pods (node interface down)
    Given a kubernetes <kubeConfig>
    And cluster is clean of test pods
    And wait <nodeCleanSecs> to see there are no taints
    And <podsPerNode> pods per node with <nVol> volumes and <nDev> devices using <driverType> and <storageClass> in <deploySecs>
    Then validate that all pods are running within <deploySecs> seconds
    When I fail <workers> worker nodes and <primary> primary nodes with <failure> failure for <failSecs> seconds
    Then validate that all pods are running within <runSecs> seconds
    And labeled pods are on a different node
    And the taints for the failed nodes are removed within <nodeCleanSecs> seconds
    Then finally cleanup everything

    Examples:
      | kubeConfig | podsPerNode | nVol  | nDev  | driverType | storageClass  | workers     | primary | failure         | failSecs | deploySecs | runSecs | nodeCleanSecs |
     # Small number of pods, increasing number of vols and devs
      | ""         | "1-2"       | "1-1" | "0-0" | "isilon"    | "isilon" | "one-third" | "zero"  | "interfacedown" | 120      | 600        | 600     | 600           |
      | ""         | "3-5"       | "2-2" | "0-0" | "isilon"    | "isilon" | "one-third" | "zero"  | "interfacedown" | 240      | 600        | 600     | 600           |
  
  @powerstore-integration
  Scenario Outline: Basic node failover testing using test StatefulSet pods (node interface down)
    Given a kubernetes <kubeConfig>
    And cluster is clean of test pods
    And wait <nodeCleanSecs> to see there are no taints
    And <podsPerNode> pods per node with <nVol> volumes and <nDev> devices using <driverType> and <storageClass> in <deploySecs>
    Then validate that all pods are running within <deploySecs> seconds
    When I fail <workers> worker nodes and <primary> primary nodes with <failure> failure for <failSecs> seconds
    Then validate that all pods are running within <runSecs> seconds
    And labeled pods are on a different node
    And the taints for the failed nodes are removed within <nodeCleanSecs> seconds
    Then finally cleanup everything

    Examples:
      | kubeConfig | podsPerNode | nVol  | nDev  | driverType | storageClass  | workers     | primary | failure         | failSecs | deploySecs | runSecs | nodeCleanSecs |
     # Small number of pods, increasing number of vols and devs
      | ""         | "1-2"       | "1-1" | "0-0" | "powerstore"    | "powerstore-nfs" | "one-third" | "zero"  | "interfacedown" | 120      | 600        | 600     | 600           |
      | ""         | "3-5"       | "2-2" | "0-0" | "powerstore"    | "powerstore-nfs" | "one-third" | "zero"  | "interfacedown" | 240      | 600        | 600     | 600           |
      # Small number of pods, increasing number of vols and devs
      | ""         | "1-2"       | "1-1" | "0-0" | "powerstore"    | "powerstore-iscsi" | "one-third" | "zero"  | "interfacedown" | 120      | 600        | 600     | 600           |
      | ""         | "3-5"       | "2-2" | "0-0" | "powerstore"    | "powerstore-iscsi" | "one-third" | "zero"  | "interfacedown" | 240      | 600        | 600     | 600           |
      # Small number of pods, increasing number of vols and devs
      #| ""         | "1-2"       | "1-1" | "0-0" | "powerstore"    | "powerstore-nvmetcp" | "one-third" | "zero"  | "interfacedown" | 120      | 600        | 600     | 600           |
      #| ""         | "3-5"       | "2-2" | "0-0" | "powerstore"    | "powerstore-nvmetcp" | "one-third" | "zero"  | "interfacedown" | 240      | 600        | 600     | 600           |



  @unity-integration
  Scenario Outline: Basic node failover testing using test StatefulSet pods (node interface down)
    Given a kubernetes <kubeConfig>
    And cluster is clean of test pods
    And wait <nodeCleanSecs> to see there are no taints
    And <podsPerNode> pods per node with <nVol> volumes and <nDev> devices using <driverType> and <storageClass> in <deploySecs>
    Then validate that all pods are running within <deploySecs> seconds
    When I fail <workers> worker nodes and <primary> primary nodes with <failure> failure for <failSecs> seconds
    Then validate that all pods are running within <runSecs> seconds
    And labeled pods are on a different node
    And the taints for the failed nodes are removed within <nodeCleanSecs> seconds
    Then finally cleanup everything

    Examples:
      | kubeConfig | podsPerNode | nVol  | nDev  | driverType | storageClass | workers     | primary | failure         | failSecs | deploySecs | runSecs | nodeCleanSecs |
      # Small number of pods, increasing number of vols and devs
      | ""         | "1-2"       | "1-1" | "0-0" | "unity"    | "unity-nfs"  | "one-third" | "zero"  | "interfacedown" | 600      | 900        | 900     | 900           |
      # Slightly more pods, increasing number of vols and devs
      | ""         | "3-5"       | "2-2" | "0-0" | "unity"    | "unity-nfs"  | "one-third" | "zero"  | "interfacedown" | 600      | 900        | 900     | 900           |

  @powermax-integration
  Scenario Outline: Basic node failover testing using test StatefulSet pods (node interface down)
    Given a kubernetes <kubeConfig>
    And cluster is clean of test pods
    And wait <nodeCleanSecs> to see there are no taints
    And <podsPerNode> pods per node with <nVol> volumes and <nDev> devices using <driverType> and <storageClass> in <deploySecs>
    Then validate that all pods are running within <deploySecs> seconds
    When I fail <workers> worker nodes and <primary> primary nodes with <failure> failure for <failSecs> seconds
    Then validate that all pods are running within <runSecs> seconds
    And labeled pods are on a different node
    And the taints for the failed nodes are removed within <nodeCleanSecs> seconds
    Then finally cleanup everything

    Examples:
      | kubeConfig | podsPerNode | nVol  | nDev  | driverType | storageClass  | workers     | primary | failure         | failSecs | deploySecs | runSecs | nodeCleanSecs |
     # Small number of pods, increasing number of vols and devs
      | ""         | "1-2"       | "1-1" | "0-0" | "powermax"    | "powermax-nfs" | "one-third" | "zero"  | "interfacedown" | 120      | 600        | 600     | 600           |
      | ""         | "3-5"       | "2-2" | "0-0" | "powermax"    | "powermax-nfs" | "one-third" | "zero"  | "interfacedown" | 240      | 600        | 600     | 600           |
      # Small number of pods, increasing number of vols and devs
      | ""         | "1-2"       | "1-1" | "0-0" | "powermax"    | "powermax-iscsi" | "one-third" | "zero"  | "interfacedown" | 120      | 600        | 600     | 600         |
      | ""         | "3-5"       | "2-2" | "0-0" | "powermax"    | "powermax-iscsi" | "one-third" | "zero"  | "interfacedown" | 240      | 600        | 600     | 600         |
      # Small number of pods, increasing number of vols and devs
      #| ""         | "1-2"       | "1-1" | "0-0" | "powermax"    | "powermax-nvmetcp" | "one-third" | "zero"  | "interfacedown" | 120      | 600        | 600     | 600         |
      #| ""         | "3-5"       | "2-2" | "0-0" | "powermax"    | "powermax-nvmetcp" | "one-third" | "zero"  | "interfacedown" | 240      | 600        | 600     | 600         |

  @unity-integration
  Scenario Outline: Basic node failover testing using test StatefulSet pods (node kubelet down)
    Given a kubernetes <kubeConfig>
    And cluster is clean of test pods
    And wait <nodeCleanSecs> to see there are no taints
    And <podsPerNode> pods per node with <nVol> volumes and <nDev> devices using <driverType> and <storageClass> in <deploySecs>
    Then validate that all pods are running within <deploySecs> seconds
    When I fail <workers> worker nodes and <primary> primary nodes with <failure> failure for <failSecs> seconds
    Then validate that all pods are running within <runSecs> seconds
    And labeled pods are on a different node
    And the taints for the failed nodes are removed within <nodeCleanSecs> seconds
    Then finally cleanup everything

    Examples:
      | kubeConfig | podsPerNode | nVol  | nDev  | driverType | storageClass  | workers     | primary | failure       | failSecs | deploySecs | runSecs | nodeCleanSecs |
      # Small number of pods, increasing number of vols and devs
      | ""         | "1-2"       | "1-1" | "1-1" | "unity"    | "unity-iscsi" | "one-third" | "zero"  | "kubeletdown" | 600      | 900        | 900     | 900           |
      # Slightly more pods, increasing number of vols and devs
      | ""         | "3-5"       | "2-2" | "2-2" | "unity"    | "unity-iscsi" | "one-third" | "zero"  | "kubeletdown" | 600      | 900        | 900     | 900           |

@unity-integration
  Scenario Outline: Basic node failover testing using test StatefulSet pods (node kubelet down)
    Given a kubernetes <kubeConfig>
    And cluster is clean of test pods
    And wait <nodeCleanSecs> to see there are no taints
    And <podsPerNode> pods per node with <nVol> volumes and <nDev> devices using <driverType> and <storageClass> in <deploySecs>
    Then validate that all pods are running within <deploySecs> seconds
    When I fail <workers> worker nodes and <primary> primary nodes with <failure> failure for <failSecs> seconds
    Then validate that all pods are running within <runSecs> seconds
    And labeled pods are on a different node
    And the taints for the failed nodes are removed within <nodeCleanSecs> seconds
    Then finally cleanup everything

    Examples:
      | kubeConfig | podsPerNode | nVol  | nDev  | driverType | storageClass | workers     | primary | failure       | failSecs | deploySecs | runSecs | nodeCleanSecs |
      # Small number of pods, increasing number of vols and devs
      | ""         | "1-2"       | "1-1" | "0-0" | "unity"    | "unity-nfs"  | "one-third" | "zero"  | "kubeletdown" | 600      | 900        | 900     | 900           |
      # Slightly more pods, increasing number of vols and devs
      | ""         | "3-5"       | "2-2" | "0-0" | "unity"    | "unity-nfs"  | "one-third" | "zero"  | "kubeletdown" | 600      | 900        | 900     | 900           |


@powerscale-integration
  Scenario Outline: Basic node failover testing using test StatefulSet pods (node kubelet down)
    Given a kubernetes <kubeConfig>
    And cluster is clean of test pods
    And wait <nodeCleanSecs> to see there are no taints
    And <podsPerNode> pods per node with <nVol> volumes and <nDev> devices using <driverType> and <storageClass> in <deploySecs>
    Then validate that all pods are running within <deploySecs> seconds
    When I fail <workers> worker nodes and <primary> primary nodes with <failure> failure for <failSecs> seconds
    Then validate that all pods are running within <runSecs> seconds
    And labeled pods are on a different node
    And the taints for the failed nodes are removed within <nodeCleanSecs> seconds
    Then finally cleanup everything

    Examples:
      | kubeConfig | podsPerNode | nVol  | nDev  | driverType | storageClass | workers     | primary | failure       | failSecs | deploySecs | runSecs | nodeCleanSecs |
      # Small number of pods, increasing number of vols and devs
      | ""         | "1-2"       | "1-1" | "0-0" | "isilon"    | "isilon"  | "one-third" | "zero"  | "kubeletdown" | 600      | 900        | 900     | 900           |
      # Slightly more pods, increasing number of vols and devs
      | ""         | "3-5"       | "1-1" | "0-0" | "isilon"    | "isilon"  | "one-third" | "zero"  | "kubeletdown" | 600      | 900        | 900     | 900           |

  @powerstore-integration
  Scenario Outline: Basic node failover testing using test StatefulSet pods (node kubelet down)
    Given a kubernetes <kubeConfig>
    And cluster is clean of test pods
    And wait <nodeCleanSecs> to see there are no taints
    And <podsPerNode> pods per node with <nVol> volumes and <nDev> devices using <driverType> and <storageClass> in <deploySecs>
    Then validate that all pods are running within <deploySecs> seconds
    When I fail <workers> worker nodes and <primary> primary nodes with <failure> failure for <failSecs> seconds
    Then validate that all pods are running within <runSecs> seconds
    And labeled pods are on a different node
    And the taints for the failed nodes are removed within <nodeCleanSecs> seconds
    Then finally cleanup everything

    Examples:
      | kubeConfig | podsPerNode | nVol  | nDev  | driverType | storageClass | workers     | primary | failure       | failSecs | deploySecs | runSecs | nodeCleanSecs |
      # Small number of pods, increasing number of vols and devs
      | ""         | "1-2"       | "1-1" | "0-0" | "powerstore"    | "powerstore-nfs"  | "one-third" | "zero"  | "kubeletdown" | 600      | 900        | 900     | 900           |
      # Slightly more pods, increasing number of vols and devs
      | ""         | "3-5"       | "1-1" | "0-0" | "powerstore"    | "powerstore-nfs"  | "one-third" | "zero"  | "kubeletdown" | 600      | 900        | 900     | 900           |
      # Small number of pods, increasing number of vols and devs
      | ""         | "1-2"       | "1-1" | "0-0" | "powerstore"    | "powerstore-iscsi"  | "one-third" | "zero"  | "kubeletdown" | 600      | 900        | 900     | 900           |
      # Slightly more pods, increasing number of vols and devs
      | ""         | "3-5"       | "1-1" | "0-0" | "powerstore"    | "powerstore-iscsi"  | "one-third" | "zero"  | "kubeletdown" | 600      | 900        | 900     | 900           |
      # Small number of pods, increasing number of vols and devs
      #| ""         | "1-2"       | "1-1" | "0-0" | "powerstore"    | "powerstore-nvmetcp"  | "one-third" | "zero"  | "kubeletdown" | 600      | 900        | 900     | 900           |
      # Slightly more pods, increasing number of vols and devs
      #| ""         | "3-5"       | "1-1" | "0-0" | "powerstore"    | "powerstore-nvmetcp"  | "one-third" | "zero"  | "kubeletdown" | 600      | 900        | 900     | 900           |

  @powerstore-short-integration
  Scenario Outline: Basic node failover testing using test StatefulSet pods (node kubelet down)
    Given a kubernetes <kubeConfig>
    And cluster is clean of test pods
    And wait <nodeCleanSecs> to see there are no taints
    And <podsPerNode> pods per node with <nVol> volumes and <nDev> devices using <driverType> and <storageClass> in <deploySecs>
    Then validate that all pods are running within <deploySecs> seconds
    When I fail <workers> worker nodes and <primary> primary nodes with <failure> failure for <failSecs> seconds
    Then validate that all pods are running within <runSecs> seconds
    And labeled pods are on a different node
    And the taints for the failed nodes are removed within <nodeCleanSecs> seconds
    Then finally cleanup everything

    Examples:
      | kubeConfig | podsPerNode | nVol  | nDev  | driverType | storageClass | workers     | primary | failure       | failSecs | deploySecs | runSecs | nodeCleanSecs |
      # Uncomment the storageclass to use. The default is set to nvme which is supported by nightly qualification.
      #| ""         | "1-1"       | "1-1" | "0-0" | "powerstore"    | "powerstore-nfs"  | "one-third" | "zero"  | "kubeletdown" | 600      | 900        | 900     | 900           |
      #| ""         | "1-1"       | "1-1" | "0-0" | "powerstore"    | "powerstore-iscsi"  | "one-third" | "zero"  | "kubeletdown" | 600      | 900        | 900     | 900           |
      | ""         | "1-1"       | "1-1" | "0-0" | "powerstore"    | "powerstore-nvmetcp"  | "one-third" | "zero"  | "kubeletdown" | 600      | 900        | 900     | 900           |

  @powermax-integration
  Scenario Outline: Basic node failover testing using test StatefulSet pods (node kubelet down)
    Given a kubernetes <kubeConfig>
    And cluster is clean of test pods
    And wait <nodeCleanSecs> to see there are no taints
    And <podsPerNode> pods per node with <nVol> volumes and <nDev> devices using <driverType> and <storageClass> in <deploySecs>
    Then validate that all pods are running within <deploySecs> seconds
    When I fail <workers> worker nodes and <primary> primary nodes with <failure> failure for <failSecs> seconds
    Then validate that all pods are running within <runSecs> seconds
    And labeled pods are on a different node
    And the taints for the failed nodes are removed within <nodeCleanSecs> seconds
    Then finally cleanup everything

    Examples:
      | kubeConfig | podsPerNode | nVol  | nDev  | driverType | storageClass | workers     | primary | failure       | failSecs | deploySecs | runSecs | nodeCleanSecs |
      # Small number of pods, increasing number of vols and devs
      | ""         | "1-2"       | "1-1" | "0-0" | "powermax"    | "powermax-nfs"  | "one-third" | "zero"  | "kubeletdown" | 600      | 900        | 900     | 900           |
      # Slightly more pods, increasing number of vols and devs
      | ""         | "3-5"       | "1-1" | "0-0" | "powermax"    | "powermax-nfs"  | "one-third" | "zero"  | "kubeletdown" | 600      | 900        | 900     | 900           |
      # Small number of pods, increasing number of vols and devs
      | ""         | "1-2"       | "1-1" | "0-0" | "powermax"    | "powermax-iscsi"  | "one-third" | "zero"  | "kubeletdown" | 600      | 900        | 900     | 900           |
      # Slightly more pods, increasing number of vols and devs
      | ""         | "3-5"       | "1-1" | "0-0" | "powermax"    | "powermax-iscsi"  | "one-third" | "zero"  | "kubeletdown" | 600      | 900        | 900     | 900           |
      # Small number of pods, increasing number of vols and devs
      #| ""         | "1-2"       | "1-1" | "0-0" | "powermax"    | "powermax-nvmetcp"  | "one-third" | "zero"  | "kubeletdown" | 600      | 900        | 900     | 900           |
      # Slightly more pods, increasing number of vols and devs
      #| ""         | "3-5"       | "1-1" | "0-0" | "powermax"    | "powermax-nvmetcp"  | "one-third" | "zero"  | "kubeletdown" | 600      | 900        | 900     | 900           |

  @powermax-short-integration
  Scenario Outline: Basic node failover testing using test StatefulSet pods (node kubelet down)
    Given a kubernetes <kubeConfig>
    And cluster is clean of test pods
    And wait <nodeCleanSecs> to see there are no taints
    And <podsPerNode> pods per node with <nVol> volumes and <nDev> devices using <driverType> and <storageClass> in <deploySecs>
    Then validate that all pods are running within <deploySecs> seconds
    When I fail <workers> worker nodes and <primary> primary nodes with <failure> failure for <failSecs> seconds
    Then validate that all pods are running within <runSecs> seconds
    And labeled pods are on a different node
    And the taints for the failed nodes are removed within <nodeCleanSecs> seconds
    Then finally cleanup everything

    Examples:
      | kubeConfig | podsPerNode | nVol  | nDev  | driverType | storageClass | workers     | primary | failure       | failSecs | deploySecs | runSecs | nodeCleanSecs |
      | ""         | "1-1"       | "1-1" | "0-0" | "powermax"    | "powermax-nfs"  | "one-third" | "zero"  | "kubeletdown" | 600      | 900        | 900     | 900           |
      | ""         | "1-1"       | "1-1" | "1-1" | "powermax"    | "powermax-iscsi"  | "one-third" | "zero"  | "kubeletdown" | 600      | 900        | 900     | 900           |
      #| ""         | "1-1"       | "1-1" | "0-0" | "powermax"    | "powermax-nvmetcp"  | "one-third" | "zero"  | "kubeletdown" | 600      | 900        | 900     | 900           |

  @powerflex-integration
  Scenario Outline: Basic node failover testing using test StatefulSet pods (node slow reboots)
    Given a kubernetes <kubeConfig>
    And cluster is clean of test pods
    And wait <nodeCleanSecs> to see there are no taints
    And <podsPerNode> pods per node with <nVol> volumes and <nDev> devices using <driverType> and <storageClass> in <deploySecs>
    Then validate that all pods are running within <deploySecs> seconds
    When I fail <workers> worker nodes and <primary> primary nodes with <failure> failure for <failSecs> seconds
    Then validate that all pods are running within <runSecs> seconds
    And labeled pods are on a different node
    And the taints for the failed nodes are removed within <nodeCleanSecs> seconds
    Then finally cleanup everything

    Examples:
      | kubeConfig | podsPerNode | nVol  | nDev  | driverType | storageClass | workers     | primary | failure  | failSecs | deploySecs | runSecs | nodeCleanSecs |
      # Small number of pods, increasing number of vols and devs
      | ""         | "1-2"       | "1-1" | "1-1" | "vxflexos" | "vxflexos"   | "one-third" | "zero"  | "reboot" | 120      | 240        | 300     | 600           |
      | ""         | "1-2"       | "2-2" | "2-2" | "vxflexos" | "vxflexos"   | "one-third" | "zero"  | "reboot" | 120      | 240        | 300     | 600           |
      | ""         | "1-2"       | "4-4" | "4-4" | "vxflexos" | "vxflexos"   | "one-third" | "zero"  | "reboot" | 120      | 240        | 300     | 600           |
      # Slightly more pods, increasing number of vols and devs
      | ""         | "3-5"       | "1-1" | "1-1" | "vxflexos" | "vxflexos"   | "one-third" | "zero"  | "reboot" | 240      | 240        | 300     | 600           |
      | ""         | "3-5"       | "2-2" | "2-2" | "vxflexos" | "vxflexos"   | "one-third" | "zero"  | "reboot" | 240      | 240        | 300     | 600           |
      | ""         | "3-5"       | "4-4" | "4-4" | "vxflexos" | "vxflexos"   | "one-third" | "zero"  | "reboot" | 240      | 240        | 300     | 600           |

#  @unity-integration
#  Scenario Outline: Basic node failover testing using test StatefulSet pods (node slow reboots)
#    Given a kubernetes <kubeConfig>
#    And cluster is clean of test pods
#    And wait <nodeCleanSecs> to see there are no taints
#    And <podsPerNode> pods per node with <nVol> volumes and <nDev> devices using <driverType> and <storageClass> in <deploySecs>
#    Then validate that all pods are running within <deploySecs> seconds
#    When I fail <workers> worker nodes and <primary> primary nodes with <failure> failure for <failSecs> seconds
#    Then validate that all pods are running within <runSecs> seconds
#    And labeled pods are on a different node
#    And the taints for the failed nodes are removed within <nodeCleanSecs> seconds
#    Then finally cleanup everything
#
#    Examples:
#      | kubeConfig | podsPerNode | nVol  | nDev  | driverType | storageClass  | workers     | primary | failure  | failSecs | deploySecs | runSecs | nodeCleanSecs |
#      # Small number of pods, increasing number of vols and devs
#      | ""         | "1-2"       | "1-1" | "1-1" | "unity"    | "unity-iscsi" | "one-third" | "zero"  | "reboot" | 600      | 900        | 900     | 600           |
#      | ""         | "1-2"       | "2-2" | "2-2" | "unity"    | "unity-iscsi" | "one-third" | "zero"  | "reboot" | 600      | 900        | 900     | 600           |
#      | ""         | "1-2"       | "4-4" | "4-4" | "unity"    | "unity-iscsi" | "one-third" | "zero"  | "reboot" | 600      | 900        | 900     | 600           |
#      # Slightly more pods, increasing number of vols and devs
#      | ""         | "3-5"       | "1-1" | "1-1" | "unity"    | "unity-iscsi" | "one-third" | "zero"  | "reboot" | 600      | 1500       | 600     | 600           |
#      | ""         | "3-5"       | "2-2" | "2-2" | "unity"    | "unity-iscsi" | "one-third" | "zero"  | "reboot" | 600      | 1500       | 600     | 600           |
#      | ""         | "3-5"       | "4-4" | "4-4" | "unity"    | "unity-iscsi" | "one-third" | "zero"  | "reboot" | 600      | 1500       | 600     | 600           |

#  @unity-integration
#  Scenario Outline: Basic node failover testing using test StatefulSet pods (node slow reboots)
#    Given a kubernetes <kubeConfig>
#    And cluster is clean of test pods
#    And wait <nodeCleanSecs> to see there are no taints
#    And <podsPerNode> pods per node with <nVol> volumes and <nDev> devices using <driverType> and <storageClass> in <deploySecs>
#    Then validate that all pods are running within <deploySecs> seconds
#    When I fail <workers> worker nodes and <primary> primary nodes with <failure> failure for <failSecs> seconds
#    Then validate that all pods are running within <runSecs> seconds
#    And labeled pods are on a different node
#    And the taints for the failed nodes are removed within <nodeCleanSecs> seconds
#    Then finally cleanup everything
#
#    Examples:
#      | kubeConfig | podsPerNode | nVol  | nDev  | driverType | storageClass | workers     | primary | failure  | failSecs | deploySecs | runSecs | nodeCleanSecs |
#      # Small number of pods, increasing number of vols and devs
#      | ""         | "1-2"       | "1-1" | "0-0" | "unity"    | "unity-nfs"  | "one-third" | "zero"  | "reboot" | 600      | 900        | 900     | 600           |
#      | ""         | "1-2"       | "2-2" | "0-0" | "unity"    | "unity-nfs"  | "one-third" | "zero"  | "reboot" | 600      | 900        | 900     | 600           |
#      | ""         | "1-2"       | "4-4" | "0-0" | "unity"    | "unity-nfs"  | "one-third" | "zero"  | "reboot" | 600      | 900        | 900     | 600           |
#      # Slightly more pods, increasing number of vols and devs
#      | ""         | "3-5"       | "1-1" | "0-0" | "unity"    | "unity-nfs"  | "one-third" | "zero"  | "reboot" | 600      | 1500       | 900     | 900           |
#      | ""         | "3-5"       | "2-2" | "0-0" | "unity"    | "unity-nfs"  | "one-third" | "zero"  | "reboot" | 600      | 1500       | 900     | 900           |
#      | ""         | "3-5"       | "4-4" | "0-0" | "unity"    | "unity-nfs"  | "one-third" | "zero"  | "reboot" | 600      | 1500       | 900     | 900           |


  @powerflex-integration
  Scenario Outline: Deploy pods when there are failed nodes already
    Given a kubernetes <kubeConfig>
    And cluster is clean of test pods
    And wait <nodeCleanSecs> to see there are no taints
    When I fail <workers> worker nodes and <primary> primary nodes with <failure> failure for <failSecs> seconds
    And <podsPerNode> pods per node with <nVol> volumes and <nDev> devices using <driverType> and <storageClass> in <deploySecs>
    Then validate that all pods are running within <deploySecs> seconds
    And the taints for the failed nodes are removed within <nodeCleanSecs> seconds
    Then finally cleanup everything
    And <podsPerNode> pods per node with <nVol> volumes and <nDev> devices using <driverType> and <storageClass> in <deploySecs>
    Then validate that all pods are running within <deploySecs> seconds
    Then finally cleanup everything
    Examples:
      | kubeConfig | podsPerNode | nVol  | nDev  | driverType | storageClass | workers     | primary | failure         | failSecs | deploySecs | nodeCleanSecs |
      | ""         | "1-2"       | "1-1" | "1-1" | "vxflexos" | "vxflexos"   | "one-third" | "zero"  | "interfacedown" | 120      | 300        | 300           |
      | ""         | "1-2"       | "1-1" | "1-1" | "vxflexos" | "vxflexos"   | "one-third" | "zero"  | "reboot"        | 120      | 300        | 300           |

  @unity-integration
  Scenario Outline: Deploy pods when there are failed nodes already
    Given a kubernetes <kubeConfig>
    And cluster is clean of test pods
    And wait <nodeCleanSecs> to see there are no taints
    When I fail <workers> worker nodes and <primary> primary nodes with <failure> failure for <failSecs> seconds
    And <podsPerNode> pods per node with <nVol> volumes and <nDev> devices using <driverType> and <storageClass> in <deploySecs>
    Then validate that all pods are running within <deploySecs> seconds
    And the taints for the failed nodes are removed within <nodeCleanSecs> seconds
    Then finally cleanup everything
    And <podsPerNode> pods per node with <nVol> volumes and <nDev> devices using <driverType> and <storageClass> in <deploySecs>
    Then validate that all pods are running within <deploySecs> seconds
    Then finally cleanup everything
    Examples:
      | kubeConfig | podsPerNode | nVol  | nDev  | driverType | storageClass  | workers     | primary | failure         | failSecs | deploySecs | nodeCleanSecs |
      | ""         | "1-2"       | "1-1" | "1-1" | "unity"    | "unity-iscsi" | "one-third" | "zero"  | "interfacedown" | 600      | 900        | 900           |
#      | ""         | "1-2"       | "1-1" | "1-1" | "unity"    | "unity-iscsi" | "one-third" | "zero"  | "reboot"        | 240      | 900        | 900           |

  @unity-integration
  Scenario Outline: Deploy pods when there are failed nodes already
    Given a kubernetes <kubeConfig>
    And cluster is clean of test pods
    And wait <nodeCleanSecs> to see there are no taints
    When I fail <workers> worker nodes and <primary> primary nodes with <failure> failure for <failSecs> seconds
    And <podsPerNode> pods per node with <nVol> volumes and <nDev> devices using <driverType> and <storageClass> in <deploySecs>
    Then validate that all pods are running within <deploySecs> seconds
    And the taints for the failed nodes are removed within <nodeCleanSecs> seconds
    Then finally cleanup everything
    And <podsPerNode> pods per node with <nVol> volumes and <nDev> devices using <driverType> and <storageClass> in <deploySecs>
    Then validate that all pods are running within <deploySecs> seconds
    Then finally cleanup everything
    Examples:
      | kubeConfig | podsPerNode | nVol  | nDev  | driverType | storageClass | workers     | primary | failure         | failSecs | deploySecs | nodeCleanSecs |
      | ""         | "1-2"       | "1-1" | "0-0" | "unity"    | "unity-nfs"  | "one-third" | "zero"  | "interfacedown" | 600      | 900        | 900           |
#      | ""         | "1-2"       | "1-1" | "0-0" | "unity"    | "unity-nfs"  | "one-third" | "zero"  | "reboot"        | 240      | 900        | 900           |


  @powerscale-integration
  Scenario Outline: Deploy pods when there are failed nodes already
    Given a kubernetes <kubeConfig>
    And cluster is clean of test pods
    And wait <nodeCleanSecs> to see there are no taints
    When I fail <workers> worker nodes and <primary> primary nodes with <failure> failure for <failSecs> seconds
    And <podsPerNode> pods per node with <nVol> volumes and <nDev> devices using <driverType> and <storageClass> in <deploySecs>
    Then validate that all pods are running within <deploySecs> seconds
    And the taints for the failed nodes are removed within <nodeCleanSecs> seconds
    Then finally cleanup everything
    And <podsPerNode> pods per node with <nVol> volumes and <nDev> devices using <driverType> and <storageClass> in <deploySecs>
    Then validate that all pods are running within <deploySecs> seconds
    Then finally cleanup everything
    Examples:
      | kubeConfig | podsPerNode | nVol  | nDev  | driverType | storageClass  | workers     | primary | failure         | failSecs | deploySecs | nodeCleanSecs |
      | ""         | "1-2"       | "1-1" | "0-0" | "isilon"   | "isilon"      | "one-third" | "zero"  | "interfacedown" | 600      | 900        | 900           |
  
  @powerstore-integration
  Scenario Outline: Deploy pods when there are failed nodes already
    Given a kubernetes <kubeConfig>
    And cluster is clean of test pods
    And wait <nodeCleanSecs> to see there are no taints
    When I fail <workers> worker nodes and <primary> primary nodes with <failure> failure for <failSecs> seconds
    And <podsPerNode> pods per node with <nVol> volumes and <nDev> devices using <driverType> and <storageClass> in <deploySecs>
    Then validate that all pods are running within <deploySecs> seconds
    And the taints for the failed nodes are removed within <nodeCleanSecs> seconds
    Then finally cleanup everything
    And <podsPerNode> pods per node with <nVol> volumes and <nDev> devices using <driverType> and <storageClass> in <deploySecs>
    Then validate that all pods are running within <deploySecs> seconds
    Then finally cleanup everything
    Examples:
      | kubeConfig | podsPerNode | nVol  | nDev  | driverType | storageClass  | workers     | primary | failure         | failSecs | deploySecs | nodeCleanSecs |
      | ""         | "1-2"       | "1-1" | "0-0" | "powerstore"   | "powerstore-nfs"      | "one-third" | "zero"  | "interfacedown" | 600      | 900        | 900           |
      | ""         | "1-2"       | "1-1" | "0-0" | "powerstore"   | "powerstore-iscsi"      | "one-third" | "zero"  | "interfacedown" | 600      | 900        | 900           |
      #| ""         | "1-2"       | "1-1" | "0-0" | "powerstore"   | "powerstore-nvmetcp"      | "one-third" | "zero"  | "interfacedown" | 600      | 900        | 900           |

  @powermax-integration
  Scenario Outline: Deploy pods when there are failed nodes already
    Given a kubernetes <kubeConfig>
    And cluster is clean of test pods
    And wait <nodeCleanSecs> to see there are no taints
    When I fail <workers> worker nodes and <primary> primary nodes with <failure> failure for <failSecs> seconds
    And <podsPerNode> pods per node with <nVol> volumes and <nDev> devices using <driverType> and <storageClass> in <deploySecs>
    Then validate that all pods are running within <deploySecs> seconds
    And the taints for the failed nodes are removed within <nodeCleanSecs> seconds
    Then finally cleanup everything
    And <podsPerNode> pods per node with <nVol> volumes and <nDev> devices using <driverType> and <storageClass> in <deploySecs>
    Then validate that all pods are running within <deploySecs> seconds
    Then finally cleanup everything
    Examples:
      | kubeConfig | podsPerNode | nVol  | nDev  | driverType | storageClass  | workers     | primary | failure         | failSecs | deploySecs | nodeCleanSecs |
      | ""         | "1-2"       | "1-1" | "0-0" | "powermax"   | "powermax-nfs"      | "one-third" | "zero"  | "interfacedown" | 600      | 900        | 900           |
      | ""         | "1-2"       | "1-1" | "0-0" | "powermax"   | "powermax-iscsi"      | "one-third" | "zero"  | "interfacedown" | 600      | 900        | 900           |
      #| ""         | "1-2"       | "1-1" | "0-0" | "powermax"   | "powermax-nvmetcp"      | "one-third" | "zero"  | "interfacedown" | 600      | 900        | 900           |

  @powerflex-integration
  Scenario Outline: Short failure window tests
    Given a kubernetes <kubeConfig>
    And cluster is clean of test pods
    And wait <nodeCleanSecs> to see there are no taints
    And <podsPerNode> pods per node with <nVol> volumes and <nDev> devices using <driverType> and <storageClass> in <deploySecs>
    Then validate that all pods are running within <deploySecs> seconds
    When I fail <workers> worker nodes and <primary> primary nodes with <failure> failure for <failSecs> seconds
    Then validate that all pods are running within <runSecs> seconds
    And labeled pods are on a different node
    And the taints for the failed nodes are removed within <nodeCleanSecs> seconds
    Then finally cleanup everything
    Examples:
      | kubeConfig | podsPerNode | nVol  | nDev  | driverType | storageClass | workers     | primary | failure         | failSecs | deploySecs | runSecs | nodeCleanSecs |
      | ""         | "1-2"       | "1-1" | "1-1" | "vxflexos" | "vxflexos"   | "one-third" | "zero"  | "interfacedown" | 60       | 300        | 120     | 300           |
      | ""         | "1-2"       | "1-1" | "1-1" | "vxflexos" | "vxflexos"   | "one-third" | "zero"  | "reboot"        | 45       | 300        | 120     | 300           |

  @powerflex-integration
  Scenario Outline: Failover test that includes unlabelled pods
    Given a kubernetes <kubeConfig>
    And cluster is clean of test pods
    And wait <nodeCleanSecs> to see there are no taints
    And <podsPerNode> pods per node with <nVol> volumes and <nDev> devices using <driverType> and <storageClass> in <deploySecs>
    And <podsPerNode> unprotected pods per node with <nVol> volumes and <nDev> devices using <driverType> and <storageClass> in <deploySecs>
    Then validate that all pods are running within <deploySecs> seconds
    When I fail <workers> worker nodes and <primary> primary nodes with <failure> failure for <failSecs> seconds
    Then validate that all pods are running within <runSecs> seconds
    And labeled pods are on a different node
    And the taints for the failed nodes are removed within <nodeCleanSecs> seconds
    Then finally cleanup everything

    Examples:
      | kubeConfig | podsPerNode | nVol  | nDev  | driverType | storageClass | workers     | primary | failure         | failSecs | deploySecs | runSecs | nodeCleanSecs |
      | ""         | "1-1"       | "1-1" | "1-1" | "vxflexos" | "vxflexos"   | "one-third" | "zero"  | "interfacedown" | 120      | 240        | 300     | 600           |
      | ""         | "1-1"       | "1-1" | "1-1" | "vxflexos" | "vxflexos"   | "one-third" | "zero"  | "reboot"        | 120      | 240        | 300     | 600           |

  @unity-integration
  Scenario Outline: Failover test that includes unlabelled pods
    Given a kubernetes <kubeConfig>
    And cluster is clean of test pods
    And wait <nodeCleanSecs> to see there are no taints
    And <podsPerNode> pods per node with <nVol> volumes and <nDev> devices using <driverType> and <storageClass> in <deploySecs>
    And <podsPerNode> unprotected pods per node with <nVol> volumes and <nDev> devices using <driverType> and <storageClass> in <deploySecs>
    Then validate that all pods are running within <deploySecs> seconds
    When I fail <workers> worker nodes and <primary> primary nodes with <failure> failure for <failSecs> seconds
    Then validate that all pods are running within <runSecs> seconds
    And labeled pods are on a different node
    And the taints for the failed nodes are removed within <nodeCleanSecs> seconds
    Then finally cleanup everything

    Examples:
      | kubeConfig | podsPerNode | nVol  | nDev  | driverType | storageClass  | workers     | primary | failure         | failSecs | deploySecs | runSecs | nodeCleanSecs |
      | ""         | "1-1"       | "1-1" | "1-1" | "unity"    | "unity-iscsi" | "one-third" | "zero"  | "interfacedown" | 600      | 900        | 900     | 900           |
#      | ""         | "1-1"       | "1-1" | "1-1" | "unity"    | "unity-iscsi" | "one-third" | "zero"  | "reboot"        | 240      | 900        | 900     | 900           |

  @powerflex-short-integration
  Scenario Outline: Basic node failover testing using test StatefulSet pods (node interface down)
    Given a kubernetes <kubeConfig>
    And cluster is clean of test pods
    And wait <nodeCleanSecs> to see there are no taints
    And <podsPerNode> pods per node with <nVol> volumes and <nDev> devices using <driverType> and <storageClass> in <deploySecs>
    Then validate that all pods are running within <deploySecs> seconds
    When I fail <workers> worker nodes and <primary> primary nodes with <failure> failure for <failSecs> seconds
    Then validate that all pods are running within <runSecs> seconds
    And labeled pods are on a different node
    And the taints for the failed nodes are removed within <nodeCleanSecs> seconds
    Then finally cleanup everything

    Examples:
      | kubeConfig | podsPerNode | nVol  | nDev  | driverType | storageClass | workers     | primary | failure         | failSecs | deploySecs | runSecs | nodeCleanSecs |
      | ""         | "1-1"       | "1-1" | "1-1" | "vxflexos" | "vxflexos"   | "one-third" | "zero"  | "interfacedown" | 120      | 240        | 300     | 300           |
      | ""         | "2-2"       | "2-2" | "2-2" | "vxflexos" | "vxflexos"   | "one-third" | "zero"  | "interfacedown" | 120      | 240        | 300     | 600           |

  @unity-short-integration
  Scenario Outline: Basic node failover testing using test StatefulSet pods (node interface down)
    Given a kubernetes <kubeConfig>
    And cluster is clean of test pods
    And wait <nodeCleanSecs> to see there are no taints
    And <podsPerNode> pods per node with <nVol> volumes and <nDev> devices using <driverType> and <storageClass> in <deploySecs>
    Then validate that all pods are running within <deploySecs> seconds
    When I fail <workers> worker nodes and <primary> primary nodes with <failure> failure for <failSecs> seconds
    Then validate that all pods are running within <runSecs> seconds
    And labeled pods are on a different node
    And the taints for the failed nodes are removed within <nodeCleanSecs> seconds
    Then finally cleanup everything

    Examples:
      | kubeConfig | podsPerNode | nVol  | nDev  | driverType | storageClass  | workers     | primary | failure         | failSecs | deploySecs | runSecs | nodeCleanSecs |
      | ""         | "2-2"       | "2-2" | "2-2" | "unity"    | "unity-iscsi" | "one-third" | "zero"  | "interfacedown" | 600      | 900        | 900     | 900           |
      | ""         | "2-2"       | "2-2" | "0-0" | "unity"    | "unity-nfs"   | "one-third" | "zero"  | "interfacedown" | 600      | 900        | 900     | 900           |

  @powerscale-short-integration
  Scenario Outline: Basic node failover testing using test StatefulSet pods (node interface down)
    Given a kubernetes <kubeConfig>
    And cluster is clean of test pods
    And wait <nodeCleanSecs> to see there are no taints
    And <podsPerNode> pods per node with <nVol> volumes and <nDev> devices using <driverType> and <storageClass> in <deploySecs>
    Then validate that all pods are running within <deploySecs> seconds
    When I fail <workers> worker nodes and <primary> primary nodes with <failure> failure for <failSecs> seconds
    Then validate that all pods are running within <runSecs> seconds
    And labeled pods are on a different node
    And the taints for the failed nodes are removed within <nodeCleanSecs> seconds
    Then finally cleanup everything

    Examples:
      | kubeConfig | podsPerNode | nVol  | nDev  | driverType | storageClass | workers     | primary | failure         | failSecs | deploySecs | runSecs | nodeCleanSecs |
      | ""         | "1-1"       | "1-1" | "0-0" | "isilon" | "isilon"   | "one-third" | "zero"  | "interfacedown" | 120      | 240        | 300     | 300           |
  
  @powerstore-short-integration
  Scenario Outline: Basic node failover testing using test StatefulSet pods (node interface down)
    Given a kubernetes <kubeConfig>
    And cluster is clean of test pods
    And wait <nodeCleanSecs> to see there are no taints
    And <podsPerNode> pods per node with <nVol> volumes and <nDev> devices using <driverType> and <storageClass> in <deploySecs>
    Then validate that all pods are running within <deploySecs> seconds
    When I fail <workers> worker nodes and <primary> primary nodes with <failure> failure for <failSecs> seconds
    Then validate that all pods are running within <runSecs> seconds
    And labeled pods are on a different node
    And the taints for the failed nodes are removed within <nodeCleanSecs> seconds
    Then finally cleanup everything

    Examples:
      | kubeConfig | podsPerNode | nVol  | nDev  | driverType | storageClass | workers     | primary | failure         | failSecs | deploySecs | runSecs | nodeCleanSecs |
      # Uncomment the storageclass to use. The default is set to nvme which is supported by nightly qualification.
      #| ""         | "1-1"       | "1-1" | "0-0" | "powerstore" | "powerstore-nfs"   | "one-third" | "zero"  | "interfacedown" | 120      | 240        | 300     | 300           |
      #| ""         | "1-1"       | "1-1" | "0-0" | "powerstore" | "powerstore-iscsi"   | "one-third" | "zero"  | "interfacedown" | 120      | 240        | 300     | 300           |
      | ""         | "1-1"       | "1-1" | "0-0" | "powerstore" | "powerstore-nvmetcp"   | "one-third" | "zero"  | "interfacedown" | 240      | 480        | 300     | 300           |

  @powermax-short-integration
  Scenario Outline: Basic node failover testing using test StatefulSet pods (node interface down)
    Given a kubernetes <kubeConfig>
    And cluster is clean of test pods
    And wait <nodeCleanSecs> to see there are no taints
    And <podsPerNode> pods per node with <nVol> volumes and <nDev> devices using <driverType> and <storageClass> in <deploySecs>
    Then validate that all pods are running within <deploySecs> seconds
    When I fail <workers> worker nodes and <primary> primary nodes with <failure> failure for <failSecs> seconds
    Then validate that all pods are running within <runSecs> seconds
    And labeled pods are on a different node
    And the taints for the failed nodes are removed within <nodeCleanSecs> seconds
    Then finally cleanup everything

    Examples:
      | kubeConfig | podsPerNode | nVol  | nDev  | driverType | storageClass | workers     | primary | failure         | failSecs | deploySecs | runSecs | nodeCleanSecs |
      | ""         | "1-1"       | "1-1" | "0-0" | "powermax" | "powermax-nfs"   | "one-third" | "zero"  | "interfacedown" | 120      | 240        | 300     | 300           |
      | ""         | "1-1"       | "1-1" | "0-0" | "powermax" | "powermax-iscsi"   | "one-third" | "zero"  | "interfacedown" | 120      | 240        | 300     | 300           |
      #| ""         | "1-1"       | "1-1" | "0-0" | "powermax" | "powermax-nvmetcp"   | "one-third" | "zero"  | "interfacedown" | 120      | 240        | 300     | 300           |

  @powerflex-short-integration
  Scenario Outline: Basic node failover testing using test StatefulSet pods (node slow reboots)
    Given a kubernetes <kubeConfig>
    And cluster is clean of test pods
    And wait <nodeCleanSecs> to see there are no taints
    And <podsPerNode> pods per node with <nVol> volumes and <nDev> devices using <driverType> and <storageClass> in <deploySecs>
    Then validate that all pods are running within <deploySecs> seconds
    When I fail <workers> worker nodes and <primary> primary nodes with <failure> failure for <failSecs> seconds
    Then validate that all pods are running within <runSecs> seconds
    And labeled pods are on a different node
    And the taints for the failed nodes are removed within <nodeCleanSecs> seconds
    Then finally cleanup everything

    Examples:
      | kubeConfig | podsPerNode | nVol  | nDev  | driverType | storageClass | workers     | primary | failure  | failSecs | deploySecs | runSecs | nodeCleanSecs |
      | ""         | "1-1"       | "1-1" | "1-1" | "vxflexos" | "vxflexos"   | "one-third" | "zero"  | "reboot" | 120      | 240        | 300     | 600           |
      | ""         | "2-2"       | "2-2" | "2-2" | "vxflexos" | "vxflexos"   | "one-third" | "zero"  | "reboot" | 120      | 240        | 300     | 600           |

  @powerflex-short-integration
  Scenario Outline: Basic node failover testing using test StatefulSet pods (node kubelet down)
    Given a kubernetes <kubeConfig>
    And cluster is clean of test pods
    And wait <nodeCleanSecs> to see there are no taints
    And <podsPerNode> pods per node with <nVol> volumes and <nDev> devices using <driverType> and <storageClass> in <deploySecs>
    Then validate that all pods are running within <deploySecs> seconds
    When I fail <workers> worker nodes and <primary> primary nodes with <failure> failure for <failSecs> seconds
    Then validate that all pods are running within <runSecs> seconds
    And labeled pods are on a different node
    And the taints for the failed nodes are removed within <nodeCleanSecs> seconds
    Then finally cleanup everything

    Examples:
      | kubeConfig | podsPerNode | nVol  | nDev  | driverType | storageClass | workers     | primary | failure         | failSecs | deploySecs | runSecs | nodeCleanSecs |
      | ""         | "1-1"       | "1-1" | "1-1" | "vxflexos" | "vxflexos"   | "one-third" | "zero"  | "kubeletdown"   | 600      | 900        | 900     | 900           |
      | ""         | "2-2"       | "2-2" | "2-2" | "vxflexos" | "vxflexos"   | "one-third" | "zero"  | "kubeletdown"   | 600      | 900        | 900     | 900           |

  @unity-short-integration
  Scenario Outline: Basic node failover testing using test StatefulSet pods (node kubelet down)
    Given a kubernetes <kubeConfig>
    And cluster is clean of test pods
    And wait <nodeCleanSecs> to see there are no taints
    And <podsPerNode> pods per node with <nVol> volumes and <nDev> devices using <driverType> and <storageClass> in <deploySecs>
    Then validate that all pods are running within <deploySecs> seconds
    When I fail <workers> worker nodes and <primary> primary nodes with <failure> failure for <failSecs> seconds
    Then validate that all pods are running within <runSecs> seconds
    And labeled pods are on a different node
    And the taints for the failed nodes are removed within <nodeCleanSecs> seconds
    Then finally cleanup everything

    Examples:
      | kubeConfig | podsPerNode | nVol  | nDev  | driverType | storageClass  | workers     | primary | failure       | failSecs | deploySecs | runSecs | nodeCleanSecs |
      | ""         | "2-2"       | "2-2" | "2-2" | "unity"    | "unity-iscsi" | "one-third" | "zero"  | "kubeletdown" | 600      | 900        | 900     | 900           |
      | ""         | "2-2"       | "2-2" | "0-0" | "unity"    | "unity-nfs"   | "one-third" | "zero"  | "kubeletdown" | 600      | 900        | 900     | 900           |

  @powerflex-array-interface
  Scenario Outline: Multi networked nodes with a failure against the array interface network
    Given a kubernetes <kubeConfig>
    And cluster is clean of test pods
    And wait <nodeCleanSecs> to see there are no taints
    And <podsPerNode> pods per node with <nVol> volumes and <nDev> devices using <driverType> and <storageClass> in <deploySecs>
    Then validate that all pods are running within <deploySecs> seconds
    When I fail <workers> worker nodes and <primary> primary nodes with <failure> failure for <failSecs> and I expect these taints <taints>
    Then validate that all pods are running within <runSecs> seconds
    And labeled pods are on a different node
    And the taints for the failed nodes are removed within <nodeCleanSecs> seconds
    Then finally cleanup everything

    Examples:
      | kubeConfig | podsPerNode | nVol  | nDev  | driverType | storageClass | workers     | primary | failure                     | taints                             | failSecs | deploySecs | runSecs | nodeCleanSecs |
      | ""         | "1-1"       | "1-1" | "1-1" | "vxflexos" | "vxflexos"   | "one-third" | "zero"  | "interfacedown:INTERFACE_A" | "vxflexos.podmon.storage.dell.com" | 120      | 240        | 300     | 300           |
      | ""         | "1-1"       | "1-1" | "1-1" | "vxflexos" | "vxflexos"   | "one-third" | "zero"  | "reboot:INTERFACE_A"        | "vxflexos.podmon.storage.dell.com" | 120      | 240        | 300     | 300           |

  @unity-array-interface
  Scenario Outline: Multi networked nodes with a failure against the array interface network
    Given a kubernetes <kubeConfig>
    And cluster is clean of test pods
    And wait <nodeCleanSecs> to see there are no taints
    And <podsPerNode> pods per node with <nVol> volumes and <nDev> devices using <driverType> and <storageClass> in <deploySecs>
    Then validate that all pods are running within <deploySecs> seconds
    When I fail <workers> worker nodes and <primary> primary nodes with <failure> failure for <failSecs> and I expect these taints <taints>
    Then validate that all pods are running within <runSecs> seconds
    And labeled pods are on a different node
    And the taints for the failed nodes are removed within <nodeCleanSecs> seconds
    Then finally cleanup everything

    Examples:
      | kubeConfig | podsPerNode | nVol  | nDev  | driverType | storageClass  | workers     | primary | failure                     | taints                          | failSecs | deploySecs | runSecs | nodeCleanSecs |
      | ""         | "1-1"       | "1-1" | "1-1" | "unity"    | "unity-iscsi" | "one-third" | "zero"  | "interfacedown:INTERFACE_A" | "unity.podmon.storage.dell.com" | 120      | 900        | 900     | 300           |
      | ""         | "1-1"       | "1-1" | "1-1" | "unity"    | "unity-iscsi" | "one-third" | "zero"  | "reboot:INTERFACE_A"        | "unity.podmon.storage.dell.com" | 120      | 900        | 900     | 300           |
      | ""         | "1-1"       | "1-1" | "1-1" | "unity"    | "unity-nfs"   | "one-third" | "zero"  | "interfacedown:INTERFACE_B" | "unity.podmon.storage.dell.com" | 120      | 900        | 900     | 300           |
      | ""         | "1-1"       | "1-1" | "1-1" | "unity"    | "unity-nfs"   | "one-third" | "zero"  | "reboot:INTERFACE_B"        | "unity.podmon.storage.dell.com" | 120      | 900        | 900     | 300           |

  @powerstore-array-interface
  Scenario Outline: Multi networked nodes with a failure against the array interface network
    Given a kubernetes <kubeConfig>
    And cluster is clean of test pods
    And wait <nodeCleanSecs> to see there are no taints
    And <podsPerNode> pods per node with <nVol> volumes and <nDev> devices using <driverType> and <storageClass> in <deploySecs>
    Then validate that all pods are running within <deploySecs> seconds
    When I fail <workers> worker nodes and <primary> primary nodes with <failure> failure for <failSecs> and I expect these taints <taints>
    Then validate that all pods are running within <runSecs> seconds
    And labeled pods are on a different node
    And the taints for the failed nodes are removed within <nodeCleanSecs> seconds
    Then finally cleanup everything

    Examples:
      | kubeConfig | podsPerNode | nVol  | nDev  | driverType    | storageClass         | workers     | primary | failure                     | taints                               | failSecs | deploySecs | runSecs | nodeCleanSecs |
      | ""         | "1-1"       | "1-1" | "1-1" | "powerstore"  | "powerstore-nfs"     | "one-third" | "zero"  | "interfacedown:INTERFACE_A" | "powerstore.podmon.storage.dell.com" | 120      | 900        | 900     | 300           |
      | ""         | "1-1"       | "1-1" | "1-1" | "powerstore"  | "powerstore-nfs"     | "one-third" | "zero"  | "reboot:INTERFACE_A"        | "powerstore.podmon.storage.dell.com" | 120      | 900        | 900     | 300           |
      | ""         | "1-1"       | "1-1" | "1-1" | "powerstore"  | "powerstore-iscsi"   | "one-third" | "zero"  | "interfacedown:INTERFACE_B" | "powerstore.podmon.storage.dell.com" | 120      | 900        | 900     | 300           |
      | ""         | "1-1"       | "1-1" | "1-1" | "powerstore"  | "powerstore-iscsi"   | "one-third" | "zero"  | "reboot:INTERFACE_B"        | "powerstore.podmon.storage.dell.com" | 120      | 900        | 900     | 300           |
      | ""         | "1-1"       | "1-1" | "1-1" | "powerstore"  | "powerstore-nvmetcp" | "one-third" | "zero"  | "interfacedown:INTERFACE_C" | "powerstore.podmon.storage.dell.com" | 120      | 900        | 900     | 300           |
      | ""         | "1-1"       | "1-1" | "1-1" | "powerstore"  | "powerstore-nvmetcp" | "one-third" | "zero"  | "reboot:INTERFACE_C"        | "powerstore.podmon.storage.dell.com" | 120      | 900        | 900     | 300           |

  @powermax-array-interface
  Scenario Outline: Multi networked nodes with a failure against the array interface network
    Given a kubernetes <kubeConfig>
    And cluster is clean of test pods
    And wait <nodeCleanSecs> to see there are no taints
    And <podsPerNode> pods per node with <nVol> volumes and <nDev> devices using <driverType> and <storageClass> in <deploySecs>
    Then validate that all pods are running within <deploySecs> seconds
    When I fail <workers> worker nodes and <primary> primary nodes with <failure> failure for <failSecs> and I expect these taints <taints>
    Then validate that all pods are running within <runSecs> seconds
    And labeled pods are on a different node
    And the taints for the failed nodes are removed within <nodeCleanSecs> seconds
    Then finally cleanup everything

    Examples:
      | kubeConfig | podsPerNode | nVol  | nDev  | driverType    | storageClass         | workers     | primary | failure                     | taints                               | failSecs | deploySecs | runSecs | nodeCleanSecs |
      | ""         | "1-1"       | "1-1" | "1-1" | "powermax"    | "powermax-nfs"       | "one-third" | "zero"  | "interfacedown:INTERFACE_A" | "powermax.podmon.storage.dell.com"   | 120      | 900        | 900     | 300           |
      | ""         | "1-1"       | "1-1" | "1-1" | "powermax"    | "powermax-nfs"       | "one-third" | "zero"  | "reboot:INTERFACE_A"        | "powermax.podmon.storage.dell.com"   | 120      | 900        | 900     | 300           |
      | ""         | "1-1"       | "1-1" | "1-1" | "powermax"    | "powermax-iscsi"     | "one-third" | "zero"  | "interfacedown:INTERFACE_B" | "powermax.podmon.storage.dell.com"   | 120      | 900        | 900     | 300           |
      | ""         | "1-1"       | "1-1" | "1-1" | "powermax"    | "powermax-iscsi"     | "one-third" | "zero"  | "reboot:INTERFACE_B"        | "powermax.podmon.storage.dell.com"   | 120      | 900        | 900     | 300           |
      | ""         | "1-1"       | "1-1" | "1-1" | "powermax"    | "powermax-nvmetcp"   | "one-third" | "zero"  | "interfacedown:INTERFACE_C" | "powermax.podmon.storage.dell.com"   | 120      | 900        | 900     | 300           |
      | ""         | "1-1"       | "1-1" | "1-1" | "powermax"    | "powermax-nvmetcp"   | "one-third" | "zero"  | "reboot:INTERFACE_C"        | "powermax.podmon.storage.dell.com"   | 120      | 900        | 900     | 300           |

  @powerscale-integration
  Scenario Outline: Basic node failover testing using test StatefulSet pods (node slow reboots)
    Given a kubernetes <kubeConfig>
    And cluster is clean of test pods
    And wait <nodeCleanSecs> to see there are no taints
    And <podsPerNode> pods per node with <nVol> volumes and <nDev> devices using <driverType> and <storageClass> in <deploySecs>
    Then validate that all pods are running within <deploySecs> seconds
    When I fail <workers> worker nodes and <primary> primary nodes with <failure> failure for <failSecs> seconds
    Then validate that all pods are running within <runSecs> seconds
    And labeled pods are on a different node
    And the taints for the failed nodes are removed within <nodeCleanSecs> seconds
    Then finally cleanup everything
    Examples:
      | kubeConfig | podsPerNode | nVol  | nDev  | driverType | storageClass | workers     | primary | failure  | failSecs | deploySecs | runSecs | nodeCleanSecs |
       #Small number of pods, increasing number of vols and devs
      | ""         | "1-2"       | "1-1" | "0-0" | "isilon" | "isilon"   | "one-third" | "zero"  | "reboot" | 240      | 600        | 600     | 900           |
      | ""         | "1-2"       | "2-2" | "0-0" | "isilon" | "isilon"   | "one-third" | "zero"  | "reboot" | 600      | 900        | 900     | 1200           |
      # Slightly more pods, increasing number of vols and devs
      | ""         | "3-5"       | "1-1" | "0-0" | "isilon" | "isilon"   | "one-third" | "zero"  | "reboot" | 600      | 900        | 900     | 1800           |
    # | ""         | "3-5"       | "2-2" | "0-0" | "isilon" | "isilon"   | "one-third" | "zero"  | "reboot" | 240      | 240        | 300     | 600           |
    # | ""         | "5-10"       | "1-1" | "0-0" | "isilon" | "isilon"   | "one-third" | "zero"  | "reboot" | 1200      | 2000       | 2000     | 2000           |
  
  @powerstore-integration
  Scenario Outline: Basic node failover testing using test StatefulSet pods (node slow reboots)
    Given a kubernetes <kubeConfig>
    And cluster is clean of test pods
    And wait <nodeCleanSecs> to see there are no taints
    And <podsPerNode> pods per node with <nVol> volumes and <nDev> devices using <driverType> and <storageClass> in <deploySecs>
    Then validate that all pods are running within <deploySecs> seconds
    When I fail <workers> worker nodes and <primary> primary nodes with <failure> failure for <failSecs> seconds
    Then validate that all pods are running within <runSecs> seconds
    And labeled pods are on a different node
    And the taints for the failed nodes are removed within <nodeCleanSecs> seconds
    Then finally cleanup everything
    Examples:
      | kubeConfig | podsPerNode | nVol  | nDev  | driverType | storageClass | workers     | primary | failure  | failSecs | deploySecs | runSecs | nodeCleanSecs |
      #Small number of pods, increasing number of vols and devs
      | ""         | "1-2"       | "1-1" | "0-0" | "powerstore" | "powerstore-nfs"   | "one-third" | "zero"  | "reboot" | 240      | 600        | 600     | 600          |
      # Slightly more pods, increasing number of vols and devs
      | ""         | "3-5"       | "1-1" | "0-0" | "powerstore" | "powerstore-nfs"   | "one-third" | "zero"  | "reboot" | 240      | 900        | 900     | 900           |
      #Small number of pods, increasing number of vols and devs
      | ""         | "1-2"       | "1-1" | "0-0" | "powerstore" | "powerstore-iscsi"   | "one-third" | "zero"  | "reboot" | 240      | 600        | 600     | 600          |
      # Slightly more pods, increasing number of vols and devs
      | ""         | "3-5"       | "1-1" | "0-0" | "powerstore" | "powerstore-iscsi"   | "one-third" | "zero"  | "reboot" | 240      | 900        | 900     | 900           |
      #Small number of pods, increasing number of vols and devs
      #| ""         | "1-2"       | "1-1" | "0-0" | "powerstore" | "powerstore-nvmetcp"   | "one-third" | "zero"  | "reboot" | 240      | 600        | 600     | 600          |
      # Slightly more pods, increasing number of vols and devs
      #| ""         | "3-5"       | "1-1" | "0-0" | "powerstore" | "powerstore-nvmetcp"   | "one-third" | "zero"  | "reboot" | 240      | 900        | 900     | 900           |

  @powermax-integration
  Scenario Outline: Basic node failover testing using test StatefulSet pods (node slow reboots)
    Given a kubernetes <kubeConfig>
    And cluster is clean of test pods
    And wait <nodeCleanSecs> to see there are no taints
    And <podsPerNode> pods per node with <nVol> volumes and <nDev> devices using <driverType> and <storageClass> in <deploySecs>
    Then validate that all pods are running within <deploySecs> seconds
    When I fail <workers> worker nodes and <primary> primary nodes with <failure> failure for <failSecs> seconds
    Then validate that all pods are running within <runSecs> seconds
    And labeled pods are on a different node
    And the taints for the failed nodes are removed within <nodeCleanSecs> seconds
    Then finally cleanup everything
    Examples:
      | kubeConfig | podsPerNode | nVol  | nDev  | driverType | storageClass     | workers     | primary | failure  | failSecs | deploySecs | runSecs | nodeCleanSecs |
      #Small number of pods, increasing number of vols and devs
      | ""         | "1-2"       | "1-1" | "0-0" | "powermax" | "powermax-nfs"   | "one-third" | "zero"  | "reboot" | 240      | 600        | 600     | 600           |
      # Slightly more pods, increasing number of vols and devs
      | ""         | "3-5"       | "1-1" | "0-0" | "powermax" | "powermax-nfs"   | "one-third" | "zero"  | "reboot" | 240      | 900        | 900     | 900           |
      #Small number of pods, increasing number of vols and devs
      | ""         | "1-2"       | "1-1" | "0-0" | "powermax" | "powermax-iscsi" | "one-third" | "zero"  | "reboot" | 240      | 600        | 600     | 600           |
      # Slightly more pods, increasing number of vols and devs
      | ""         | "3-5"       | "1-1" | "0-0" | "powermax" | "powermax-iscsi" | "one-third" | "zero"  | "reboot" | 240      | 900        | 900     | 900           |
      #Small number of pods, increasing number of vols and devs
      #| ""         | "1-2"       | "1-1" | "0-0" | "powermax" | "powermax-nvmetcp"   | "one-third" | "zero"  | "reboot" | 240      | 600        | 600     | 600          |
      # Slightly more pods, increasing number of vols and devs
      #| ""         | "3-5"       | "1-1" | "0-0" | "powermax" | "powermax-nvmetcp"   | "one-third" | "zero"  | "reboot" | 240      | 900        | 900     | 900           |

  @powerstore-short-integration
  Scenario Outline: Basic node failover testing using test StatefulSet pods (node slow reboots)
    Given a kubernetes <kubeConfig>
    And cluster is clean of test pods
    And wait <nodeCleanSecs> to see there are no taints
    And <podsPerNode> pods per node with <nVol> volumes and <nDev> devices using <driverType> and <storageClass> in <deploySecs>
    Then validate that all pods are running within <deploySecs> seconds
    When I fail <workers> worker nodes and <primary> primary nodes with <failure> failure for <failSecs> seconds
    Then validate that all pods are running within <runSecs> seconds
    And labeled pods are on a different node
    And the taints for the failed nodes are removed within <nodeCleanSecs> seconds
    Then finally cleanup everything
    Examples:
      | kubeConfig | podsPerNode | nVol  | nDev  | driverType | storageClass | workers     | primary | failure  | failSecs | deploySecs | runSecs | nodeCleanSecs |
      # Uncomment the storageclass to use. The default is set to nvme which is supported by nightly qualification.
      #| ""         | "1-1"       | "1-1" | "0-0" | "powerstore" | "powerstore-nfs"   | "one-third" | "zero"  | "reboot" | 240      | 600        | 600     | 600          |
      #| ""         | "1-1"       | "1-1" | "0-0" | "powerstore" | "powerstore-iscsi"   | "one-third" | "zero"  | "reboot" | 240      | 600        | 600     | 600          |
      | ""         | "1-1"       | "1-1" | "0-0" | "powerstore" | "powerstore-nvmetcp"   | "one-third" | "zero"  | "reboot" | 240      | 600        | 600     | 600          |

  @powermax-short-integration
  Scenario Outline: Basic node failover testing using test StatefulSet pods (node slow reboots)
    Given a kubernetes <kubeConfig>
    And cluster is clean of test pods
    And wait <nodeCleanSecs> to see there are no taints
    And <podsPerNode> pods per node with <nVol> volumes and <nDev> devices using <driverType> and <storageClass> in <deploySecs>
    Then validate that all pods are running within <deploySecs> seconds
    When I fail <workers> worker nodes and <primary> primary nodes with <failure> failure for <failSecs> seconds
    Then validate that all pods are running within <runSecs> seconds
    And labeled pods are on a different node
    And the taints for the failed nodes are removed within <nodeCleanSecs> seconds
    Then finally cleanup everything
    Examples:
      | kubeConfig | podsPerNode | nVol  | nDev  | driverType | storageClass     | workers     | primary | failure  | failSecs | deploySecs | runSecs | nodeCleanSecs |
      | ""         | "1-1"       | "1-1" | "0-0" | "powermax" | "powermax-nfs"   | "one-third" | "zero"  | "reboot" | 240      | 600        | 600     | 600           |
      | ""         | "1-1"       | "1-1" | "0-0" | "powermax" | "powermax-iscsi" | "one-third" | "zero"  | "reboot" | 240      | 600        | 600     | 600           |
      #| ""         | "1-1"       | "1-1" | "0-0" | "powermax" | "powermax-nvmetcp"   | "one-third" | "zero"  | "reboot" | 240      | 600        | 600     | 600          |

  @powerflex-integration
  Scenario Outline: Basic node failover testing using test StatefulSet pods (driver pods down)
    Given a kubernetes <kubeConfig>
    And cluster is clean of test pods
    And wait <nodeCleanSecs> to see there are no taints
    And <podsPerNode> pods per node with <nVol> volumes and <nDev> devices using <driverType> and <storageClass> in <deploySecs>
    Then validate that all pods are running within <deploySecs> seconds
    When I fail <workers> worker driver pod with <failure> failure for <failSecs> and I expect these taints <taints>
    And the taints for the failed nodes are removed within <nodeCleanSecs> seconds
    Then finally cleanup everything

    Examples:
      | kubeConfig | podsPerNode | nVol  | nDev  | driverType | storageClass | workers     | primary | failure     | taints                           | failSecs | deploySecs | runSecs | nodeCleanSecs |
      | ""         | "1-2"       | "1-1" | "1-1" | "vxflexos" | "vxflexos"   | "one-third" | "zero"  | "driverpod" | "offline.vxflexos.storage.dell.com" | 120      | 240        | 300     | 600           |
      | ""         | "1-2"       | "2-2" | "2-2" | "vxflexos" | "vxflexos"   | "one-third" | "zero"  | "driverpod" | "offline.vxflexos.storage.dell.com" | 120      | 240        | 300     | 600           |
      | ""         | "2-5"       | "2-2" | "2-2" | "vxflexos" | "vxflexos"   | "one-third" | "zero"  | "driverpod" | "offline.vxflexos.storage.dell.com" | 120      | 240        | 300     | 600           |


@unity-integration
  Scenario Outline: Basic node failover testing using test StatefulSet pods (driver pods down)
    Given a kubernetes <kubeConfig>
    And cluster is clean of test pods
    And wait <nodeCleanSecs> to see there are no taints
    And <podsPerNode> pods per node with <nVol> volumes and <nDev> devices using <driverType> and <storageClass> in <deploySecs>
    Then validate that all pods are running within <deploySecs> seconds
    When I fail <workers> worker driver pod with <failure> failure for <failSecs> and I expect these taints <taints>
    And the taints for the failed nodes are removed within <nodeCleanSecs> seconds
    Then finally cleanup everything

    Examples:
      | kubeConfig | podsPerNode | nVol  | nDev  | driverType | storageClass   | workers     | primary | failure     |  taints                          | failSecs | deploySecs | runSecs | nodeCleanSecs |
      | ""         | "1-2"       | "1-1" | "0-0" | "unity"    | "unity-nfs"    | "one-third" | "zero"  | "driverpod" | "offline.unity.storage.dell.com" | 120      | 300        | 300     | 600           | 
      | ""         | "1-3"       | "2-2" | "0-0" | "unity"    | "unity-nfs"    | "one-third" | "zero"  | "driverpod" | "offline.unity.storage.dell.com" | 120      | 300        | 300     | 600           |
      | ""         | "1-2"       | "1-1" | "0-0" | "unity"    | "unity-iscsi"  | "one-third" | "zero"  | "driverpod" | "offline.unity.storage.dell.com" | 120      | 300        | 300     | 600           | 
      | ""         | "1-3"       | "2-2" | "0-0" | "unity"    | "unity-iscsi"  | "one-third" | "zero"  | "driverpod" | "offline.unity.storage.dell.com" | 120      | 300        | 300     | 600           |

@powerstore-integration
  Scenario Outline: Basic node failover testing using test StatefulSet pods (driver pods down)
    Given a kubernetes <kubeConfig>
    And cluster is clean of test pods
    And wait <nodeCleanSecs> to see there are no taints
    And <podsPerNode> pods per node with <nVol> volumes and <nDev> devices using <driverType> and <storageClass> in <deploySecs>
    Then validate that all pods are running within <deploySecs> seconds
    When I fail <workers> worker driver pod with <failure> failure for <failSecs> and I expect these taints <taints>
    And the taints for the failed nodes are removed within <nodeCleanSecs> seconds
    Then finally cleanup everything

    Examples:
      | kubeConfig | podsPerNode | nVol  | nDev  | driverType    | storageClass          | workers     | primary | failure     |  taints                               | failSecs | deploySecs | runSecs | nodeCleanSecs |
      | ""         | "1-2"       | "1-1" | "0-0" | "powerstore"  | "powerstore-nfs"      | "one-third" | "zero"  | "driverpod" | "offline.powerstore.storage.dell.com" | 120      | 300        | 300     | 600           | 
      | ""         | "1-3"       | "2-2" | "0-0" | "powerstore"  | "powerstore-nfs"      | "one-third" | "zero"  | "driverpod" | "offline.powerstore.storage.dell.com" | 120      | 300        | 300     | 600           |
      | ""         | "1-2"       | "1-1" | "0-0" | "powerstore"  | "powerstore-iscsi"    | "one-third" | "zero"  | "driverpod" | "offline.powerstore.storage.dell.com" | 120      | 300        | 300     | 600           | 
      | ""         | "1-3"       | "2-2" | "0-0" | "powerstore"  | "powerstore-iscsi"    | "one-third" | "zero"  | "driverpod" | "offline.powerstore.storage.dell.com" | 120      | 300        | 300     | 600           |
      #| ""         | "1-2"       | "1-1" | "0-0" | "powerstore"  | "powerstore-nvmetcp"  | "one-third" | "zero"  | "driverpod" | "offline.powerstore.storage.dell.com" | 120      | 300        | 300     | 600           | 
      #| ""         | "1-3"       | "2-2" | "0-0" | "powerstore"  | "powerstore-nvmetcp"  | "one-third" | "zero"  | "driverpod" | "offline.powerstore.storage.dell.com" | 120      | 300        | 300     | 600           |

  @powerstore-short-integration
  Scenario Outline: Basic node failover testing using test StatefulSet pods (driver pods down)
    Given a kubernetes <kubeConfig>
    And cluster is clean of test pods
    And wait <nodeCleanSecs> to see there are no taints
    And <podsPerNode> pods per node with <nVol> volumes and <nDev> devices using <driverType> and <storageClass> in <deploySecs>
    Then validate that all pods are running within <deploySecs> seconds
    When I fail <workers> worker driver pod with <failure> failure for <failSecs> and I expect these taints <taints>
    And the taints for the failed nodes are removed within <nodeCleanSecs> seconds
    Then finally cleanup everything

    Examples:
      | kubeConfig | podsPerNode | nVol  | nDev  | driverType    | storageClass          | workers     | primary | failure     |  taints                               | failSecs | deploySecs | runSecs | nodeCleanSecs |
      # Uncomment the storageclass to use. The default is set to nvme which is supported by nightly qualification.
      #| ""         | "1-1"       | "1-1" | "0-0" | "powerstore"  | "powerstore-nfs"      | "one-third" | "zero"  | "driverpod" | "offline.powerstore.storage.dell.com" | 120      | 300        | 300     | 600           |
      #| ""         | "1-1"       | "1-1" | "0-0" | "powerstore"  | "powerstore-iscsi"    | "one-third" | "zero"  | "driverpod" | "offline.powerstore.storage.dell.com" | 120      | 300        | 300     | 600           |
      | ""         | "1-1"       | "1-1" | "0-0" | "powerstore"  | "powerstore-nvmetcp"  | "one-third" | "zero"  | "driverpod" | "offline.powerstore.storage.dell.com" | 120      | 300        | 300     | 600           |

  @powerscale-short-integration
    Scenario Outline: Basic node failover testing using test StatefulSet pods (node slow reboots)
      Given a kubernetes <kubeConfig>
      And cluster is clean of test pods
      And wait <nodeCleanSecs> to see there are no taints
      And <podsPerNode> pods per node with <nVol> volumes and <nDev> devices using <driverType> and <storageClass> in <deploySecs>
      Then validate that all pods are running within <deploySecs> seconds
      When I fail <workers> worker nodes and <primary> primary nodes with <failure> failure for <failSecs> seconds
      Then validate that all pods are running within <runSecs> seconds
      And labeled pods are on a different node
      And the taints for the failed nodes are removed within <nodeCleanSecs> seconds
      Then finally cleanup everything
      Examples:
        | kubeConfig | podsPerNode | nVol  | nDev  | driverType | storageClass | workers     | primary | failure  | failSecs | deploySecs | runSecs | nodeCleanSecs |
        | ""         | "1-2"       | "1-1" | "0-0" | "isilon" | "isilon"   | "one-third" | "zero"  | "reboot" | 240      | 600        | 600     | 600          |
        | ""         | "1-2"       | "2-2" | "0-0" | "isilon" | "isilon"   | "one-third" | "zero"  | "reboot" | 600      | 900        | 900     | 900           |


  @powerscale-short-integration
    Scenario Outline: Basic node failover testing using test StatefulSet pods (node kubelet down)
      Given a kubernetes <kubeConfig>
      And cluster is clean of test pods
      And wait <nodeCleanSecs> to see there are no taints
      And <podsPerNode> pods per node with <nVol> volumes and <nDev> devices using <driverType> and <storageClass> in <deploySecs>
      Then validate that all pods are running within <deploySecs> seconds
      When I fail <workers> worker nodes and <primary> primary nodes with <failure> failure for <failSecs> seconds
      Then validate that all pods are running within <runSecs> seconds
      And labeled pods are on a different node
      And the taints for the failed nodes are removed within <nodeCleanSecs> seconds
      Then finally cleanup everything

      Examples:
        | kubeConfig | podsPerNode | nVol  | nDev  | driverType | storageClass | workers     | primary | failure       | failSecs | deploySecs | runSecs | nodeCleanSecs |
        | ""         | "1-2"       | "1-1" | "0-0" | "isilon"   | "isilon"     | "one-third" | "zero"  | "kubeletdown" | 600      | 900        | 900     | 900           |
        | ""         | "3-5"       | "1-1" | "0-0" | "isilon"   | "isilon"     | "one-third" | "zero"  | "kubeletdown" | 600      | 900        | 900     | 900           |

  @powermax-integration
  Scenario Outline: Basic node failover testing using test StatefulSet pods (driver pods down)
    Given a kubernetes <kubeConfig>
    And cluster is clean of test pods
    And wait <nodeCleanSecs> to see there are no taints
    And <podsPerNode> pods per node with <nVol> volumes and <nDev> devices using <driverType> and <storageClass> in <deploySecs>
    Then validate that all pods are running within <deploySecs> seconds
    When I fail <workers> worker driver pod with <failure> failure for <failSecs> and I expect these taints <taints>
    And the taints for the failed nodes are removed within <nodeCleanSecs> seconds
    Then finally cleanup everything

    Examples:
      | kubeConfig | podsPerNode | nVol  | nDev  | driverType    | storageClass          | workers     | primary | failure     |  taints                               | failSecs | deploySecs | runSecs | nodeCleanSecs |
      | ""         | "1-2"       | "1-1" | "0-0" | "powermax"  | "powermax-nfs"      | "one-third" | "zero"  | "driverpod" | "offline.powermax.storage.dell.com" | 120      | 300        | 300     | 600           |
      | ""         | "1-3"       | "2-2" | "0-0" | "powermax"  | "powermax-nfs"      | "one-third" | "zero"  | "driverpod" | "offline.powermax.storage.dell.com" | 120      | 300        | 300     | 600           |
      | ""         | "1-2"       | "1-1" | "0-0" | "powermax"  | "powermax-iscsi"    | "one-third" | "zero"  | "driverpod" | "offline.powermax.storage.dell.com" | 120      | 300        | 300     | 600           |
      | ""         | "1-3"       | "2-2" | "0-0" | "powermax"  | "powermax-iscsi"    | "one-third" | "zero"  | "driverpod" | "offline.powermax.storage.dell.com" | 120      | 300        | 300     | 600           |
      #| ""         | "1-2"       | "1-1" | "0-0" | "powermax"  | "powermax-nvmetcp"  | "one-third" | "zero"  | "driverpod" | "offline.powermax.storage.dell.com" | 120      | 300        | 300     | 600           |
      #| ""         | "1-3"       | "2-2" | "0-0" | "powermax"  | "powermax-nvmetcp"  | "one-third" | "zero"  | "driverpod" | "offline.powermax.storage.dell.com" | 120      | 300        | 300     | 600           |

  @powermax-short-integration
  Scenario Outline: Basic node failover testing using test StatefulSet pods (driver pods down)
    Given a kubernetes <kubeConfig>
    And cluster is clean of test pods
    And wait <nodeCleanSecs> to see there are no taints
    And <podsPerNode> pods per node with <nVol> volumes and <nDev> devices using <driverType> and <storageClass> in <deploySecs>
    Then validate that all pods are running within <deploySecs> seconds
    When I fail <workers> worker driver pod with <failure> failure for <failSecs> and I expect these taints <taints>
    And the taints for the failed nodes are removed within <nodeCleanSecs> seconds
    Then finally cleanup everything

    Examples:
      | kubeConfig | podsPerNode | nVol  | nDev  | driverType    | storageClass          | workers     | primary | failure     |  taints                               | failSecs | deploySecs | runSecs | nodeCleanSecs |
      | ""         | "1-1"       | "1-1" | "0-0" | "powermax"    | "powermax-nfs"        | "one-third" | "zero"  | "driverpod" | "offline.powermax.storage.dell.com"   | 120      | 300        | 300     | 600           |
      | ""         | "1-1"       | "1-1" | "0-0" | "powermax"    | "powermax-iscsi"      | "one-third" | "zero"  | "driverpod" | "offline.powermax.storage.dell.com"   | 120      | 300        | 300     | 600           |
      #| ""         | "1-1"       | "1-1" | "0-0" | "powermax"  | "powermax-nvmetcp"  | "one-third" | "zero"  | "driverpod" | "offline.powermax.storage.dell.com" | 120      | 300        | 300     | 600           |
