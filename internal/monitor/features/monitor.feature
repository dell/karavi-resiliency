Feature: Monitor generic code
  As a podmon developer
  I want to test the monitor generic code
  So that it is known to work

  @monitor
  Scenario Outline: Test StartPodMonitorHandler
    Given a controller monitor "vxflex"
    And a pod for node <podnode> with <nvol> volumes condition ""
    And pod monitor mode <mode>
    And I induce error <error>
    When I call StartPodMonitor with key "podmn" and value "csi-vxflexos"
    And I send a pod event type <eventtype>
    Then I close the Watcher
    And the last log message contains <errormsg>

    Examples:
      | podnode | nvol | error            | mode         | eventtype | errormsg                       |
      | "node1" | 0    | "Watch"          | "none"       | "None"    | "none"                         |
      | "node1" | 0    | "none"           | "none"       | "None"    | "Setup of PodWatcher complete" |
      | "node1" | 0    | "none"           | "none"       | "Add"     | "PodMonitor.Mode not set"      |
      | "node1" | 0    | "none"           | "controller" | "Add"     | "podMonitorHandler"            |
      | "node1" | 0    | "GetPod"         | "controller" | "Add"     | "GetPod error"                 |
      | "node1" | 0    | "BadWatchObject" | "controller" | "Add"     | "podMonitorHandler nil pod"    |
      | "node1" | 0    | "none"           | "node"       | "Add"     | "nodeModePodHandler"           |
      | "node1" | 0    | "none"           | "standalone" | "Add"     | "podMonitorHandler"            |
      | "node1" | 0    | "GetPod"         | "standalone" | "Add"     | "GetPod error"                 |
      | "node1" | 0    | "none"           | "none"       | "Modify"  | "PodMonitor.Mode not set"      |
      | "node1" | 0    | "none"           | "none"       | "Delete"  | "PodMonitor.Mode not set"      |
      | "node1" | 0    | "none"           | "none"       | "Stop"    | "PodWatcher stopped..."        |
      | "node1" | 0    | "none"           | "none"       | "Error"   | "Setup of PodWatcher complete" |

  @monitor
  Scenario Outline: Test StartNodeMonitorHandler
    Given a controller monitor "vxflex"
    And a pod for node <podnode> with <nvol> volumes condition ""
    And I induce error <error>
    When I call StartNodeMonitor with key "podmon" and value "csi-vxflexos"
    And I send a node event type <eventtype>
    Then I close the Watcher
    And the last log message contains <errormsg>

    Examples:
      | podnode | nvol | error            | eventtype | errormsg                        |
      | "node1" | 0    | "Watch"          | "None"    | "none"                          |
      | "node1" | 0    | "none"           | "None"    | "Setup of NodeWatcher complete" |
      | "node1" | 0    | "none"           | "Add"     | "node name: node1"              |
      | "node1" | 0    | "BadWatchObject" | "Add"     | "nodeMonitorHandler nil node"   |
      | "node1" | 0    | "none"           | "Modify"  | "node name: node1"              |
      | "node1" | 0    | "none"           | "Delete"  | "node name: node1"              |
      | "node1" | 0    | "none"           | "Stop"    | "NodeWatcher stopped..."        |
      | "node1" | 0    | "none"           | "Error"   | "Setup of NodeWatcher complete" |

  @monitor
  Scenario Outline: Test Lock/Unlock and getPodKey
    Given a controller monitor "vxflex"
    And a pod for node <podnode> with <nvol> volumes condition ""
    When I call test lock and getPodKey
   # The previous step will fail if there is an error

    Examples:
      | podnode | nvol |
      | "node1" | 0    |

