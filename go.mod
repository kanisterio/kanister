module github.com/kanisterio/kanister

go 1.12

replace (
	github.com/googleapis/gnostic => github.com/googleapis/gnostic v0.4.1
	github.com/graymeta/stow => github.com/kastenhq/stow v0.2.6-kasten.1
	github.com/rook/operator-kit => github.com/kastenhq/operator-kit v0.0.0-20180316185208-859e831cc18d
	gopkg.in/check.v1 => github.com/kastenhq/check v0.0.0-20180626002341-0264cfcea734
)

require (
	github.com/Azure/azure-sdk-for-go v49.1.0+incompatible
	github.com/Azure/go-autorest/autorest v0.11.18
	github.com/Azure/go-autorest/autorest/azure/auth v0.5.3
	github.com/Azure/go-autorest/autorest/to v0.4.0
	github.com/Azure/go-autorest/autorest/validation v0.3.1 // indirect
	github.com/BurntSushi/toml v0.3.1
	github.com/IBM-Cloud/ibm-cloud-cli-sdk v0.3.0 // indirect
	github.com/IBM/ibmcloud-storage-volume-lib v1.0.2-beta02.0.20190828145158-1da4543a60af
	github.com/Masterminds/semver v1.4.2
	github.com/Masterminds/sprig v2.15.0+incompatible
	github.com/aokoli/goutils v1.1.0 // indirect
	github.com/asaskevich/govalidator v0.0.0-20200907205600-7a23bdc65eef // indirect
	github.com/aws/aws-sdk-go v1.38.17
	github.com/dnaeon/go-vcr v1.0.1 // indirect
	github.com/docker/spdystream v0.0.0-20181023171402-6480d4af844c // indirect
	github.com/elazarl/goproxy v0.0.0-20190711103511-473e67f1d7d2 // indirect
	github.com/elazarl/goproxy/ext v0.0.0-20190711103511-473e67f1d7d2 // indirect
	github.com/ghodss/yaml v1.0.1-0.20190212211648-25d852aebe32
	github.com/go-logr/logr v0.3.0 // indirect
	github.com/go-openapi/strfmt v0.19.3
	github.com/go-sql-driver/mysql v1.5.0
	github.com/golang/mock v1.5.0
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/google/uuid v1.2.0
	github.com/googleapis/gnostic v0.5.3 // indirect
	github.com/graymeta/stow v0.0.0-00010101000000-000000000000
	github.com/huandu/xstrings v1.2.0 // indirect
	github.com/imdario/mergo v0.3.11 // indirect
	github.com/jarcoal/httpmock v1.0.4 // indirect
	github.com/jpillora/backoff v1.0.0
	github.com/json-iterator/go v1.1.10
	github.com/kelseyhightower/envconfig v1.4.0 // indirect
	github.com/kopia/kopia v0.8.2-0.20210412184047-667d171714af
	github.com/kubernetes-csi/external-snapshotter/client/v4 v4.0.0
	github.com/lib/pq v1.9.0
	github.com/luci/go-render v0.0.0-20160219211803-9a04cc21af0f
	github.com/mitchellh/mapstructure v1.4.0

	//pinned openshift to release-4.5 branch
	github.com/openshift/api v0.0.0-20200526144822-34f54f12813a
	github.com/openshift/client-go v0.0.0-20200521150516-05eb9880269c
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.10.0
	github.com/renier/xmlrpc v0.0.0-20170708154548-ce4a1a486c03 // indirect
	github.com/satori/go.uuid v1.2.0
	github.com/sirupsen/logrus v1.8.1
	github.com/softlayer/softlayer-go v0.0.0-20190615201252-ba6e7f295217 // indirect
	github.com/spf13/cobra v1.1.1
	github.com/vmware/govmomi v0.22.2-0.20200329013745-f2eef8fc745f
	go.mongodb.org/mongo-driver v1.1.2 // indirect
	go.uber.org/multierr v1.6.0 // indirect
	go.uber.org/zap v1.16.0
	golang.org/x/net v0.0.0-20210520170846-37e1c6afe023 // indirect
	golang.org/x/oauth2 v0.0.0-20210402161424-2e8d93401602
	golang.org/x/time v0.0.0-20201208040808-7e3f01d25324 // indirect
	google.golang.org/api v0.44.0
	gopkg.in/check.v1 v1.0.0-20200902074654-038fdea0a05b
	gopkg.in/tomb.v2 v2.0.0-20161208151619-d5d1b5820637

	//pinned k8s.io to v0.20.1 tag
	k8s.io/api v0.20.1
	k8s.io/apiextensions-apiserver v0.20.1
	k8s.io/apimachinery v0.20.1
	k8s.io/client-go v0.20.1
)
