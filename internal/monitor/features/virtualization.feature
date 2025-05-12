Feature: Virtual Machine Integration Test
  As a CSM for Resiliency developer
  I want to test OpenShift Virtualization with CSM for Resiliency in a OpenShift environment
  So that it is known to work on various vm clean up cases and give consistent results

  @powerflex-vm-int-setup-check
  Scenario Outline: Validate that we have a valid k8s configuration for the Virtual Machine integration tests
    Given a kubernetes <kubeConfig>
    Then Check OpenShift Virtualization is installed in the cluster
    And test environmental variables are set
    And these CSI driver <driverNames> are configured on the system
    And these storageClasses <storageClasses> exist in the cluster
    And there is a <namespace> in the cluster
    And there are driver pods in <namespace> with this <name> prefix
    And can logon to nodes and drop test scripts
    Examples:
      | kubeConfig | driverNames                | namespace  | name       | storageClasses          |
      | ""         | "csi-vxflexos.dellemc.com" | "vxflexos" | "vxflexos" | "vxflexos" |

#   @powerscale-vm-int-setup-check
#   Scenario Outline: Validate that we have a valid k8s configuration for the Virtual Machine integration tests
#     Given a kubernetes <kubeConfig>
#     Then Check OpenShift Virtualization is installed in the cluster
#     And test environmental variables are set
#     And these CSI driver <driverNames> are configured on the system
#     And these storageClasses <storageClasses> exist in the cluster
#     And there is a <namespace> in the cluster
#     And there are driver pods in <namespace> with this <name> prefix
#     And can logon to nodes and drop test scripts
#     Examples:
#       | kubeConfig | driverNames             | namespace | name    | storageClasses          |
#       | ""         | "csi-isilon.dellemc.com" | "isilon"   | "isilon" | "isilon" |
  
 
#   @powermax-vm-int-setup-check
#   Scenario Outline: Validate that we have a valid k8s configuration for the Virtual Machine integration tests
#     Given a kubernetes <kubeConfig>
#     Then Check OpenShift Virtualization is installed in the cluster
#     And test environmental variables are set
#     And these CSI driver <driverNames> are configured on the system
#     And these storageClasses <storageClasses> exist in the cluster
#     And there is a <namespace> in the cluster
#     And there are driver pods in <namespace> with this <name> prefix
#     And can logon to nodes and drop test scripts
#     Examples:
#       | kubeConfig | driverNames                  | namespace    | name       | storageClasses                |
#       | ""         | "csi-powermax.dellemc.com"   | "powermax"   | "powermax" | "powermax-iscsi, powermax-nfs" |

 
  @powerstore-vm-int-setup-check
  Scenario Outline: Validate that we have a valid k8s configuration for the Virtual Machine integration tests
    Given a kubernetes <kubeConfig>
    Then Check OpenShift Virtualization is installed in the cluster
    And test environmental variables are set 
    And these CSI driver <driverNames> are configured on the system
    And these storageClasses <storageClasses> exist in the cluster
    And there is a <namespace> in the cluster
    And there are driver pods in <namespace> with this <name> prefix
    And can logon to nodes and drop test scripts
    Examples:
      | kubeConfig | driverNames                  | namespace      | name         | storageClasses   |
      | ""         | "csi-powerstore.dellemc.com" | "powerstore"   | "powerstore" | "powerstore-nfs" |

  @powerstore-vm-integration
  Scenario Outline: Basic node failover testing using test VM's (node interface down)
    Given a kubernetes <kubeConfig>
    And cluster is clean of test vms
    And wait <nodeCleanSecs> to see there are no taints
    And <vmsPerNode> vms per node with <nVol> volumes and <nDev> devices using <driverType> and <storageClass> in <deploySecs>
    Then validate that all pods are running within <deploySecs> seconds
    Then initial disk write and verify on all VMs succeeds
    When I fail <workers> worker nodes and <primary> primary nodes with <failure> failure for <failSecs> seconds
    Then validate that all pods are running within <runSecs> seconds
    And labeled pods are on a different node
    Then post failover disk content verification on all VMs succeeds
    And the taints for the failed nodes are removed within <nodeCleanSecs> seconds
    Then finally cleanup everything

    Examples:
      | kubeConfig | vmsPerNode  | nVol  | nDev  | driverType | storageClass         | workers     | primary | failure         | failSecs | deploySecs | runSecs | nodeCleanSecs |
      # Uncomment the storageclass to use. The default is set to nvme which is supported by nightly qualification.
      | ""         | "1-1"       | "1-1" | "0-0" | "powerstore" | "powerstore-nfs"   | "one-third" | "zero"  | "interfacedown" | 400      | 300        | 350     | 400           |
      #| ""         | "1-1"       | "0-0" | "1-1" | "powerstore" | "powerstore-iscsi"   | "one-third" | "zero"  | "interfacedown" | 400      | 300        | 350     | 400           |
      #| ""         | "1-1"       | "0-0" | "1-1" | "powerstore" | "powerstore-nvmetcp"   | "one-third" | "zero"  | "interfacedown" | 400      | 300        | 350     | 400           |

  @powerstore-vm-integration
  Scenario Outline: Basic node failover testing using test VM's (node slow reboots)
    Given a kubernetes <kubeConfig>
    And cluster is clean of test vms
    And wait <nodeCleanSecs> to see there are no taints
    And <vmsPerNode> vms per node with <nVol> volumes and <nDev> devices using <driverType> and <storageClass> in <deploySecs>
    Then validate that all pods are running within <deploySecs> seconds
    Then initial disk write and verify on all VMs succeeds
    When I fail <workers> worker nodes and <primary> primary nodes with <failure> failure for <failSecs> seconds
    Then validate that all pods are running within <runSecs> seconds
    And labeled pods are on a different node
    Then post failover disk content verification on all VMs succeeds
    And the taints for the failed nodes are removed within <nodeCleanSecs> seconds
    Then finally cleanup everything
    Examples:
      | kubeConfig | vmsPerNode | nVol  | nDev  | driverType | storageClass | workers     | primary | failure  | failSecs | deploySecs | runSecs | nodeCleanSecs |
      # Uncomment the storageclass to use. The default is set to nvme which is supported by nightly qualification.
      | ""         | "1-1"      | "1-1" | "0-0" | "powerstore" | "powerstore-nfs"   | "one-third" | "zero"  | "reboot" | 300      | 600        | 600     | 600          |
      #| ""         | "1-1"      | "0-0" | "1-1" | "powerstore" | "powerstore-iscsi"   | "one-third" | "zero"  | "reboot" | 240      | 600        | 600     | 600          |
      #| ""         | "1-1"      | "0-0" | "1-1" | "powerstore" | "powerstore-nvmetcp"   | "one-third" | "zero"  | "reboot" | 240      | 600        | 600     | 600          |

  @powerstore-vm-integration
  Scenario Outline: Basic node failover testing using test VM's (node kubelet down)
    Given a kubernetes <kubeConfig>
    And cluster is clean of test vms
    And wait <nodeCleanSecs> to see there are no taints
    And <vmsPerNode> vms per node with <nVol> volumes and <nDev> devices using <driverType> and <storageClass> in <deploySecs>
    Then validate that all pods are running within <deploySecs> seconds
    Then initial disk write and verify on all VMs succeeds
    When I fail <workers> worker nodes and <primary> primary nodes with <failure> failure for <failSecs> seconds
    Then validate that all pods are running within <runSecs> seconds
    And labeled pods are on a different node
    Then post failover disk content verification on all VMs succeeds
    And the taints for the failed nodes are removed within <nodeCleanSecs> seconds
    Then finally cleanup everything

    Examples:
      | kubeConfig | vmsPerNode | nVol  | nDev  | driverType | storageClass | workers     | primary | failure       | failSecs | deploySecs | runSecs | nodeCleanSecs |
      # Uncomment the storageclass to use. The default is set to nvme which is supported by nightly qualification.
      | ""         | "1-1"      | "1-1" | "0-0" | "powerstore"    | "powerstore-nfs"  | "one-third" | "zero"  | "kubeletdown" | 600      | 900        | 900     | 900           |
      #| ""         | "1-1"      | "0-0" | "1-1" | "powerstore"    | "powerstore-iscsi"  | "one-third" | "zero"  | "kubeletdown" | 600      | 900        | 900     | 900           |
      #| ""         | "1-1"      | "0-0" | "1-1" | "powerstore"    | "powerstore-nvmetcp"  | "one-third" | "zero"  | "kubeletdown" | 600      | 900        | 900     | 900           |

  @powerstore-vm-integration
  Scenario Outline: Basic node failover testing using test VM's (driver pods down)
    Given a kubernetes <kubeConfig>
    And cluster is clean of test vms
    And wait <nodeCleanSecs> to see there are no taints
    And <vmsPerNode> vms per node with <nVol> volumes and <nDev> devices using <driverType> and <storageClass> in <deploySecs>
    Then validate that all pods are running within <deploySecs> seconds
    Then initial disk write and verify on all VMs succeeds
    When I fail <workers> worker driver pod with <failure> failure for <failSecs> and I expect these taints <taints>
    And the taints for the failed nodes are removed within <nodeCleanSecs> seconds
    Then finally cleanup everything

    Examples:
      | kubeConfig | vmsPerNode | nVol  | nDev  | driverType    | storageClass          | workers     | primary | failure     |  taints                               | failSecs | deploySecs | runSecs | nodeCleanSecs |
      # Uncomment the storageclass to use. The default is set to nvme which is supported by nightly qualification.
      | ""         | "1-1"      | "1-1" | "0-0" | "powerstore"  | "powerstore-nfs"      | "one-third" | "zero"  | "driverpod" | "offline.powerstore.storage.dell.com" | 120      | 300        | 300     | 600           |
      #| ""         | "1-1"      | "0-0" | "1-1" | "powerstore"  | "powerstore-iscsi"    | "one-third" | "zero"  | "driverpod" | "offline.powerstore.storage.dell.com" | 120      | 300        | 300     | 600           |
      #| ""         | "1-1"      | "0-0" | "1-1" | "powerstore"  | "powerstore-nvmetcp"  | "one-third" | "zero"  | "driverpod" | "offline.powerstore.storage.dell.com" | 120      | 300        | 300     | 600           |

  @powerflex-vm-integration
  Scenario Outline: Basic node failover testing using test VM's (node interface down)
    Given a kubernetes <kubeConfig>
    And cluster is clean of test vms
    And wait <nodeCleanSecs> to see there are no taints
    And <vmsPerNode> vms per node with <nVol> volumes and <nDev> devices using <driverType> and <storageClass> in <deploySecs>
    Then validate that all pods are running within <deploySecs> seconds
    Then initial disk write and verify on all VMs succeeds
    When I fail <workers> worker nodes and <primary> primary nodes with <failure> failure for <failSecs> seconds
    Then validate that all pods are running within <runSecs> seconds
    And labeled pods are on a different node
    Then post failover disk content verification on all VMs succeeds
    And the taints for the failed nodes are removed within <nodeCleanSecs> seconds
    Then finally cleanup everything

    Examples:
      | kubeConfig | vmsPerNode | nVol  | nDev  | driverType | storageClass | workers     | primary | failure         | failSecs | deploySecs | runSecs | nodeCleanSecs |
      | ""         | "1-1"      | "0-0" | "1-1" | "vxflexos" | "vxflexos"   | "one-third" | "zero"  | "interfacedown" | 450      | 300        | 350     | 500           |
      | ""         | "2-2"      | "0-0" | "2-2" | "vxflexos" | "vxflexos"   | "one-third" | "zero"  | "interfacedown" | 500      | 400        | 500     | 650           |

  @powerflex-vm-integration
  Scenario Outline: Basic node failover testing using test VM's (node slow reboots)
    Given a kubernetes <kubeConfig>
    And cluster is clean of test vms
    And wait <nodeCleanSecs> to see there are no taints
    And <vmsPerNode> vms per node with <nVol> volumes and <nDev> devices using <driverType> and <storageClass> in <deploySecs>
    Then validate that all pods are running within <deploySecs> seconds
    Then initial disk write and verify on all VMs succeeds
    When I fail <workers> worker nodes and <primary> primary nodes with <failure> failure for <failSecs> seconds
    Then validate that all pods are running within <runSecs> seconds
    And labeled pods are on a different node
    Then post failover disk content verification on all VMs succeeds
    And the taints for the failed nodes are removed within <nodeCleanSecs> seconds
    Then finally cleanup everything

    Examples:
      | kubeConfig | vmsPerNode | nVol  | nDev  | driverType | storageClass | workers     | primary | failure  | failSecs | deploySecs | runSecs | nodeCleanSecs |
      | ""         | "1-1"      | "0-0" | "1-1" | "vxflexos" | "vxflexos"   | "one-third" | "zero"  | "reboot" | 400      | 300        | 350     | 400           |
      | ""         | "2-2"      | "0-0" | "2-2" | "vxflexos" | "vxflexos"   | "one-third" | "zero"  | "reboot" | 450      | 400        | 300     | 600           |

  @powerflex-vm-integration
  Scenario Outline: Basic node failover testing using test VM's (node kubelet down)
    Given a kubernetes <kubeConfig>
    And cluster is clean of test vms
    And wait <nodeCleanSecs> to see there are no taints
    And <vmsPerNode> vms per node with <nVol> volumes and <nDev> devices using <driverType> and <storageClass> in <deploySecs>
    Then validate that all pods are running within <deploySecs> seconds
    Then initial disk write and verify on all VMs succeeds
    When I fail <workers> worker nodes and <primary> primary nodes with <failure> failure for <failSecs> seconds
    Then validate that all pods are running within <runSecs> seconds
    And labeled pods are on a different node
    Then post failover disk content verification on all VMs succeeds
    And the taints for the failed nodes are removed within <nodeCleanSecs> seconds
    Then finally cleanup everything

    Examples:
      | kubeConfig | vmsPerNode | nVol  | nDev  | driverType | storageClass | workers     | primary | failure         | failSecs | deploySecs | runSecs | nodeCleanSecs |
      | ""         | "1-1"      | "0-0" | "1-1" | "vxflexos" | "vxflexos"   | "one-third" | "zero"  | "kubeletdown"   | 600      | 900        | 900     | 900           |
      | ""         | "2-2"      | "0-0" | "2-2" | "vxflexos" | "vxflexos"   | "one-third" | "zero"  | "kubeletdown"   | 600      | 900        | 900     | 900           |

