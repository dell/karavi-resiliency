<!--
Copyright (c) 2021 Dell Inc., or its subsidiaries. All Rights Reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0
-->

# Reporting Problems

If you experience a problem with Karavi Resiliency it is important you provide us as much information as possible so that we can diagnose the issue and improve Karavi Resiliency. If possible, please submit all the following data (listed in decreasing order of importance):

* The controller-podmon logs. If there are multiple controllers deployed, submit the podmon logs for the controller that has been elected leader, or submit the logs for all the podmon controllers.
* The node-podmon log for any involved nodes (nodes containing pods that have not been able to achieve the Ready state).
* The output of "kubectl describe pod ..." for the failed pod(s).
* The output of "kubectl get pvc -n namespace" for the namespace(s) of the failed pod.
* The events for the namespace(s) that have failed pods ("kubectl get events -n namespace")
* The CSI driver node logs for any involved nodes.
* The CSI driver controller logs. If there are multiple controllers, submit the driver logs for the controller that has been elected leader, or submit the logs for all the driver controllers.