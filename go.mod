module github.com/aliyun/aliyun_assist_client

go 1.20

require (
	bou.ke/monkey v1.0.2
	github.com/Microsoft/go-winio v0.4.17
	github.com/agiledragon/gomonkey/v2 v2.11.0
	github.com/aliyun/alibaba-cloud-sdk-go v1.61.1491
	github.com/coreos/go-systemd/v22 v22.3.2
	github.com/creack/pty v1.1.11
	github.com/docker/docker v24.0.4+incompatible
	github.com/fabiokung/shm v0.0.0-20150728212823-2852b0d79bae
	github.com/go-logr/logr v1.2.3
	github.com/godbus/dbus/v5 v5.0.6
	github.com/golang/protobuf v1.5.2
	github.com/google/shlex v0.0.0-20191202100458-e7afc7fbc510
	github.com/google/uuid v1.2.0
	github.com/gorhill/cronexpr v0.0.0-20180427100037-88b0669f7d75
	github.com/gorilla/websocket v1.4.2
	github.com/hectane/go-acl v0.0.0-20190604041725-da78bae5fc95
	github.com/jarcoal/httpmock v1.0.8
	github.com/kirinlabs/HttpRequest v1.1.1
	github.com/lestrrat-go/strftime v1.0.4
	github.com/marcsauter/single v0.0.0-20201009143647-9f8d81240be2
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd
	github.com/modern-go/reflect2 v1.0.2
	github.com/pkg/errors v0.9.1
	github.com/rodaine/table v1.0.1
	github.com/shirou/gopsutil v3.21.4+incompatible
	github.com/shirou/gopsutil/v3 v3.22.10
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.8.1
	github.com/tidwall/gjson v1.9.3
	github.com/viney-shih/go-lock v1.0.1
	github.com/yookoala/realpath v1.0.0
	go.etcd.io/bbolt v1.3.7
	go.uber.org/atomic v1.11.0
	golang.org/x/net v0.17.0
	golang.org/x/sys v0.13.0
	golang.org/x/term v0.13.0
	golang.org/x/text v0.13.0
	google.golang.org/grpc v1.40.0
	gopkg.in/ini.v1 v1.66.2
	k8s.io/cri-api v0.24.3
	k8s.io/klog/v2 v2.60.1
	k8s.io/kubernetes v1.24.3
	k8s.io/utils v0.0.0-20220210201930-3a6ce19ff2f9
)

require (
	github.com/StackExchange/wmi v0.0.0-20210224194228-fe8f1750fd46 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/docker/distribution v2.8.2+incompatible // indirect
	github.com/docker/go-connections v0.4.0 // indirect
	github.com/docker/go-units v0.4.0 // indirect
	github.com/go-ole/go-ole v1.2.6 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/lufia/plan9stats v0.0.0-20211012122336-39d0f177ccd0 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.0.2 // indirect
	github.com/pborman/uuid v1.2.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/power-devops/perfstat v0.0.0-20210106213030-5aafc221ea8c // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.0 // indirect
	github.com/tklauser/go-sysconf v0.3.10 // indirect
	github.com/tklauser/numcpus v0.4.0 // indirect
	github.com/yusufpapurcu/wmi v1.2.2 // indirect
	golang.org/x/sync v0.1.0 // indirect
	google.golang.org/genproto v0.0.0-20220107163113-42d7afdf6368 // indirect
	google.golang.org/protobuf v1.28.1 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	k8s.io/apimachinery v0.24.3 // indirect
	k8s.io/apiserver v0.24.3 // indirect
	k8s.io/component-base v0.24.3 // indirect
)

// Dependency replacements listed below MUST be maintained via script
// tools/update-k8s-dependencies.sh
replace (
	k8s.io/api => k8s.io/api v0.24.3
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.24.3
	k8s.io/apimachinery => k8s.io/apimachinery v0.24.4-rc.0
	k8s.io/apiserver => k8s.io/apiserver v0.24.3
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.24.3
	k8s.io/client-go => k8s.io/client-go v0.24.3
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.24.3
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.24.3
	k8s.io/code-generator => k8s.io/code-generator v0.24.4-rc.0
	k8s.io/component-base => k8s.io/component-base v0.24.3
	k8s.io/component-helpers => k8s.io/component-helpers v0.24.3
	k8s.io/controller-manager => k8s.io/controller-manager v0.24.3
	k8s.io/cri-api => k8s.io/cri-api v0.25.0-alpha.0
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.24.3
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.24.3
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.24.3
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.24.3
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.24.3
	k8s.io/kubectl => k8s.io/kubectl v0.24.3
	k8s.io/kubelet => k8s.io/kubelet v0.24.3
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.24.3
	k8s.io/metrics => k8s.io/metrics v0.24.3
	k8s.io/mount-utils => k8s.io/mount-utils v0.24.4-rc.0
	k8s.io/pod-security-admission => k8s.io/pod-security-admission v0.24.3
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.24.3
	k8s.io/sample-cli-plugin => k8s.io/sample-cli-plugin v0.24.3
	k8s.io/sample-controller => k8s.io/sample-controller v0.24.3
)
