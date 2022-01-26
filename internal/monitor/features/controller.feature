Feature: Controller Monitor
  As a podmon developer
  I want to test the controller monitor
  So that it is known to work

  @controller-mode
  Scenario Outline: Test controllerCleanupPod
    Given a controller monitor "vxflex"
    And a pod for node <podnode> with <nvol> volumes condition ""
    And I induce error <error>
    When I call controllerCleanupPod for node <node>
    Then the return status is <retstatus>
    And the last log message contains <errormsg>

    Examples:
      | podnode | nvol | error                            | node    | retstatus | errormsg                                             |
      | "node1" | 0    | "none"                           | "node1" | "true"    | "Successfully cleaned up pod"                        |
      | "node1" | 2    | "none"                           | "node1" | "true"    | "Successfully cleaned up pod"                        |
      | "node1" | 2    | "CSIExtensionsNotPresent"        | "node1" | "true"    | "Successfully cleaned up pod"                        |
      | "node1" | 2    | "GetVolumeAttachments"           | "node1" | "false"   | "induced GetVolumeAttachments error"                 |
      | "node1" | 2    | "GetPersistentVolumesInPod"      | "node1" | "false"   | "Could not get PersistentVolumes: induced"           |
      | "node1" | 2    | "DeleteVolumeAttachment"         | "node1" | "false"   | "Couldn't delete VolumeAttachment"                   |
      | "node1" | 2    | "DeletePod"                      | "node1" | "false"   | "Delete pod failed"                                  |
      | "node1" | 2    | "ControllerUnpublishVolume"      | "node1" | "false"   | "errors calling ControllerUnpublishVolume to fence"  |
      | "node1" | 2    | "ValidateVolumeHostConnectivity" | "node1" | "false"   | "Aborting pod cleanup because array still connected" |
      | "node1" | 2    | "CreateEvent"                    | "node1" | "true"    | "Successfully cleaned up pod"                        |

  @controller-mode
  Scenario Outline: test controllerModePodHandler
    Given a controller monitor "vxflex"
    And a pod for node <podnode> with <nvol> volumes condition <condition> affinity <affin>
    And a node <podnode> with taint <nodetaint>
    And I induce error <error>
    When I call controllerModePodHandler with event <eventtype>
    Then the pod is cleaned <cleaned>
    And a controllerPodInfo is present <info>
    And the last log message contains <errormsg>

    Examples:
      | podnode | nvol | condition     | affin   | nodetaint | error         | eventtype | cleaned | info    | errormsg                           |
      | "node1" | 2    | "Initialized" | "false" | "noexec"  | "none"        | "Updated" | "true"  | "false" | "Successfully cleaned up pod"      |
      | "node1" | 2    | "NotReady"    | "false" | "noexec"  | "none"        | "Updated" | "true"  | "false" | "Successfully cleaned up pod"      |
      | "node1" | 2    | "NotReady"    | "false" | "nosched" | "none"        | "Updated" | "true"  | "false" | "Successfully cleaned up pod"      |
      | "node1" | 2    | "CrashLoop"   | "false" | "none"    | "none"        | "Updated" | "false" | "false" | "cleaning up CrashLoopBackOff pod" |
      | "node1" | 2    | "NotReady"    | "false" | "nosched" | "none"        | "Deleted" | "false" | "false" | "none"                             |
      | "node1" | 2    | "Ready"       | "false" | "none"    | "none"        | "Updated" | "false" | "true"  | "none"                             |
      | "node1" | 2    | "Ready"       | "true"  | "none"    | "none"        | "Updated" | "false" | "true"  | "none"                             |
      | "node1" | 2    | "NotReady"    | "false" | "noexec"  | "GetPod"      | "Updated" | "false" | "false" | "GetPod failed"                    |
      | "node1" | 2    | "NotReady"    | "false" | "noexec"  | "GetNode"     | "Updated" | "false" | "false" | "GetNode failed"                   |
      | "node1" | 2    | "Ready"       | "false" | "noexec"  | "CreateEvent" | "Updated" | "false" | "true"  | "none"                             |
      | "node1" | 2    | "NotReady"    | "false" | "noexec"  | "CreateEvent" | "Updated" | "true"  | "false" | "Successfully cleaned up pod"      |
      | "node1" | 2    | "CrashLoop"   | "false" | "noexec"  | "CreateEvent" | "Updated" | "true"  | "false" | "Successfully cleaned up pod"      |
      | "node1" | 2    | "Initialized" | "false" | "noexec"  | "CreateEvent" | "Updated" | "true"  | "false" | "Successfully cleaned up pod"      |

  @controller-mode
  Scenario Outline: test ArrayConnectivityMonitor
    Given a controller monitor "vxflex"
    And a pod for node <podnode> with <nvol> volumes condition <condition> affinity <affin>
    And I induce error <error>
    When I call controllerModePodHandler with event "Updated"
    And I call ArrayConnectivityMonitor
    Then the pod is cleaned <cleaned>
    And the last log message contains <errormsg>

    Examples:
      | podnode | nvol | condition | affin   | error              | cleaned | errormsg                      |
      | "node1" | 2    | "Ready"   | "true"  | "NodeNotConnected" | "true"  | "none"                        |
      | "node1" | 2    | "Ready"   | "false" | "NodeConnected"    | "false" | "Connected true"              |
      | "node1" | 2    | "Ready"   | "false" | "NodeNotConnected" | "true"  | "Successfully cleaned up pod" |
      | "node1" | 2    | "Ready"   | "false" | "CreateEvent"      | "true"  | "Successfully cleaned up pod" |

  @controller-mode
  Scenario Outline: test PodAffinityLabels
    Given a controller pod with podaffinitylabels
    And create a pod for node <podnode> with <nvol> volumes condition <condition> affinity <affin> errorcase <errorcase>
    And I induce error <error>
    When I call getPodAffinityLabels
    Then the pod is cleaned <cleaned>

  Examples:
    | podnode | nvol | condition     | affin   | nodetaint | error         | errorcase       | cleaned |
    | "node1" | 2    | "Ready"       | "true"  | "none"    | "none"        | "podaffinity"   | "false" |
    | "node1" | 2    | "Ready"       | "true"  | "none"    | "none"        | "topology"      | "false" |
    | "node1" | 2    | "Ready"       | "true"  | "none"    | "none"        | "required"      | "false" |
    | "node1" | 2    | "Ready"       | "true"  | "none"    | "none"        | "labelselector" | "false" |
    | "node1" | 2    | "Ready"       | "true"  | "none"    | "none"        | "operator"      | "false" |
