module tkestack.io/kstone

go 1.16

require (
	github.com/codegangsta/inject v0.0.0-20150114235600-33e0aa1cb7c0 // indirect
	github.com/coreos/etcd-operator v0.9.4
	github.com/gin-gonic/gin v1.7.2
	github.com/go-martini/martini v0.0.0-20170121215854-22fa46961aab
	github.com/go-openapi/spec v0.20.3 // indirect
	github.com/go-openapi/swag v0.19.15 // indirect
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/google/uuid v1.1.2 // indirect
	github.com/mozillazg/go-httpheader v0.3.0 // indirect
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.16.0
	github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring v0.48.1
	github.com/prometheus-operator/prometheus-operator/pkg/client v0.48.1
	github.com/prometheus/client_golang v1.11.0
	github.com/spf13/cobra v1.1.3
	github.com/spf13/pflag v1.0.5
	github.com/tencentyun/cos-go-sdk-v5 v0.7.31
	go.etcd.io/etcd/api/v3 v3.5.0
	go.etcd.io/etcd/client/pkg/v3 v3.5.0
	go.etcd.io/etcd/client/v2 v2.305.0-alpha.0
	go.etcd.io/etcd/client/v3 v3.5.0
	golang.org/x/oauth2 v0.0.0-20210323180902-22b0adad7558 // indirect
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
	k8s.io/api v0.21.3
	k8s.io/apimachinery v0.21.3
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/code-generator v0.21.3
	k8s.io/klog/v2 v2.8.0
	k8s.io/kubectl v0.21.3
	sigs.k8s.io/controller-runtime v0.9.1
	sigs.k8s.io/yaml v1.2.0
)

replace (
	github.com/coreos/etcd-operator v0.9.4 => ./third_party/etcd-operator
	k8s.io/client-go v12.0.0+incompatible => k8s.io/client-go v0.21.1
)
