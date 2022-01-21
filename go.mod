module github.com/nmstate/kubernetes-nmstate

go 1.17

require (
	github.com/evanphx/json-patch v4.12.0+incompatible
	github.com/github-release/github-release v0.10.0
	github.com/go-logr/logr v1.2.0
	github.com/gofrs/flock v0.8.0
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/nmstate/kubernetes-nmstate/api v0.0.0
	github.com/nmstate/nmpolicy v0.2.1
	github.com/onsi/ginkgo/v2 v2.1.3
	github.com/onsi/gomega v1.17.0
	github.com/openshift/cluster-network-operator v0.0.0-20200922032245-f47200e8dbc0
	github.com/operator-framework/operator-registry v1.19.5
	github.com/phoracek/networkmanager-go v0.3.0
	github.com/pkg/errors v0.9.1
	github.com/qinqon/kube-admission-webhook v0.18.0
	github.com/spf13/pflag v1.0.5
	github.com/tidwall/gjson v1.9.3
	github.com/tidwall/sjson v1.1.7
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/api v0.23.1
	k8s.io/apimachinery v0.23.1
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/kubectl v0.23.1
	k8s.io/release v0.12.0
	nmstate.io/go/nmstate v0.0.0
	sigs.k8s.io/controller-runtime v0.11.0
	sigs.k8s.io/controller-tools v0.6.0
	sigs.k8s.io/yaml v1.3.0
)