#   @powerscale-vm-integration
#   Scenario Outline: Basic node failover testing using test VM's (node interface down)
#     Given a kubernetes <kubeConfig>
#     And cluster is clean of test vms
#     And wait <nodeCleanSecs> to see there are no taints
#     And <vmsPerNode> vms per node with <nVol> volumes and <nDev> devices using <driverType> and <storageClass> in <deploySecs>
#     Then validate that all pods are running within <deploySecs> seconds
#     Then initial disk write and verify on all VMs succeeds
#     When I fail <workers> worker nodes and <primary> primary nodes with <failure> failure for <failSecs> seconds
#     Then validate that all pods are running within <runSecs> seconds
#     And labeled pods are on a different node
#     Then post failover disk content verification on all VMs succeeds
#     And the taints for the failed nodes are removed within <nodeCleanSecs> seconds
#     Then finally cleanup everything

#     Examples:
#       | kubeConfig | vmsPerNode | nVol  | nDev  | driverType | storageClass | workers     | primary | failure         | failSecs | deploySecs | runSecs | nodeCleanSecs |
#       | ""         | "1-1"      | "1-1" | "0-0" | "isilon" | "isilon"   | "one-third" | "zero"  | "interfacedown" | 450      | 300        | 350     | 400           |
  
#   @powerscale-vm-integration
#     Scenario Outline: Basic node failover testing using test VM's (node slow reboots)
#       Given a kubernetes <kubeConfig>
#       And cluster is clean of test vms
#       And wait <nodeCleanSecs> to see there are no taints
#       And <vmsPerNode> vms per node with <nVol> volumes and <nDev> devices using <driverType> and <storageClass> in <deploySecs>
#       Then validate that all pods are running within <deploySecs> seconds
#       Then initial disk write and verify on all VMs succeeds
#       When I fail <workers> worker nodes and <primary> primary nodes with <failure> failure for <failSecs> seconds
#       Then validate that all pods are running within <runSecs> seconds
#       And labeled pods are on a different node
#       Then post failover disk content verification on all VMs succeeds
#       And the taints for the failed nodes are removed within <nodeCleanSecs> seconds
#       Then finally cleanup everything
#       Examples:
#         | kubeConfig | vmsPerNode | nVol  | nDev  | driverType | storageClass | workers     | primary | failure  | failSecs | deploySecs | runSecs | nodeCleanSecs |
#         | ""         | "1-2"      | "1-1" | "0-0" | "isilon" | "isilon"   | "one-third" | "zero"  | "reboot" | 450      | 600        | 600     | 600          |
#         | ""         | "1-2"      | "2-2" | "0-0" | "isilon" | "isilon"   | "one-third" | "zero"  | "reboot" | 600      | 900        | 900     | 900           |


