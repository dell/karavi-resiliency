module podmon

go 1.13

replace  github.com/dell/dell-csi-extensions/podmon => ./dell-csi-extensions/podmon

require (
	github.com/dell/dell-csi-extensions/podmon v0.0.0
	github.com/container-storage-interface/spec v1.1.0
	github.com/cucumber/godog v0.10.0
	github.com/dell/gofsutil v1.3.0
	github.com/kubernetes-csi/csi-lib-utils v0.7.0
	github.com/sirupsen/logrus v1.6.0
	github.com/stretchr/testify v1.6.1
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d // indirect
	golang.org/x/time v0.0.0-20200630173020-3af7569d3a1e // indirect
	google.golang.org/grpc v1.33.2
	k8s.io/api v0.18.6
	k8s.io/apimachinery v0.18.6
	k8s.io/client-go v0.18.6
	k8s.io/utils v0.0.0-20200731180307-f00132d28269 // indirect
)