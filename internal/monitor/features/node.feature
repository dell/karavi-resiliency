Feature: Controller Monitor
  As a podmon developer
  I want to test the controller monitor
  So that it is known to work on node mode clean up cases

  @node-mode
  Scenario Outline: Testing monitor.nodeModePodHandler
    Given a controller monitor "vxflex"
    And node <nodeName> env vars set
    And a pod for node <nodeName> with <vols> volumes condition ""
    And I induce error <csiVolumePathError>
    And I induce error <induceError>
    When I call nodeModePodHandler for node <node> with event <eventType>
    Then I expect podMonitor to have <nMounts> mounts
    And the last log message contains <errorMsg>

    Examples:
      | nodeName | vols | node    | eventType  | nMounts | csiVolumePathError     | induceError           | errorMsg |
      | "node1"  | 1    | "node1" | "none"     | 0       | "none"                 | "none"                | "none"   |
      | "node1"  | 1    | "node1" | "ADDED"    | 1       | "none"                 | "none"                | "none"   |
      | "node1"  | 1    | "node1" | "ADDED"    | 0       | "none"                 | "GetPersistentVolume" | "none"   |
      | "node1"  | 1    | "node1" | "MODIFIED" | 0       | "none"                 | "GetPersistentVolume" | "none"   |
      | "node1"  | 1    | "node1" | "MODIFIED" | 1       | "none"                 | "none"                | "none"   |
      | "node1"  | 1    | "node1" | "DELETED"  | 0       | "none"                 | "none"                | "none"   |
      | "node1"  | 1    | "node1" | "DELETED"  | 0       | "CSIVolumePathDirRead" | "none"                | "none"   |
      | "node1"  | 1    | "node1" | "DELETED"  | 0       | "none"                 | "GetPersistentVolume" | "none"   |
      | "node1"  | 1    | "node1" | "BOOKMARK" | 0       | "none"                 | "none"                | "none"   |
      | "node1"  | 1    | "node1" | "ERROR"    | 0       | "none"                 | "none"                | "none"   |

  @node-mode
  Scenario Outline: Testing monitor.nodeModePodHandler multiple calls
    Given a controller monitor "vxflex"
    And node <nodeName> env vars set
    And a pod for node <nodeName> with <vols> volumes condition ""
    When I call nodeModePodHandler for node <node> with event <eventType>
    And I induce error <induceError>
    When I call nodeModePodHandler for node <node> with event <eventType>
    Then I expect podMonitor to have <nMounts> mounts
    And the last log message contains <errorMsg>

    Examples:
      | nodeName | vols | node    | eventType  | nMounts | induceError           | errorMsg                    |
      | "node1"  | 1    | "node1" | "none"     | 0       | "none"                | "none"                      |
      | "node1"  | 1    | "node1" | "ADDED"    | 1       | "CSIVolumePathDirRead"| "Skipped Storing podInfo"   |
      | "node1"  | 1    | "node1" | "MODIFIED" | 1       | "CSIVolumePathDirRead"| "Skipped Storing podInfo"   |
      | "node1"  | 1    | "node1" | "DELETED"  | 0       | "none"                | "none"                      |

  @node-mode
  Scenario Outline: Testing monitor.nodeModeCleanupPods
    Given a controller monitor <driver>
    And node <nodeName> env vars set
    And I have a <pods> pods for node <nodeName> with <vols> volumes <devs> devices condition ""
    And the controller cleaned up <cleaned> pods for node <nodeName>
    And I induce error <k8apiErr>
    And I induce error <unMountErr>
    And I induce error <rmDirErr>
    And I induce error <taintErr>
    When I call nodeModeCleanupPods for node <nodeName>
    And the last log message contains <errorMsg>

    Examples:
      | driver | nodeName | pods | vols | devs | cleaned | unMountErr | rmDirErr    | taintErr       | k8apiErr              | errorMsg                           |
      | vxflex | "node1"  | 1    | 1    | 1    | 1       | "none"     | "none"      | "none"         | "none"                | "none"                             |
      | vxflex | "node1"  | 1    | 1    | 1    | 1       | "none"     | "none"      | "none"         | "GetContainerInfo"    | "none"                             |
      | vxflex | "node1"  | 1    | 1    | 1    | 1       | "none"     | "none"      | "none"         | "ContainerRunning"    | "Couldn't completely cleanup node" |
      | vxflex | "node1"  | 1    | 1    | 1    | 1       | "none"     | "none"      | "none"         | "NodeUnpublishVolume" | "Couldn't completely cleanup node" |
      | vxflex | "node1"  | 1    | 1    | 1    | 1       | "none"     | "none"      | "K8sTaint"     | "none"                | "Failed to remove taint against node1 node" |
      | vxflex | "node1"  | 1    | 1    | 1    | 1       | "none"     | "none"      | "K8sTaint"     | "NodeUnpublishVolume" | "Couldn't completely cleanup node" |
      | vxflex | "node1"  | 1    | 1    | 1    | 1       | "none"     | "RemoveDir" | "none"         | "none"                | "Couldn't completely cleanup node" |
      | vxflex | "node1"  | 1    | 1    | 1    | 1       | "none"     | "RemoveDir" | "none"         | "NodeUnpublishVolume" | "Couldn't completely cleanup node" |
      | vxflex | "node1"  | 1    | 1    | 1    | 1       | "none"     | "RemoveDir" | "K8sTaint"     | "NodeUnpublishVolume" | "Couldn't completely cleanup node" |
      | vxflex | "node1"  | 1    | 1    | 1    | 1       | "Unmount"  | "none"      | "none"         | "none"                | "none"                             |
      | vxflex | "node1"  | 1    | 1    | 1    | 1       | "Unmount"  | "none"      | "none"         | "NodeUnpublishVolume" | "Couldn't completely cleanup node" |
      | vxflex | "node1"  | 1    | 1    | 1    | 1       | "Unmount"  | "none"      | "K8sTaint"     | "none"                | "Failed to remove taint against node1 node" |
      | vxflex | "node1"  | 1    | 1    | 1    | 1       | "Unmount"  | "none"      | "K8sTaint"     | "NodeUnpublishVolume" | "Couldn't completely cleanup node" |
      | vxflex | "node1"  | 1    | 1    | 1    | 1       | "Unmount"  | "RemoveDir" | "none"         | "none"                | "Couldn't completely cleanup node" |
      | vxflex | "node1"  | 1    | 1    | 1    | 1       | "Unmount"  | "RemoveDir" | "none"         | "NodeUnpublishVolume" | "Couldn't completely cleanup node" |
      | vxflex | "node1"  | 1    | 1    | 1    | 1       | "Unmount"  | "RemoveDir" | "K8sTaint"     | "NodeUnpublishVolume" | "Couldn't completely cleanup node" |
      | unity  | "node1"  | 1    | 1    | 1    | 1       | "none"     | "none"      | "none"         | "none"                | "none"                             |
      | unity  | "node1"  | 1    | 1    | 1    | 1       | "none"     | "none"      | "none"         | "NodeUnpublishVolume" | "Couldn't completely cleanup node" |
      | unity  | "node1"  | 1    | 1    | 1    | 1       | "none"     | "none"      | "none"         | "NodeUnstageVolume"   | "Couldn't completely cleanup node" |
      | unity  | "node1"  | 1    | 1    | 1    | 1       | "none"     | "none"      | "none"         | "NodeUnpublishNFSShareNotFound" | "none" |
      | unity  | "node1"  | 1    | 1    | 1    | 1       | "none"     | "none"      | "none"         | "NodeUnstageNFSShareNotFound"   | "none" |
      | isilon | "node1"  | 1    | 1    | 1    | 1       | "none"     | "none"      | "none"         | "none"                | "none"                             |
      | isilon | "node1"  | 1    | 1    | 1    | 1       | "none"     | "none"      | "none"         | "GetContainerInfo"    | "none"                             |
      | isilon | "node1"  | 1    | 1    | 1    | 1       | "none"     | "none"      | "none"         | "ContainerRunning"    | "Couldn't completely cleanup node" |
      | isilon | "node1"  | 1    | 1    | 1    | 1       | "none"     | "none"      | "none"         | "NodeUnpublishVolume" | "Couldn't completely cleanup node" |

      | powerstore | "node1"  | 1    | 1    | 1    | 1       | "none"     | "none"      | "none"     | "none"                          | "none"                                      |
      | powerstore | "node1"  | 1    | 1    | 1    | 1       | "none"     | "none"      | "none"     | "GetContainerInfo"              | "none"                                      |
      | powerstore | "node1"  | 1    | 1    | 1    | 1       | "none"     | "none"      | "none"     | "NodeUnpublishVolume"           | "Couldn't completely cleanup node"          |
      | powerstore | "node1"  | 1    | 1    | 1    | 1       | "none"     | "none"      | "none"     | "NodeUnstageVolume"             | "Couldn't completely cleanup node"          |
      | powermax   | "node1"  | 1    | 1    | 1    | 1       | "none"     | "none"      | "none"     | "none"                          | "none"                                      |
      | powermax   | "node1"  | 1    | 1    | 1    | 1       | "none"     | "none"      | "none"     | "GetContainerInfo"              | "none"                                      |
      | powermax   | "node1"  | 1    | 1    | 1    | 1       | "none"     | "none"      | "none"     | "NodeUnpublishVolume"           | "Couldn't completely cleanup node"          |
      | powermax   | "node1"  | 1    | 1    | 1    | 1       | "none"     | "none"      | "none"     | "NodeUnstageVolume"             | "Couldn't completely cleanup node"          |
      # Multiple pod tests
      | vxflex | "node1"  | 3    | 2    | 1    | 3       | "none"     | "none"      | "none"         | "none"                | "none"                             |
      | vxflex | "node1"  | 3    | 2    | 1    | 2       | "none"     | "none"      | "none"         | "none"                | "Couldn't completely cleanup node" |
      | vxflex | "node1"  | 3    | 2    | 1    | 2       | "none"     | "none"      | "none"         | "NodeUnpublishVolume" | "Couldn't completely cleanup node" |
      | vxflex | "node1"  | 3    | 2    | 1    | 2       | "none"     | "none"      | "K8sTaint"     | "none"                | "Couldn't completely cleanup node" |
      | vxflex | "node1"  | 3    | 2    | 1    | 2       | "none"     | "none"      | "K8sTaint"     | "NodeUnpublishVolume" | "Couldn't completely cleanup node" |
      | vxflex | "node1"  | 3    | 2    | 1    | 2       | "none"     | "RemoveDir" | "none"         | "none"                | "Couldn't completely cleanup node" |
      | vxflex | "node1"  | 3    | 2    | 1    | 2       | "none"     | "RemoveDir" | "none"         | "NodeUnpublishVolume" | "Couldn't completely cleanup node" |
      | vxflex | "node1"  | 3    | 2    | 1    | 2       | "none"     | "RemoveDir" | "K8sTaint"     | "NodeUnpublishVolume" | "Couldn't completely cleanup node" |
      | vxflex | "node1"  | 3    | 2    | 1    | 2       | "Unmount"  | "none"      | "none"         | "none"                | "Couldn't completely cleanup node" |
      | vxflex | "node1"  | 3    | 2    | 1    | 2       | "Unmount"  | "none"      | "none"         | "NodeUnpublishVolume" | "Couldn't completely cleanup node" |
      | vxflex | "node1"  | 3    | 2    | 1    | 2       | "Unmount"  | "none"      | "K8sTaint"     | "none"                | "Couldn't completely cleanup node" |
      | vxflex | "node1"  | 3    | 2    | 1    | 2       | "Unmount"  | "none"      | "K8sTaint"     | "NodeUnpublishVolume" | "Couldn't completely cleanup node" |
      | vxflex | "node1"  | 3    | 2    | 1    | 2       | "Unmount"  | "RemoveDir" | "none"         | "none"                | "Couldn't completely cleanup node" |
      | vxflex | "node1"  | 3    | 2    | 1    | 2       | "Unmount"  | "RemoveDir" | "none"         | "NodeUnpublishVolume" | "Couldn't completely cleanup node" |
      | vxflex | "node1"  | 3    | 2    | 1    | 2       | "Unmount"  | "RemoveDir" | "K8sTaint"     | "NodeUnpublishVolume" | "Couldn't completely cleanup node" |
      | powerstore | "node1"  | 3    | 2    | 1    | 3       | "none"     | "none"      | "none"         | "none"                | "none"                             |
      | powerstore | "node1"  | 3    | 2    | 1    | 2       | "none"     | "none"      | "none"         | "none"                | "Couldn't completely cleanup node" |
      | powerstore | "node1"  | 3    | 2    | 1    | 2       | "none"     | "none"      | "none"         | "NodeUnpublishVolume" | "Couldn't completely cleanup node" |
      | powerstore | "node1"  | 3    | 2    | 1    | 2       | "none"     | "none"      | "none"         | "NodeUnstageVolume"   | "Couldn't completely cleanup node" |
      | powerstore | "node1"  | 3    | 2    | 1    | 2       | "Unmount"  | "none"      | "none"         | "none"                | "Couldn't completely cleanup node" |
      | powerstore | "node1"  | 3    | 2    | 1    | 2       | "none"     | "RemoveDir" | "none"         | "none"                | "Couldn't completely cleanup node" |
      | powermax   | "node1"  | 3    | 2    | 1    | 2       | "none"     | "none"      | "K8sTaint"     | "none"                | "Couldn't completely cleanup node" |
      | powermax   | "node1"  | 3    | 2    | 1    | 3       | "none"     | "none"      | "none"         | "none"                | "none"                             |
      | powermax   | "node1"  | 3    | 2    | 1    | 2       | "none"     | "none"      | "none"         | "none"                | "Couldn't completely cleanup node" |
      | powermax   | "node1"  | 3    | 2    | 1    | 2       | "none"     | "none"      | "none"         | "NodeUnpublishVolume" | "Couldn't completely cleanup node" |
      | powermax   | "node1"  | 3    | 2    | 1    | 2       | "none"     | "none"      | "none"         | "NodeUnstageVolume"   | "Couldn't completely cleanup node" |
      | powermax   | "node1"  | 3    | 2    | 1    | 2       | "Unmount"  | "none"      | "none"         | "none"                | "Couldn't completely cleanup node" |
      | powermax   | "node1"  | 3    | 2    | 1    | 2       | "none"     | "RemoveDir" | "none"         | "none"                | "Couldn't completely cleanup node" |
      | powermax   | "node1"  | 3    | 2    | 1    | 2       | "none"     | "none"      | "K8sTaint"     | "none"                | "Couldn't completely cleanup node" |


  @node-mode
  Scenario Outline: Testing monitor.StartApiMonitor (Loop Invocation Function)
    Given a controller monitor "vxflex"
    And node <nodeName> env vars set
    And a node <nodeName> with taint <nodeTaint>
    And I allow nodeApiMonitor loop to run <loopCount>
    When I call StartAPIMonitor
    Then the last log message contains <errorMsg>

    Examples:
      | nodeName | nodeTaint | loopCount | errorMsg                                          |
      | "node1"  | "none"    | 1         | "none"                                            |
      | ""       | "none"    | 1         | "KUBE_NODE_NAME environment variable must be set" |

  @node-mode
  Scenario Outline: Testing monitor.apiMonitorLoop (K8S GetNode API Checker)
    Given a controller monitor "vxflex"
    And node <nodeName> env vars set
    And a node <nodeName> with taint <nodeTaint>
    And I allow nodeApiMonitor loop to run <loopCount>
    And I induce error <inducedErr> for <maxFailTimes>
    When I call apiMonitorLoop for <nodeName>
    Then the last log message contains <errorMsg>

    Examples:
      | nodeName | nodeTaint        | loopCount | inducedErr           | maxFailTimes | errorMsg                            |
      | "node1"  | "none"           | 2         | "GetNodeWithTimeout" | "-1"         | "Lost API connectivity from node"   |
      | "node1"  | "none"           | 2         | "GetNodeWithTimeout" | "3"          | "none"                              |
      | "node1"  | "none"           | 2         | "GetNodeWithTimeout" | "4"          | "API connectivity restored to node" |
      | "node1"  | "noexec"         | 2         | "GetNodeWithTimeout" | "5"          | "Lost API connectivity from node"   |
      | "node1"  | "nosched"        | 2         | "GetNodeWithTimeout" | "5"          | "Lost API connectivity from node"   |

      | "node1"  | "podmon-nosched" | 2         | "GetNodeWithTimeout" | "5"          | "Lost API connectivity from node"   |
      | "node1"  | "podmon-noexec"  | 2         | "GetNodeWithTimeout" | "5"          | "Lost API connectivity from node"   |

      | "node1"  | "noexec"         | 3         | "GetNodeWithTimeout" | "6"          | "API connectivity restored to node" |
      | "node1"  | "nosched"        | 3         | "GetNodeWithTimeout" | "6"          | "API connectivity restored to node" |

      | "node1"  | "podmon-nosched" | 3         | "GetNodeWithTimeout" | "6"          | "Cleanup of pods complete"          |
      | "node1"  | "podmon-noexec"  | 3         | "GetNodeWithTimeout" | "6"          | "API connectivity restored to node" |

  @node-mode
  Scenario Outline: Testing monitor.nodeModeCleanupPods with privateMountDir
    Given a controller monitor <driver>
    And node <nodeName> env vars set
    And I have a <pods> pods for node <nodeName> with <vols> volumes <devs> devices condition ""
    And the controller cleaned up <cleaned> pods for node <nodeName>
    And I induce error <k8apiErr>
    And I induce error <unMountErr>
    And I induce error <rmDirErr>
    And I induce error <taintErr>
    When I call nodeModeCleanupPods for node <nodeName> with empty private mount
    And the last log message contains <errorMsg>

    Examples:
      | driver     | nodeName | pods | vols | devs | cleaned | unMountErr | rmDirErr    | taintErr       | k8apiErr     | errorMsg |
      | vxflex     | "node1"  | 1    | 1    | 1    | 1       | "none"     | "none"      | "none"         | "none"       | "none"   |
      | isilon     | "node1"  | 1    | 1    | 1    | 1       | "none"     | "none"      | "none"         | "none"       | "none"   |
      | powerstore | "node1"  | 1    | 1    | 1    | 1       | "none"     | "none"      | "none"         | "none"       | "none"   |
      | powermax   | "node1"  | 1    | 1    | 1    | 1       | "none"     | "none"      | "none"         | "none"       | "none"   |