#   @powerscale-vm-integration
#     Scenario Outline: Basic node failover testing using test VM's (node kubelet down)
#       Given a kubernetes <kubeConfig>
#       And cluster is clean of test vms
#       And wait <nodeCleanSecs> to see there are no taints
#       And <vmsPerNode> vms per node with <nVol> volumes and <nDev> devices using <driverType> and <storageClass> in <deploySecs>
#       Then validate that all pods are running within <deploySecs> seconds
#       Then initial disk write and verify on all VMs succeeds
#       When I fail <workers> worker nodes and <primary> primary nodes with <failure> failure for <failSecs> seconds
#       Then validate that all pods are running within <runSecs> seconds
#       And labeled pods are on a different node
#       Then post failover disk content verification on all VMs succeeds
#       And the taints for the failed nodes are removed within <nodeCleanSecs> seconds
#       Then finally cleanup everything

#       Examples:
#         | kubeConfig | vmsPerNode | nVol  | nDev  | driverType | storageClass | workers     | primary | failure       | failSecs | deploySecs | runSecs | nodeCleanSecs |
#         | ""         | "1-2"      | "1-1" | "0-0" | "isilon"   | "isilon"     | "one-third" | "zero"  | "kubeletdown" | 600      | 900        | 900     | 900           |
#         | ""         | "3-5"      | "1-1" | "0-0" | "isilon"   | "isilon"     | "one-third" | "zero"  | "kubeletdown" | 600      | 900        | 900     | 900           |



