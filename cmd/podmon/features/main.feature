Feature: Podmon Main
  As a podmon developer
  I want to test the main function
  So that the podmon startup works as expected

  Scenario Outline: Test the main routine in node mode
    Given a podmon instance
    And Podmon env vars set to <k8sHostValue>:<k8sPort>
    And I invoke main with arguments <args>
    Then the last log message contains <message>

    Examples:
      | k8sHostValue | k8sPort | args                                                                                           | message                 |
      | "localhost"  | "1234"  | "--mode=node --leaderelection=true"                                                            | "leader election: true" |
      | "localhost"  | "1234"  | "--mode=node --leaderelection=true --csisock='csi.sock'"                                       | "leader election: true" |
      | "localhost"  | "1234"  | "--mode=node --leaderelection=false"                                                           | "podmon alive"          |
      | "localhost"  | "1234"  | "--mode=node --leaderelection=false --csisock='csi.sock' "                                     | "podmon alive"          |
      # Skip array connection check
      | "localhost"  | "1234"  | "--mode=node --leaderelection=true --csisock='csi.sock' --skipArrayConnectionValidation=true"  | "leader election: true" |
      | "localhost"  | "1234"  | "--mode=node --leaderelection=false --csisock='csi.sock' --skipArrayConnectionValidation=true" | "podmon alive"          |

  Scenario Outline: Test the main routine in controller mode
    Given a podmon instance
    And Podmon env vars set to <k8sHostValue>:<k8sPort>
    And I invoke main with arguments <args>
    Then the last log message contains <message>
    Examples:
      | k8sHostValue | k8sPort | args                                                                                                 | message                 |
      | "localhost"  | "1234"  | "--leaderelection=true"                                                                              | "leader election: true" |
      | "localhost"  | "1234"  | "--mode=controller --leaderelection=true"                                                            | "leader election: true" |
      | "localhost"  | "1234"  | "--mode=controller --leaderelection=true --csisock='csi.sock'"                                       | "leader election: true" |
      | "localhost"  | "1234"  | "--mode=controller --leaderelection=false"                                                           | "podmon alive"          |
      | "localhost"  | "1234"  | "--mode=controller --leaderelection=false --csisock='csi.sock'"                                      | "podmon alive"          |
      # Skip array connection check
      | "localhost"  | "1234"  | "--mode=controller --leaderelection=true --csisock='csi.sock' --skipArrayConnectionValidation=true"  | "leader election: true" |
      | "localhost"  | "1234"  | "--mode=controller --leaderelection=false --csisock='csi.sock' --skipArrayConnectionValidation=true" | "podmon alive"          |

  Scenario Outline: Test the main routine in standalone mode
    Given a podmon instance
    And Podmon env vars set to <k8sHostValue>:<k8sPort>
    And I invoke main with arguments <args>
    Then the last log message contains <message>

    Examples:
      | k8sHostValue | k8sPort | args                                                                                                 | message                 |
      | "localhost"  | "1234"  | "--mode=standalone --leaderelection=true"                                                            | "leader election: true" |
      | "localhost"  | "1234"  | "--mode=standalone --leaderelection=true --csisock='csi.sock'"                                       | "leader election: true" |
      | "localhost"  | "1234"  | "--mode=standalone --leaderelection=false"                                                           | "podmon alive"          |
      | "localhost"  | "1234"  | "--mode=standalone --leaderelection=false --csisock='csi.sock'"                                      | "podmon alive"          |
      # Skip array connection check
      | "localhost"  | "1234"  | "--mode=standalone --leaderelection=false --csisock='csi.sock' --skipArrayConnectionValidation=true" | "podmon alive"          |
      | "localhost"  | "1234"  | "--mode=standalone --leaderelection=true --csisock='csi.sock' --skipArrayConnectionValidation=true"  | "leader election: true" |

  Scenario Outline: Test the main routine with negative test cases
    Given a podmon instance
    And Podmon env vars set to <k8sHostValue>:<k8sPort>
    And I induce error <induceErr>
    And I invoke main with arguments <args>
    Then the last log message contains <message>

    Examples:
      | k8sHostValue | k8sPort | args                                       | induceErr         | message                                |
      | "localhost"  | "1234"  | "--mode=blah --labelvalue=utest"           | "none"            | "invalid mode"                         |
      | "localhost"  | "1234"  | "--mode=controller"                        | "Connect"         | "Connect error"                        |
      | "localhost"  | "1234"  | "--mode=node"                              | "Connect"         | "Connect error"                        |
      | "localhost"  | "1234"  | "--mode=standalone"                        | "Connect"         | "Connect error"                        |
      # Fail leader election
      | "localhost"  | "1234"  | "--mode=controller --leaderelection=true"  | "LeaderElection"  | "failed to initialize leader election" |
      | "localhost"  | "1234"  | "--mode=node --leaderelection=true"        | "LeaderElection"  | "failed to initialize leader election" |
      | "localhost"  | "1234"  | "--mode=standalone --leaderelection=true"  | "LeaderElection"  | "failed to initialize leader election" |
      # Fail StartAPIMonitor
      | "localhost"  | "1234"  | "--mode=controller --leaderelection=false" | "StartAPIMonitor" | "podmon alive"                         |
      | "localhost"  | "1234"  | "--mode=node --leaderelection=false"       | "StartAPIMonitor" | "Couldn't start API monitor:"          |
      | "localhost"  | "1234"  | "--mode=standalone --leaderelection=false" | "StartAPIMonitor" | "podmon alive"                         |

  Scenario Outline: Check on CSIExtensionsPresent flag
    Given a podmon instance
    And Podmon env vars set to <k8sHostValue>:<k8sPort>
    And I induce error <induceErr>
    And I invoke main with arguments <args>
    Then CSIExtensionsPresent is <csiExtPresent>

    Examples:
      | k8sHostValue | k8sPort | args                                                           | induceErr                        | csiExtPresent |
      | "localhost"  | "1234"  | "--mode=controller --leaderelection=true --csisock='csi.sock'" | "none"                           | "true"        |
      | "localhost"  | "1234"  | "--mode=node --leaderelection=true --csisock='csi.sock'"       | "none"                           | "true"        |
      | "localhost"  | "1234"  | "--mode=standalone --leaderelection=true --csisock='csi.sock'" | "none"                           | "true"        |
      # Negative test cases
      | "localhost"  | "1234"  | "--mode=controller --leaderelection=true --csisock='csi.sock'" | "ValidateVolumeHostConnectivity" | "false"       |
      | "localhost"  | "1234"  | "--mode=node --leaderelection=true --csisock='csi.sock'"       | "ValidateVolumeHostConnectivity" | "false"       |
      | "localhost"  | "1234"  | "--mode=standalone --leaderelection=true --csisock='csi.sock'" | "ValidateVolumeHostConnectivity" | "false"       |

  Scenario Outline: Different driver paths
    Given a podmon instance
    And Podmon env vars set to <k8sHostValue>:<k8sPort>
    And I invoke main with arguments <args>
    Then the last log message contains <message>

    Examples:
      | k8sHostValue | k8sPort | args                      | message                 |
      | "localhost"  | "1234"  | "--driverPath=unity"      | "leader election: true" |
      | "localhost"  | "1234"  | "--driverPath=vxflexos"   | "leader election: true" |
      | "localhost"  | "1234"  | "--driverPath=isilon"     | "leader election: true" |
      | "localhost"  | "1234"  | "--driverPath=powerstore" | "leader election: true" |
      | "localhost"  | "1234"  | "--driverPath=powermax"   | "leader election: true" |

  Scenario Outline: Test using driver ConfigMap
    Given a podmon instance
    And Podmon env vars set to <k8sHostValue>:<k8sPort>
    And I invoke main with arguments <args>
    Then the last log message contains <message>

    Examples:
      | k8sHostValue | k8sPort | args                                                                                | message                                  |
      # Use a default for testing
      | "localhost"  | "1234"  | ""                                                                                  | "leader election: true"                  |
      | "localhost"  | "1234"  | "-driver-config-params=resources/driver-config-params2.yaml"                        | "leader election: true"                  |
      | "localhost"  | "1234"  | "--mode=node"                                                                       | "leader election: true"                  |
      | "localhost"  | "1234"  | "--mode=node -driver-config-params=resources/driver-config-params2.yaml"            | "leader election: true"                  |
      # Error cases
      | "localhost"  | "1234"  | "--driver-config-params= "                                                          | "--driver-config-params cannot be empty" |
      | "localhost"  | "1234"  | "--driver-config-params=fake"                                                       | "unable to read driver config file"      |
      # Bad data in ConfigMaps
      | "localhost"  | "1234"  | "--driver-config-params=resources/driver-config-params-bad-format1.yaml"            | "leader election: true"                  |
      | "localhost"  | "1234"  | "--driver-config-params=resources/driver-config-params-bad-format2.yaml"            | "leader election: true"                  |
      | "localhost"  | "1234"  | "--driver-config-params=resources/driver-config-params-bad-level1.yaml"             | "error with configuration parameters"    |
      | "localhost"  | "1234"  | "--mode=node --driver-config-params=resources/driver-config-params-bad-level2.yaml" | "error with configuration parameters"    |
      | "localhost"  | "1234"  | "--driver-config-params=resources/driver-config-params-bad-value1.yaml"             | "error with configuration parameters"    |
      | "localhost"  | "1234"  | "--driver-config-params=resources/driver-config-params-bad-value2.yaml"             | "error with configuration parameters"    |
      | "localhost"  | "1234"  | "--driver-config-params=resources/driver-config-params-bad-value3.yaml"             | "error with configuration parameters"    |
      | "localhost"  | "1234"  | "--driver-config-params=resources/driver-config-params-bad-value4.yaml"             | "error with configuration parameters"    |
      | "localhost"  | "1234"  | "--driver-config-params=resources/driver-config-params-bad-value5.yaml"             | "error with configuration parameters"    |
