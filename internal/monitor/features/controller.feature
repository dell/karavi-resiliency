Feature: Controller Monitor
  As a podmon developer
  I want to test the controller monitor
  So that it is known to work

@controller-mode
  Scenario Outline: Test controllerCleanupPod
   Given a controller monitor
   And a pod for node <podnode> with <nvol> volumes condition ""
   And I induce error <error>
   When I call controllerCleanupPod for node <node>
   Then the return status is <retstatus>
   And the last log message contains <errormsg>

  Examples:
   | podnode       | nvol | error                            | node        | retstatus | errormsg                                                  |
   | "node1"       | 0    | "none"                           | "node1"     | "true"    | "Successfully cleaned up pod"                             |
   | "node1"       | 2    | "none"                           | "node1"     | "true"    | "Successfully cleaned up pod"                             |
   | "node1"       | 2    | "CSIExtensionsNotPresent"        | "node1"     | "true"    | "Successfully cleaned up pod"                             |
   | "node1"       | 2    | "GetVolumeAttachments"           | "node1"     | "false"   | "induced GetVolumeAttachments error"                      |
   | "node1"       | 2    | "IsVolumeAttachmentToPod"        | "node1"     | "false"   | "Aborting cleanup because could not determine if VA"      |
   | "node1"       | 2    | "GetPersistentVolumeClaim"       | "node1"     | "false"   | "Aborting cleanup because could not determine if VA"      |
   | "node1"       | 2    | "DeleteVolumeAttachment"         | "node1"     | "false"   | "Couldn't delete VolumeAttachment"                        |
   | "node1"       | 2    | "DeletePod"                      | "node1"     | "false"   | "Delete pod failed"                                       |
   | "node1"       | 2    | "ControllerUnpublishVolume"      | "node1"     | "false"   | "errors calling ControllerUnpublishVolume to fence"       |
   | "node1"       | 2    | "ValidateVolumeHostConnectivity" | "node1"     | "false"   | "Aborting pod cleanup because array still connected"      |
   | "node1"       | 2    | "GetVolumeHandleFromVA"          | "node1"     | "false"   | "could not getVolumeHandleFromVA"                         |

@controller-mode
  Scenario Outline: test controllerModePodHandler
   Given a controller monitor
   And a pod for node <podnode> with <nvol> volumes condition <condition>
   And a node <podnode> with taint <nodetaint>
   And I induce error <error>
   When I call controllerModePodHandler with event <eventtype>
   Then the pod is cleaned <cleaned>
   And a controllerPodInfo is present <info>
   And the last log message contains <errormsg>

  Examples:
   | podnode       | nvol | condition     | nodetaint               | error                           | eventtype | cleaned | info    | errormsg                        |
   | "node1"       | 2    | "Initialized" | "noexec"                | "none"                          | "Updated" | "true"  | "false" | "Successfully cleaned up pod"   |
   | "node1"       | 2    | "NotReady"    | "noexec"                | "none"                          | "Updated" | "true"  | "false" | "Successfully cleaned up pod"   |
   | "node1"       | 2    | "NotReady"    | "nosched"               | "none"                          | "Updated" | "true"  | "false" | "Successfully cleaned up pod"   |
   | "node1"       | 2    | "NotReady"    | "nosched"               | "none"                          | "Deleted" | "false" | "false" | "none"                          |
   | "node1"       | 2    | "Ready"       | "none"                  | "none"                          | "Updated" | "false" | "true"  | "none"                          |
   | "node1"       | 2    | "NotReady"    | "noexec"                | "GetPod"                        | "Updated" | "false" | "false" | "GetPod failed"                 |
   | "node1"       | 2    | "NotReady"    | "noexec"                | "GetNode"                       | "Updated" | "false" | "false" | "GetNode failed"                |

@controller-mode
  Scenario Outline: test ArrayConnectivityMonitor
   Given a controller monitor
   And a pod for node <podnode> with <nvol> volumes condition <condition>
   And I induce error <error>
   When I call controllerModePodHandler with event "Updated"
   And I call ArrayConnectivityMonitor
   Then the pod is cleaned <cleaned>
   And the last log message contains <errormsg>

  Examples:
   | podnode       | nvol | condition     |  error                           | cleaned |  errormsg                        |
   | "node1"       | 2    | "Ready"       |  "NodeConnected"                 | "false" |  "Connected true"                |
   | "node1"       | 2    | "Ready"       |  "NodeNotConnected"              | "true"  |  "Successfully cleaned up pod"   |