#   @powermax-vm-integration
#   Scenario Outline: Basic node failover testing using test VM's (node interface down)
#     Given a kubernetes <kubeConfig>
#     And cluster is clean of test vms
#     And wait <nodeCleanSecs> to see there are no taints
#     And <vmsPerNode> vms per node with <nVol> volumes and <nDev> devices using <driverType> and <storageClass> in <deploySecs>
#     Then validate that all pods are running within <deploySecs> seconds
#     Then initial disk write and verify on all VMs succeeds
#     When I fail <workers> worker nodes and <primary> primary nodes with <failure> failure for <failSecs> seconds
#     Then validate that all pods are running within <runSecs> seconds
#     And labeled pods are on a different node
#     Then post failover disk content verification on all VMs succeeds
#     And the taints for the failed nodes are removed within <nodeCleanSecs> seconds
#     Then finally cleanup everything

#     Examples:
#       | kubeConfig | vmsPerNode | nVol  | nDev  | driverType | storageClass | workers     | primary | failure         | failSecs | deploySecs | runSecs | nodeCleanSecs |
#       | ""         | "1-1"      | "1-1" | "0-0" | "powermax" | "powermax-nfs"   | "one-third" | "zero"  | "interfacedown" | 450      | 400        | 350     | 600           |
#       | ""         | "1-1"      | "0-0" | "1-1" | "powermax" | "powermax-iscsi"   | "one-third" | "zero"  | "interfacedown" | 450      | 400        | 350     | 600           |
#       #| ""         | "1-1"      | "0-0" | "1-1" | "powermax" | "powermax-nvmetcp"   | "one-third" | "zero"  | "interfacedown" | 450      | 400        | 350     | 600           |


