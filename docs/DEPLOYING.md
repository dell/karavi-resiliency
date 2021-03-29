<!--
Copyright (c) 2021 Dell Inc., or its subsidiaries. All Rights Reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0
-->

# Deploying Karavi Resiliency

Karavi Resiliency is deployed as part of the CSI driver deployment. The drivers can be deployed either by a _helm chart_ or by the _Dell CSI Operator_. For the alpha (Tech. Preview) phase, only _helm chart_ installation is supported.

For information on the PowerFlex CSI driver, see (PowerFlex CSI Driver)[https://github.com/dell/csi-powerflex].

Configure all the helm chart parameters described below before deploying the drivers.

## Helm Chart Installation

These installation instructions apply to the helm chart in the github.com/dell/csi-powerflex repository version v1.4.0. There was a change
identified after the PowerFlex driver release that needs to be made to the helm chart, specifically to the file helm/csi-vxflexos/templates/node.yaml. It is a simple two line addition to the podmon container section of the chart. Please make this change before deploying podmon.

The diff is as follows:

```
@@ -113,8 +113,10 @@ spec:
           volumeMounts:
             - name: kubelet-pods
               mountPath: /var/lib/kubelet/pods
+              mountPropagation: "Bidirectional"
             - name: driver-path
               mountPath: /var/lib/kubelet/plugins/vxflexos.emc.dell.com
+              mountPropagation: "Bidirectional"
             - name: usr-bin
               mountPath: /usr-bin
```


For reference, the entire node.yaml file with the change applied is available here: [node.yaml](node.yaml).

The drivers that support Helm chart deployment allow Karavi Resiliency to be _optionally_ deployed by variables in the chart. There is a _podmon_ block specified in the _values.yaml_ file of the chart that will look similar the text below by default:

```
# Podmon is an optional feature under development and tech preview.
# Enable this feature only after contact support for additional information
podmon:
  enabled: false
  image: 
  #controller:
  #  args:
  #    - "-csisock=unix:/var/run/csi/csi.sock"
  #    - "-labelvalue=csi-vxflexos"
  #    - "-mode=controller"
  #node:
  #  args:
  #    - "-csisock=unix:/var/lib/kubelet/plugins/vxflexos.emc.dell.com/csi_sock"
  #    - "-labelvalue=csi-vxflexos"
  #    - "-mode=node"
  #    - "-leaderelection=false"
```

To deploy Karavi Resiliency with the driver, the following changes are requried:
1. Enable Karavi Resiliency by changing the podmon.enabled boolean to true. This will enable both controller-podmon and node-podmon.
2. Specify the podmon image to be used as podmon.image.
3. Specify arguments to controller-podmon in the podmon.controller.args block. See "Podmon Arguments" below. Note that some arguments are required. Note that the arguments supplied to controller-podmon are different than those supplied to node-podmon.
4. Specify arguments to node-podmon in the podmon.node.args block. See "Podmon Arguments" below. Note that some arguments are required. Note that the arguments supplied to controller-podmon are different than those supplied to node-podmon.

## Podmon Arguments
  
|Argument | Required | Description | Applicability |
|---------|----------|-------------|---------------|
| enabled | Required | Boolean "true" enables Karavi Resiliency deployment with the driver in a helm installation. | top level |
| image   | Required | Must be set to a repository where the podmon image can be pulled. | controller & node |
|mode     | Required | Must be set to "controller" for controller-podmon and "node" for node-podmon. | controller & node |
|csisock  | Required | This should be left as set in the helm template for the driver. For controller: "-csisock=unix:/var/run/csi/csi.sock". For node it will vary depending on the driver's identity, e.g. "-csisock=unix:/var/lib/kubelet/plugins/vxflexos.emc.dell.com/csi_sock" | controller & node |
| leaderelection | Required | Boolean value that should be set true for controller and false for node. The default value is true. | controller & node |
| skipArrayConnectionValidation | Optional | Boolean value that if set to true will cause controllerPodCleanup to skip the validation that no I/O is ongong before cleaning up the pod. | controller |
| labelKey | Optional | String value that sets the label key used to denote pods to be monitored by Karavi Resiliency. It will make life easier if this key is the same for all driver types, and drivers are differentiated by different labelValues (see below). If the label keys are the same across all drivers you can do "kubectl get pods -A -l labelKey" to find all the Karavi Resiliency protected pods. labelKey defaults to "podmon.dellemc.com/driver". | controller & node |
| labelValue | Required | String that sets the value that denotes pods to be monitored by Karavi Resiliency. This must be specific for each driver. Defaults to "csi-vxflexos" | controller & node |
| arrayConnectivityPollRate | Optional | The minimum polling rate in seconds to determine if array has connectivity to a node. Should not be set to less than 5 seconds. See the specific section for each array type for additional guidance. | controller |
| arrayConnectivityConnectionLossThreshold | Optional | Gives the number of failed connection polls that will be deemed to indicate array connectivity loss. Should not be set to less than 3. See the specific section for each array type for additional guidance. | controller |

## PowerFlex Specific Recommendations

PowerFlex supports a very robust array connection validation mechanism that can detect changes in connectivity in about two seconds and can detect whether I/O has occured over a five second sample. For that reason it is recommended to set "skipArrayConnectionValidation=false" (which is the default) and to set "arrayConnectivityPollRate=5" (5 seconds) and "arrayConnectivityConnectionLossThreshold=3" to 3 or more.

Here is a typical deployment used for testing:

```
podmon:
  image: image_repository_host_ip:5000/podmon:v0.0.54
  enabled: true
  controller:
    args:
      - "-csisock=unix:/var/run/csi/csi.sock"
      - "-labelvalue=csi-vxflexos"
      - "-mode=controller"
      - "-arrayConnectivityPollRate=5"
      - "-arrayConnectivityConnectionLossThreshold=3"
  node:
    args:
      - "-csisock=unix:/var/lib/kubelet/plugins/vxflexos.emc.dell.com/csi_sock"
      - "-labelvalue=csi-vxflexos"
      - "-mode=node"
      - "-leaderelection=false"

```