@node-mode
  Scenario Outline: Testing monitor.nodeModeCleanupPods
    Given a controller monitor <driver>
    And node <nodeName> env vars set
    And I have a <pods> pods for node <nodeName> with <vols> volumes <devs> devices condition ""
    And the controller cleaned up <cleaned> pods for node <nodeName>
    And I induce error <k8apiErr>
    And I induce error <unMountErr>
    And I induce error <rmDirErr>
    And I induce error <taintErr>
    When I call nodeModeCleanupPods for node <nodeName>
    And the last log message contains <errorMsg>

    Examples:
      | driver | nodeName | pods | vols | devs | cleaned | unMountErr | rmDirErr    | taintErr       | k8apiErr   | errorMsg                           |
      | vxflex | "node1"  | 1    | 0    | 0    |0        | "none"     | "none"      | "none"         | "none"     | "none"                             |
      | vxflex | "node1"  | 1    | 0    | 0    |0        | "none"     | "none"      | "none"         | "none"     | "none"                             |

@node-mode
  Scenario Outline: Testing monitor.nodeModeCleanupPods
    Given a controller monitor <driver>
    And node <nodeName> env vars set
    And I have a <pods> pods for node <nodeName> with <vols> volumes <devs> devices condition "" with pod <phase> phase
    And I induce error <k8apiErr>
    And I induce error <unMountErr>
    And I induce error <rmDirErr>
    And I induce error <taintErr>
    When I call nodeModeCleanupPods for node <nodeName>
    And the last log message contains <errorMsg>

    Examples:
      | driver | nodeName | pods | vols | devs | cleaned | unMountErr | rmDirErr    | taintErr       | k8apiErr   | errorMsg    | phase        |
      | vxflex | "node1"  | 1    | 1    | 1    | 1       | "none"     | "none"      | "none"         | "none"     | "none"      | "running"    |
      | vxflex | "node1"  | 1    | 0    | 0    |0        | "none"     | "none"      | "none"         | "none"     | "none"      | "pending"    |