#   @powermax-vm-integration
#   Scenario Outline: Basic node failover testing using test VM's (node kubelet down)
#     Given a kubernetes <kubeConfig>
#     And cluster is clean of test vms
#     And wait <nodeCleanSecs> to see there are no taints
#     And <vmsPerNode> vms per node with <nVol> volumes and <nDev> devices using <driverType> and <storageClass> in <deploySecs>
#     Then validate that all pods are running within <deploySecs> seconds
#     Then initial disk write and verify on all VMs succeeds
#     When I fail <workers> worker nodes and <primary> primary nodes with <failure> failure for <failSecs> seconds
#     Then validate that all pods are running within <runSecs> seconds
#     And labeled pods are on a different node
#     Then post failover disk content verification on all VMs succeeds
#     And the taints for the failed nodes are removed within <nodeCleanSecs> seconds
#     Then finally cleanup everything

#     Examples:
#       | kubeConfig | vmsPerNode | nVol  | nDev  | driverType    | storageClass        | workers     | primary | failure       | failSecs | deploySecs | runSecs | nodeCleanSecs |
#       | ""         | "1-1"      | "1-1" | "0-0" | "powermax"    | "powermax-nfs"      | "one-third" | "zero"  | "kubeletdown" | 600      | 900        | 900     | 900           |
#       | ""         | "1-1"      | "0-0" | "1-1" | "powermax"    | "powermax-iscsi"    | "one-third" | "zero"  | "kubeletdown" | 600      | 900        | 900     | 900           |
#       #| ""         | "1-1"      | "0-0" | "1-1" | "powermax"    | "powermax-nvmetcp"  | "one-third" | "zero"  | "kubeletdown" | 600      | 900        | 900     | 900           |

 
#   @powermax-vm-integration
#   Scenario Outline: Basic node failover testing using test VM's (node slow reboots)
#     Given a kubernetes <kubeConfig>
#     And cluster is clean of test vms
#     And wait <nodeCleanSecs> to see there are no taints
#     And <vmsPerNode> vms per node with <nVol> volumes and <nDev> devices using <driverType> and <storageClass> in <deploySecs>
#     Then validate that all pods are running within <deploySecs> seconds
#     Then initial disk write and verify on all VMs succeeds
#     When I fail <workers> worker nodes and <primary> primary nodes with <failure> failure for <failSecs> seconds
#     Then validate that all pods are running within <runSecs> seconds
#     And labeled pods are on a different node
#     Then post failover disk content verification on all VMs succeeds
#     And the taints for the failed nodes are removed within <nodeCleanSecs> seconds
#     Then finally cleanup everything
#     Examples:
#       | kubeConfig | vmsPerNode | nVol  | nDev  | driverType | storageClass     | workers     | primary | failure  | failSecs | deploySecs | runSecs | nodeCleanSecs |
#       | ""         | "1-1"      | "1-1" | "0-0" | "powermax" | "powermax-nfs"   | "one-third" | "zero"  | "reboot" | 450      | 600        | 600     | 600           |
#       | ""         | "1-1"      | "0-0" | "1-1" | "powermax" | "powermax-iscsi" | "one-third" | "zero"  | "reboot" | 450      | 600        | 600     | 600           |
#       #| ""         | "1-1"      | "0-0" | "1-1" | "powermax" | "powermax-nvmetcp"   | "one-third" | "zero"  | "reboot" | 450      | 600        | 600     | 600          |

