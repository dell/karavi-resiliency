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
      | ""         | "csi-vxflexos.dellemc.com" | "vxflexos" | "vxflexos" | "vxflexos,vxflexos-xfs,vxflexos-nvmetcp" |

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
  
  @powerstore-metro-int-setup-check
  Scenario Outline: Validate that we have a valid k8s configuration for the PowerStore metro integration tests
    Given a kubernetes <kubeConfig>
    And test environmental variables are set
    And these CSI driver <driverNames> are configured on the system
    And these storageClasses <storageClasses> exist in the cluster
    And there is a <namespace> in the cluster
    And there are driver pods in <namespace> with this <name> prefix
    And <cliTool> is installed on this machine
    And can logon to nodes and drop test scripts
    Examples:
      | kubeConfig | driverNames                  | namespace      | name         | storageClasses     | cliTool  |
      | ""         | "csi-powerstore.dellemc.com" | "powerstore"   | "powerstore" | "powerstore-metro" | "pstcli" |

  @powerstore-metro-int-nonuniform-setup-check
  Scenario Outline: Validate that we have a valid k8s configuration for the PowerStore non-uniform metro integration tests
    Given a kubernetes <kubeConfig>
    And test metro environmental variables are set
    And these CSI driver <driverNames> are configured on the system
    And these storageClasses <storageClasses> exist in the cluster
    And there is a <namespace> in the cluster
    And there are driver pods in <namespace> with this <name> prefix
    And <cliTool> is installed on this machine
    And can logon to nodes and drop test scripts
    And there are nodes labelled <preferred>
    And there are nodes labelled <nonpreferred>
    Examples:
      | kubeConfig | driverNames                  | namespace      | name         | storageClasses     | cliTool  | preferred   | nonpreferred |
      | ""         | "csi-powerstore.dellemc.com" | "powerstore"   | "powerstore" | "powerstore-metro" | "pstcli" | "ZoneA" | "ZoneB" |

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
      | ""         | "1-2"       | "1-1" | "1-1" | "vxflexos" | "vxflexos"   | "one-third" | "zero"  | "interfacedown" | 900      | 900        | 900     | 900           |
      | ""         | "1-2"       | "2-2" | "2-2" | "vxflexos" | "vxflexos"   | "one-third" | "zero"  | "interfacedown" | 900      | 900        | 900     | 900           |
      | ""         | "1-2"       | "4-4" | "4-4" | "vxflexos" | "vxflexos"   | "one-third" | "zero"  | "interfacedown" | 900      | 900        | 900     | 900           |
      # Slightly more pods, increasing number of vols and devs
      | ""         | "3-5"       | "1-1" | "1-1" | "vxflexos" | "vxflexos"   | "one-third" | "zero"  | "interfacedown" | 900      | 900        | 900     | 900           |
      | ""         | "3-5"       | "2-2" | "0-0" | "vxflexos" | "vxflexos"   | "one-third" | "zero"  | "interfacedown" | 1800     | 1800       | 1800    | 1800          |
      | ""         | "3-5"       | "4-4" | "4-4" | "vxflexos" | "vxflexos"   | "one-third" | "zero"  | "interfacedown" | 900      | 900        | 900     | 900           |
      
      # Small number of pods, increasing number of vols and devs
      #| ""         | "1-2"       | "1-1" | "1-1" | "vxflexos" | "vxflexos-nvmetcp"   | "one-third" | "zero"  | "interfacedown" | 900      | 900        | 900     | 900           |
      #| ""         | "1-2"       | "2-2" | "2-2" | "vxflexos" | "vxflexos-nvmetcp"   | "one-third" | "zero"  | "interfacedown" | 900      | 900        | 900     | 900           |
      #| ""         | "1-2"       | "4-4" | "4-4" | "vxflexos" | "vxflexos-nvmetcp"   | "one-third" | "zero"  | "interfacedown" | 900      | 900        | 900     | 900           |
      # Slightly more pods, increasing number of vols and devs
      #| ""         | "3-5"       | "1-1" | "1-1" | "vxflexos" | "vxflexos-nvmetcp"   | "one-third" | "zero"  | "interfacedown" | 900      | 900        | 900     | 900           |
      #| ""         | "3-5"       | "2-2" | "0-0" | "vxflexos" | "vxflexos-nvmetcp"   | "one-third" | "zero"  | "interfacedown" | 1800     | 1800       | 1800    | 1800          |
      #| ""         | "3-5"       | "4-4" | "4-4" | "vxflexos" | "vxflexos-nvmetcp"   | "one-third" | "zero"  | "interfacedown" | 900      | 900        | 900     | 900           |

  @powerflex-sanity-test
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
      # Slightly more pods, increasing number of vols and devs
      | ""         | "3-5"       | "2-2" | "0-0" | "vxflexos" | "vxflexos"   | "one-third" | "zero"  | "interfacedown" | 1800     | 1800       | 1800    | 1800          |
      #| ""        | "3-5"       | "2-2" | "0-0" | "vxflexos" | "vxflexos-nvmetcp"   | "one-third" | "zero"  | "interfacedown" | 1800     | 1800       | 1800    | 1800          |

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

      # Small number of pods, increasing number of vols and devs
      #| ""         | "1-2"       | "1-1" | "1-1" | "vxflexos" | "vxflexos-nvmetcp"   | "one-third" | "zero"  | "kubeletdown" | 900      | 1200        | 1200     | 1200           |
      #| ""         | "1-2"       | "2-2" | "2-2" | "vxflexos" | "vxflexos-nvmetcp"   | "one-third" | "zero"  | "kubeletdown" | 900      | 1200        | 1200     | 1200           |
      #| ""         | "1-2"       | "4-4" | "4-4" | "vxflexos" | "vxflexos-nvmetcp"   | "one-third" | "zero"  | "kubeletdown" | 900      | 1200        | 1200     | 1200           |
      # Slightly more pods, increasing number of vols and devs
      #| ""         | "3-5"       | "1-1" | "1-1" | "vxflexos" | "vxflexos-nvmetcp"   | "one-third" | "zero"  | "kubeletdown" | 900      | 1200        | 1200     | 1200           |
      #| ""         | "3-5"       | "2-2" | "2-2" | "vxflexos" | "vxflexos-nvmetcp"   | "one-third" | "zero"  | "kubeletdown" | 900      | 1200        | 1200     | 1200           |
      #| ""         | "3-5"       | "4-4" | "4-4" | "vxflexos" | "vxflexos-nvmetcp"   | "one-third" | "zero"  | "kubeletdown" | 900      | 1200        | 1200     | 1200           |

  @unity-integration @unity-sanity-test
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

  @powerscale-integration @powerscale-sanity-test
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
      | ""         | "1-2"       | "1-1" | "0-0" | "isilon"   | "isilon"     | "one-third" | "zero"  | "interfacedown" | 120      | 900        | 900     | 900           |
      | ""         | "3-5"       | "2-2" | "0-0" | "isilon"   | "isilon"     | "one-third" | "zero"  | "interfacedown" | 240      | 900        | 900     | 900           |
  
  @powerstore-integration @powerstore-sanity-test
  Scenario Outline: Basic node failover testing using test StatefulSet pods (node interface down)
    Given a kubernetes <kubeConfig>
    And cluster is clean of test pods
    And wait <nodeCleanSecs> to see there are no taints
    And <podsPerNode> pods per node with <nVol> volumes and <nDev> devices using <driverType> and <storageClass> in <deploySecs>
    Then validate that all pods are running within <deploySecs> seconds
    When I fail <workers> worker nodes and <primary> primary nodes with <failure> failure for <failSecs> seconds
    Then wait up to <failSecs> seconds for pods to switch nodes
    Then validate that all pods are running within <runSecs> seconds
    And labeled pods are on a different node
    And the taints for the failed nodes are removed within <nodeCleanSecs> seconds
    Then finally cleanup everything

    Examples:
      | kubeConfig | podsPerNode | nVol  | nDev  | driverType | storageClass  | workers     | primary | failure         | failSecs | deploySecs | runSecs | nodeCleanSecs |
     # Small number of pods, increasing number of vols and devs
      #| ""         | "1-2"       | "1-1" | "0-0" | "powerstore"    | "powerstore-nfs" | "one-third" | "zero"  | "interfacedown" | 120      | 600        | 600     | 600           |
      #| ""         | "3-5"       | "2-2" | "0-0" | "powerstore"    | "powerstore-nfs" | "one-third" | "zero"  | "interfacedown" | 240      | 600        | 600     | 600           |
      # Small number of pods, increasing number of vols and devs
      | ""         | "1-2"       | "1-1" | "0-0" | "powerstore"    | "powerstore-iscsi" | "one-third" | "zero"  | "interfacedown" | 120      | 600        | 600     | 600           |
      | ""         | "3-5"       | "2-2" | "0-0" | "powerstore"    | "powerstore-iscsi" | "one-third" | "zero"  | "interfacedown" | 240      | 600        | 600     | 600           |
      # Small number of pods, increasing number of vols and devs
      #| ""         | "1-2"       | "1-1" | "0-0" | "powerstore"    | "powerstore-nvmetcp" | "one-third" | "zero"  | "interfacedown" | 120      | 600        | 600     | 600           |
      #| ""         | "3-5"       | "2-2" | "0-0" | "powerstore"    | "powerstore-nvmetcp" | "one-third" | "zero"  | "interfacedown" | 240      | 600        | 600     | 600           |

  @powerstore-integration @powerstore-metro-integration
  Scenario Outline: Preferred cluster node failure hosting metro volumes testing using StatefulSet pods (node interface down)
    Given a kubernetes <kubeConfig>
    And cluster is clean of test pods
    And wait <nodeCleanSecs> to see there are no taints
    And label <workers> node as <preferred> site
    And <podsPerNode> pods per node with <nVol> volumes and <nDev> devices using <driverType> and <storageClass> in <deploySecs> with <preferred> affinity
    Then validate that all pods are running within <deploySecs> seconds
    And all pods are running on <preferred> node
    When I fail labeled <preferred> nodes with <failure> failure for <failSecs> seconds
    Then wait up to <failSecs> seconds for pods to switch nodes
    And validate that all pods are running within <runSecs> seconds
    And labeled pods are on a different node
    And the taints for the failed nodes are removed within <nodeCleanSecs> seconds
    Then finally cleanup everything

    Examples:
      | kubeConfig | podsPerNode | nVol  | nDev  | driverType   | storageClass       | workers     | primary | failure         | failSecs | deploySecs | runSecs | nodeCleanSecs | preferred |
      | ""         | "1-1"       | "1-1" | "0-0" | "powerstore" | "powerstore-metro" | "one-third" | "zero"  | "interfacedown" | 240      | 600        | 600     | 600           | "ZoneA"    |

  @powerstore-integration @powerstore-metro-integration
  Scenario Outline: Preferred site node failover to preferred node (w/ metro, multiple preferred nodes)
    Given a kubernetes <kubeConfig>
    And there are at least <nNodes> worker nodes which are ready
    And cluster is clean of test pods
    And wait <nodeCleanSecs> to see there are no taints
    And label <workers> node as <preferred> site
    And <podsPerNode> pods per node with <nVol> volumes and <nDev> devices using <driverType> and <storageClass> in <deploySecs> with <preferred> affinity
    Then validate that all pods are running within <deploySecs> seconds
    And all pods are running on <preferred> node
    When I fail <workers> nodes with label <preferred> with <failure> failure for <failSecs> seconds
    Then wait up to <failSecs> seconds for pods to switch nodes
    And validate that all pods are running within <runSecs> seconds
    And labeled pods are on a <preferred> node
    And the taints for the failed nodes are removed within <nodeCleanSecs> seconds
    Then finally cleanup everything

    Examples:
      | kubeConfig | nNodes | podsPerNode | nVol  | nDev  | driverType   | storageClass       | workers     |  failure         | failSecs | deploySecs | runSecs | nodeCleanSecs | preferred |
      | ""         | 4      | "1-1"       | "1-1" | "0-0" | "powerstore" | "powerstore-metro" | "one-half"  |  "interfacedown" | 240      | 600        | 600     | 600           | "ZoneA"    |
  
  @powerstore-integration @powerstore-metro-integration
  Scenario Outline: Recovery of preferred-site node testing using test StatefulSet pods (node interface down)
    Given a kubernetes <kubeConfig>
    And cluster is clean of test pods
    And wait <nodeCleanSecs> to see there are no taints
    And label <workers> node as <preferred> site
    And <podsPerNode> pods per node with <nVol> volumes and <nDev> devices using <driverType> and <storageClass> in <deploySecs> with <preferred> affinity
    Then validate that all pods are running within <deploySecs> seconds
    And all pods are running on <preferred> node
    When I fail labeled <preferred> nodes with <failure> failure for <failSecs> seconds
    Then wait up to <failSecs> seconds for pods to switch nodes
    And validate that all pods are running within <deploySecs> seconds
    And labeled pods are on a different node
    And pods are scheduled on the non preferred nodes
    And the taints for the failed nodes are removed within <nodeCleanSecs> seconds
    And verify pods do not migrate for <migrateSecs> seconds
    Then finally cleanup everything

    Examples:
      | kubeConfig | podsPerNode | nVol  | nDev  | driverType   | storageClass       | workers     | primary | failure         | failSecs | deploySecs | nodeCleanSecs | preferred | migrateSecs |
      | ""         | "1-1"       | "1-1" | "0-0" | "powerstore" | "powerstore-metro" | "one-third" | "zero"  | "interfacedown" | 300      | 600        | 600           | "ZoneA"    | 150         |

  @powerstore-integration @powerstore-metro-integration
  Scenario Outline: Recovery of preferred metro array on preferred node testing using test StatefulSet pods (iptables drop iscsi)
    Given a kubernetes <kubeConfig>
    And a driver namespace name <driverNamespaceName>
    And a driver secret name <driverSecretName>
    And cluster is clean of test pods
    And wait <nodeCleanSecs> to see there are no taints
    And label <workers> node as <preferred> site
    And <podsPerNode> pods per node with <nVol> volumes and <nDev> devices using <driverType> and <storageClass> in <deploySecs> with <preferred> affinity
    Then validate that all pods are running within <deploySecs> seconds
    And all pods are running on <preferred> node
    Then the connection fails between the preferred metro array and the nodes with <preferred> label
    And nodes with pods and with <preferred> label have taint <taint> within <failSecs> seconds
    Then verify pods do not migrate for <migrateSecs> seconds
    When the connection is restored between the preferred metro array and the nodes with <preferred> label
    And validate that all pods are running within <runSecs> seconds
    And all pods are running on <preferred> node
    And the taints for the failed nodes are removed within <nodeCleanSecs> seconds
    And verify pods do not migrate for <migrateSecs> seconds
    Then finally cleanup everything

    Examples:
      | kubeConfig  | podsPerNode | nVol  | nDev  | driverNamespaceName | driverSecretName      | driverType    | storageClass        | workers     | migrateSecs | failSecs  | deploySecs  | nodeCleanSecs | runSecs | preferred | taint                                 |
      | ""          | "1-1"       | "1-1" | "0-0" | "powerstore"        | "powerstore-config"   | "powerstore"  | "powerstore-metro"  | "one-third" | 300         | 300       | 300         | 300           | 300     | "ZoneA"    | "powerstore.podmon.storage.dell.com"  |
  
  @powerstore-integration @powerstore-metro-integration
  Scenario Outline: Non-Preferred array failure from the preferred node testing using test StatefulSet pods (iptables drop iscsi)
    Given a kubernetes <kubeConfig>
    And a driver namespace name <driverNamespaceName>
    And a driver secret name <driverSecretName>
    And cluster is clean of test pods
    And wait <nodeCleanSecs> to see there are no taints
    And label <workers> node as <preferred> site
    And <podsPerNode> pods per node with <nVol> volumes and <nDev> devices using <driverType> and <storageClass> in <deploySecs> with <preferred> affinity
    Then validate that all pods are running within <deploySecs> seconds
    And all pods are running on <preferred> node
    Then the connection fails between the non preferred metro array and the nodes with <preferred> label
    Then verify pods do not migrate for <migrateSecs> seconds
    And validate that all pods are running within <runSecs> seconds
    And all pods are running on <preferred> node
    When the connection is restored between the non preferred metro array and the nodes with <preferred> label
    And validate that all pods are running within <runSecs> seconds
    And all pods are running on <preferred> node
    And the taints for the failed nodes are removed within <nodeCleanSecs> seconds
    And verify pods do not migrate for <migrateSecs> seconds
    Then finally cleanup everything

    Examples:
      | kubeConfig  | podsPerNode | nVol  | nDev  | driverNamespaceName | driverSecretName      | driverType    | storageClass        | workers     | migrateSecs | failSecs  | deploySecs  | nodeCleanSecs | runSecs | preferred | taint                                 |
      | ""          | "1-1"       | "1-1" | "0-0" | "powerstore"        | "powerstore-config"   | "powerstore"  | "powerstore-metro"  | "one-third" | 300         | 300       | 300         | 300           | 300     | "ZoneA"    | "powerstore.podmon.storage.dell.com"  |

  @powerstore-integration @powerstore-metro-integration
  Scenario Outline: All Non-Preferred-site Node Failure - Few nodes having preferred node labels not set
    Given a kubernetes <kubeConfig>
    And cluster is clean of test pods
    And wait <nodeCleanSecs> to see there are no taints
    And label <workers> node as <preferred> site
    And <podsPerNode> pods per node with <nVol> volumes and <nDev> devices using <driverType> and <storageClass> in <deploySecs> with <preferred> affinity
    Then validate that all pods are running within <deploySecs> seconds
    And all pods are running on <preferred> node
    When I fail non <preferred> nodes with <failure> failure for <failSecs> seconds
    Then validate that all pods are running within <deploySecs> seconds
    And all pods are running on <preferred> node
    Then finally cleanup everything

    Examples:
      | kubeConfig | podsPerNode | nVol  | nDev  | driverType   | storageClass         | workers      | failure         | failSecs | deploySecs | runSecs | nodeCleanSecs | preferred |
      | ""         | "1-1"       | "1-1" | "0-0" | "powerstore" | "powerstore-metro"   | "one-half"   | "interfacedown" | 120      | 600        | 600     | 600           | "ZoneA"|
    
  @powerstore-metro-integration-nonuniform
  Scenario Outline:  All nodes on preferred site fail; test pod moves to non-preferred node; when preferred site nodes come back, pod does not move
    Given a kubernetes <kubeConfig>
    And a driver namespace name <driverNamespaceName>
    And the arrays in storageclass <storageClass> are in non uniform configuration
    And cluster is clean of test pods but may have labels
    And there are nodes labelled <preferred>
    And there are nodes labelled <nonpreferred>
    And wait <nodeCleanSecs> to see there are no taints
    And <podsPerNode> pods per node with <nVol> volumes and <nDev> devices using <driverType> and <storageClass> in <deploySecs> with <preferred> affinity
    Then validate that all pods are running within <deploySecs> seconds
    And all pods are running on <preferred> node
    When I fail labeled <preferred> nodes with <failure> failure for <failSecs> seconds
    Then wait up to <failSecs> seconds for pods to switch nodes
    And validate that all pods are running within <deploySecs> seconds
    And labeled pods are on a different node
    And pods are scheduled on the non preferred nodes
    And the taints for the failed nodes are removed within <nodeCleanSecs> seconds
    And verify pods do not migrate for <migrateSecs> seconds
    Then finally cleanup everything except labels

    Examples:
      | kubeConfig | podsPerNode | nVol  | nDev  | driverType   | storageClass       | workers     | primary | failure         | failSecs | deploySecs | nodeCleanSecs | preferred | migrateSecs | driverNamespaceName | nonpreferred |
      | ""         | "1-1"       | "1-1" | "0-0" | "powerstore" | "powerstore-metro" | "one-third" | "zero"  | "interfacedown" | 300      | 600        | 600           | "ZoneA"    | 150         | "powerstore"  |  "ZoneB"  |

  @powerstore-metro-integration-nonuniform
  Scenario Outline: Non-Uniform Metro; Metro connectivity and Preferred site nodes fail - pod run on non-preferred nodes after manual promotion
    Given a kubernetes <kubeConfig>
    And a driver namespace name <driverNamespaceName>
    And the arrays in storageclass <storageClass> are in non uniform configuration
    And cluster is clean of test pods but may have labels
    And there are nodes labelled <preferred>
    And there are nodes labelled <nonpreferred>
    And wait <nodeCleanSecs> to see there are no taints
    And <podsPerNode> pods per node with <nVol> volumes and <nDev> devices using <driverType> and <storageClass> in <deploySecs> with <preferred> affinity
    Then validate that all pods are running within <deploySecs> seconds
    And all pods are running on <preferred> node
    And I ensure that all metro volumes in test namespaces on <storageClass> for <driverType> are stable
    When I disrupt metro connectivity between arrays in storage class <storageClass>
    And I fail labeled <preferred> nodes with <failure> failure for <failSecs> seconds
    Then wait up to <failSecs> seconds for pods to switch nodes
    And validate that all pods are not running within <deploySecs> seconds
    And I execute <actionA> on preferred array on <storageClass> for <driverType> for metro volumes
    And I execute <actionB> on non preferred array on <storageClass> for <driverType> for metro volumes
    And validate that all pods are running within <deploySecs> seconds
    And labeled pods are on a different node
    And pods are scheduled on the non preferred nodes
    And the taints for the failed nodes are removed within <nodeCleanSecs> seconds
    And I restore metro connectivity between arrays in storage class <storageClass>
    Then finally cleanup everything except labels

    Examples:
      | kubeConfig | podsPerNode | nVol  | nDev  | driverType   | storageClass       | workers     | primary | failure         | failSecs | deploySecs | runSecs | nodeCleanSecs | preferred | migrateSecs | driverNamespaceName | nonpreferred | actionA | actionB |
      | ""         | "1-1"       | "1-1" | "0-0" | "powerstore" | "powerstore-metro" | "one-third" | "zero"  | "interfacedown" | 240      | 240        | 600     | 600           | "ZoneA"    | 150         | "powerstore"  |  "ZoneB"  | "demote"  |  "promote"  |

