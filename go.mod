module github.com/nmstate/kubernetes-nmstate

go 1.11

require (
	cloud.google.com/go v0.39.0 // indirect
	github.com/NYTimes/gziphandler v1.0.1 // indirect
	github.com/emicklei/go-restful v2.9.5+incompatible // indirect
	github.com/evanphx/json-patch v4.4.0+incompatible // indirect
	github.com/go-openapi/spec v0.19.2
	github.com/gobuffalo/envy v1.7.0 // indirect
	github.com/golang/groupcache v0.0.0-20190702054246-869f871628b6 // indirect
	github.com/gophercloud/gophercloud v0.1.0 // indirect
	github.com/gregjones/httpcache v0.0.0-20190611155906-901d90724c79 // indirect
	github.com/grpc-ecosystem/grpc-gateway v1.9.0 // indirect
	github.com/onsi/ginkgo v1.8.0
	github.com/onsi/gomega v1.5.0
	github.com/operator-framework/operator-sdk v0.9.0
	github.com/prometheus/client_golang v0.9.3 // indirect
	github.com/prometheus/common v0.4.1 // indirect
	github.com/prometheus/procfs v0.0.0-20190528151240-3cb620ac02d0 // indirect
	github.com/sirupsen/logrus v1.4.2 // indirect
	github.com/spf13/cobra v0.0.4 // indirect
	github.com/spf13/pflag v1.0.3
	go.uber.org/atomic v1.4.0 // indirect
	go.uber.org/zap v1.10.0 // indirect
	golang.org/x/oauth2 v0.0.0-20190523182746-aaccbc9213b0 // indirect
	google.golang.org/appengine v1.6.0 // indirect
	google.golang.org/genproto v0.0.0-20190522204451-c2c4e71fbf69 // indirect
	google.golang.org/grpc v1.21.0 // indirect
	k8s.io/api v0.0.0-20190612125737-db0771252981
	k8s.io/apimachinery v0.0.0-20190612125636-6a5db36e93ad
	k8s.io/client-go v11.0.0+incompatible
	k8s.io/klog v0.3.2 // indirect
	k8s.io/kube-openapi v0.0.0-20190709113604-33be087ad058
	sigs.k8s.io/controller-runtime v0.1.10
	sigs.k8s.io/controller-tools v0.1.11 // indirect
	sigs.k8s.io/yaml v1.1.0
)

replace (
	github.com/Sirupsen/logrus => github.com/Sirupsen/logrus v1.0.7-0.20180904202135-3791101e143b
	k8s.io/api => k8s.io/api v0.0.0-20181213150558-05914d821849
	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20181127025237-2b1284ed4c93
	k8s.io/client-go => k8s.io/client-go v0.0.0-20181213151034-8d9ed539ba31
)