#    @powermax-vm-integration
#   Scenario Outline: Basic node failover testing using test VM's (driver pods down)
#     Given a kubernetes <kubeConfig>
#     And cluster is clean of test vms
#     And wait <nodeCleanSecs> to see there are no taints
#     And <vmsPerNode> vms per node with <nVol> volumes and <nDev> devices using <driverType> and <storageClass> in <deploySecs>
#     Then validate that all pods are running within <deploySecs> seconds
#     Then initial disk write and verify on all VMs succeeds
#     When I fail <workers> worker driver pod with <failure> failure for <failSecs> and I expect these taints <taints>
#     And the taints for the failed nodes are removed within <nodeCleanSecs> seconds
#     Then finally cleanup everything

#     Examples:
#       | kubeConfig | vmsPerNode | nVol  | nDev  | driverType    | storageClass          | workers     | primary | failure     |  taints                               | failSecs | deploySecs | runSecs | nodeCleanSecs |
#       | ""         | "1-1"      | "1-1" | "0-0" | "powermax"    | "powermax-nfs"        | "one-third" | "zero"  | "driverpod" | "offline.powermax.storage.dell.com"   | 350      | 300        | 300     | 600           |
#       | ""         | "1-1"      | "0-0" | "1-1" | "powermax"    | "powermax-iscsi"      | "one-third" | "zero"  | "driverpod" | "offline.powermax.storage.dell.com"   | 350      | 300        | 300     | 600           |
#       #| ""         | "1-1"      | "0-0" | "1-1" | "powermax"  | "powermax-nvmetcp"  | "one-third" | "zero"  | "driverpod" | "offline.powermax.storage.dell.com" | 350      | 300        | 300     | 600           |