@powerstore-metro-integration-nonuniform
  Scenario Outline: Non-Uniform Metro; Volume offline on both arrays; workload runs after manual promotion of non-preferred
    Given a kubernetes <kubeConfig>
    And a driver namespace name <driverNamespaceName>
    And the arrays in storageclass <storageClass> are in non uniform configuration
    And cluster is clean of test pods but may have labels
    And there are nodes labelled <preferred>
    And there are nodes labelled <nonpreferred>
    And wait <nodeCleanSecs> to see there are no taints
    And <podsPerNode> pods per node with <nVol> volumes and <nDev> devices using <driverType> and <storageClass> in <deploySecs> with <preferred> affinity
    Then validate that all pods are running within <deploySecs> seconds
    And all pods are running on <preferred> node
    And I ensure that all metro volumes in test namespaces on <storageClass> for <driverType> are stable
    When I disrupt metro connectivity between arrays in storage class <storageClass>
    And I execute <actionA> on preferred array on <storageClass> for <driverType> for metro volumes
    And I execute <actionA> on non preferred array on <storageClass> for <driverType> for metro volumes
    And I fail labeled <preferred> nodes with <failure> failure for <failSecs> seconds
    Then wait up to <failSecs> seconds for pods to switch nodes
    And validate that all pods are not running within <deploySecs> seconds
    And I execute <actionA> on preferred array on <storageClass> for <driverType> for metro volumes
    And I execute <actionB> on non preferred array on <storageClass> for <driverType> for metro volumes
    And validate that all pods are running within <deploySecs> seconds
    And labeled pods are on a different node
    And pods are scheduled on the non preferred nodes
    And the taints for the failed nodes are removed within <nodeCleanSecs> seconds
    And I restore metro connectivity between arrays in storage class <storageClass>
    Then finally cleanup everything except labels

    Examples:
      | kubeConfig | podsPerNode | nVol  | nDev  | driverType   | storageClass       | workers     | primary | failure         | failSecs | deploySecs | runSecs | nodeCleanSecs | preferred | migrateSecs | driverNamespaceName | nonpreferred | actionA | actionB |
      | ""         | "1-1"       | "1-1" | "0-0" | "powerstore" | "powerstore-metro" | "one-third" | "zero"  | "interfacedown" | 240      | 240        | 600     | 600           | "ZoneA"    | 150         | "powerstore"  |  "ZoneB"  | "demote"  |  "promote"  |

  @powerstore-metro-integration-nonuniform
  Scenario Outline: Non-Uniform Metro; Volume offline on both arrays; workload runs after manual promotion of preferred
    Given a kubernetes <kubeConfig>
    And a driver namespace name <driverNamespaceName>
    And the arrays in storageclass <storageClass> are in non uniform configuration
    And cluster is clean of test pods but may have labels
    And there are nodes labelled <preferred>
    And there are nodes labelled <nonpreferred>
    And wait <nodeCleanSecs> to see there are no taints
    And <podsPerNode> pods per node with <nVol> volumes and <nDev> devices using <driverType> and <storageClass> in <deploySecs> with <nonpreferred> affinity
    Then validate that all pods are running within <deploySecs> seconds
    And all pods are running on <nonpreferred> node
    And I ensure that all metro volumes in test namespaces on <storageClass> for <driverType> are stable
    When I disrupt metro connectivity between arrays in storage class <storageClass>
    And I execute <actionB> on preferred array on <storageClass> for <driverType> for metro volumes
    And I execute <actionB> on non preferred array on <storageClass> for <driverType> for metro volumes
    And I fail labeled <nonpreferred> nodes with <failure> failure for <failSecs> seconds
    Then wait up to <failSecs> seconds for pods to switch nodes
    And validate that all pods are not running within <deploySecs> seconds
    And I execute <actionA> on preferred array on <storageClass> for <driverType> for metro volumes
    And I execute <actionB> on non preferred array on <storageClass> for <driverType> for metro volumes
    And validate that all pods are running within <deploySecs> seconds
    And labeled pods are on a different node
    And all pods are running on <preferred> node
    And the taints for the failed nodes are removed within <nodeCleanSecs> seconds
    And I restore metro connectivity between arrays in storage class <storageClass>
    Then finally cleanup everything except labels

    Examples:
      | kubeConfig | podsPerNode | nVol  | nDev  | driverType   | storageClass       | workers     | primary | failure         | failSecs | deploySecs | runSecs | nodeCleanSecs | preferred | migrateSecs | driverNamespaceName | nonpreferred | actionA | actionB |
      | ""         | "1-1"       | "1-1" | "0-0" | "powerstore" | "powerstore-metro" | "one-third" | "zero"  | "interfacedown" | 240      | 240        | 600     | 600           | "ZoneA"    | 150         | "powerstore"  |  "ZoneB"  | "promote"  |  "demote"  |


  @pstcli-integration-validation
  Scenario Outline: PowerStore Metro - pstcli Validation
    Given a kubernetes <kubeConfig>
    And a driver namespace name <driverNamespaceName>
    And finally cleanup everything except labels
    And wait <nodeCleanSecs> to see there are no taints
    And there are nodes labelled <preferred>
    And I verify that <storageClass> is immediate binding
    Then I deploy <nVols> on <storageClass> for <driverType>
    And I ensure all volumes on <storageClass> for <driverType> are Bound
    And I ensure that all metro volumes on <storageClass> for <driverType> are stable
    Then I execute <actionA> on preferred array on <storageClass> for <driverType> for metro volumes
    Then I execute <actionB> on preferred array on <storageClass> for <driverType> for metro volumes
    Then I execute <actionC> on preferred array on <storageClass> for <driverType> for metro volumes
    Then I execute <actionD> on preferred array on <storageClass> for <driverType> for metro volumes
    Then I soft fracture the metro volumes on <storageClass> for <driverType>
    Then I deploy pods for all metro volumes on <storageClass> for <driverType> with <preferred> affinity
    And I clean up all metro pods and volumes for <driverType>

    Examples:
      | kubeConfig | nVols | driverNamespaceName | driverType   | storageClass         | workers      | nodeCleanSecs | preferred  | actionA  | actionB   | actionC   | actionD  |
      | ""         | 1     | "powerstore"        | "powerstore" | "powerstore-metro"   | "one-half"   | 600           | "zone1"    | "pause"  | "demote"  | "promote" | "resume" |

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

  @powermax-integration @powermax-sanity-test
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
    And skip if <failure> is not compatible with <driverType>
    And cluster is clean of test pods
    And wait <nodeCleanSecs> to see there are no taints
    And <podsPerNode> pods per node with <nVol> volumes and <nDev> devices using <driverType> and <storageClass> in <deploySecs>
    Then validate that all pods are running within <deploySecs> seconds
    When I fail <workers> worker nodes and <primary> primary nodes with <failure> failure for <failSecs> seconds
    Then wait up to <failSecs> seconds for pods to switch nodes
    Then validate that all pods are running within <runSecs> seconds
    And labeled pods are on a different node
    And the taints for the failed nodes are removed within <nodeCleanSecs> seconds
    Then finally cleanup everything

    Examples:
      | kubeConfig | podsPerNode | nVol  | nDev  | driverType   | storageClass         | workers     | primary | failure       | failSecs | deploySecs | runSecs | nodeCleanSecs |
      # Small number of pods, increasing number of vols and devs
      | ""         | "1-2"       | "1-1" | "0-0" | "powerstore" | "powerstore-nfs"     | "one-third" | "zero"  | "kubeletdown" | 600      | 900        | 900     | 900           |
      # Slightly more pods, increasing number of vols and devs
      | ""         | "3-5"       | "1-1" | "0-0" | "powerstore" | "powerstore-nfs"     | "one-third" | "zero"  | "kubeletdown" | 600      | 900        | 900     | 900           |
      # Small number of pods, increasing number of vols and devs
      | ""         | "1-2"       | "1-1" | "0-0" | "powerstore" | "powerstore-iscsi"   | "one-third" | "zero"  | "kubeletdown" | 600      | 900        | 900     | 900           |
      # Slightly more pods, increasing number of vols and devs
      | ""         | "3-5"       | "1-1" | "0-0" | "powerstore" | "powerstore-iscsi"   | "one-third" | "zero"  | "kubeletdown" | 600      | 900        | 900     | 900           |
      # Small number of pods, increasing number of vols and devs
      #| ""         | "1-2"      | "1-1" | "0-0" | "powerstore" | "powerstore-nvmetcp" | "one-third" | "zero"  | "kubeletdown" | 600      | 900        | 900     | 900           |
      # Slightly more pods, increasing number of vols and devs
      #| ""         | "3-5"      | "1-1" | "0-0" | "powerstore" | "powerstore-nvmetcp" | "one-third" | "zero"  | "kubeletdown" | 600      | 900        | 900     | 900           |

  @powerstore-short-integration
  Scenario Outline: Basic node failover testing using test StatefulSet pods (node kubelet down)
    Given a kubernetes <kubeConfig>
    And skip if <failure> is not compatible with <driverType>
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

      # Small number of pods, increasing number of vols and devs
      #| ""         | "1-2"       | "1-1" | "1-1" | "vxflexos" | "vxflexos-nvmetcp"   | "one-third" | "zero"  | "reboot" | 120      | 240        | 300     | 600           |
      #| ""         | "1-2"       | "2-2" | "2-2" | "vxflexos" | "vxflexos-nvmetcp"   | "one-third" | "zero"  | "reboot" | 120      | 240        | 300     | 600           |
      #| ""         | "1-2"       | "4-4" | "4-4" | "vxflexos" | "vxflexos-nvmetcp"   | "one-third" | "zero"  | "reboot" | 120      | 240        | 300     | 600           |
      # Slightly more pods, increasing number of vols and devs
      #| ""         | "3-5"       | "1-1" | "1-1" | "vxflexos" | "vxflexos-nvmetcp"   | "one-third" | "zero"  | "reboot" | 240      | 240        | 300     | 600           |
      #| ""         | "3-5"       | "2-2" | "2-2" | "vxflexos" | "vxflexos-nvmetcp"   | "one-third" | "zero"  | "reboot" | 240      | 240        | 300     | 600           |
      #| ""         | "3-5"       | "4-4" | "4-4" | "vxflexos" | "vxflexos-nvmetcp"   | "one-third" | "zero"  | "reboot" | 240      | 240        | 300     | 600           |

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
    And I set the correct driver type to <driverType>
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
      #| ""         | "1-2"       | "1-1" | "1-1" | "vxflexos" | "vxflexos-nvmetcp"   | "one-third" | "zero"  | "interfacedown" | 120      | 300        | 300           |
      #| ""         | "1-2"       | "1-1" | "1-1" | "vxflexos" | "vxflexos-nvmetcp"   | "one-third" | "zero"  | "reboot"        | 120      | 300        | 300           |

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
    And I set the correct driver type to <driverType>
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
    And I set the correct driver type to <driverType>
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
      | kubeConfig | podsPerNode | nVol  | nDev  | driverType   | storageClass           | workers     | primary | failure         | failSecs | deploySecs | nodeCleanSecs |
      | ""         | "1-2"       | "1-1" | "0-0" | "powerstore" | "powerstore-nfs"       | "one-third" | "zero"  | "interfacedown" | 300      | 900        | 900           |
      | ""         | "1-2"       | "1-1" | "0-0" | "powerstore" | "powerstore-iscsi"     | "one-third" | "zero"  | "interfacedown" | 300      | 900        | 900           |
      # | ""         | "1-2"       | "1-1" | "0-0" | "powerstore"   | "powerstore-nvmetcp" | "one-third" | "zero"  | "interfacedown" | 600      | 900        | 900           |

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
      #| ""         | "1-2"       | "1-1" | "1-1" | "vxflexos" | "vxflexos-nvmetcp"   | "one-third" | "zero"  | "interfacedown" | 60       | 300        | 120     | 300           |
      #| ""         | "1-2"       | "1-1" | "1-1" | "vxflexos" | "vxflexos-nvmetcp"   | "one-third" | "zero"  | "reboot"        | 45       | 300        | 120     | 300           |

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
      #| ""         | "1-1"       | "1-1" | "1-1" | "vxflexos" | "vxflexos-nvmetcp"   | "one-third" | "zero"  | "interfacedown" | 120      | 240        | 300     | 600           |
      #| ""         | "1-1"       | "1-1" | "1-1" | "vxflexos" | "vxflexos-nvmetcp"   | "one-third" | "zero"  | "reboot"        | 120      | 240        | 300     | 600           |

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
      | ""         | "1-1"       | "1-1" | "0-0" | "vxflexos" | "vxflexos"   | "one-third" | "zero"  | "interfacedown" | 900      | 900        | 900     | 900           |
      #| ""         | "1-1"       | "1-1" | "0-0" | "vxflexos" | "vxflexos-nvmetcp"   | "one-third" | "zero"  | "interfacedown" | 900      | 900        | 900     | 900           |

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
      | ""         | "1-1"       | "1-1" | "0-0" | "isilon" | "isilon"   | "one-third" | "zero"  | "interfacedown" | 900      | 900        | 900     | 900           |
  
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
      | ""         | "1-1"       | "1-1" | "0-0" | "vxflexos" | "vxflexos"   | "one-third" | "zero"  | "reboot" | 900      | 900        | 900     | 900           |
      #| ""         | "1-1"       | "1-1" | "0-0" | "vxflexos" | "vxflexos-nvmetcp"   | "one-third" | "zero"  | "reboot" | 900      | 900        | 900     | 1200           |

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
      | ""         | "1-1"       | "1-1" | "0-0" | "vxflexos" | "vxflexos"   | "one-third" | "zero"  | "kubeletdown"   | 900      | 900        | 900     | 900           |
      #| ""         | "1-1"       | "1-1" | "0-0" | "vxflexos" | "vxflexos-nvmetcp"   | "one-third" | "zero"  | "kubeletdown"   | 900      | 900        | 900     | 900           |

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
    Then wait up to <failSecs> seconds for pods to switch nodes
    Then validate that all pods are running within <runSecs> seconds
    And labeled pods are on a different node
    And the taints for the failed nodes are removed within <nodeCleanSecs> seconds
    Then finally cleanup everything
    Examples:
      | kubeConfig | podsPerNode | nVol  | nDev  | driverType   | storageClass         | workers     | primary | failure  | failSecs | deploySecs | runSecs | nodeCleanSecs |
      #Small number of pods, increasing number of vols and devs
      | ""         | "1-2"       | "1-1" | "0-0" | "powerstore" | "powerstore-nfs"     | "one-third" | "zero"  | "reboot" | 240      | 600        | 600     | 600           |
      # Slightly more pods, increasing number of vols and devs
      | ""         | "3-5"       | "1-1" | "0-0" | "powerstore" | "powerstore-nfs"     | "one-third" | "zero"  | "reboot" | 240      | 900        | 900     | 900           |
      #Small number of pods, increasing number of vols and devs
      | ""         | "1-2"       | "1-1" | "0-0" | "powerstore" | "powerstore-iscsi"   | "one-third" | "zero"  | "reboot" | 240      | 600        | 600     | 600           |
      # Slightly more pods, increasing number of vols and devs
      | ""         | "3-5"       | "1-1" | "0-0" | "powerstore" | "powerstore-iscsi"   | "one-third" | "zero"  | "reboot" | 240      | 900        | 900     | 900           |
      #Small number of pods, increasing number of vols and devs
      # | ""         | "1-2"       | "1-1" | "0-0" | "powerstore" | "powerstore-nvmetcp" | "one-third" | "zero"  | "reboot" | 240      | 600        | 600     | 600           |
      # Slightly more pods, increasing number of vols and devs
      # | ""         | "3-5"       | "1-1" | "0-0" | "powerstore" | "powerstore-nvmetcp" | "one-third" | "zero"  | "reboot" | 240      | 900        | 900     | 900           |

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
      #| ""         | "1-2"       | "1-1" | "1-1" | "vxflexos" | "vxflexos-nvmetcp"   | "one-third" | "zero"  | "driverpod" | "offline.vxflexos.storage.dell.com" | 120      | 240        | 300     | 600           |
      #| ""         | "1-2"       | "2-2" | "2-2" | "vxflexos" | "vxflexos-nvmetcp"   | "one-third" | "zero"  | "driverpod" | "offline.vxflexos.storage.dell.com" | 120      | 240        | 300     | 600           |
      #| ""         | "2-5"       | "2-2" | "2-2" | "vxflexos" | "vxflexos-nvmetcp"   | "one-third" | "zero"  | "driverpod" | "offline.vxflexos.storage.dell.com" | 120      | 240        | 300     | 600           |      


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
      | ""         | "1-1"       | "1-1" | "0-0" | "isilon" | "isilon"   | "one-third" | "zero"  | "reboot" | 600      | 900        | 1200     | 1200          |

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
      | ""         | "1-1"       | "1-1" | "0-0" | "isilon"   | "isilon"     | "one-third" | "zero"  | "kubeletdown" | 600      | 900        | 900     | 900           |

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


  @powerstore-metro-integration-uniform
  Scenario Outline: Metro Uniform with connectivity between powerstore 1 and 2 failure, all nodes down in ZoneA
    Given a kubernetes <kubeConfig>
    And a driver namespace name <driverNamespaceName>
    And the arrays in storageclass <storageClass> are in uniform configuration
    And there are nodes labelled <preferred>
    And there are nodes labelled <nonpreferred>
    And wait <nodeCleanSecs> to see there are no taints
    And <podsPerNode> pods per node with <nVol> volumes and <nDev> devices using <driverType> and <storageClass> in <deploySecs> with <preferred> affinity
    Then validate that all pods are running within <deploySecs> seconds
    And all pods are running on <preferred> node
    And I ensure that all metro volumes in test namespaces on <storageClass> for <driverType> are stable
    When I disrupt metro connectivity between arrays in storage class <storageClass>
    And I fail labeled <preferred> nodes with <failure> failure for <failSecs> seconds
    Then wait up to <failSecs> seconds for pods to switch nodes
    And pods are scheduled on the non preferred nodes
    And validate that all pods are running within <deploySecs> seconds
    And labeled pods are on a different node
    And I restore metro connectivity between arrays in storage class <storageClass>
    And the taints for the failed nodes are removed within <nodeCleanSecs> seconds
    Then finally cleanup everything except labels

    Examples:
    | kubeConfig | driverNames                    | driverType   | nDev   | nVol  | podsPerNode | storageClass       | primary | failure         | failSecs | deploySecs | runSecs | nodeCleanSecs  | preferred | workers     | migrateSecs | nonpreferred | driverNamespaceName    |
    |         "" | "csi-powerstore.dellemc.com"   | "powerstore" | "0-0"  | "1-1" | "1-1"       | "powerstore-metro" | "zero"  | "interfacedown" | 600      | 600        | 600     | 600            | "ZoneA"   | "one-third" | 150         | "ZoneB"      | "powerstore"           |
 
  @powerstore-metro-integration-uniform
  Scenario Outline:  In Uniform configuration, all on nonpreferred site fail; test pod moves to preferred node; when nonpreferred site nodes come back, pod does not move
    Given a kubernetes <kubeConfig>
    And a driver namespace name <driverNamespaceName>
    And the taints for the failed nodes are removed within <nodeCleanSecs> seconds
    And the arrays in storageclass <storageClass> are in uniform configuration
    And there are nodes labelled <preferred>
    And there are nodes labelled <nonpreferred>
    And wait 30 to see there are no taints
    And <podsPerNode> pods per node with <nVol> volumes and <nDev> devices using <driverType> and <storageClass> in <deploySecs> with <nonpreferred> affinity
    Then validate that all pods are running within <deploySecs> seconds
    And I ensure that all metro volumes in test namespaces on <storageClass> for <driverType> are stable
    And all pods are running on <nonpreferred> node
    When I disrupt metro connectivity between arrays in storage class <storageClass>
    And block connection for <preferred> node to remote array in <storageClass> 
    When I fail labeled <nonpreferred> nodes with <failure> failure for <failSecs> seconds
    Then wait up to <failSecs> seconds for pods to switch nodes
    And validate that all pods are running within <deploySecs> seconds
    And labeled pods are on a different node
    And all pods are running on <preferred> node
    Then Check that there "are" volumejournals
    And the taints for the failed nodes are removed within <nodeCleanSecs> seconds
    And verify pods do not migrate for <migrateSecs> seconds
    And I restore metro connectivity between arrays in storage class <storageClass>
    And restore connection for <preferred> node to remote array in <storageClass> 
    Then finally cleanup everything except labels
    Then Check that there "are not" volumejournals
    And clear out all volumejournals

    Examples:
      | kubeConfig | podsPerNode | nVol  | nDev  | driverType   | storageClass       | workers     | primary | failure         | failSecs | deploySecs | nodeCleanSecs |  migrateSecs | driverNamespaceName | preferred | nonpreferred| 
      | ""         | "1-1"       | "1-1" | "0-0" | "powerstore" | "powerstore-metro" | "one-third" | "zero"  | "interfacedown" | 900      | 900        | 600           |  150         | "powerstore"        |  "ZoneA"  | "ZoneB"     | 

  @powerstore-metro-integration-uniform
  Scenario Outline: Uniform Metro; Volume offline on both arrays; workload runs after manual promotion of nonpreferred
    Given a kubernetes <kubeConfig>
    And a driver namespace name <driverNamespaceName>
    And the arrays in storageclass <storageClass> are in uniform configuration
    And cluster is clean of test pods but may have labels
    And there are nodes labelled <preferred>
    And there are nodes labelled <nonpreferred>
    And wait <nodeCleanSecs> to see there are no taints
    And <podsPerNode> pods per node with <nVol> volumes and <nDev> devices using <driverType> and <storageClass> in <deploySecs> with <preferred> affinity
    Then validate that all pods are running within <deploySecs> seconds
    And all pods are running on <preferred> node
    And I ensure that all metro volumes in test namespaces on <storageClass> for <driverType> are stable
    When I disrupt metro connectivity between arrays in storage class <storageClass>
    And I execute <actionA> on preferred array on <storageClass> for <driverType> for metro volumes
    And I execute <actionA> on non preferred array on <storageClass> for <driverType> for metro volumes
    And I fail labeled <preferred> nodes with <failure> failure for <failSecs> seconds
    Then wait up to <failSecs> seconds for pods to switch nodes
    And validate that all pods are not running within <deploySecs> seconds
    And I execute <actionA> on preferred array on <storageClass> for <driverType> for metro volumes
    And I execute <actionB> on non preferred array on <storageClass> for <driverType> for metro volumes
    And validate that all pods are running within <deploySecs> seconds
    And labeled pods are on a different node
    And all pods are running on <nonpreferred> node
    And the taints for the failed nodes are removed within <nodeCleanSecs> seconds
    And I restore metro connectivity between arrays in storage class <storageClass>
    Then finally cleanup everything except labels

    Examples:
      | kubeConfig | podsPerNode | nVol  | nDev  | driverType   | storageClass       | workers     | primary | failure         | failSecs | deploySecs | runSecs | nodeCleanSecs | preferred | migrateSecs | driverNamespaceName | nonpreferred | actionA | actionB |
      | ""         | "1-1"       | "1-1" | "0-0" | "powerstore" | "powerstore-metro" | "one-third" | "zero"  | "interfacedown" | 240      | 240        | 600     | 600           | "ZoneA"    | 150         | "powerstore"  |  "ZoneB"  | "demote"  |  "promote"  |

  @powerstore-metro-integration-uniform
  Scenario Outline: Uniform Metro; Volume offline on both arrays; workload runs after manual promotion of preferred
    Given a kubernetes <kubeConfig>
    And a driver namespace name <driverNamespaceName>
    And the arrays in storageclass <storageClass> are in uniform configuration
    And cluster is clean of test pods but may have labels
    And there are nodes labelled <preferred>
    And there are nodes labelled <nonpreferred>
    And wait <nodeCleanSecs> to see there are no taints
    And <podsPerNode> pods per node with <nVol> volumes and <nDev> devices using <driverType> and <storageClass> in <deploySecs> with <nonpreferred> affinity
    Then validate that all pods are running within <deploySecs> seconds
    And all pods are running on <nonpreferred> node
    And I ensure that all metro volumes in test namespaces on <storageClass> for <driverType> are stable
    When I disrupt metro connectivity between arrays in storage class <storageClass>
    And I execute <actionB> on preferred array on <storageClass> for <driverType> for metro volumes
    And I execute <actionB> on non preferred array on <storageClass> for <driverType> for metro volumes
    And I fail labeled <nonpreferred> nodes with <failure> failure for <failSecs> seconds
    Then wait up to <failSecs> seconds for pods to switch nodes
    And validate that all pods are not running within <deploySecs> seconds
    And I execute <actionA> on preferred array on <storageClass> for <driverType> for metro volumes
    And I execute <actionB> on non preferred array on <storageClass> for <driverType> for metro volumes
    And validate that all pods are running within <deploySecs> seconds
    And labeled pods are on a different node
    And all pods are running on <preferred> node
    And the taints for the failed nodes are removed within <nodeCleanSecs> seconds
    And I restore metro connectivity between arrays in storage class <storageClass>
    Then finally cleanup everything except labels

    Examples:
      | kubeConfig | podsPerNode | nVol  | nDev  | driverType   | storageClass       | workers     | primary | failure         | failSecs | deploySecs | runSecs | nodeCleanSecs | preferred | migrateSecs | driverNamespaceName | nonpreferred | actionA | actionB |
      | ""         | "1-1"       | "1-1" | "0-0" | "powerstore" | "powerstore-metro" | "one-third" | "zero"  | "interfacedown" | 240      | 240        | 600     | 600           | "ZoneA"    | 150         | "powerstore"  |  "ZoneB"  | "promote"  |  "demote"  |

  @powerstore-metro-integration-nonuniform
  Scenario Outline: Complete Site failure for Non-Uniform Configuration; ControllerUnpublish does not work when the only array from which volume was published is down. 
    Given a kubernetes <kubeConfig>
    And a driver namespace name <driverNamespaceName>
    And the arrays in storageclass <storageClass> are in non uniform configuration
    And cluster is clean of test pods but may have labels
    And there are nodes labelled <preferred>
    And there are nodes labelled <nonpreferred>
    And wait <nodeCleanSecs> to see there are no taints
    And <podsPerNode> pods per node with <nVol> volumes and <nDev> devices using <driverType> and <storageClass> in <deploySecs> with <preferred> affinity
    Then validate that all pods are running within <deploySecs> seconds
    And all pods are running on <preferred> node
    When I disrupt metro connectivity between arrays in storage class <storageClass>
    And block connection for <nonpreferred> node to local array in <storageClass>
    When I fail labeled <preferred> nodes with <failure> failure for <failSecs> seconds
    Then check for terminating pod for <driverType> with the <preferred> label within <failSecs> seconds
    And the taints for the failed nodes are removed within <nodeCleanSecs> seconds
    And I restore metro connectivity between arrays in storage class <storageClass>
    And restore connection for <nonpreferred> node to local array in <storageClass>
    Then validate that all pods are running within <deploySecs> seconds
    And all pods are running on <preferred> node
    Then finally cleanup everything except labels

    Examples:
      | kubeConfig | podsPerNode | nVol  | nDev  | driverType   | storageClass       | workers     | primary | failure         | failSecs | deploySecs | nodeCleanSecs |  migrateSecs | driverNamespaceName | preferred | nonpreferred| 
      | ""         | "1-1"       | "1-1" | "0-0" | "powerstore" | "powerstore-metro" | "one-third" | "zero"  | "interfacedown" | 600      | 600        | 600           |  150         | "powerstore"        |  "ZoneA"  | "ZoneB"     | 
