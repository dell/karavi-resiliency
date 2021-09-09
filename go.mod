module podmon

go 1.13

require (
	github.com/bramvdbogaerde/go-scp v0.0.0-20201229172121-7a6c0268fa67
	github.com/container-storage-interface/spec v1.1.0
	github.com/cucumber/godog v0.10.0
	github.com/dell/dell-csi-extensions/podmon v1.0.0
	github.com/dell/gofsutil v1.3.0
	github.com/fsnotify/fsnotify v1.4.9
	github.com/golang/mock v1.5.0
	github.com/kubernetes-csi/csi-lib-utils v0.7.0
	github.com/sirupsen/logrus v1.6.0
	github.com/spf13/viper v1.8.1
	github.com/stretchr/testify v1.7.0
	golang.org/x/crypto v0.0.0-20201216223049-8b5274cf687f
	golang.org/x/time v0.0.0-20200630173020-3af7569d3a1e // indirect
	google.golang.org/grpc v1.38.0
	k8s.io/api v0.18.6
	k8s.io/apimachinery v0.18.6
	k8s.io/client-go v0.18.6
	k8s.io/cri-api v0.21.0
	k8s.io/utils v0.0.0-20200731180307-f00132d28269 // indirect
)