require (
	cloud.google.com/go v0.93.3 // indirect
	cloud.google.com/go/storage v1.16.1 // indirect
	github.com/Azure/go-ansiterm v0.0.0-20210617225240-d185dfc1b5a1 // indirect
	github.com/BurntSushi/toml v0.3.1 // indirect
	github.com/MakeNowJust/heredoc v0.0.0-20170808103936-bb23615498cd // indirect
	github.com/Masterminds/goutils v1.1.0 // indirect
	github.com/Masterminds/semver v1.5.0 // indirect
	github.com/Masterminds/sprig v2.22.0+incompatible // indirect
	github.com/Microsoft/go-winio v0.5.0 // indirect
	github.com/Microsoft/hcsshim v0.8.23 // indirect
	github.com/ProtonMail/go-crypto v0.0.0-20210428141323-04723f9f07d7 // indirect
	github.com/PuerkitoBio/purell v1.1.1 // indirect
	github.com/PuerkitoBio/urlesc v0.0.0-20170810143723-de5bf2ad4578 // indirect
	github.com/StackExchange/wmi v1.2.1 // indirect
	github.com/VividCortex/ewma v1.1.1 // indirect
	github.com/acomagu/bufpipe v1.0.3 // indirect
	github.com/antlr/antlr4/runtime/Go/antlr v0.0.0-20210826220005-b48c857c3a0e // indirect
	github.com/asaskevich/govalidator v0.0.0-20200108200545-475eaeb16496 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/blang/semver v3.5.1+incompatible // indirect
	github.com/blang/semver/v4 v4.0.0 // indirect
	github.com/cespare/xxhash/v2 v2.1.2 // indirect
	github.com/cheggaaa/pb/v3 v3.0.8 // indirect
	github.com/containerd/cgroups v1.0.1 // indirect
	github.com/containerd/containerd v1.5.10 // indirect
	github.com/containerd/continuity v0.1.0 // indirect
	github.com/containerd/stargz-snapshotter/estargz v0.7.0 // indirect
	github.com/containerd/ttrpc v1.1.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/docker/cli v20.10.7+incompatible // indirect
	github.com/docker/distribution v2.7.1+incompatible // indirect
	github.com/docker/docker v20.10.7+incompatible // indirect
	github.com/docker/docker-credential-helpers v0.6.3 // indirect
	github.com/docker/go-connections v0.4.0 // indirect
	github.com/docker/go-metrics v0.0.1 // indirect
	github.com/docker/go-units v0.4.0 // indirect
	github.com/dustin/go-humanize v1.0.0 // indirect
	github.com/emirpasic/gods v1.12.0 // indirect
	github.com/fatih/color v1.12.0 // indirect
	github.com/felixge/httpsnoop v1.0.1 // indirect
	github.com/fsnotify/fsnotify v1.5.1 // indirect
	github.com/ghodss/yaml v1.0.1-0.20190212211648-25d852aebe32 // indirect
	github.com/go-errors/errors v1.0.1 // indirect
	github.com/go-git/gcfg v1.5.0 // indirect
	github.com/go-git/go-billy/v5 v5.3.1 // indirect
	github.com/go-git/go-git/v5 v5.4.2 // indirect
	github.com/go-logr/zapr v1.2.0 // indirect
	github.com/go-ole/go-ole v1.2.6 // indirect
	github.com/go-openapi/jsonpointer v0.19.5 // indirect
	github.com/go-openapi/jsonreference v0.19.5 // indirect
	github.com/go-openapi/swag v0.19.14 // indirect
	github.com/go-task/slim-sprig v0.0.0-20210107165309-348f09dbbbc0 // indirect
	github.com/gobuffalo/flect v0.2.3 // indirect
	github.com/godbus/dbus/v5 v5.0.4 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang-migrate/migrate/v4 v4.6.2 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/golang/snappy v0.0.3 // indirect
	github.com/gomarkdown/markdown v0.0.0-20210514010506-3b9f47219fe7 // indirect
	github.com/google/cel-go v0.9.0 // indirect
	github.com/google/go-cmp v0.5.6 // indirect
	github.com/google/go-containerregistry v0.6.0 // indirect
	github.com/google/go-github/v39 v39.2.0 // indirect
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/google/gofuzz v1.1.0 // indirect
	github.com/google/licenseclassifier/v2 v2.0.0-alpha.1 // indirect
	github.com/google/pprof v0.0.0-20210720184732-4bb14d4b1be1 // indirect
	github.com/google/shlex v0.0.0-20191202100458-e7afc7fbc510 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/googleapis/gax-go/v2 v2.1.0 // indirect
	github.com/googleapis/gnostic v0.5.5 // indirect
	github.com/gorilla/mux v1.8.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway v1.16.0 // indirect
	github.com/h2non/filetype v1.1.1 // indirect
	github.com/h2non/go-is-svg v0.0.0-20160927212452-35e8c4b0612c // indirect
	github.com/huandu/xstrings v1.2.0 // indirect
	github.com/imdario/mergo v0.3.12 // indirect
	github.com/in-toto/in-toto-golang v0.3.3 // indirect
	github.com/inconshreveable/mousetrap v1.0.0 // indirect
	github.com/jbenet/go-context v0.0.0-20150711004518-d14ea06fba99 // indirect
	github.com/joelanford/ignore v0.0.0-20210607151042-0d25dc18b62d // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/kevinburke/rest v0.0.0-20210222204520-f7a2e216372f // indirect
	github.com/kevinburke/ssh_config v1.1.0 // indirect
	github.com/klauspost/compress v1.13.0 // indirect
	github.com/mailru/easyjson v0.7.6 // indirect
	github.com/mattn/go-colorable v0.1.8 // indirect
	github.com/mattn/go-isatty v0.0.14 // indirect
	github.com/mattn/go-runewidth v0.0.12 // indirect
	github.com/mattn/go-sqlite3 v1.10.0 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.2-0.20181231171920-c182affec369 // indirect
	github.com/mitchellh/copystructure v1.0.0 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/mitchellh/go-wordwrap v1.0.0 // indirect
	github.com/mitchellh/hashstructure/v2 v2.0.2 // indirect
	github.com/mitchellh/mapstructure v1.4.2 // indirect
	github.com/mitchellh/reflectwalk v1.0.1 // indirect
	github.com/mmarkdown/mmark v2.0.40+incompatible // indirect
	github.com/moby/locker v1.0.1 // indirect
	github.com/moby/spdystream v0.2.0 // indirect
	github.com/moby/sys/mountinfo v0.4.1 // indirect
	github.com/moby/term v0.0.0-20210610120745-9d4ed1856297 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/monochromegane/go-gitignore v0.0.0-20200626010858-205db1a8cc00 // indirect
	github.com/morikuni/aec v1.0.0 // indirect
	github.com/nozzle/throttler v0.0.0-20180817012639-2ea982251481 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.0.2 // indirect
	github.com/operator-framework/api v0.10.5 // indirect
	github.com/otiai10/copy v1.2.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/prometheus/client_golang v1.12.1 // indirect
	github.com/prometheus/client_model v0.2.0 // indirect
	github.com/prometheus/common v0.32.1 // indirect
	github.com/prometheus/procfs v0.7.3 // indirect
	github.com/rivo/uniseg v0.2.0 // indirect
	github.com/russross/blackfriday v1.5.2 // indirect
	github.com/saschagrunert/go-modiff v1.3.0 // indirect
	github.com/sergi/go-diff v1.2.0 // indirect
	github.com/shibumi/go-pathspec v1.2.0 // indirect
	github.com/shirou/gopsutil/v3 v3.21.10 // indirect
	github.com/sirupsen/logrus v1.8.1 // indirect
	github.com/spf13/cobra v1.2.1 // indirect
	github.com/spiegel-im-spiegel/errs v1.0.2 // indirect
	github.com/spiegel-im-spiegel/go-cvss v0.4.0 // indirect
	github.com/stoewer/go-strcase v1.2.0 // indirect
	github.com/stretchr/testify v1.7.0 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.0 // indirect
	github.com/tomnomnom/linkheader v0.0.0-20180905144013-02ca5825eb80 // indirect
	github.com/voxelbrain/goptions v0.0.0-20180630082107-58cddc247ea2 // indirect
	github.com/xanzy/ssh-agent v0.3.0 // indirect
	github.com/xlab/treeprint v0.0.0-20181112141820-a009c3971eca // indirect
	go.etcd.io/bbolt v1.3.6 // indirect
	go.opencensus.io v0.23.0 // indirect
	go.opentelemetry.io/contrib v0.20.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.20.0 // indirect
	go.opentelemetry.io/otel v0.20.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp v0.20.0 // indirect
	go.opentelemetry.io/otel/metric v0.20.0 // indirect
	go.opentelemetry.io/otel/sdk v0.20.0 // indirect
	go.opentelemetry.io/otel/sdk/export/metric v0.20.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v0.20.0 // indirect
	go.opentelemetry.io/otel/trace v0.20.0 // indirect
	go.opentelemetry.io/proto/otlp v0.7.0 // indirect
	go.starlark.net v0.0.0-20200306205701-8dd3e2ee1dd5 // indirect
	go.uber.org/atomic v1.7.0 // indirect
	go.uber.org/multierr v1.6.0 // indirect
	go.uber.org/zap v1.19.1 // indirect
	golang.org/x/crypto v0.0.0-20210817164053-32db794688a5 // indirect
	golang.org/x/mod v0.5.1 // indirect
	golang.org/x/net v0.0.0-20211209124913-491a49abca63 // indirect
	golang.org/x/oauth2 v0.0.0-20210819190943-2bc19b11175f // indirect
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c // indirect
	golang.org/x/sys v0.0.0-20220114195835-da31bd327af9 // indirect
	golang.org/x/term v0.0.0-20210615171337-6886f2dfbf5b // indirect
	golang.org/x/text v0.3.7 // indirect
	golang.org/x/time v0.0.0-20210723032227-1f47c861a9ac // indirect
	golang.org/x/tools v0.1.7 // indirect
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
	gomodules.xyz/jsonpatch/v2 v2.2.0 // indirect
	google.golang.org/api v0.56.0 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/genproto v0.0.0-20210831024726-fe130286e0e2 // indirect
	google.golang.org/grpc v1.41.0 // indirect
	google.golang.org/protobuf v1.27.1 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/warnings.v0 v0.1.2 // indirect
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
	k8s.io/apiextensions-apiserver v0.23.0 // indirect
	k8s.io/apiserver v0.23.0 // indirect
	k8s.io/cli-runtime v0.23.1 // indirect
	k8s.io/component-base v0.23.1 // indirect
	k8s.io/klog/v2 v2.30.0 // indirect
	k8s.io/kube-openapi v0.0.0-20211115234752-e816edb12b65 // indirect
	k8s.io/utils v0.0.0-20210930125809-cb0fa318a74b // indirect
	sigs.k8s.io/apiserver-network-proxy/konnectivity-client v0.0.25 // indirect
	sigs.k8s.io/json v0.0.0-20211020170558-c049b76a60c6 // indirect
	sigs.k8s.io/kustomize/api v0.10.1 // indirect
	sigs.k8s.io/kustomize/kyaml v0.13.0 // indirect
	sigs.k8s.io/mdtoc v1.1.0 // indirect
	sigs.k8s.io/release-sdk v0.2.1-0.20211005095109-f50f5112261c // indirect
	sigs.k8s.io/release-utils v0.3.0 // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.2.0 // indirect
)

replace (
	github.com/Azure/go-autorest => github.com/Azure/go-autorest v14.2.0+incompatible // Required by OLM
	// Using containerd 1.4.0+ resolves an issue with invalid error logging
	// from an init function in containerd. This replace can be removed when
	// one of our direct dependencies begins using containerd v1.4.0+
	github.com/gogo/protobuf => github.com/gogo/protobuf v1.3.2
	github.com/mattn/go-sqlite3 => github.com/mattn/go-sqlite3 v1.10.0
	github.com/nmstate/kubernetes-nmstate/api => ./api
	golang.org/x/text => golang.org/x/text v0.3.3 // Required to fix CVE-2020-14040
	k8s.io/client-go => k8s.io/client-go v0.23.1
	nmstate.io/go/nmstate => github.com/nmstate/nmstate/rust/src/go/nmstate v0.0.0-20220411120213-64e310b5e86a
)

exclude github.com/spf13/viper v1.3.2 // Required to fix CVE-2018-1098
